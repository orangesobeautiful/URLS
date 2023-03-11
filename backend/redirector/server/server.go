package server

import (
	"crypto/tls"
	"log"
	"net"

	"URLS/internal/common"
	"URLS/internal/fh"
	"URLS/redirector/configs"
	"URLS/redirector/controllers"

	"github.com/gowo9/fhlogger/fhzap"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

const serviceName string = "redirector"

func NewFHServer(logger *zap.Logger) *fasthttp.Server {
	s := &fasthttp.Server{
		Logger:                fh.NewInternalLogger(logger),
		NoDefaultServerHeader: true,
		ReadBufferSize:        4096,
		WriteBufferSize:       4096,
		MaxRequestBodySize:    fasthttp.DefaultMaxRequestBodySize,
		Concurrency:           fasthttp.DefaultConcurrency,
		TLSConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
				tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			},
		},
	}
	return s
}

func Run() {
	var err error
	cfgInfo := new(configs.RDSCfgInfo)
	if err = common.ParseConfigFile(serviceName, "", cfgInfo); err != nil {
		log.Fatalf("common.ParseConfigFile failed, err=%s", err)
	}

	logger, err := cfgInfo.GenZapConfig()
	if err != nil {
		log.Fatalf("GenZapConfig failed, err=%s", err)
	}
	sugar := logger.Sugar()

	rdCtrl, err := controllers.NewRDController(cfgInfo, logger)
	if err != nil {
		log.Fatal()
	}

	addr := cfgInfo.GetListenAddr()
	var ln net.Listener
	var tlsConfig *tls.Config
	tlsConfig, err = common.GenInternalTLSConfig(serviceName)
	if err != nil {
		logger.Fatal("common.GenInternalTLSConfig failed", zap.Error(err))
	}

	ln, err = tls.Listen("tcp4", addr, tlsConfig)
	if err != nil {
		sugar.Fatalw("net listener failed", "addr", addr, "err", err)
	}

	restServer := NewFHServer(logger)
	restServer.TLSConfig = tlsConfig
	restServer.Handler = fhzap.New(logger).Combined(rdCtrl.GetRestHandler())

	logger.Sugar().Infof("%s listen %s", serviceName, addr)
	err = restServer.Serve(ln)
	_ = logger.Sync()
	if err != nil {
		sugar.Fatalw("fasthttp.Serve failed", "err", err)
	}
}
