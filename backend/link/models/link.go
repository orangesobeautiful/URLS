package models

import (
	"URLS/internal/common"
	"URLS/internal/utils/bsonext"
	linkPB "URLS/proto/gen/go/link/v1"
	"context"
	"net/url"
	"sort"
	"time"

	"github.com/qiniu/qmgo"
	"github.com/qiniu/qmgo/field"
	"github.com/qiniu/qmgo/options"
	"github.com/speps/go-hashids/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	officialOpts "go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const linkCollName string = "links" + collSuffix

var linkColl *qmgo.Collection

var shortLinkHash *hashids.HashID

func initLinkCollIndex(ctx context.Context) (err error) {
	uniqueOpts := officialOpts.Index()
	uniqueOpts.SetUnique(true)

	typeOpts := officialOpts.Index()
	typeOpts.SetPartialFilterExpression(bson.M{"type": bson.M{"$gt": LTDirect}})

	deletedOpts := officialOpts.Index()
	deletedOpts.SetPartialFilterExpression(bson.M{"deleted": bson.M{"$eq": false}})

	tagsOpts := officialOpts.Index()
	tagsOpts.SetPartialFilterExpression(bson.M{"tags": bson.M{"$exists": true}})

	err = linkColl.CreateIndexes(ctx, []options.IndexModel{
		{Key: []string{"type"}, IndexOptions: typeOpts},
		{Key: []string{"deleted"}, IndexOptions: deletedOpts},
		{Key: []string{"short", "host"}, IndexOptions: uniqueOpts},
		{Key: []string{"creator"}},
		{Key: []string{"tags"}, IndexOptions: tagsOpts},
		{Key: []string{"totalclicks"}},
	})
	if err != nil {
		return
	}

	// init short link hashID

	hashSlat, err := hashIDSlatGet(ctx)
	if err != nil {
		return
	}

	hd := hashids.NewData()
	hd.Salt = hashSlat
	hd.MinLength = 5
	shortLinkHash, err = hashids.NewWithData(hd)
	if err != nil {
		return
	}

	return nil
}

// LinkType 短網址的類型
type LinkType int32

const (
	_        LinkType = iota
	LTDirect          // 直接導向
)

func LinkTypeFromInteger[T constraints.Integer](i T) (LinkType, bool) {
	conv := LinkType(i)
	switch conv {
	case LTDirect:
		return conv, true
	default:
		return 0, false
	}
}

type UTMInfo struct {
	Source   string
	Medium   string
	Campaign string
	Term     string
	Content  string
}

func UTMInfoFromPB(info *linkPB.UTMInfo) *UTMInfo {
	return &UTMInfo{
		Source:   info.GetSource(),
		Medium:   info.GetMedium(),
		Campaign: info.GetCampaign(),
		Term:     info.GetTerm(),
		Content:  info.GetContent(),
	}
}

func (u *UTMInfo) ConvertToMap() (res map[string]string) {
	const utmColumNum = 5
	res = make(map[string]string, utmColumNum)
	if u.Source != "" {
		res["utm_source"] = u.Source
	}
	if u.Medium != "" {
		res["utm_medium"] = u.Medium
	}
	if u.Campaign != "" {
		res["utm_campaign"] = u.Campaign
	}
	if u.Term != "" {
		res["utm_term"] = u.Term
	}
	if u.Content != "" {
		res["utm_content"] = u.Content
	}

	return
}

// LinkInfo 網址資訊
type LinkInfo struct {
	field.DefaultField `bson:",inline"`

	Type     LinkType           `bson:"type"`             // 短網址的類型
	Deleted  bool               `bson:"deleted"`          // 是否已被刪除
	Short    string             `bson:"short"`            // 縮短後的網址
	Host     string             `bson:"host"`             // 短網址的 host，預設為空
	Dest     string             `bson:"dest"`             // 要導向的網址
	IsCustom bool               `bson:"iscustom"`         // 是不是客製化的短網址
	Querys   map[string]string  `bson:"querys,omitempty"` // 自定義參數
	Creator  primitive.ObjectID `bson:"creator"`          // 建立者

	Note string   `bson:"note"`           // 備註訊息
	Tags []string `bson:"tags,omitempty"` // 標籤

	TotalClicks   uint64            `bson:"totalclicks"`             // 總點擊次數
	CountryClicks map[string]uint64 `bson:"countryclicks,omitempty"` // 國家來源 map[ISO3166]count
	OSClicks      map[string]uint64 `bson:"osclicks,omitempty"`      // 作業系統來源
	DeviceClicks  map[string]uint64 `bson:"deviceclicks,omitempty"`  // 裝置來源 map[(pc、tablet、phone ...)]count
	BrowserClicks map[string]uint64 `bson:"browserclicks,omitempty"` // 瀏覽器來源

	DeleteAt time.Time `bson:"deleteAt,omitempty"` // 被刪除的時間
}

// FullDest 回傳包含 query 的目的地網址
func (l *LinkInfo) FullDest() string {
	u, _ := url.Parse(l.Dest)

	if len(l.Querys) > 0 {
		query := u.Query()
		keyList := maps.Keys(l.Querys)
		sort.Strings(keyList)

		for _, key := range keyList {
			query.Add(key, l.Querys[key])
		}
		u.RawQuery = query.Encode()
	}

	return u.String()
}

// LinkCreate 根據指定資料建立短網址到資料庫
func LinkCreate(ctx context.Context, custom, host, dest string,
	utmInfo *UTMInfo, creator primitive.ObjectID, note string, tags []string) (
	*LinkInfo, error) {
	var isCustom bool
	var short string
	var err error
	if custom == "" {
		var encVal []int64
		encVal, err = LinkCounterNext(ctx)
		if err != nil {
			return nil, err
		}
		short, err = shortLinkHash.EncodeInt64(encVal)
		if err != nil {
			logger.Error("shortLinkHash.EncodeInt64 faeild", zap.Int64s("enc value", encVal), zap.Error(err))
			err = common.GRPCErrInternal
			return nil, err
		}
	} else {
		short = custom
		isCustom = true
	}

	newLink := LinkInfo{
		Type:     LTDirect,
		IsCustom: isCustom,
		Host:     host,
		Short:    short,
		Dest:     dest,
		Creator:  creator,
		Querys:   utmInfo.ConvertToMap(),
		Note:     note,
		Tags:     tags,
	}
	_, err = linkColl.InsertOne(ctx, &newLink)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			// 短網址(short)已有相同的
			err = status.Error(codes.AlreadyExists, "this link already exists")
			return nil, err
		}
		logger.Error("new link insert to db faeild", zap.Error(err))
		err = common.GRPCErrInternal
		return nil, err
	}

	return &newLink, nil
}

