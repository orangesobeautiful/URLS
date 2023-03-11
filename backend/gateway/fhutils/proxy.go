package fhutils

import (
	"crypto/tls"

	"github.com/valyala/fasthttp"
)

// ReverseProxy fasthttp Reverse Proxy
type ReverseProxy struct {
	hc           *fasthttp.HostClient
	ErrorHandler func(ctx *fasthttp.RequestCtx, err error)
}

// NewReverseProxy create a new fasthttp proxy
func NewReverseProxy(addr string, isTLS bool, tlsCfg *tls.Config) *ReverseProxy {
	return &ReverseProxy{
		hc: &fasthttp.HostClient{
			Addr:      addr,
			IsTLS:     isTLS,
			TLSConfig: tlsCfg,
		},
	}
}

func (rp *ReverseProxy) Handler(ctx *fasthttp.RequestCtx) {
	req := &ctx.Request
	resp := &ctx.Response
	rp.prepareRequest(req)

	if rp.hc.IsTLS {
		req.URI().SetSchemeBytes([]byte("https"))
	} else {
		req.URI().SetSchemeBytes([]byte("http"))
	}

	req.Header.SetProtocolBytes([]byte("HTTP/1.1"))

	if err := rp.hc.Do(req, resp); err != nil && rp.ErrorHandler != nil {
		rp.ErrorHandler(ctx, err)
	}
	rp.postprocessResponse(resp)
}

func (rp *ReverseProxy) prepareRequest(req *fasthttp.Request) {
	req.Header.Del("Connection")
}

func (rp *ReverseProxy) postprocessResponse(resp *fasthttp.Response) {
	resp.Header.Del("Connection")
}
