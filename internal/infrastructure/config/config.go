package config

import (
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config 应用全局配置
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Tracing  TracingConfig  `mapstructure:"tracing"`
}

// AppConfig 应用基础配置
type AppConfig struct {
	Name  string `mapstructure:"name"`
	Env   string `mapstructure:"env"` // development | production | test
	Debug bool   `mapstructure:"debug"`
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	HTTP HTTPServerConfig `mapstructure:"http"`
}

// HTTPServerConfig HTTP 服务器参数
type HTTPServerConfig struct {
	Port            int `mapstructure:"port"`             // 监听端口，默认 8080
	ReadTimeout     int `mapstructure:"read_timeout"`     // 秒，默认 30
	WriteTimeout    int `mapstructure:"write_timeout"`    // 秒，默认 60
	ShutdownTimeout int `mapstructure:"shutdown_timeout"` // 优雅关闭等待秒，默认 10
}

// LoggerConfig Zap 日志配置
type LoggerConfig struct {
	Level      string `mapstructure:"level"`       // debug | info | warn | error
	Format     string `mapstructure:"format"`      // json | console
	OutputPath string `mapstructure:"output_path"` // stdout 或文件路径
}

// JWTConfig JWT 配置
type JWTConfig struct {
	Secret     string `mapstructure:"secret"`
	ExpireHour int    `mapstructure:"expire_hour"` // token 有效小时数
}

// DatabaseConfig PostgreSQL 数据库配置
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Name            string `mapstructure:"name"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	SSLMode         string `mapstructure:"ssl_mode"`
	MaxOpenConns    int    `mapstructure:"max_open_conns"`    // 最大打开连接数，默认 100
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`    // 最大空闲连接数，默认 10
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // 连接最大存活分钟数，默认 60
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr         string `mapstructure:"addr"`
	Password     string `mapstructure:"password"`
	DB           int    `mapstructure:"db"`
	PoolSize     int    `mapstructure:"pool_size"`      // 连接池大小，默认 10
	MinIdleConns int    `mapstructure:"min_idle_conns"` // 最小空闲连接数，默认 5
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
	// 将环境变量中的 _ 映射为 .，如 DATABASE_HOST -> database.host
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 设置默认值
	setDefaults(v)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	// 显式绑定所有需要从环境变量覆盖的配置项
	bindEnvKeys(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// bindEnvKeys 显式绑定环境变量键到 viper 配置键
func bindEnvKeys(v *viper.Viper) {
	envBindings := map[string]string{
		"app.env":              "APP_ENV",
		"app.debug":            "APP_DEBUG",
		"database.host":        "DATABASE_HOST",
		"database.port":        "DATABASE_PORT",
		"database.name":        "DATABASE_NAME",
		"database.user":        "DATABASE_USER",
		"database.password":    "DATABASE_PASSWORD",
		"database.ssl_mode":    "DATABASE_SSL_MODE",
		"redis.addr":           "REDIS_ADDR",
		"redis.password":       "REDIS_PASSWORD",
		"redis.db":             "REDIS_DB",
		"jwt.secret":           "JWT_SECRET",
		"tracing.enabled":      "TRACING_ENABLED",
		"tracing.endpoint":     "TRACING_ENDPOINT",
		"tracing.service_name": "TRACING_SERVICE_NAME",
		"tracing.sample_rate":  "TRACING_SAMPLE_RATE",
		"server.http.port":     "SERVER_HTTP_PORT",
		"logger.level":         "LOGGER_LEVEL",
		"logger.format":        "LOGGER_FORMAT",
		"logger.output_path":   "LOGGER_OUTPUT_PATH",
	}
	for key, env := range envBindings {
		v.BindEnv(key, env)
	}
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("app.env", "development")
	v.SetDefault("app.debug", false)
	v.SetDefault("server.http.port", 8080)
	v.SetDefault("server.http.read_timeout", 30)
	v.SetDefault("server.http.write_timeout", 60)
	v.SetDefault("server.http.shutdown_timeout", 10)
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "json")
	v.SetDefault("logger.output_path", "stdout")
	v.SetDefault("jwt.expire_hour", 24)
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.conn_max_lifetime", 60)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.min_idle_conns", 5)
	v.SetDefault("tracing.enabled", false)
	v.SetDefault("tracing.sample_rate", 1.0)
}
