package valueobject_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/dysodeng/ai-adp/internal/domain/shared/valueobject"
)

func TestPagination_Offset(t *testing.T) {
	p := valueobject.NewPagination(2, 20)
	assert.Equal(t, 20, p.Offset())
	assert.Equal(t, 20, p.Limit())
}

func TestPagination_DefaultLimit(t *testing.T) {
	p := valueobject.NewPagination(1, 0)
	assert.Equal(t, 20, p.Limit()) // 默认 limit 20
}

func TestPagination_Page1IsDefault(t *testing.T) {
	p := valueobject.NewPagination(0, 10)
	assert.Equal(t, 1, p.Page())
	assert.Equal(t, 0, p.Offset())
}
