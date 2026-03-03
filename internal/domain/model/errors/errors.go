package errors

import "github.com/dysodeng/ai-adp/internal/domain/shared/errors"

var (
	ErrModelConfigNotFound = errors.New("MODEL_CONFIG_NOT_FOUND", "model config not found")
	ErrNoDefaultModel      = errors.New("NO_DEFAULT_MODEL", "no default model configured for this capability")
	ErrModelDisabled       = errors.New("MODEL_DISABLED", "model config is disabled")
)
