package controllers

import (
	"context"

	"URLS/internal/common"
	userPB "URLS/proto/gen/go/user/v1"
	"URLS/user/models"
	"URLS/user/pkg/sessions"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	cookiePath   string = "/"
	cookieSecure bool   = false
)

// Signin 登入
func (uc *UserController) Signin(ctx context.Context, req *userPB.SigninRequest) (resp *userPB.SigninResponse, err error) {
	var previousSessionID string

	cookies := common.CtxReadCookies(ctx, common.SigninCookieName)
	if len(cookies) != 0 && cookies[0].Value != "" {
		previousSessionID = sessions.GetSessionIDFromCookie(ctx, uc.cookieCodes, cookies[0].Value)
	}

	// 驗證帳號密碼是否正確

	user, exist, err := models.UserFindByEmail(ctx, req.GetEmail())
	if err != nil {
		return
	} else if !exist {
		err = status.Error(codes.Unauthenticated, "account or password was incorrect")
		return
	}

	// 比較密碼

	if !models.IsPwdEqual(user.PHType, user.PHBytes, []byte(req.GetPwd())) {
		err = status.Error(codes.Unauthenticated, "account or password was incorrect")
		return
	}

	authCookie, err := sessions.GenerateSession(ctx, uc.redisDB, uc.cfg.AuthCookie, uc.cookieCodes, user.Id.Hex())
	if err != nil {
		uc.Logger.Error("sessions.GenerateSession failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}
	authCookie.Path = cookiePath
	authCookie.Secure = cookieSecure

	// 如果原先有登入 => 刪除之前登入的 session

	if previousSessionID != "" {
		_, err = sessions.DelSession(ctx, uc.redisDB, previousSessionID)
		if err != nil {
			uc.Logger.Error("sessions.DelSession failed", zap.Error(err))
			err = common.GRPCErrInternal
			return
		}
	}

	if err = common.CtxSetCookie(ctx, authCookie); err != nil {
		uc.Logger.Error("common.CtxSetCookie", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	resp = &userPB.SigninResponse{Id: user.Id.Hex()}

	return resp, nil
}

func (uc *UserController) GetAuthInfo(ctx context.Context, req *userPB.GetAuthInfoRequest) (
	resp *userPB.GetAuthInfoResponse, err error) {
	if err = uc.IsInternalCall(ctx); err != nil {
		return
	}

	sessionID := sessions.GetSessionIDFromCookie(ctx, uc.cookieCodes, req.GetSigninCookieValue())
	if sessionID == "" {
		err = common.GRPCErrUnauthenticated
		return
	}

	sessionValues, err := sessions.GetSessionValues(ctx, uc.redisDB, sessionID)
	if err != nil {
		uc.Logger.Error("sessions.GetSession failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}
	if sessionValues.UserIDHex == "" {
		err = common.GRPCErrUnauthenticated
		return
	}

	return &userPB.GetAuthInfoResponse{UserIdHex: sessionValues.UserIDHex}, nil
}

// Logout 登出
func (uc *UserController) Logout(ctx context.Context, req *emptypb.Empty) (resp *userPB.LogoutResponse, err error) {
	sessionID, err := uc.getSessIDFromCtxCookie(ctx)
	if err != nil {
		return
	}

	toDelCookie, err := sessions.DelSession(ctx, uc.redisDB, sessionID)
	if err != nil {
		uc.Logger.Error("sessions.DelSession failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}
	toDelCookie.Path = cookiePath
	toDelCookie.Secure = cookieSecure

	if err = common.CtxSetCookie(ctx, toDelCookie); err != nil {
		uc.Logger.Error("common.CtxSetCookie", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	return &userPB.LogoutResponse{Msg: "log out success"}, nil
}
