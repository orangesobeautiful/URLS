package models

import (
	"context"

	"github.com/qiniu/qmgo"
	"go.uber.org/zap"
)

const collSuffix string = "-link"

var logger *zap.Logger
var mgoDB *qmgo.Database

func InitModels(ctx context.Context, db *qmgo.Database, inLogger *zap.Logger) (err error) {
	mgoDB = db
	logger = inLogger

	otherColl = mgoDB.Collection(otherCollName)
	linkColl = mgoDB.Collection(linkCollName)

	err = initIndex(ctx)
	return
}

// initIndex 初始化所有 collection 的 index
func initIndex(ctx context.Context) (err error) {
	var initFuncList = []func(context.Context) error{
		initOtherCollIndex,
		initLinkCollIndex,
	}

	for _, f := range initFuncList {
		err = f(ctx)
		if err != nil {
			return
		}
	}

	return
}
