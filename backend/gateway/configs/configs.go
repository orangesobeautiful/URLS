package configs

import "URLS/internal/common"

type CORSInfo struct {
	Enable          bool
	AllowAllOrigins bool     // 允許所有的 Origins, 會覆蓋 AllowOrigins 的設定值
	AllowOrigins    []string // 只允許特定的 Origins
}

type WebInfo struct {
	StaticRoot string
}

type GWSCfgInfo struct {
	common.BaseCfgInfo `mapstructure:",squash"`

	SSL      bool
	CertPath string
	KeyPath  string

	CORS CORSInfo

	WebInfo WebInfo
}
