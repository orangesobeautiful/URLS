package controllers

import (
	"URLS/gateway/fhutils"
	"URLS/internal/common"
	"net/http"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

const (
	badGatewayBody         string = "{\"code\":14,\"message\":\"bad gateway\",\"details\":[]}"
	serviceUnavailableBody string = "{\"code\":14,\"message\":\"serevice unavailable\",\"details\":[]}"
	gatewayTimeout         string = "{\"code\":14,\"message\":\"gateway timeout\",\"details\":[]}"
)

const (
	webPrefix        string = "/web/"
	apiPrefix        string = "/api"
	userSrvPrefix    string = "/user/"
	linkSrvPrefix    string = "/link/"
	redirectorPrefix string = ""
)

var userRProxy, linkRProxy, redirectorRProxy *fhutils.ReverseProxy

func (pr *ProxyRouter) preDeal(ctx *fasthttp.RequestCtx) {
	var ipSet, countrySet bool
	if pr.cfg.CFSupport {
		cfIP := string(ctx.Request.Header.Peek("Cf-Connecting-Ip"))
		if cfIP != "" {
			ipSet = true
			ctx.Request.Header.Set(common.HderNameGWIP, cfIP)
		}
		cfCountry := string(ctx.Request.Header.Peek("Cf-Ipcountry"))
		if cfCountry != "" {
			countrySet = true
			ctx.Request.Header.Set(common.HderNameGWCountry, cfCountry)
		}
	}

	if !ipSet {
		ctx.Request.Header.Set(common.HderNameGWIP, ctx.RemoteIP().String())
	}
	if !countrySet {
		ctx.Request.Header.Set(common.HderNameGWCountry, "")
	}
}

func (pr *ProxyRouter) errorHandler(ctx *fasthttp.RequestCtx, err error) {
	ctx.Response.SetStatusCode(http.StatusBadGateway)
	ctx.Response.Header.Set("Content-Type", "application/json")
	_, _ = ctx.WriteString(badGatewayBody)
	pr.logger.Error("proxy failed", zap.Error(err))
}

func (pr *ProxyRouter) InitReverseProxy() (err error) {
	tlsCfg, err := common.GenInternalTLSConfig(common.InternalTLSServerName)
	if err != nil {
		return
	}

	userRProxy = fhutils.NewReverseProxy(pr.cfg.SrvcAddrMap.User.REST, true, tlsCfg)
	userRProxy.ErrorHandler = pr.errorHandler
	linkRProxy = fhutils.NewReverseProxy(pr.cfg.SrvcAddrMap.Link.REST, true, tlsCfg)
	linkRProxy.ErrorHandler = pr.errorHandler
	redirectorRProxy = fhutils.NewReverseProxy(pr.cfg.SrvcAddrMap.RD.REST, true, tlsCfg)
	redirectorRProxy.ErrorHandler = pr.errorHandler

	return nil
}

func (pr *ProxyRouter) UserRProxy(ctx *fasthttp.RequestCtx) {
	pr.preDeal(ctx)
	ctx.Request.URI().SetPathBytes(ctx.Path()[len(apiPrefix+userSrvPrefix)-1:])
	userRProxy.Handler(ctx)
}

func (pr *ProxyRouter) LinkRProxy(ctx *fasthttp.RequestCtx) {
	pr.preDeal(ctx)
	ctx.Request.URI().SetPathBytes(ctx.Path()[len(apiPrefix+linkSrvPrefix)-1:])
	linkRProxy.Handler(ctx)
}

func (pr *ProxyRouter) RedictroRProxy(ctx *fasthttp.RequestCtx) {
	pr.preDeal(ctx)
	redirectorRProxy.Handler(ctx)
}
