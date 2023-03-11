package fhutils

import (
	"URLS/internal/utils/strconvext"
	"net/http"
	"strings"

	"github.com/valyala/fasthttp"
	"golang.org/x/exp/slices"
)

// TODO: 優化宣告方法與程式碼流程&完善功能

var allowHeaders = []string{"Origin", "Content-Length", "Content-Type"}
var allowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

type FHCORS struct {
	AllowAllOrigins bool
	AllowOrigins    []string

	allowHeadersBytes []byte
	allowMethodsBytes []byte
}

func NewFHCORS() *FHCORS {
	var allowHeadersBytes, allowMethodsBytes []byte
	if len(allowHeaders) > 0 {
		allowHeadersBytes = []byte(strings.Join(allowHeaders, ","))
	}
	if len(allowMethods) > 0 {
		allowMethodsBytes = []byte(strings.Join(allowMethods, ","))
	}

	return &FHCORS{
		allowHeadersBytes: allowHeadersBytes,
		allowMethodsBytes: allowMethodsBytes,
	}
}

func (cors *FHCORS) setResponseHeader(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	ctx.Response.Header.SetBytesV("Access-Control-Allow-Origin", ctx.Request.Header.Peek("Origin"))
	ctx.Response.Header.Add("Vary", "Origin")
	ctx.Response.Header.Add("Vary", "Access-Control-Request-Method")
	ctx.Response.Header.Add("Vary", "Access-Control-Request-Headers")

	if len(cors.allowHeadersBytes) > 0 {
		ctx.Response.Header.SetBytesV("Access-Control-Allow-Headers", cors.allowHeadersBytes)
	}
	if len(cors.allowMethodsBytes) > 0 {
		ctx.Response.Header.SetBytesV("Access-Control-Allow-Methods", cors.allowMethodsBytes)
	}
}

func (cors *FHCORS) corsCheck(ctx *fasthttp.RequestCtx) (isCORS, continueRun bool) {
	origin := strconvext.B2S(ctx.Request.Header.Peek("Origin"))
	if origin == "" {
		continueRun = true
		return
	}
	host := strconvext.B2S(ctx.Request.Host())

	if origin == "http://"+host || origin == "https://"+host {
		continueRun = true
		return
	}

	isCORS = true
	if !cors.validateOrigin(origin) {
		ctx.SetStatusCode(http.StatusForbidden)
		continueRun = false
		return
	}

	if strconvext.B2S(ctx.Method()) == "OPTIONS" {
		cors.setResponseHeader(ctx)
		continueRun = false
		return
	}

	continueRun = true
	return
}

func (cors *FHCORS) validateOrigin(origin string) bool {
	if cors.AllowAllOrigins {
		return true
	}

	return slices.Contains(cors.AllowOrigins, origin)
}

func (cors *FHCORS) Combined(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		isCORS, continueRun := cors.corsCheck(ctx)
		if !continueRun {
			return
		}

		next(ctx)

		if isCORS {
			cors.setResponseHeader(ctx)
		}
	}
}
