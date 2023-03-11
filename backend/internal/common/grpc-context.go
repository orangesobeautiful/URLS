package common

import (
	"context"
	"net/http"
	"net/textproto"
	"strings"

	"golang.org/x/net/http/httpguts"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func CtxSetHeader(ctx context.Context, key string, values ...string) error {
	md := metadata.MD{strings.ToLower(key): values}
	return grpc.SetHeader(ctx, md)
}

func CtxSetCookie(ctx context.Context, cookie *http.Cookie) error {
	return CtxSetHeader(ctx, "Set-Cookie", cookie.String())
}

func isNotToken(r rune) bool {
	return !httpguts.IsTokenRune(r)
}

func isCookieNameValid(raw string) bool {
	if raw == "" {
		return false
	}
	return strings.IndexFunc(raw, isNotToken) < 0
}

func validCookieValueByte(b byte) bool {
	return 0x20 <= b && b < 0x7f && b != '"' && b != ';' && b != '\\'
}

func parseCookieValue(raw string, allowDoubleQuote bool) (string, bool) {
	// Strip the quotes, if present.
	if allowDoubleQuote && len(raw) > 1 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	for i := 0; i < len(raw); i++ {
		if !validCookieValueByte(raw[i]) {
			return "", false
		}
	}
	return raw, true
}

// CtxReadCookies 讀取 gRPC gateway Context 中的 cookies
//
// 解析方法從 http 中照搬
func CtxReadCookies(ctx context.Context, filter string) []*http.Cookie {
	var cookieLines []string
	var cookieHeaderExist bool
	md, _ := metadata.FromIncomingContext(ctx)
	cookieLines, cookieHeaderExist = md[MetadataPrefix+"cookie"]
	if !cookieHeaderExist || len(cookieLines) == 0 {
		return []*http.Cookie{}
	}

	cookies := make([]*http.Cookie, 0, len(cookieLines)+strings.Count(cookieLines[0], ";"))
	for _, line := range cookieLines {
		line = textproto.TrimString(line)

		var part string
		for len(line) > 0 { // continue since we have rest
			part, line, _ = strings.Cut(line, ";")
			part = textproto.TrimString(part)
			if part == "" {
				continue
			}
			name, val, _ := strings.Cut(part, "=")
			if !isCookieNameValid(name) {
				continue
			}
			if filter != "" && filter != name {
				continue
			}
			val, ok := parseCookieValue(val, true)
			if !ok {
				continue
			}
			cookies = append(cookies, &http.Cookie{Name: name, Value: val})
		}
	}
	return cookies
}
