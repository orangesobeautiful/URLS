package controllers

import (
	"URLS/gateway/configs"
	"URLS/gateway/fhutils"
	"path/filepath"

	"github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type ProxyRouter struct {
	handler fasthttp.RequestHandler
	cfg     *configs.GWSCfgInfo
	logger  *zap.Logger
}

func (pr *ProxyRouter) Handler() fasthttp.RequestHandler {
	return pr.handler
}

func (ProxyRouter) pingHander(ctx *fasthttp.RequestCtx) {
	_, _ = ctx.WriteString("ok")
}

func NewGWRouter(logger *zap.Logger, cfg *configs.GWSCfgInfo) (pr *ProxyRouter, err error) {
	pr = new(ProxyRouter)
	pr.logger = logger.WithOptions(
		zap.WithCaller(false),
		zap.AddStacktrace(zap.PanicLevel))
	pr.cfg = cfg

	err = pr.InitReverseProxy()
	if err != nil {
		return nil, err
	}

	r := router.New()

	apiGroup := r.Group(apiPrefix)
	apiGroup.ANY(userSrvPrefix+"{nouse:*}", pr.UserRProxy)
	apiGroup.ANY(linkSrvPrefix+"{nouse:*}", pr.LinkRProxy)
	apiGroup.GET("/gateway/ping", pr.pingHander)

	var staticRoot = cfg.WebInfo.StaticRoot
	if staticRoot == "" {
		staticRoot = "public"
	}
	var webHandler func(*fasthttp.RequestCtx)
	fs := &fasthttp.FS{
		Root:               staticRoot,
		IndexNames:         []string{"index.html"},
		GenerateIndexPages: false,
		Compress:           true,
		AcceptByteRange:    false,
		PathNotFound: func(ctx *fasthttp.RequestCtx) {
			ctx.SendFile(filepath.Join(staticRoot, "web", "index.html"))
		},
	}
	webHandler = fs.NewRequestHandler()
	r.GET(webPrefix+"{nouse:*}", func(ctx *fasthttp.RequestCtx) {
		webHandler(ctx)
	})

	r.NotFound = pr.RedictroRProxy

	var hander = r.Handler
	if cfg.CORS.Enable {
		cors := fhutils.NewFHCORS()
		cors.AllowAllOrigins = cfg.CORS.AllowAllOrigins
		cors.AllowOrigins = cfg.CORS.AllowOrigins
		hander = cors.Combined(r.Handler)
	}

	pr.handler = hander

	return pr, nil
}
