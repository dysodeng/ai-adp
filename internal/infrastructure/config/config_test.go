package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

const testConfigYAML = `
app:
  name: ai-adp-test
  env: test
  debug: false
server:
  http:
    port: 9090
    read_timeout: 15
    write_timeout: 30
    shutdown_timeout: 5
logger:
  level: debug
  format: json
  output_path: stdout
jwt:
  secret: test-secret
  expire_hour: 12
database:
  host: localhost
  port: 5432
  name: ai_adp_test
  user: postgres
  password: secret
  ssl_mode: disable
  max_open_conns: 50
  max_idle_conns: 5
  conn_max_lifetime: 30
redis:
  addr: localhost:6379
  db: 1
  pool_size: 5
  min_idle_conns: 2
tracing:
  enabled: true
  endpoint: localhost:4317
  service_name: ai-adp-test
  sample_rate: 0.5
`

func writeTempConfig(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	t.Cleanup(func() { os.Remove(f.Name()) })
	_, _ = f.WriteString(content)
	f.Close()
	return f.Name()
}

func TestLoad_AllSections(t *testing.T) {
	path := writeTempConfig(t, testConfigYAML)
	cfg, err := config.Load(path)
	require.NoError(t, err)

	// App
	assert.Equal(t, "ai-adp-test", cfg.App.Name)
	assert.Equal(t, "test", cfg.App.Env)
	assert.False(t, cfg.App.Debug)

	// Server
	assert.Equal(t, 9090, cfg.Server.HTTP.Port)
	assert.Equal(t, 15, cfg.Server.HTTP.ReadTimeout)
	assert.Equal(t, 5, cfg.Server.HTTP.ShutdownTimeout)

	// Logger
	assert.Equal(t, "debug", cfg.Logger.Level)
	assert.Equal(t, "json", cfg.Logger.Format)

	// JWT
	assert.Equal(t, "test-secret", cfg.JWT.Secret)
	assert.Equal(t, 12, cfg.JWT.ExpireHour)

	// Database
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 50, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)
	assert.Equal(t, 30, cfg.Database.ConnMaxLifetime)

	// Redis
	assert.Equal(t, 5, cfg.Redis.PoolSize)

	// Tracing
	assert.True(t, cfg.Tracing.Enabled)
	assert.Equal(t, "ai-adp-test", cfg.Tracing.ServiceName)
	assert.InDelta(t, 0.5, cfg.Tracing.SampleRate, 0.001)
}

func TestLoad_Defaults(t *testing.T) {
	// Minimal config — defaults should fill in the rest
	minimal := `
app:
  name: minimal
database:
  host: localhost
  port: 5432
  name: db
  user: user
redis:
  addr: localhost:6379
`
	path := writeTempConfig(t, minimal)
	cfg, err := config.Load(path)
	require.NoError(t, err)

	assert.Equal(t, 8080, cfg.Server.HTTP.Port)
	assert.Equal(t, 30, cfg.Server.HTTP.ReadTimeout)
	assert.Equal(t, "info", cfg.Logger.Level)
	assert.Equal(t, "json", cfg.Logger.Format)
	assert.Equal(t, 24, cfg.JWT.ExpireHour)
	assert.Equal(t, 100, cfg.Database.MaxOpenConns)
	assert.Equal(t, 10, cfg.Redis.PoolSize)
	assert.False(t, cfg.Tracing.Enabled)
	assert.InDelta(t, 1.0, cfg.Tracing.SampleRate, 0.001)
}
