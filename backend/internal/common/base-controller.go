package common

import (
	"context"
	"encoding/base64"
	"os"

	linkPB "URLS/proto/gen/go/link/v1"
	rdPB "URLS/proto/gen/go/redirector/v1"
	userPB "URLS/proto/gen/go/user/v1"

	"github.com/qiniu/qmgo"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const SigninCookieName = "urlssi"

const SrvcAuthCtxKey = "srvcauth"

// ServiceConnection 各個微服務間的連線
type ServiceConnection struct {
	SrvcKeyBS64 string
	User        userPB.UserServiceClient
	Link        linkPB.LinkServiceClient
	RD          rdPB.RDServiceClient
}

// GetRequestMetadata 實作 credentials.PerRPCCredentials 介面
func (sc *ServiceConnection) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{
		SrvcAuthCtxKey: sc.SrvcKeyBS64,
	}, nil
}

// RequireTransportSecurity 實作 credentials.PerRPCCredentials 介面
func (ServiceConnection) RequireTransportSecurity() bool {
	return true
}

func (sc *ServiceConnection) clientGRPCDialOpts() ([]grpc.DialOption, error) {
	tlsConfig, err := GenInternalTLSConfig(InternalTLSServerName)
	if err != nil {
		return nil, err
	}

	return []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)),
		grpc.WithPerRPCCredentials(sc),
	}, nil
}

func (sc *ServiceConnection) GenUserConn(addr string) (err error) {
	dialOpts, err := sc.clientGRPCDialOpts()
	if err != nil {
		return
	}
	usConn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return
	}
	sc.User = userPB.NewUserServiceClient(usConn)

	return
}

func (sc *ServiceConnection) GenLinkConn(addr string) (err error) {
	dialOpts, err := sc.clientGRPCDialOpts()
	if err != nil {
		return
	}
	lsConn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return
	}
	sc.Link = linkPB.NewLinkServiceClient(lsConn)

	return
}

func (sc *ServiceConnection) GenRDConn(addr string) (err error) {
	dialOpts, err := sc.clientGRPCDialOpts()
	if err != nil {
		return
	}
	rsConn, err := grpc.Dial(addr, dialOpts...)
	if err != nil {
		return
	}
	sc.RD = rdPB.NewRDServiceClient(rsConn)

	return
}

type BaseController struct {
	Logger      *zap.Logger
	MgoClient   *qmgo.Client
	MgoDB       *qmgo.Database
	SrvcKeyBS64 string
	SrvcConn    *ServiceConnection
}

// NewBaseController create new base controller
func NewBaseController(cfgInfo *BaseCfgInfo, logger *zap.Logger) (bc *BaseController, err error) {
	// 設定 mongodb 連線

	mgoClient, err := qmgo.NewClient(
		context.Background(),
		&qmgo.Config{
			Uri: cfgInfo.MgoDB.URI,
		})
	if err != nil {
		return
	}

	mgoDB := mgoClient.Database(cfgInfo.MgoDB.DBName)

	// 讀取服務之間的驗證檔案

	srvcKeyB64, err := os.ReadFile(cfgInfo.SrvcKeyPath)
	if err != nil {
		return
	}
	srvcKeyBs, err := base64.StdEncoding.DecodeString(string(srvcKeyB64))
	if err != nil {
		return
	}
	srvcKeyStr := base64.URLEncoding.EncodeToString(srvcKeyBs)

	bc = &BaseController{
		Logger:      logger,
		MgoClient:   mgoClient,
		MgoDB:       mgoDB,
		SrvcKeyBS64: srvcKeyStr,
		SrvcConn: &ServiceConnection{
			SrvcKeyBS64: srvcKeyStr,
		},
	}
	return bc, nil
}

type AuthReqLevel int

const (
	_ AuthReqLevel = iota

	AuthSignin // 必須登入
	AuthEither // 有登入無登入皆可
)

// SigninSessValsInfo Sign in Session Values Informations
type SigninSessValsInfo struct {
	UserIDHex string
}

