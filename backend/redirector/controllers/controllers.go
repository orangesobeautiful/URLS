package controllers

import (
	"context"
	"fmt"

	"URLS/internal/common"
	linkModels "URLS/link/models"
	"URLS/redirector/configs"
	"URLS/redirector/models"

	"github.com/redis/go-redis/v9"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type RedirectorController struct {
	*common.BaseController
	cfg *configs.RDSCfgInfo

	handler fasthttp.RequestHandler

	redisDB *redis.Client
}

const RedirectorRedisIdx = 2

func NewRDController(cfgInfo *configs.RDSCfgInfo, logger *zap.Logger) (ctrl *RedirectorController, err error) {
	bc, err := common.NewBaseController(&cfgInfo.BaseCfgInfo, logger)
	if err != nil {
		logger.Error("common.NewBaseController failed", zap.Error(err))
		return
	}

	// init redis connection

	redisOpts, err := cfgInfo.GenRedisOptions()
	if err != nil {
		return
	}
	redisOpts.DB = RedirectorRedisIdx
	rClient := redis.NewClient(redisOpts)

	bgCtx := context.Background()
	_, err = rClient.Ping(bgCtx).Result()
	if err != nil {
		logger.Error("redis client ping failed", zap.Error(err))
		return
	}

	// init models
	err = linkModels.InitModels(bgCtx, bc.MgoDB, logger)
	if err != nil {
		err = fmt.Errorf("InitIndex failed, err=%s", err)
		return
	}
	models.InitModels(rClient, logger)

	ctrl = &RedirectorController{
		BaseController: bc,
		cfg:            cfgInfo,
		redisDB:        rClient,
	}
	ctrl.handler = ctrl.redirectorHandler

	return ctrl, nil
}

func (rd *RedirectorController) GetRestHandler() func(ctx *fasthttp.RequestCtx) {
	return rd.handler
}
