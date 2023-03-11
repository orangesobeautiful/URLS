package controllers

import (
	"context"
	"fmt"
	"time"

	"URLS/internal/common"
	userPB "URLS/proto/gen/go/user/v1"
	"URLS/user/configs"
	"URLS/user/models"
	"URLS/user/pkg/sessions"

	"github.com/gorilla/securecookie"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const AuthRedisIdx = 1

type UserController struct {
	*common.BaseController
	userPB.UnimplementedUserServiceServer

	cfg         *configs.USCfgInfo
	redisDB     *redis.Client
	cookieCodes []securecookie.Codec
}

func NewUserController(cfgInfo *configs.USCfgInfo, logger *zap.Logger) (uc *UserController, err error) {
	bc, err := common.NewBaseController(&cfgInfo.BaseCfgInfo, logger)
	if err != nil {
		return
	}

	// init redis connection

	redisOpts, err := cfgInfo.GenRedisOptions()
	if err != nil {
		return
	}
	redisOpts.DB = AuthRedisIdx

	rClient := redis.NewClient(redisOpts)
	bgCtx := context.Background()
	_, err = rClient.Ping(bgCtx).Result()
	if err != nil {
		return
	}

	err = models.InitModels(context.Background(), bc.MgoDB, bc.Logger)
	if err != nil {
		err = fmt.Errorf("InitIndex failed, err=%s", err)
		return
	}

	var cookieKeyPairs = make([][]byte, 0, len(cfgInfo.CookieKeyPairs))
	for _, keyStr := range cfgInfo.CookieKeyPairs {
		cookieKeyPairs = append(cookieKeyPairs, []byte(keyStr))
	}

	uc = &UserController{
		BaseController: bc,
		cfg:            cfgInfo,
		redisDB:        rClient,
		cookieCodes:    securecookie.CodecsFromPairs(cookieKeyPairs...),
	}
	uc.setNextQuotaResetTimer(0)

	return uc, nil
}

func (uc *UserController) setNextQuotaResetTimer(td time.Duration) {
	var nextDuration time.Duration
	if td > 0 {
		nextDuration = td
	} else {
		nowTime := time.Now()
		const updateDay = 1
		var nextUpdateTime time.Time
		if nowTime.Day() >= updateDay {
			nextUpdateTime = time.Date(nowTime.Year(), nowTime.Month()+1, updateDay, 0, 0, 0, 0, nowTime.Location())
		} else {
			nextUpdateTime = time.Date(nowTime.Year(), nowTime.Month(), updateDay, 0, 0, 0, 0, nowTime.Location())
		}
		nextDuration = nextUpdateTime.Sub(nowTime)
	}
	time.AfterFunc(nextDuration, uc.resetUserQuota)
}

func (uc *UserController) resetUserQuota() {
	uc.Logger.Info("reset user quota")
	bgCtx := context.Background()
	err := models.UserQuotaReset(bgCtx)
	if err != nil {
		// 失敗的話每10分鐘重試一次
		const retryMin = 10
		uc.setNextQuotaResetTimer(retryMin * time.Minute)
	}
}

// Ping 用來測試服務是否依然在線
//
// GET /ping
func (uc *UserController) Ping(ctx context.Context, req *emptypb.Empty) (resp *userPB.PingResponse, err error) {
	resp = &userPB.PingResponse{Msg: "user ping ok"}

	return
}

// getSessIDFromCtxCookie 從 context 的 cookies 中解析出 sessionID
func (uc *UserController) getSessIDFromCtxCookie(ctx context.Context) (sessionID string, err error) {
	cookies := common.CtxReadCookies(ctx, common.SigninCookieName)
	if len(cookies) == 0 || cookies[0].Value == "" {
		err = common.GRPCErrUnauthenticated
		return
	}

	sessionID = sessions.GetSessionIDFromCookie(ctx, uc.cookieCodes, cookies[0].Value)
	if sessionID == "" {
		err = common.GRPCErrUnauthenticated
		return
	}

	return
}

// getSessIDFromCtxCookie 從 context 的 cookies 中解析出 sessionID，並取得 session values
func (uc *UserController) getSessFromCtxCookie(ctx context.Context) (sessVals common.SigninSessValsInfo, err error) {
	sessionID, err := uc.getSessIDFromCtxCookie(ctx)
	if err != nil {
		return
	}

	sessVals, err = sessions.GetSessionValues(ctx, uc.redisDB, sessionID)
	if err != nil {
		uc.Logger.Error("sessions.GetSession failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	return
}

// initRequestUser 初始化發送請求的使用者，如果發生錯誤時會處理好對應的 log 訊息和回傳 gRPC 錯誤
func (uc *UserController) initRequestUser(ctx context.Context) (userInfo *models.UserInfo, err error) {
	sessVals, err := uc.getSessFromCtxCookie(ctx)
	if err != nil {
		return
	}

	if sessVals.UserIDHex == "" {
		err = common.GRPCErrUnauthenticated
		return
	}

	var exist bool
	objID, _ := primitive.ObjectIDFromHex(sessVals.UserIDHex)
	userInfo, exist, err = models.UserFindByID(ctx, objID)
	if err != nil {
		uc.Logger.Error("uc.FindUserByObjectID failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	if !exist {
		err = common.GRPCErrUnauthenticated
		return
	}

	return
}

// findUserByObjectID 根據 ID 查詢資料庫中的使用者資料，如果發生錯誤時會處理好對應的 log 訊息和回傳 gRPC 錯誤
func (uc *UserController) findUserByObjectIDWithNotFoundErr(
	ctx context.Context,
	userID primitive.ObjectID) (userInfo *models.UserInfo, err error) {
	var exist bool
	userInfo, exist, err = models.UserFindByID(ctx, userID)
	if err != nil {
		return
	} else if !exist {
		err = status.Error(codes.NotFound, "user was not found")
		return
	}

	return
}
