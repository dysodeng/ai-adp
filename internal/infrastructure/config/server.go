package config

import "github.com/spf13/viper"

// Server 服务配置
type Server struct {
	HTTP      HTTPConfig      `mapstructure:"http"`
	GRPC      GRPCConfig      `mapstructure:"grpc"`
	WebSocket WebSocketConfig `mapstructure:"websocket"`
	Event     EventConfig     `mapstructure:"event"`
	Health    HealthConfig    `mapstructure:"health"`
}

// HTTPConfig HTTP服务配置
type HTTPConfig struct {
	Enabled         bool   `mapstructure:"enabled"`
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	ReadTimeout     int    `mapstructure:"read_timeout"`     // 秒，默认 30
	WriteTimeout    int    `mapstructure:"write_timeout"`    // 秒，默认 60
	ShutdownTimeout int    `mapstructure:"shutdown_timeout"` // 优雅关闭等待秒，默认 10
}

// GRPCConfig gRPC配置
type GRPCConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Namespace string `mapstructure:"namespace"`
}

// WebSocketConfig WebSocket配置
type WebSocketConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Host    string `mapstructure:"host"`
	Port    int    `mapstructure:"port"`
}

// EventConfig 事件消费者服务配置
type EventConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Driver  string `mapstructure:"driver"`
}

type HealthConfig struct {
	Enabled bool `mapstructure:"enabled"`
	Port    int  `mapstructure:"port"`
}

func serverBindEnv(v *viper.Viper) {
	_ = v.BindEnv("http.port", "SERVER_HTTP_PORT")
	_ = v.BindEnv("grpc.port", "SERVER_GRPC_PORT")
	_ = v.BindEnv("websocket.port", "SERVER_WEBSOCKET_PORT")
	_ = v.BindEnv("health.port", "SERVER_HEALTH_PORT")

	v.SetDefault("http.port", 8080)
	v.SetDefault("http.read_timeout", 30)
	v.SetDefault("http.write_timeout", 60)
	v.SetDefault("http.shutdown_timeout", 10)
}
