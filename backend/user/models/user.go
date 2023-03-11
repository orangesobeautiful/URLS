package models

import (
	"URLS/internal/common"
	"URLS/internal/utils/bsonext"
	"context"

	"github.com/qiniu/qmgo"
	"github.com/qiniu/qmgo/field"
	"github.com/qiniu/qmgo/options"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	officialOpts "go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const UserCollName string = "users" + collSuffix

var userColl *qmgo.Collection

func initUserCollIndex(ctx context.Context) (err error) {
	emailOpts := officialOpts.Index()
	emailOpts.SetUnique(true)

	err = userColl.CreateOneIndex(ctx,
		options.IndexModel{Key: []string{"email"}, IndexOptions: emailOpts})

	return
}

const (
	userNormalLinkQuota = 50
	userCustomLinkQuota = 10
)

// UserRole 使用者權限
type UserRole uint32

const (
	_         UserRole = iota
	UROwner            // 網站擁有者
	URManager          // 管理員
	URNormal           // 一般使用者
)

func UserRoleFromInteger[T constraints.Integer](i T) (UserRole, bool) {
	if int64(i) < int64(UROwner) || int64(i) > int64(URNormal) {
		return 0, false
	}
	return UserRole(i), true
}

// IsValid 是否為有效的 UserRole
func (ur UserRole) IsValid() bool {
	if ur > URNormal || ur < UROwner {
		return false
	}
	return true
}

// UserInfo 使用者資訊
type UserInfo struct {
	field.DefaultField `bson:",inline"`

	Email   string `bson:"email"`
	PHType  PHType `bson:"phtype"`  // PHType Hash 密碼的方式
	PHBytes []byte `bson:"phbytes"` // PHBytes Hash 過後的密碼

	Role UserRole `bson:"role"`

	NormalLinkQuota uint64 `bson:"normallinkquota"` // 一般短網址額度
	NormalLinkUsage uint64 `bson:"normallinkusage"` // 一般短網址使用量
	CustomLinkQuota uint64 `bson:"customlinkquota"` // 自訂短網址額度
	CustomLinkUsage uint64 `bson:"customlinkusage"` // 自訂短網址使用量

	LinkTags []string `bson:"linktags,omitempty"` // 使用者定義的短網址 tags
}

// IsManager 使用者權限是否至少為管理員等級
func (u *UserInfo) IsManager() bool {
	switch u.Role {
	case UROwner, URManager:
		return true
	}

	return false
}

// UserRegister 註冊使用者
func UserRegister(
	ctx context.Context, email, pwd, userAgent, remoteIP string, country string) (newUserID primitive.ObjectID, err error) {
	// 生成使用者資料

	phType, pwdHash, err := PwdHash(pwd)
	if err != nil {
		logger.Error("PwdHash failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	regInfo := RegisterInfo{
		RegisterIP:      remoteIP,
		RegisterCountry: country,
		RegisterUA:      userAgent,
	}
	_, err = regInfoColl.InsertOne(ctx, &regInfo)
	if err != nil {
		logger.Error("new register info insert to db faeild", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	newUser := UserInfo{
		DefaultField: field.DefaultField{
			Id:       regInfo.Id,
			CreateAt: regInfo.CreateAt,
			UpdateAt: regInfo.UpdateAt,
		},
		Email:           email,
		PHType:          phType,
		PHBytes:         pwdHash,
		Role:            URNormal,
		NormalLinkQuota: userNormalLinkQuota,
		CustomLinkQuota: userCustomLinkQuota,
	}
	_, err = userColl.InsertOne(ctx, &newUser)
	if err != nil {
		_ = regInfo.Delete(ctx)
		if mongo.IsDuplicateKeyError(err) {
			// email 已被註冊過
			err = status.Error(codes.AlreadyExists, "email has been registered")
		} else {
			logger.Error("new user insert to db faeild", zap.Error(err))
			err = common.GRPCErrInternal
		}
		return
	}

	return newUser.Id, nil
}

// UserFindByEmail 根據 email 尋找使用者
func UserFindByEmail(ctx context.Context, email string) (u *UserInfo, exist bool, err error) {
	u = new(UserInfo)
	err = userColl.Find(ctx, bson.M{"email": email}).One(&u)
	if err != nil {
		u = nil
		if qmgo.IsErrNoDocuments(err) {
			exist = false
			err = nil
			return
		}
		logger.Error("find user by email failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	exist = true
	return
}

// UserFindByID 根據 id 尋找使用者
//
// 已自動處理好內部錯誤情形
func UserFindByID(ctx context.Context, id primitive.ObjectID) (u *UserInfo, exist bool, err error) {
	u = new(UserInfo)
	err = userColl.Find(ctx, bsonext.ID(id)).One(&u)
	if err != nil {
		u = nil
		if qmgo.IsErrNoDocuments(err) {
			exist = false
			err = nil
			return
		}
		logger.Error("find user by id failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	exist = true
	return
}

// UserQuotaReset 將所有使用者的使用額度清0
func UserQuotaReset(ctx context.Context) (err error) {
	_, err = userColl.UpdateAll(ctx, bson.M{}, bsonext.Set(bson.M{"normallinkusage": 0, "customlinkusage": 0}))
	if err != nil {
		logger.Error("reset user quota failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	return
}

// UpdatePassword 更新密碼
//
// 已自動處理好內部錯誤情形
func (u *UserInfo) UpdatePassword(ctx context.Context, newPwd string) (err error) {
	newPHType, newPHByte, err := PwdHash(newPwd)
	if err != nil {
		logger.Error("models.PwdHash", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	err = userColl.UpdateOne(ctx,
		bsonext.ID(u.Id),
		bsonext.Set(bson.M{"phtype": newPHType, "phbytes": newPHByte}))
	if err != nil {
		logger.Error("mgo update user pwd failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	u.PHType = newPHType
	u.PHBytes = newPHByte

	return
}

// UpdatePassword 更新使用者權限
//
// 已自動處理好內部錯誤情形
func (u *UserInfo) UpdateRole(ctx context.Context, newRole UserRole) (err error) {
	err = userColl.UpdateOne(ctx, bsonext.ID(u.Id), bsonext.Set(bson.M{"role": newRole}))
	if err != nil {
		logger.Error("mgo update user role failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	return
}

type UserPatchInfo struct {
	PNormalLinkQuota    bool
	NormalLinkQuota     uint64
	PCustomLinkQuota    bool
	CustomLinkQuota     uint64
	NormalLinkUsageDiff int64
	CustomLinkUsageDiff int64
}

// Patch 更新使用者資料
func (u *UserInfo) Patch(ctx context.Context, pInfo *UserPatchInfo) (err error) {
	setCol := bson.M{}
	var incList bsonext.IncList
	if pInfo.PNormalLinkQuota {
		setCol["normallinkquota"] = pInfo.NormalLinkQuota
	}
	if pInfo.PCustomLinkQuota {
		setCol["customlinkquota"] = pInfo.CustomLinkQuota
	}
	if pInfo.NormalLinkUsageDiff != 0 {
		incList.ADD("normallinkusage", pInfo.NormalLinkUsageDiff)
	}
	if pInfo.CustomLinkUsageDiff != 0 {
		incList.ADD("customlinkusage", pInfo.CustomLinkUsageDiff)
	}
	// TODO: 優化 bsonext 無法直接處理多項 $ 的問題
	updateCol := bson.M{
		"$set": bsonext.Set(setCol)["$set"],
		"$inc": bsonext.Inc(incList)["$inc"],
	}

	err = userColl.UpdateOne(ctx,
		bsonext.ID(u.Id),
		updateCol)
	if err != nil {
		logger.Error("user patch failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	return nil
}

// Delete 刪除使用者
//
// 已自動處理好內部錯誤情形
func (u *UserInfo) Delete(ctx context.Context) (err error) {
	err = userColl.Remove(ctx, bsonext.ID(u.Id))
	if err != nil {
		logger.Error("delete user failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	toDelRegInfo := &RegisterInfo{}
	toDelRegInfo.Id = u.Id
	err = toDelRegInfo.Delete(ctx)
	if err != nil {
		logger.Error("delete register info failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	return
}
