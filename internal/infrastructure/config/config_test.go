package config_test

import (
	"os"
	"testing"

	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testConfigYAML = `
app:
  name: ai-adp-test
  environment: test
  debug: false
server:
  http:
    port: 9090
    read_timeout: 15
    write_timeout: 30
    shutdown_timeout: 5
security:
  jwt:
    secret: test-secret
database:
  host: localhost
  port: 5432
  database: ai_adp_test
  username: postgres
  password: secret
  max_open_conns: 50
  max_idle_conns: 5
redis:
  main:
    mode: standalone
    host: 127.0.0.1
    port: "6379"
    db: 0
    pool:
      min_idle_conns: 10
      max_retries: 3
      pool_size: 100
  cache:
    mode: standalone
    host: 127.0.0.1
    port: "6379"
    db: 1
    pool:
      min_idle_conns: 10
      max_retries: 3
      pool_size: 100
cache:
  driver: redis
  serializer: json
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
	assert.Equal(t, "test", cfg.App.Environment)
	assert.False(t, cfg.App.Debug)

	// Server
	assert.Equal(t, 9090, cfg.Server.HTTP.Port)
	assert.Equal(t, 15, cfg.Server.HTTP.ReadTimeout)
	assert.Equal(t, 5, cfg.Server.HTTP.ShutdownTimeout)

	// Security
	assert.Equal(t, "test-secret", cfg.Security.JWT.Secret)

	// Database
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 50, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)

	// Redis
	assert.Equal(t, "standalone", cfg.Redis.Main.Mode)
	assert.Equal(t, "127.0.0.1", cfg.Redis.Main.Host)
	assert.Equal(t, 100, cfg.Redis.Main.Pool.PoolSize)
	assert.Equal(t, 1, cfg.Redis.Cache.DB)

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
  database: db
  username: user
redis:
  main:
    host: 127.0.0.1
    port: "6379"
`
	path := writeTempConfig(t, minimal)
	cfg, err := config.Load(path)
	require.NoError(t, err)

	assert.Equal(t, 100, cfg.Database.MaxOpenConns)
	assert.Equal(t, "127.0.0.1", cfg.Redis.Main.Host)
	assert.False(t, cfg.Tracing.Enabled)
	assert.InDelta(t, 1.0, cfg.Tracing.SampleRate, 0.001)
}
