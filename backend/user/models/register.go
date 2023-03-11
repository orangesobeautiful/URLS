package models

import (
	"URLS/internal/common"
	"URLS/internal/utils/bsonext"
	"context"

	"github.com/qiniu/qmgo"
	"github.com/qiniu/qmgo/field"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

const RegInfoCollName string = "register_info" + collSuffix

var regInfoColl *qmgo.Collection

// RegisterInfo 使用者註冊時的資訊
type RegisterInfo struct {
	field.DefaultField `bson:",inline"`

	RegisterIP      string `bson:"registerip"`      // 註冊時的 IP
	RegisterUA      string `bson:"registerua"`      // 註冊時的 User Agent
	RegisterCountry string `bson:"registercountry"` // 註冊時的國家
}

func RegisterInfoFindByID(ctx context.Context, id primitive.ObjectID) (r *RegisterInfo, exist bool, err error) {
	r = new(RegisterInfo)
	err = regInfoColl.Find(ctx, bsonext.ID(id)).One(&r)
	if err != nil {
		r = nil
		if qmgo.IsErrNoDocuments(err) {
			exist = false
			err = nil
			return
		}
		logger.Error("find register info by id failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	exist = true
	return
}
func (r *RegisterInfo) Delete(ctx context.Context) (err error) {
	return regInfoColl.Remove(ctx, bsonext.ID(r.Id))
}
