package server

import (
	"crypto/tls"
	"log"
	"net"

	"URLS/gateway/configs"
	"URLS/gateway/controllers"
	"URLS/internal/common"
	"URLS/internal/fh"

	"github.com/gowo9/fhlogger/fhzap"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/reuseport"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const serviceName string = "gateway"

func Run() {
	var err error
	cfgInfo := new(configs.GWSCfgInfo)
	if err = common.ParseConfigFile(serviceName, "", cfgInfo); err != nil {
		log.Fatalf("common.ParseConfigFile failed, err=%s", err)
	}
	logger, err := cfgInfo.GenZapConfig()
	if err != nil {
		log.Fatalf("GenZapConfig failed, err=%s", err)
	}
	sugar := logger.Sugar()

	r, err := controllers.NewGWRouter(logger, cfgInfo)
	if err != nil {
		logger.Fatal("controllers.NewGWRouter failed", zap.Error(err))
	}
	addr := cfgInfo.GetListenAddr()

	var reuse = false
	var ln net.Listener
	if reuse {
		ln, err = reuseport.Listen("tcp4", addr)
		if err != nil {
			sugar.Fatalw("reuseport listener failed", "addr", addr, "err", err)
		}
	} else {
		ln, err = net.Listen("tcp4", addr)
		if err != nil {
			sugar.Fatalw("net listener failed", "addr", addr, "err", err)
		}
	}

	fhZap := fhzap.New(logger.WithOptions(zap.WithCaller(false)),
		fhzap.WithPreCtxDealFunc(func(ctx *fasthttp.RequestCtx) []zapcore.Field {
			const defaultFieldsNum = 6

			var ip string
			if cfgInfo.CFSupport {
				ip = string(ctx.Request.Header.Peek("Cf-Connecting-Ip"))
			} else {
				ip = ctx.RemoteIP().String()
			}

			zapFields := make([]zapcore.Field, 0, defaultFieldsNum)
			zapFields = append(zapFields,
				zap.String("ip", ip),
				zap.ByteString("method", ctx.Request.Header.Method()),
				zap.String("uri", string(ctx.RequestURI())),
			)

			uaCopy := string(ctx.UserAgent())
			if uaCopy != "" {
				zapFields = append(zapFields, zap.String("agent", uaCopy))
			}

			return zapFields
		}))

	s := &fasthttp.Server{
		Handler:               fhZap.Combined(r.Handler()),
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

	logger.Sugar().Infof("gateway listen %s", addr)
	if cfgInfo.SSL {
		err = s.ServeTLS(ln, cfgInfo.CertPath, cfgInfo.KeyPath)
	} else {
		err = s.Serve(ln)
	}

	_ = logger.Sync()
	if err != nil {
		sugar.Fatalw("fasthttp Server failed", "err", err)
	}
}
