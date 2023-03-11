package controllers

import (
	"context"
	"net/url"
	"strconv"
	"unicode"

	"URLS/internal/common"
	"URLS/link/models"
	linkPB "URLS/proto/gen/go/link/v1"
	userPB "URLS/proto/gen/go/user/v1"
	rdModels "URLS/redirector/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const utmMaxLen = 100
const noteMaxLen = 100
const tagsMaxLen = 15
const tagStrMaxLen = 15

const pageSizeMax = 30

// tagsArgumentCheck 檢查 tags 參數是否有效
func tagsArgumentCheck(tagList []string) (err error) {
	if len(tagList) > tagsMaxLen {
		err = status.Error(codes.InvalidArgument, "length of tag is greater than "+strconv.Itoa(tagsMaxLen))
		return
	}
	for _, tag := range tagList {
		if len(tag) > tagStrMaxLen {
			err = status.Error(codes.InvalidArgument, "one length of tag is greater than "+strconv.Itoa(tagStrMaxLen))
			return
		}
		if tag == "" {
			err = status.Error(codes.InvalidArgument, "tag can not be empty string")
			return
		}
	}

	return
}

func (lc *LinkController) LinkCreate(ctx context.Context, req *linkPB.LinkCreateRequest) (resp *linkPB.LinkCreateResponse, err error) {
	// 請求資料檢查

	_, err = url.ParseRequestURI(req.GetDest())
	if err != nil {
		err = status.Error(codes.InvalidArgument, "destination link is not a valid url")
		return
	}
	custom := req.GetCustom()
	if custom != "" {
		for _, runeValue := range custom {
			if !unicode.IsLetter(runeValue) && !unicode.IsNumber(runeValue) &&
				runeValue != rune('-') && runeValue != rune('_') &&
				!unicode.Is(unicode.Han, runeValue) {
				err = status.Error(codes.InvalidArgument, "custom link contains invalid characters")
				return
			}
		}
	}
	if len(req.GetUtmInfo().Source) > utmMaxLen ||
		len(req.GetUtmInfo().Medium) > utmMaxLen ||
		len(req.GetUtmInfo().Campaign) > utmMaxLen ||
		len(req.GetUtmInfo().Term) > utmMaxLen ||
		len(req.GetUtmInfo().Content) > utmMaxLen {
		err = status.Error(codes.InvalidArgument, "length of utm is greater than "+strconv.Itoa(utmMaxLen))
		return
	}

	if len(req.GetNote()) > noteMaxLen {
		err = status.Error(codes.InvalidArgument, "length of note is greater than "+strconv.Itoa(noteMaxLen))
		return
	}
	if err = tagsArgumentCheck(req.GetTags()); err != nil {
		return
	}

	// 使用者身分驗證與剩餘額度確認

	userInfo, err := lc.UserRequestGet(ctx)
	if err != nil {
		return
	}
	if userInfo.NormalLinkUsage >= userInfo.NormalLinkQuota ||
		custom != "" && userInfo.CustomLinkUsage >= userInfo.CustomLinkQuota {
		err = status.Error(codes.ResourceExhausted, "your quota was exceeded")
		return
	}

	if err != nil {
		lc.Logger.Error("lc.SrvcConn.User.LinkTagsAdd failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	// 資料庫添加資料

	newLink, err := models.LinkCreate(ctx,
		custom, "", req.GetDest(), models.UTMInfoFromPB(req.GetUtmInfo()), userInfo.ID, req.GetNote(), req.GetTags())
	if err != nil {
		return
	}

	err = rdModels.LinkAdd(ctx, newLink)
	if err != nil {
		_ = newLink.Delete(ctx)
		return
	}

	// 更新使用者的使用額度

	quotaUpdateInfo := &userPB.LinkQuotaUpdateRequest{
		UserIdHex:       userInfo.IDHex,
		NormalUsageDiff: 1,
	}
	if custom != "" {
		quotaUpdateInfo.CustomUsageDiff = 1
	}

	_, err = lc.SrvcConn.User.LinkQuotaUpdate(ctx, quotaUpdateInfo)
	if err != nil {
		lc.Logger.Error("User.LinkQuotaUpdate failed", zap.Error(err))
		err = common.GRPCErrInternal
		return
	}

	resp = &linkPB.LinkCreateResponse{
		Msg: "success",
	}
	return resp, nil
}

func (lc *LinkController) mLinkInfoToPBLinkInfo(mLink *models.LinkInfo) *linkPB.LinkInfo {
	var short string
	if mLink.Host == "" {
		short = lc.cfg.RDDomain + "/" + mLink.Short
	} else {
		short = mLink.Host + "/" + mLink.Short
	}
	return &linkPB.LinkInfo{
		IdHex:        mLink.Id.Hex(),
		Type:         int32(mLink.Type),
		Short:        short,
		Host:         mLink.Host,
		FullDest:     mLink.FullDest(),
		IsCustom:     mLink.IsCustom,
		CreatorIdHex: mLink.Creator.Hex(),
		Note:         mLink.Note,
		Tags:         mLink.Tags,

		TotalClicks:   mLink.TotalClicks,
		CountryClicks: mLink.CountryClicks,
		OsClicks:      mLink.OSClicks,
		DeviceClicks:  mLink.DeviceClicks,
		BrowserClicks: mLink.BrowserClicks,

		CreateAt: timestamppb.New(mLink.CreateAt),
	}
}

func (lc *LinkController) LinkList(ctx context.Context, req *linkPB.LinkListRequest) (resp *linkPB.LinkListResponse, err error) {
	// 請求資料檢查

	var toListUserID primitive.ObjectID
	if !req.GetAllUser() {
		toListUserID, err = primitive.ObjectIDFromHex(req.GetUserIdHex())
		if err != nil {
			err = status.Error(codes.InvalidArgument, "user id format is invalid")
			return
		}
	}
	switch req.GetSortBy() {
	case "totalclicks", "createAt":
	default:
		err = status.Error(codes.InvalidArgument, "sort_by is invalid")
		return
	}
	if err = tagsArgumentCheck(req.GetTags()); err != nil {
		return
	}
	if req.GetPage() == 0 {
		err = status.Error(codes.InvalidArgument, "page needs to be a value greater than 0")
		return
	}
	if req.GetPageSize() == 0 {
		err = status.Error(codes.InvalidArgument, "pagesize needs to be a value greater than 0")
		return
	}
	if req.GetPageSize() > pageSizeMax {
		err = status.Error(codes.InvalidArgument, "the maximum upper limit for pagesize is 30")
		return
	}

	// 權限檢查

	userInfo, err := lc.UserRequestGet(ctx)
	if err != nil {
		return
	}
	if req.GetAllUser() {
		if !userInfo.IsManager {
			err = common.GRPCERRPermissionDenied
			return
		}
	} else {
		if toListUserID != userInfo.ID {
			err = common.GRPCERRPermissionDenied
			return
		}
	}

	skip := int64((req.GetPage() - 1) * req.GetPageSize())
	linkList, err := models.LinkList(ctx,
		req.GetAllUser(), toListUserID,
		req.GetSortBy(), req.GetReverse(), req.GetTags(),
		skip, int64(req.GetPageSize()))
	if err != nil {
		return
	}

	var pbLinkList = make([]*linkPB.LinkInfo, 0, len(linkList))
	for _, mLink := range linkList {
		pbLinkList = append(pbLinkList, lc.mLinkInfoToPBLinkInfo(mLink))
	}

	resp = &linkPB.LinkListResponse{
		LinkInfoList: pbLinkList,
	}
	return resp, nil
}

func (lc *LinkController) LinkListCount(ctx context.Context,
	req *linkPB.LinkListCountRequest) (resp *linkPB.LinkListCountResponse, err error) {
	// 請求資料檢查

	var toListUserID primitive.ObjectID
	if !req.GetAllUser() {
		toListUserID, err = primitive.ObjectIDFromHex(req.GetUserIdHex())
		if err != nil {
			err = status.Error(codes.InvalidArgument, "user id format is invalid")
			return
		}
	}
	if err = tagsArgumentCheck(req.GetTags()); err != nil {
		return
	}

	// 權限檢查

	userInfo, err := lc.UserRequestGet(ctx)
	if err != nil {
		return
	}
	if req.GetAllUser() {
		if !userInfo.IsManager {
			err = common.GRPCERRPermissionDenied
			return
		}
	} else {
		if toListUserID != userInfo.ID {
			err = common.GRPCERRPermissionDenied
			return
		}
	}

	totalNum, err := models.LinkListCount(ctx, req.GetAllUser(), toListUserID, req.GetTags())
	if err != nil {
		return
	}

	resp = &linkPB.LinkListCountResponse{
		TotalNum: uint64(totalNum),
	}

	return resp, nil
}

func (lc *LinkController) LinkPatch(ctx context.Context, req *linkPB.LinkPatchRequest) (resp *linkPB.LinkPatchResponse, err error) {
	// 請求資料檢查

	var hasPatch = false
	if req.GetPatchNote() {
		hasPatch = true
		if len(req.GetNote()) > noteMaxLen {
			err = status.Error(codes.InvalidArgument, "length of note is greater than "+strconv.Itoa(noteMaxLen))
			return
		}
	}
	if req.GetPatchTags() {
		hasPatch = true
		if err = tagsArgumentCheck(req.GetTags()); err != nil {
			return
		}
	}
	if !hasPatch {
		err = status.Error(codes.InvalidArgument, "no patch field")
		return
	}
	toPatchLinkID, err := primitive.ObjectIDFromHex(req.GetLinkIdHex())
	if err != nil {
		err = status.Error(codes.InvalidArgument, "link id format is invalid")
		return
	}

	// 使用者身分驗證

	userInfo, err := lc.UserRequestGet(ctx)
	if err != nil {
		return
	}
	toPatchLink, exist, err := models.LinkFindByID(ctx, toPatchLinkID)
	if err != nil {
		return
	} else if !exist || userInfo.ID != toPatchLink.Creator {
		err = common.GRPCERRPermissionDenied
		return
	}

	err = toPatchLink.Patch(ctx, &models.LinkPatchInfo{
		PNote: req.GetPatchNote(),
		Note:  req.GetNote(),
		PTags: req.GetPatchTags(),
		Tags:  req.GetTags(),
	})
	if err != nil {
		return
	}

	resp = &linkPB.LinkPatchResponse{
		Msg: "success",
	}

	return resp, nil
}

func (lc *LinkController) LinkDelete(ctx context.Context, req *linkPB.LinkDeleteRequest) (resp *linkPB.LinkDeleteResponse, err error) {
	toDeleteLinkID, err := primitive.ObjectIDFromHex(req.GetLinkIdHex())
	if err != nil {
		err = status.Error(codes.InvalidArgument, "link id format is invalid")
		return
	}

	userInfo, err := lc.UserRequestGet(ctx)
	if err != nil {
		return
	}

	toDeleteLink, exist, err := models.LinkFindByID(ctx, toDeleteLinkID)
	if err != nil {
		return
	} else if !exist || userInfo.ID != toDeleteLink.Creator {
		err = common.GRPCERRPermissionDenied
		return
	}

	if !toDeleteLink.Deleted {
		err = toDeleteLink.Delete(ctx)
		if err != nil {
			return
		}
		err = rdModels.LinkDelete(ctx, toDeleteLink)
		if err != nil {
			_ = toDeleteLink.SetNoDelete(ctx)
			return
		}
	}
	resp = &linkPB.LinkDeleteResponse{
		Msg: "success",
	}

	return resp, nil
}

func (lc *LinkController) UserTagsGet(ctx context.Context, req *linkPB.UserTagsGetRequest) (resp *linkPB.UserTagsGetResponse, err error) {
	userInfo, err := lc.UserRequestGet(ctx)
	if err != nil {
		return
	}

	tags, err := models.TagsAggreByUser(ctx, userInfo.ID)
	if err != nil {
		return
	}

	resp = &linkPB.UserTagsGetResponse{
		Tags: tags,
	}
	return resp, nil
}
