package configs

import (
	"URLS/internal/common"
	"net/http"
	"strconv"
	"strings"
)

type AuthCookieInfo struct {
	DisableSecure bool
	SameSite      string
	Domain        string
}

func (info *AuthCookieInfo) GetHTTPSameSite() http.SameSite {
	i, err := strconv.ParseInt(info.SameSite, 10, 32)
	if err == nil {
		switch http.SameSite(i) {
		case http.SameSiteDefaultMode, http.SameSiteLaxMode, http.SameSiteStrictMode, http.SameSiteNoneMode:
			return http.SameSite(i)
		}
	}

	switch strings.ToLower(info.SameSite) {
	case "default":
		return http.SameSiteDefaultMode
	case "lax":
		return http.SameSiteLaxMode
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	}

	return http.SameSite(0)
}

// USCfgInfo User service config
type USCfgInfo struct {
	common.BaseCfgInfo `mapstructure:",squash"`

	CookieKeyPairs []string
	AuthCookie     AuthCookieInfo
}
