package errors_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	sharederrors "github.com/dysodeng/ai-adp/internal/domain/shared/errors"
)

func TestDomainError_Error(t *testing.T) {
	err := sharederrors.New("TENANT_NOT_FOUND", "tenant not found")
	assert.Equal(t, "tenant not found", err.Error())
	assert.Equal(t, "TENANT_NOT_FOUND", err.Code())
}

func TestDomainError_Is(t *testing.T) {
	err := sharederrors.New("TENANT_NOT_FOUND", "tenant not found")
	same := sharederrors.New("TENANT_NOT_FOUND", "different message")
	different := sharederrors.New("OTHER_ERROR", "other error")

	assert.True(t, sharederrors.Is(err, "TENANT_NOT_FOUND"))
	assert.True(t, sharederrors.Is(same, "TENANT_NOT_FOUND"))
	assert.False(t, sharederrors.Is(different, "TENANT_NOT_FOUND"))
}
