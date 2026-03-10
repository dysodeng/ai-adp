package telemetry

import (
	"github.com/dysodeng/ai-adp/internal/infrastructure/config"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/telemetry/log"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/telemetry/metrics"
	"github.com/dysodeng/ai-adp/internal/infrastructure/pkg/telemetry/trace"
)

type Monitor struct{}

func InitMonitor(cfg *config.Config) (*Monitor, error) {
	if err := log.Init(cfg); err != nil {
		return nil, err
	}
	if err := metrics.Init(cfg); err != nil {
		return nil, err
	}
	if err := trace.Init(cfg); err != nil {
		return nil, err
	}
	return &Monitor{}, nil
}
