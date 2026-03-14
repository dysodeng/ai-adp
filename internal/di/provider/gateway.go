package provider

import (
	"fmt"
	"strings"

	"github.com/dysodeng/gateway/sdk"
	"github.com/dysodeng/gateway/sdk/etcd"

	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

// GatewayRegistry 网关注册器包装，用于 DI 注入
type GatewayRegistry struct {
	Registry sdk.Registry
	Config   *config.Config
}

// ProvideGatewayRegistry 提供网关注册器
func ProvideGatewayRegistry(cfg *config.Config) (*GatewayRegistry, error) {
	if !cfg.Gateway.Enabled {
		return &GatewayRegistry{Config: cfg}, nil
	}

	switch cfg.Gateway.Type {
	case "etcd":
		return provideEtcdRegistry(cfg)
	default:
		return nil, fmt.Errorf("unsupported gateway type: %s", cfg.Gateway.Type)
	}
}

func provideEtcdRegistry(cfg *config.Config) (*GatewayRegistry, error) {
	endpoints := cfg.Gateway.Etcd.Endpoints
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("gateway etcd endpoints is empty")
	}

	// 处理逗号分隔的 endpoints（兼容环境变量单字符串格式）
	var parsedEndpoints []string
	for _, ep := range endpoints {
		for _, e := range strings.Split(ep, ",") {
			e = strings.TrimSpace(e)
			if e != "" {
				parsedEndpoints = append(parsedEndpoints, e)
			}
		}
	}

	opts := []etcd.Option{
		etcd.WithPrefix(cfg.Gateway.Etcd.Prefix),
	}
	if cfg.Gateway.Etcd.Username != "" {
		opts = append(opts, etcd.WithAuth(cfg.Gateway.Etcd.Username, cfg.Gateway.Etcd.Password))
	}

	registry, err := etcd.NewRegistry(parsedEndpoints, opts...)
	if err != nil {
		return nil, fmt.Errorf("create gateway registry failed: %w", err)
	}

	return &GatewayRegistry{
		Registry: registry,
		Config:   cfg,
	}, nil
}
