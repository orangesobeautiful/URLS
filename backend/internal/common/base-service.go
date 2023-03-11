package common

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/textproto"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/gowo9/g3server"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	gwRuntime "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/encoding/protojson"
)

const MetadataPrefix string = gwRuntime.MetadataPrefix

func toKeepHeader(hdr string) bool {
	switch hdr {
	case
		HderNameGWIP,
		HderNameGWCountry,
		"Accept",
		"Accept-Charset",
		"Accept-Language",
		"Accept-Ranges",
		"Authorization",
		"Cache-Control",
		"Content-Type",
		"Cookie",
		"Date",
		"Expect",
		"From",
		"Host",
		"If-Match",
		"If-Modified-Since",
		"If-None-Match",
		"If-Schedule-Tag-Match",
		"If-Unmodified-Since",
		"Max-Forwards",
		"Origin",
		"Pragma",
		"Referer",
		"User-Agent",
		"Via",
		"Warning":
		return true
	}
	return false
}

func inHeaderMatcherFn(key string) (string, bool) {
	switch key = textproto.CanonicalMIMEHeaderKey(key); {
	case toKeepHeader(key):
		return MetadataPrefix + key, true
	case strings.HasPrefix(key, gwRuntime.MetadataHeaderPrefix):
		return key[len(gwRuntime.MetadataHeaderPrefix):], true
	}
	return "", false
}

func outHeaderMatcherFn(key string) (string, bool) {
	key = textproto.CanonicalMIMEHeaderKey(key)
	switch key {
	case "Set-Cookie":
		return key, true
	case "Content-Type":
		return "", false
	default:
		return fmt.Sprintf("%s%s", gwRuntime.MetadataHeaderPrefix, key), true
	}
}

// LoggerCodeToLevel
func LoggerCodeToLevel(code codes.Code) zapcore.Level {
	switch code {
	case codes.OK:
		return zap.InfoLevel
	case codes.Canceled:
		return zap.InfoLevel
	case codes.Unknown:
		return zap.ErrorLevel
	case codes.InvalidArgument:
		return zap.InfoLevel
	case codes.DeadlineExceeded:
		return zap.WarnLevel
	case codes.NotFound:
		return zap.InfoLevel
	case codes.AlreadyExists:
		return zap.InfoLevel
	case codes.PermissionDenied:
		return zap.InfoLevel
	case codes.Unauthenticated:
		return zap.InfoLevel // unauthenticated requests can happen
	case codes.ResourceExhausted:
		return zap.InfoLevel
	case codes.FailedPrecondition:
		return zap.InfoLevel
	case codes.Aborted:
		return zap.InfoLevel
	case codes.OutOfRange:
		return zap.WarnLevel
	case codes.Unimplemented:
		return zap.ErrorLevel
	case codes.Internal:
		return zap.ErrorLevel
	case codes.Unavailable:
		return zap.WarnLevel
	case codes.DataLoss:
		return zap.ErrorLevel
	default:
		return zap.ErrorLevel
	}
}

func NewGRPCServer(logger *zap.Logger) *grpc.Server {
	recoveryFunc := func(p interface{}) (err error) {
		logger.Error("panic error", zap.Any("msg", p))
		err = GRPCErrInternal
		return
	}

	midLogger := logger.WithOptions(zap.WithCaller(false), zap.AddStacktrace(zap.PanicLevel))
	gs := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			grpc_zap.StreamServerInterceptor(midLogger, grpc_zap.WithLevels(LoggerCodeToLevel)),
			grpc_recovery.StreamServerInterceptor(grpc_recovery.WithRecoveryHandler(recoveryFunc)),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			grpc_zap.UnaryServerInterceptor(midLogger, grpc_zap.WithLevels(LoggerCodeToLevel)),
			grpc_recovery.UnaryServerInterceptor(grpc_recovery.WithRecoveryHandler(recoveryFunc)),
		)),
	)

	return gs
}

type Service struct {
	name    string
	address string
	logger  *zap.Logger
	s       *g3server.Server
}

