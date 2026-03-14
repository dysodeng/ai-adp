package config

import "github.com/spf13/viper"

// Gateway 网关注册配置
type Gateway struct {
	Enabled bool        `mapstructure:"enabled"`
	Type    string      `mapstructure:"type"` // etcd
	Etcd    GatewayEtcd `mapstructure:"etcd"`
}

// GatewayEtcd etcd网关注册配置
type GatewayEtcd struct {
	Endpoints []string `mapstructure:"endpoints"`
	Prefix    string   `mapstructure:"prefix"`
	Username  string   `mapstructure:"username"`
	Password  string   `mapstructure:"password"`
}

func gatewayBindEnv(v *viper.Viper) {
	_ = v.BindEnv("enabled", "GATEWAY_ENABLED")
	_ = v.BindEnv("type", "GATEWAY_TYPE")
	_ = v.BindEnv("etcd.endpoints", "GATEWAY_ETCD_ENDPOINTS")
	_ = v.BindEnv("etcd.prefix", "GATEWAY_ETCD_PREFIX")
	_ = v.BindEnv("etcd.username", "GATEWAY_ETCD_USERNAME")
	_ = v.BindEnv("etcd.password", "GATEWAY_ETCD_PASSWORD")

	v.SetDefault("enabled", false)
	v.SetDefault("type", "etcd")
	v.SetDefault("etcd.prefix", "/services/")
}