func (bc *BaseController) IsSrvckeyEqual(key string) bool {
	return bc.SrvcKeyBS64 == key
}

// IsInternalCall 判斷是否為內部呼叫
func (bc *BaseController) IsInternalCall(ctx context.Context) error {
	md, _ := metadata.FromIncomingContext(ctx)
	if srvcKeyBS64, keyExist := md[SrvcAuthCtxKey]; keyExist {
		if len(srvcKeyBS64) > 0 && bc.IsSrvckeyEqual(srvcKeyBS64[0]) {
			return nil
		}
	}

	return GRPCERRPermissionDenied
}

// AuthInit controller 統一的登入處理
func (bc *BaseController) AuthInit(ctx context.Context, authReqLV AuthReqLevel) (SigninSessValsInfo, error) {
	var sessValues SigninSessValsInfo

	cookies := CtxReadCookies(ctx, SigninCookieName)
	if len(cookies) == 0 || cookies[0].Value == "" {
		return unAuthErrDeal(authReqLV)
	}

	res, err := bc.SrvcConn.User.GetAuthInfo(ctx, &userPB.GetAuthInfoRequest{
		SigninCookieValue: cookies[0].Value,
	})
	if err != nil {
		if statusErr, convOK := status.FromError(err); convOK &&
			statusErr.Code() == codes.Unauthenticated {
			return unAuthErrDeal(authReqLV)
		}

		bc.Logger.Error("auth GetAuthInfo internal error", zap.Error(err))
		return sessValues, err
	}
	sessValues.UserIDHex = res.GetUserIdHex()

	return sessValues, nil
}

func unAuthErrDeal(authReqLV AuthReqLevel) (SigninSessValsInfo, error) {
	var sessValues SigninSessValsInfo

	switch authReqLV {
	case AuthSignin:
		return sessValues, status.Error(codes.Unauthenticated, "sign in require")
	case AuthEither:
		return sessValues, nil
	default:
		panic("undefind AuthReqLevel")
	}
}

type UserInfo struct {
	ID              primitive.ObjectID
	IDHex           string
	Email           string
	Role            uint32
	IsManager       bool
	NormalLinkQuota uint64   // 一般短網址額度
	NormalLinkUsage uint64   // 一般短網址使用量
	CustomLinkQuota uint64   // 自訂短網址額度
	CustomLinkUsage uint64   // 自訂短網址使用量
	LinkTags        []string // 使用者定義的短網址 tags
}

// UserGetByID 取得指定 ID 的 user 資料 (透過 gRPC 呼叫 user service)
func (bc *BaseController) UserGetByID(ctx context.Context, userHexID string) (*UserInfo, error) {
	resp, err := bc.SrvcConn.User.UserInfoGet(ctx, &userPB.UserInfoGetRequest{UserIdHex: userHexID})
	if err != nil {
		return nil, err
	}

	pbUserInfo := resp.GetUserInfo()
	objID, err := primitive.ObjectIDFromHex(pbUserInfo.GetIdHex())
	if err != nil {
		bc.Logger.Error("primitive.ObjectIDFromHex failed", zap.Error(err))
		return nil, err
	}

	return &UserInfo{
		ID:              objID,
		IDHex:           pbUserInfo.GetIdHex(),
		Email:           pbUserInfo.GetEmail(),
		Role:            pbUserInfo.GetRole(),
		IsManager:       pbUserInfo.GetIsManager(),
		NormalLinkQuota: pbUserInfo.GetNormalQuota(),
		NormalLinkUsage: pbUserInfo.GetNormalUsage(),
		CustomLinkQuota: pbUserInfo.GetCustomQuota(),
		CustomLinkUsage: pbUserInfo.GetCustomUsage(),
	}, nil
}

// UserRequestGet 取得發送請求的 user 資料 (透過 gRPC 呼叫 user service)
func (bc *BaseController) UserRequestGet(ctx context.Context) (*UserInfo, error) {
	sessVals, err := bc.AuthInit(ctx, AuthSignin)
	if err != nil {
		return nil, err
	}

	return bc.UserGetByID(ctx, sessVals.UserIDHex)
}
