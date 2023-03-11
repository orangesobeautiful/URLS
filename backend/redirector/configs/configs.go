package configs

import "URLS/internal/common"

// RDSCfgInfo redirector service config
type RDSCfgInfo struct {
	common.BaseCfgInfo `mapstructure:",squash"`
	WebSSL             bool
	WithoutGW          bool // 是否通過 gateway 反向代理
}
