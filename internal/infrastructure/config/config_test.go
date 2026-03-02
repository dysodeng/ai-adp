package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
)

func TestLoad_FromFile(t *testing.T) {
	content := `
app:
  name: ai-adp-test
  env: test
  port: 8080
database:
  host: localhost
  port: 5432
  name: ai_adp_test
  user: postgres
  password: secret
  ssl_mode: disable
redis:
  addr: localhost:6379
  db: 0
`
	f, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(f.Name())
	_, _ = f.WriteString(content)
	f.Close()

	cfg, err := config.Load(f.Name())
	require.NoError(t, err)
	assert.Equal(t, "ai-adp-test", cfg.App.Name)
	assert.Equal(t, 8080, cfg.App.Port)
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, "localhost:6379", cfg.Redis.Addr)
}
