package controllers

import (
	"context"

	"URLS/internal/common"
	userPB "URLS/proto/gen/go/user/v1"
	"URLS/user/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// Register 使用者註冊 API
func (uc *UserController) Register(ctx context.Context, req *userPB.RegisterRequest) (resp *userPB.RegisterResponse, err error) {
	resp = &userPB.RegisterResponse{}

	// 檢查資料格式
	if !models.PwdFormatCheck(req.GetPwd()) {
		err = status.Error(codes.InvalidArgument, "password format is invalid")
		return
	}

	// 註冊時的資訊

	var userAgent, remoteIP, country string
	if headers, ok := metadata.FromIncomingContext(ctx); ok {
		uaHeader := headers.Get(common.MetadataPrefix + "user-agent")
		if len(uaHeader) > 0 {
			userAgent = uaHeader[0]
		}

		gwIPHeader := headers.Get(common.MetadataPrefix + common.HderNameGWIPLower)
		if len(gwIPHeader) > 0 {
			remoteIP = gwIPHeader[0]
		}

		// TODO: 透過 IP 庫獲得國家資訊
		countryHearder := headers.Get(common.MetadataPrefix + common.HderNameGWCountryLower)
		if len(countryHearder) > 0 {
			country = countryHearder[0]
		}
	}

	newUserID, err := models.UserRegister(ctx, req.GetEmail(), req.GetPwd(), userAgent, remoteIP, country)
	if err != nil {
		return
	}

	resp.Id = newUserID.Hex()
	return resp, nil
}

func (uc *UserController) SelfInfoGet(ctx context.Context, req *emptypb.Empty) (resp *userPB.SelfInfoGetResponse, err error) {
	reqUser, err := uc.initRequestUser(ctx)
	if err != nil {
		return
	}

	resp = &userPB.SelfInfoGetResponse{
		UserInfo: &userPB.UserInfo{
			IdHex:       reqUser.Id.Hex(),
			Email:       reqUser.Email,
			Role:        uint32(reqUser.Role),
			IsManager:   reqUser.IsManager(),
			NormalQuota: reqUser.NormalLinkQuota,
			NormalUsage: reqUser.NormalLinkUsage,
			CustomQuota: reqUser.CustomLinkQuota,
			CustomUsage: reqUser.CustomLinkUsage,
		},
	}

	return resp, nil
}

// UserInfoGet 取得使用者資訊
func (uc *UserController) UserInfoGet(ctx context.Context, req *userPB.UserInfoGetRequest) (resp *userPB.UserInfoGetResponse, err error) {
	toGetUserID, err := primitive.ObjectIDFromHex(req.GetUserIdHex())
	if err != nil {
		err = status.Error(codes.InvalidArgument, "user id format is invalid")
		return
	}

	if err = uc.IsInternalCall(ctx); err != nil {
		var reqUser *models.UserInfo
		reqUser, err = uc.initRequestUser(ctx)
		if err != nil {
			return
		}
		if toGetUserID != reqUser.Id && reqUser.IsManager() {
			err = common.GRPCERRPermissionDenied
			return
		}
	}

	toGetUser, err := uc.findUserByObjectIDWithNotFoundErr(ctx, toGetUserID)
	if err != nil {
		return
	}

	resp = &userPB.UserInfoGetResponse{
		UserInfo: &userPB.UserInfo{
			IdHex:       toGetUserID.Hex(),
			Email:       toGetUser.Email,
			Role:        uint32(toGetUser.Role),
			IsManager:   toGetUser.IsManager(),
			NormalQuota: toGetUser.NormalLinkQuota,
			NormalUsage: toGetUser.NormalLinkUsage,
			CustomQuota: toGetUser.CustomLinkQuota,
			CustomUsage: toGetUser.CustomLinkUsage,
		},
	}

	return resp, nil
}

// UserDelete 刪除指定使用者
func (uc *UserController) UserDelete(ctx context.Context, req *userPB.UserDeleteRequest) (resp *userPB.UserDeleteResponse, err error) {
	resp = new(userPB.UserDeleteResponse)
	delUserID, err := primitive.ObjectIDFromHex(req.GetUserIdHex())
	if err != nil {
		err = status.Error(codes.InvalidArgument, "user id format is invalid")
		return
	}

	reqUser, err := uc.initRequestUser(ctx)
	if err != nil {
		return
	}

	if !reqUser.IsManager() {
		err = common.GRPCERRPermissionDenied
		return
	}

	if reqUser.Id == delUserID {
		err = status.Error(codes.FailedPrecondition, "delet self is not allow")
		return
	}

	delUser, err := uc.findUserByObjectIDWithNotFoundErr(ctx, delUserID)
	if err != nil {
		return
	}

	if delUser.IsManager() {
		err = status.Error(codes.FailedPrecondition, "delet manager is aborted")
		return
	}

	err = delUser.Delete(ctx)
	if err != nil {
		return
	}

	resp.Msg = "success delete"
	return resp, nil
}

// PwdChange 更改自己的密碼
func (uc *UserController) PwdChange(ctx context.Context, req *userPB.PwdChangeRequest) (resp *userPB.PwdChangeResponse, err error) {
	resp = new(userPB.PwdChangeResponse)
	if req.GetNewPwd() == req.GetOldPwd() {
		err = status.Error(codes.InvalidArgument, "old password is equal to new password")
		return
	} else if !models.PwdFormatCheck(req.GetNewPwd()) {
		err = status.Error(codes.InvalidArgument, "new password format is invalid")
		return
	}

	reqUser, err := uc.initRequestUser(ctx)
	if err != nil {
		return
	}

	if !models.IsPwdEqual(reqUser.PHType, reqUser.PHBytes, []byte(req.GetOldPwd())) {
		err = status.Error(codes.InvalidArgument, "old password is incorrect")
		return
	}

	// update user new password

	err = reqUser.UpdatePassword(ctx, req.GetNewPwd())
	if err != nil {
		return
	}

	resp.Msg = "update successed"
	return resp, nil
}

func (uc *UserController) LinkQuotaUpdate(ctx context.Context,
	req *userPB.LinkQuotaUpdateRequest) (resp *userPB.LinkQuotaUpdateResponse, err error) {
	toChangeUserID, err := primitive.ObjectIDFromHex(req.GetUserIdHex())
	if err != nil {
		err = status.Error(codes.InvalidArgument, "user id format is invalid")
		return
	}

	if err = uc.IsInternalCall(ctx); err != nil {
		return
	}

	toChangeUser, exist, err := models.UserFindByID(ctx, toChangeUserID)
	if err != nil {
		return
	} else if !exist {
		err = status.Error(codes.NotFound, "user was not found")
		return
	}

	err = toChangeUser.Patch(ctx, &models.UserPatchInfo{
		PNormalLinkQuota:    req.GetPatchNormalQuota(),
		NormalLinkQuota:     req.GetNormalQuota(),
		PCustomLinkQuota:    req.GetPatchCustomQuota(),
		CustomLinkQuota:     req.GetCustomQuota(),
		NormalLinkUsageDiff: req.GetNormalUsageDiff(),
		CustomLinkUsageDiff: req.GetCustomUsageDiff(),
	})
	if err != nil {
		return
	}

	resp = &userPB.LinkQuotaUpdateResponse{
		Msg: "update success",
	}
	return resp, nil
}

// RoelChange 變更使用者權限
func (uc *UserController) RoleChange(ctx context.Context, req *userPB.RoleChangeRequest) (resp *userPB.RoleChangeResponse, err error) {
	resp = new(userPB.RoleChangeResponse)
	newRole, convOK := models.UserRoleFromInteger(req.GetRole())
	if !convOK {
		err = status.Error(codes.InvalidArgument, "new role is not valid")
		return
	} else if newRole == models.UROwner {
		err = status.Error(codes.FailedPrecondition, "change role to owner is not allow")
		return
	}
	toChangeUserID, err := primitive.ObjectIDFromHex(req.GetUserIdHex())
	if err != nil {
		err = status.Error(codes.InvalidArgument, "user id format is invalid")
		return
	}

	reqUser, err := uc.initRequestUser(ctx)
	if err != nil {
		return
	}

	if reqUser.Role != models.UROwner {
		err = common.GRPCERRPermissionDenied
		return
	}

	if toChangeUserID == reqUser.Id {
		err = status.Error(codes.FailedPrecondition, "change self role is not allow")
		return
	}

	toChangeUser, exist, err := models.UserFindByID(ctx, toChangeUserID)
	if err != nil {
		return
	} else if !exist {
		err = status.Error(codes.NotFound, "user was not found")
		return
	}
	err = toChangeUser.UpdateRole(ctx, newRole)
	if err != nil {
		return
	}

	resp.Msg = "update role success"
	return resp, nil
}
