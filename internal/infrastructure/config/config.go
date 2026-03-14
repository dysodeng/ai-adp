package config

import (
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	VarPath  string = "var"
	LogPath         = VarPath + "/logs"
	TempPath        = VarPath + "/tmp"
)

var GlobalConfig *Config

// Config 应用全局配置
type Config struct {
	App      App            `mapstructure:"app"`
	Server   Server         `mapstructure:"server"`
	Security Security       `mapstructure:"security"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    Redis          `mapstructure:"redis"`
	Cache    Cache          `mapstructure:"cache"`
	Monitor  Monitor        `mapstructure:"monitor"`
	Tracing  TracingConfig  `mapstructure:"tracing"`
	Gateway  Gateway        `mapstructure:"gateway"`
}

// TracingConfig OpenTelemetry 分布式追踪配置
type TracingConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	Endpoint    string  `mapstructure:"endpoint"`     // OTLP gRPC 端点，如 localhost:4317
	ServiceName string  `mapstructure:"service_name"` // 服务名，默认使用 App.Name
	SampleRate  float64 `mapstructure:"sample_rate"`  // 采样率 0.0-1.0，默认 1.0
}

// Load 从文件加载配置，支持 .env 文件和环境变量覆盖
func Load(path string) (*Config, error) {
	// 加载 .env 文件到进程环境变量（文件不存在则忽略）
	_ = godotenv.Load()

	v := viper.New()

	v.SetConfigFile(path)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	var appConfig App
	if app := v.Sub("app"); app != nil {
		appBindEnv(app)
		if err := app.Unmarshal(&appConfig); err != nil {
			return nil, err
		}
	}

	var securityConfig Security
	if security := v.Sub("security"); security != nil {
		securityBindEnv(security)
		if err := security.Unmarshal(&securityConfig); err != nil {
			return nil, err
		}
	}

	var serverConfig Server
	if server := v.Sub("server"); server != nil {
		serverBindEnv(server)
		if err := server.Unmarshal(&serverConfig); err != nil {
			return nil, err
		}
	}

	var databaseConfig DatabaseConfig
	if database := v.Sub("database"); database != nil {
		databaseBindEnv(database)
		if err := database.Unmarshal(&databaseConfig); err != nil {
			return nil, err
		}
	}

	var redisConfig Redis
	if redis := v.Sub("redis"); redis != nil {
		redisBindEnv(redis)
		if err := redis.Unmarshal(&redisConfig); err != nil {
			return nil, err
		}
	}

	var cacheConfig Cache
	if cache := v.Sub("cache"); cache != nil {
		cacheBindEnv(cache)
		if err := cache.Unmarshal(&cacheConfig); err != nil {
			return nil, err
		}
	}

	var monitorConfig Monitor
	if monitor := v.Sub("monitor"); monitor != nil {
		monitorBindEnv(monitor)
		if err := monitor.Unmarshal(&monitorConfig); err != nil {
			return nil, err
		}
	}

	var gatewayConfig Gateway
	if gateway := v.Sub("gateway"); gateway != nil {
		gatewayBindEnv(gateway)
		if err := gateway.Unmarshal(&gatewayConfig); err != nil {
			return nil, err
		}
	}

	// 设置默认值
	setDefaults(v)

	// 显式绑定所有需要从环境变量覆盖的配置项
	bindEnvKeys(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	cfg.App = appConfig
	cfg.Server = serverConfig
	cfg.Security = securityConfig
	cfg.Database = databaseConfig
	cfg.Redis = redisConfig
	cfg.Monitor = monitorConfig
	cfg.Cache = cacheConfig
	cfg.Gateway = gatewayConfig

	GlobalConfig = &cfg

	return &cfg, nil
}

// bindEnvKeys 显式绑定环境变量键到 viper 配置键
func bindEnvKeys(v *viper.Viper) {
	envBindings := map[string]string{
		"database.host":        "DATABASE_HOST",
		"database.port":        "DATABASE_PORT",
		"database.name":        "DATABASE_NAME",
		"database.user":        "DATABASE_USER",
		"database.password":    "DATABASE_PASSWORD",
		"database.ssl_mode":    "DATABASE_SSL_MODE",
		"tracing.enabled":      "TRACING_ENABLED",
		"tracing.endpoint":     "TRACING_ENDPOINT",
		"tracing.service_name": "TRACING_SERVICE_NAME",
		"tracing.sample_rate":  "TRACING_SAMPLE_RATE",
	}
	for key, env := range envBindings {
		_ = v.BindEnv(key, env)
	}
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("tracing.enabled", false)
	v.SetDefault("tracing.sample_rate", 1.0)
}