// LinkFindByID 根據 id 尋找 link
func LinkFindByID(ctx context.Context, id primitive.ObjectID) (link *LinkInfo, exist bool, err error) {
	link = new(LinkInfo)
	err = linkColl.Find(ctx, bsonext.ID(id)).One(&link)
	if err != nil {
		link = nil
		if qmgo.IsErrNoDocuments(err) {
			exist = false
			err = nil
			return
		}
		logger.Error("find link by id failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	exist = true
	return
}

// LinkListCount 回傳根據條件會搜尋到的資料總數
func LinkListCount(ctx context.Context,
	allUser bool, userID primitive.ObjectID,
	tags []string) (totalNum int64, err error) {
	query := bson.M{
		"deleted": false,
	}
	if len(tags) > 0 {
		query["tags"] = bsonext.In(tags)
	}
	if !allUser {
		query["creator"] = userID
	}

	totalNum, err = linkColl.Find(ctx, query).Count()
	if err != nil {
		logger.Error("get list link count failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	return
}

// LinkList 根據條件回傳 link 的資料
func LinkList(ctx context.Context,
	allUser bool, userID primitive.ObjectID,
	sortBy string, reverse bool, tags []string,
	skip int64, limit int64) (linkList []*LinkInfo, err error) {
	query := bson.M{
		"deleted": false,
	}
	if !allUser {
		query["creator"] = userID
	}
	if len(tags) > 0 {
		query["tags"] = bsonext.In(tags)
	}
	if reverse {
		sortBy = "-" + sortBy
	}

	err = linkColl.Find(ctx, query).Sort(sortBy).Skip(skip).Limit(limit).All(&linkList)
	if err != nil {
		logger.Error("list link failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	return
}

func LinkClicksUpdate(ctx context.Context, short, host string, total uint64,
	country map[string]uint64,
	os map[string]uint64,
	device map[string]uint64,
	browser map[string]uint64) (err error) {
	var incList []bsonext.IncInfo
	for k, v := range country {
		incList = append(incList, bsonext.IncInfo{FieldName: "countryclicks." + k, Val: int64(v)})
	}
	for k, v := range os {
		incList = append(incList, bsonext.IncInfo{FieldName: "osclicks." + k, Val: int64(v)})
	}
	for k, v := range device {
		incList = append(incList, bsonext.IncInfo{FieldName: "deviceclicks." + k, Val: int64(v)})
	}
	for k, v := range browser {
		incList = append(incList, bsonext.IncInfo{FieldName: "browserclicks." + k, Val: int64(v)})
	}
	incList = append(incList, bsonext.IncInfo{FieldName: "totalclicks", Val: int64(total)})

	err = linkColl.UpdateOne(ctx, bson.M{"short": short, "host": host}, bsonext.Inc(incList))
	if err != nil {
		logger.Error("update link click failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	return
}

type LinkPatchInfo struct {
	PNote bool
	Note  string
	PTags bool
	Tags  []string
}

func (l *LinkInfo) Patch(ctx context.Context, pInfo *LinkPatchInfo) (err error) {
	updateCol := bson.M{}
	if pInfo.PNote {
		updateCol["note"] = pInfo.Note
	}
	if pInfo.PTags {
		updateCol["tags"] = pInfo.Tags
	}

	err = linkColl.UpdateOne(ctx, bsonext.ID(l.Id),
		bsonext.Set(updateCol))
	if err != nil {
		logger.Error("patch link failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	return
}

func (l *LinkInfo) Delete(ctx context.Context) (err error) {
	err = linkColl.UpdateOne(ctx, bsonext.ID(l.Id),
		bsonext.Set(bson.M{"deleted": true, "deleteAt": time.Now()}))
	if err != nil {
		logger.Error("delete link failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	return
}

// SetNoDelete 設為未被刪除
func (l *LinkInfo) SetNoDelete(ctx context.Context) (err error) {
	err = linkColl.UpdateOne(ctx, bsonext.ID(l.Id),
		bson.M{
			"$set":   bson.M{"deleted": false},
			"$unset": bson.M{"deleteAt": ""},
		})
	if err != nil {
		logger.Error("delete link failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	return
}

// TagsAggreByUser 回傳指定 user 的所有 tags
func TagsAggreByUser(ctx context.Context, userID primitive.ObjectID) (tags []string, err error) {
	type projectRes struct {
		Tags []string `bson:"tags"`
	}

	var res projectRes
	err = linkColl.Aggregate(ctx,
		[]bson.M{
			bsonext.Match(bson.M{"creator": userID, "deleted": false}),
			bsonext.Group(bson.M{"_id": nil, "tags": bsonext.Push("$tags")}),
			bsonext.Project(bson.M{"_id": false, "tags": bsonext.Reduce("$tags", []string{}, bsonext.SetUnion([]string{"$$value", "$$this"}))}),
		}).One(&res)
	if err != nil {
		if qmgo.IsErrNoDocuments(err) {
			err = nil
			return
		}
		logger.Error("get user tags failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	tags = res.Tags
	return
}
