package models

import (
	"URLS/internal/utils/bsonext"
	"URLS/internal/utils/bytesext"
	"context"
	"encoding/base64"
	"math"
	"strconv"

	"github.com/qiniu/qmgo"
	"github.com/qiniu/qmgo/options"
	officialOpts "go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

const otherCollName string = "other" + collSuffix

const (
	linkCounterName   string = "linkcounter"
	hashIDSlatName    string = "linkhashidslat"
	formatVersionName string = "formatVersion"
)

// curDataVersion 用來檢測資料庫格式是否需要更新
const curDataVersion = 1

// linkCounterCarryBase counter 需要進位的標準
const linkCounterCarryBase = math.MaxInt32 / 2

var otherColl *qmgo.Collection

func initOtherCollIndex(ctx context.Context) (err error) {
	uniqueOpts := officialOpts.Index()
	uniqueOpts.SetUnique(true)

	err = otherColl.CreateIndexes(ctx, []options.IndexModel{
		{Key: []string{"name"}, IndexOptions: uniqueOpts},
	})
	if err != nil {
		return
	}

	var initFuncList = []func(context.Context) error{
		linkCounterInit,
		hashIDSlatInit,
		dataVersionInit,
	}
	for _, f := range initFuncList {
		err = f(ctx)
		if err != nil {
			return
		}
	}

	return
}

func linkCounterInit(ctx context.Context) (err error) {
	lcNum, err := otherColl.Find(ctx, bsonext.Name(linkCounterName)).Count()
	if err != nil {
		return
	}
	if lcNum == 0 {
		err = linkCounterCreate(ctx)
		if err != nil {
			return
		}
	}
	return nil
}

func hashIDSlatInit(ctx context.Context) (err error) {
	num, err := otherColl.Find(ctx, bsonext.Name(hashIDSlatName)).Count()
	if err != nil {
		return
	}
	if num == 0 {
		err = hashIDSlatCreate(ctx)
		if err != nil {
			return
		}
	}
	return nil
}

func dataVersionInit(ctx context.Context) (err error) {
	num, err := otherColl.Find(ctx, bsonext.Name(formatVersionName)).Count()
	if err != nil {
		return
	}
	if num == 0 {
		err = dataVersionCreate(ctx)
		if err != nil {
			return
		}
	}
	return nil
}

// linkCounter 作為自動生成短網址的計數器
type linkCounter struct {
	Name  string  `bson:"name"`
	Value []int64 `bson:"value"`
}

func linkCounterCreate(ctx context.Context) (err error) {
	lc := &linkCounter{
		Name:  linkCounterName,
		Value: []int64{0},
	}
	_, err = otherColl.InsertOne(ctx, lc)
	return
}

// LinkCounterNext 回傳下一個 link counter 的值
func LinkCounterNext(ctx context.Context) (val []int64, err error) {
	res := linkCounter{}
	err = otherColl.Find(ctx, bsonext.Name(linkCounterName)).
		Apply(qmgo.Change{
			Update:    bsonext.Inc([]bsonext.IncInfo{{FieldName: "value.0", Val: 1}}),
			ReturnNew: true,
		}, &res)
	if err != nil {
		logger.Error("inc link counter failed", zap.Error(err))
		return
	}
	val = res.Value

	// 檢查 counter 是否到達進位標準
	// 如果到達時需要將 counter 做進位處理
	needUpdate := false
	valLen := len(res.Value)
	diffList := make([]int64, valLen+1)
	for i := 0; i < valLen; i++ {
		if res.Value[i]+diffList[i] > linkCounterCarryBase {
			needUpdate = true
			diffList[i] = res.Value[i] * -1
			diffList[i+1]++
		}
	}
	if needUpdate {
		carryList := make([]bsonext.IncInfo, valLen+1)
		for i := 0; i < valLen+1; i++ {
			if diffList[i] != 0 {
				carryList[i].FieldName = "value." + strconv.Itoa(i)
				carryList[i].Val = diffList[i]
			}
		}
		err = otherColl.UpdateOne(ctx, bsonext.Name(linkCounterName), bsonext.Inc(carryList))
		if err != nil {
			logger.Error("link counter deal failed", zap.Error(err))
		}
	}

	return val, nil
}

// settingInfo
type settingInfo struct {
	Name  string `bson:"name"`
	Value string `bson:"value"`
}

func settingCreate(ctx context.Context, name, val string) (err error) {
	sInfo := &settingInfo{
		Name:  name,
		Value: val,
	}
	_, err = otherColl.InsertOne(ctx, sInfo)
	return
}

func settingFindByName(ctx context.Context, name string) (val string, err error) {
	res := settingInfo{}
	err = otherColl.Find(ctx, bsonext.Name(name)).One(&res)
	if err != nil {
		return
	}

	val = res.Value
	return
}

func hashIDSlatCreate(ctx context.Context) (err error) {
	const randSlatLen = 32
	randBs, err := bytesext.Rand(randSlatLen)
	if err != nil {
		return
	}

	return settingCreate(ctx, hashIDSlatName, base64.StdEncoding.EncodeToString(randBs))
}

func hashIDSlatGet(ctx context.Context) (slat string, err error) {
	return settingFindByName(ctx, hashIDSlatName)
}

func dataVersionCreate(ctx context.Context) (err error) {
	return settingCreate(ctx, formatVersionName, strconv.Itoa(curDataVersion))
}