func (bs Service) Run() {
	sugar := bs.logger.Sugar()

	tcpLis, err := net.Listen("tcp", bs.address)
	if err != nil {
		bs.logger.Error("net.Listen failed", zap.String("address", bs.address), zap.Error(err))
		return
	}

	certsDir := filepath.Join("server-data", "certs")
	sugar.Infof("%s listen %s", bs.name, bs.address)
	err = bs.s.ServeTLS(tcpLis,
		filepath.Join(certsDir, "srvc.crt"),
		filepath.Join(certsDir, "srvc.key"))
	if err != nil {
		bs.logger.Error("s.Serve failed", zap.Error(err))
		return
	}
}

// NewService 建立新的 service
//
// name: service 的名字
//
// cInfo: service 自行定義的 config 類型(需要和 nCtrlFn 的第一個參數類型相同)
//
// nCtrlFn: 新建 controller 時需要呼叫的函數(第一個參數需要和 cInfo 類型相同)
//
// gRPCRSSFn: gRPC 的註冊函數 (ex: pb.RegisterXXXServiceServer)
//
// gwRegFunc: gRPC gateway 的註冊函數 (ex: pb.RegisterXXXServiceHandler)
func NewService(
	name string, cfgInfo BaseCfgInterface,
	nCtrlFn any, gRPCRSSFn any,
	gwRegFunc g3server.GatewayRegisterFunc) *Service {
	var err error
	var showName string
	switch len(name) {
	case 0, 1:
		showName = strings.ToUpper(name)
	default:
		showName = strings.ToLower(name)
		showName = strings.ToUpper(name[0:1]) + showName[1:]
	}

	// init config

	err = ParseConfigFile(name, "", cfgInfo)
	if err != nil {
		log.Fatalf("ParseConfigFile failed, err=%s", err)
	}

	// init logger

	logger, err := cfgInfo.GenZapConfig()
	if err != nil {
		log.Fatalf("logger init faeild, err=%s", err)
	}
	defer func() { _ = logger.Sync() }()
	sugar := logger.Sugar()

	sugar.Infof("Start %s Service", showName)

	// init service

	sugar.Info("Init Service")

	// create a new controller

	// 將 nCtrlFn 轉換成 func(DBInfo, *zap.Logger) (*XXXController, error) 形式來呼叫
	nCtrlFnValue := reflect.ValueOf(nCtrlFn)
	nCtrlFnArgs := []reflect.Value{reflect.ValueOf(cfgInfo), reflect.ValueOf(logger)}
	nCtrlFnRes := nCtrlFnValue.Call(nCtrlFnArgs)
	controller := nCtrlFnRes[0].Interface()
	if !nCtrlFnRes[1].IsNil() {
		err = nCtrlFnRes[1].Interface().(error)
		logger.Error("New"+showName+"Serivce failed", zap.Error(err))
		return nil
	}

	// gRPC 設定

	gs := NewGRPCServer(logger)
	grpcDialOpts := []grpc.DialOption{}

	// gRPC gateway Register Handler

	// 將 gRPCRSSFn 轉換成 pb.RegisterXXXServiceServer 形式來呼叫
	gRPCRSSFnValue := reflect.ValueOf(gRPCRSSFn)
	gRPCRSSFnArgs := []reflect.Value{reflect.ValueOf(gs), reflect.ValueOf(controller)}
	gRPCRSSFnValue.Call(gRPCRSSFnArgs)

	gwOptList := []gwRuntime.ServeMuxOption{
		gwRuntime.WithIncomingHeaderMatcher(inHeaderMatcherFn),
		gwRuntime.WithOutgoingHeaderMatcher(outHeaderMatcherFn),
		gwRuntime.WithMarshalerOption(gwRuntime.MIMEWildcard, &gwRuntime.HTTPBodyMarshaler{
			Marshaler: &gwRuntime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					EmitUnpopulated: true,
				},
				UnmarshalOptions: protojson.UnmarshalOptions{
					DiscardUnknown: true,
				},
			},
		}),
	}

	s, err := g3server.New(context.Background(), gs, gwRegFunc,
		g3server.WithGRPCDialOption(grpcDialOpts),
		g3server.WithGWServeMuxOption(gwOptList))
	if err != nil {
		sugar.Errorw("server.NewGGServer failed", "err", err)
	}

	return &Service{
		name:    name,
		address: cfgInfo.GetListenAddr(),
		logger:  logger,
		s:       s,
	}
}
