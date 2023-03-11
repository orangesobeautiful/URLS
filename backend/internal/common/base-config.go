package common

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type RunMode string

const (
	ProductionMode  RunMode = "production"
	DevelopmentMode RunMode = "development"
)

// Str2RunMode 將 string 轉換為 RunMode
func Str2RunMode(s string) (RunMode, bool) {
	switch strings.ToLower(s) {
	case "dev", DevelopmentMode.String():
		return DevelopmentMode, true
	case "pro", ProductionMode.String():
		return ProductionMode, true
	default:
		return RunMode(""), false
	}
}

func (r RunMode) String() string {
	return string(r)
}

// BaseCfgInfo 每個服務的通用設定
type BaseCfgInfo struct {
	Mode       RunMode
	ListenAddr string

	WebDomain string
	RDDomain  string

	CFSupport bool // cloud flare 功能支援

	SrvcKeyPath string
	SrvcAddrMap ServiceAddrMap // Service Address Map
	MgoDB       MgoDBInfo
	RedisDB     RedisDBInfo
	Log         LogInfo
}

// ServiceAddrInfo service 的連線地址資訊
type ServiceAddrInfo struct {
	REST string // 一般 REST API 的 address
	GRPC string // GRPC 的 address
}

// ServiceAddrMap 各個 service 的連線資訊
type ServiceAddrMap struct {
	User ServiceAddrInfo
	Link ServiceAddrInfo
	RD   ServiceAddrInfo
}

type MgoDBInfo struct {
	URI    string
	DBName string
}

type RedisDBTLSInfo struct {
	Enable     bool
	CertFile   string
	KeyFile    string
	CACertFile string
}

type RedisDBInfo struct {
	UserName string
	Password string
	Address  string
	TLS      RedisDBTLSInfo
}

type LogInfo struct {
	Level            string
	Color            bool // 輸出時受否要加上色彩字元，在 production 模式時會被強制禁用
	OutputPaths      []string
	ErrorOutputPaths []string
}

type BaseCfgInterface interface {
	GenZapConfig() (*zap.Logger, error)
	GetListenAddr() string
}

func parseCommonConfigFile(configDir string, vPtr any) (err error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)
	v.SetConfigName("common")

	if err = v.ReadInConfig(); err != nil {
		err = fmt.Errorf("read common config failed, err=%s", err)
		return
	}

	if err = v.Unmarshal(vPtr); err != nil {
		err = fmt.Errorf("unmarshal common config failed, err=%s", err)
		return
	}

	return
}

// ParseConfigFile 解析設定檔
func ParseConfigFile(serviceName, configDir string, vPtr any) (err error) {
	if configDir == "" {
		configDir = filepath.Join("server-data", "configs")
	}

	if err = parseCommonConfigFile(configDir, vPtr); err != nil {
		return
	}

	v := viper.New()
	v.SetConfigType("yaml")
	v.AddConfigPath(configDir)
	if serviceName == "" {
		panic("config name cannot be empty")
	}
	v.SetConfigName(serviceName)

	if err = v.ReadInConfig(); err != nil {
		err = fmt.Errorf("read %s config failed, err=%s", serviceName, err)
		return
	}

	if err = v.Unmarshal(vPtr); err != nil {
		err = fmt.Errorf("unmarshal %s config failed, err=%s", serviceName, err)
		return
	}

	return nil
}

func (bc *BaseCfgInfo) GenZapConfig() (logger *zap.Logger, err error) {
	runMode, convOK := Str2RunMode(bc.Mode.String())
	if !convOK {
		err = fmt.Errorf("unknow config mode \"%s\"", bc.Mode)
		return
	}

	var zapConfig zap.Config
	switch runMode {
	case DevelopmentMode:
		zapConfig = zap.NewDevelopmentConfig()
		if bc.Log.Color {
			zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		}
	case ProductionMode:
		zapConfig = zap.NewProductionConfig()
	}

	lv, err := zapcore.ParseLevel(bc.Log.Level)
	if err != nil {
		err = fmt.Errorf("parse log level failed, err=%s", err)
		return
	}
	zapConfig.Level.SetLevel(lv)
	zapConfig.OutputPaths = bc.Log.OutputPaths
	zapConfig.ErrorOutputPaths = bc.Log.ErrorOutputPaths

	logger, err = zapConfig.Build()
	return
}

func (bc *BaseCfgInfo) GenRedisOptions() (opts *redis.Options, err error) {
	opts = &redis.Options{
		Addr:     bc.RedisDB.Address,
		Username: bc.RedisDB.UserName,
		Password: bc.RedisDB.Password,
	}

	if bc.RedisDB.TLS.Enable {
		var srvcCert tls.Certificate
		srvcCert, err = tls.LoadX509KeyPair(bc.RedisDB.TLS.CertFile, bc.RedisDB.TLS.KeyFile)
		if err != nil {
			return
		}

		// Load the CA certificate
		var caCert []byte
		caCert, err = os.ReadFile(bc.RedisDB.TLS.CACertFile)
		if err != nil {
			return
		}

		// Put the CA certificate to certificate pool
		certPool := x509.NewCertPool()
		if !certPool.AppendCertsFromPEM(caCert) {
			return
		}

		// Create the TLS configuration
		opts.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{srvcCert},
			RootCAs:      certPool,
			ClientCAs:    certPool,
			MinVersion:   tls.VersionTLS12,
		}
	}

	return opts, nil
}

func (bc *BaseCfgInfo) GetListenAddr() string {
	if bc.ListenAddr == "" {
		return ":443"
	}
	return bc.ListenAddr
}

func (bc *BaseCfgInfo) GetMgoDB() MgoDBInfo {
	return bc.MgoDB
}
