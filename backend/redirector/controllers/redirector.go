package controllers

import (
	"context"
	"fmt"
	"net/http"

	"URLS/internal/common"
	"URLS/internal/utils/strconvext"
	linkModels "URLS/link/models"
	"URLS/redirector/models"

	"github.com/mileusna/useragent"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

func methodCheck(ctx *fasthttp.RequestCtx) bool {
	if !ctx.Request.Header.IsGet() {
		ctx.Response.Header.SetStatusCode(http.StatusMethodNotAllowed)
		return false
	}

	return true
}

func (rd *RedirectorController) webReirect(ctx *fasthttp.RequestCtx, p string) {
	var scheme string
	if rd.cfg.WebSSL {
		scheme = "https"
	} else {
		scheme = "http"
	}

	ctx.Redirect(scheme+"://"+rd.cfg.WebDomain+"/web"+p, http.StatusFound)
}

func (rd *RedirectorController) notFoundRedirect(ctx *fasthttp.RequestCtx) {
	// redirect to not found page
	rd.webReirect(ctx, "/link-error/not-found")
}

func (rd *RedirectorController) deletedRedirect(ctx *fasthttp.RequestCtx) {
	// redirect to deleted link page
	rd.webReirect(ctx, "/link-error/deleted")
}

func (rd *RedirectorController) redirectorHandler(ctx *fasthttp.RequestCtx) {
	if !methodCheck(ctx) {
		return
	}

	reqPath := ctx.Path()
	pathLen := len(reqPath)
	fmt.Println(string(reqPath))
	switch strconvext.B2S(reqPath) {
	case "/", "/favicon.ico":
		rd.webReirect(ctx, string(reqPath))
		return
	}
	if pathLen < 2 {
		rd.notFoundRedirect(ctx)
		return
	}
	shortPath := string(reqPath[1:])

	var reqHost string
	ctxHostStr := string(ctx.Host())
	if ctxHostStr != rd.cfg.RDDomain {
		reqHost = ctxHostStr
	}

	dest, deleted, exist, err := models.LinkGetInfo(ctx, shortPath, reqHost)
	if err != nil {
		rd.Logger.Error("models.LinkGetInfo failed", zap.Error(err))
		ctx.SetStatusCode(http.StatusInternalServerError)
		_, _ = ctx.WriteString(common.ErrMsgInternal)
		return
	}
	if deleted {
		rd.deletedRedirect(ctx)
		return
	}
	if !exist {
		rd.notFoundRedirect(ctx)
		return
	}

	ctx.Redirect(dest, http.StatusMovedPermanently)
	go rd.sourceAnalyze(shortPath, reqHost,
		string(ctx.Request.Header.UserAgent()),
		string(ctx.Request.Header.Peek(common.HderNameGWIP)),
		string(ctx.Request.Header.Peek(common.HderNameGWCountry)))
}

// sourceAnalyze 來源解析
func (rd *RedirectorController) sourceAnalyze(short, host, uaStr, ip, country string) {
	countryClick := make(map[string]uint64, 1)
	if country == "" {
		// TODO: 透過 IP 庫查詢國家
		_ = ip
	} else {
		countryClick[country] = 1
	}

	// ua parse

	ua := useragent.Parse(uaStr)

	osClick := make(map[string]uint64, 1)
	if ua.IsWindows() {
		osClick["windows"] = 1
	} else if ua.IsLinux() {
		osClick["linux"] = 1
	} else if ua.IsMacOS() {
		osClick["macos"] = 1
	} else if ua.IsAndroid() {
		osClick["android"] = 1
	} else if ua.IsIOS() {
		osClick["ios"] = 1
	} else {
		osClick["other"] = 1
	}

	deviceClick := make(map[string]uint64, 1)
	if ua.Desktop {
		deviceClick["desktop"] = 1
	} else if ua.Mobile {
		deviceClick["mobile"] = 1
	} else if ua.Tablet {
		deviceClick["tablet"] = 1
	} else {
		deviceClick["other"] = 1
	}

	browserClick := make(map[string]uint64, 1)
	if ua.IsFirefox() {
		browserClick["firefox"] = 1
	} else if ua.IsEdge() {
		browserClick["edge"] = 1
	} else if ua.IsOpera() {
		browserClick["opera"] = 1
	} else if ua.IsOperaMini() {
		browserClick["opera"] = 1
	} else if ua.IsChrome() {
		browserClick["chrome"] = 1
	} else if ua.IsSafari() {
		browserClick["safari"] = 1
	} else if ua.IsInternetExplorer() {
		browserClick["ie"] = 1
	} else {
		browserClick["other"] = 1
	}

	bgCTX := context.Background()
	_ = linkModels.LinkClicksUpdate(bgCTX, short, host, 1, countryClick, osClick, deviceClick, browserClick)
}
