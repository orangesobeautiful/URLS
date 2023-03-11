package controllers

import (
	"context"
	"fmt"

	"URLS/internal/common"
	"URLS/link/configs"
	"URLS/link/models"
	linkPB "URLS/proto/gen/go/link/v1"
	rdModels "URLS/redirector/models"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

type LinkController struct {
	*common.BaseController
	linkPB.UnimplementedLinkServiceServer

	cfg     *configs.LSCfgInfo
	redisDB *redis.Client
}

func NewLinkController(cfgInfo *configs.LSCfgInfo, logger *zap.Logger) (uc *LinkController, err error) {
	bc, err := common.NewBaseController(&cfgInfo.BaseCfgInfo, logger)
	if err != nil {
		return
	}

	err = bc.SrvcConn.GenUserConn(cfgInfo.SrvcAddrMap.User.GRPC)
	if err != nil {
		return
	}

	// init redis connection

	redisOpts, err := cfgInfo.GenRedisOptions()
	if err != nil {
		return
	}
	redisOpts.DB = rdModels.RedisIndex
	rClient := redis.NewClient(redisOpts)

	bgCtx := context.Background()
	_, err = rClient.Ping(bgCtx).Result()
	if err != nil {
		logger.Error("redis client ping failed", zap.Error(err))
		return
	}

	// init models

	err = models.InitModels(bgCtx, bc.MgoDB, bc.Logger)
	if err != nil {
		err = fmt.Errorf("InitIndex failed, err=%s", err)
		return
	}
	rdModels.InitModels(rClient, logger)

	uc = &LinkController{
		BaseController: bc,
		cfg:            cfgInfo,
		redisDB:        rClient,
	}

	return uc, nil
}

// Ping 用來測試服務是否依然在線
//
// GET /ping
func (lc *LinkController) Ping(ctx context.Context, req *emptypb.Empty) (resp *linkPB.PingResponse, err error) {
	resp = &linkPB.PingResponse{Msg: "link ping ok"}

	return
}
