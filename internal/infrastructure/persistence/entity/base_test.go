package entity_test

import (
	"testing"

	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/entity"
	"github.com/stretchr/testify/assert"
)

func TestBase_IDGeneratedOnCreate(t *testing.T) {
	b := &entity.Base{}
	assert.Empty(t, b.ID)

	err := b.GenerateID()
	assert.NoError(t, err)
	assert.NotEmpty(t, b.ID)
	assert.Len(t, b.ID, 36) // UUID v7 standard format
}

func TestBase_IDNotOverwrittenIfSet(t *testing.T) {
	b := &entity.Base{ID: "existing-id"}
	err := b.GenerateID()
	assert.NoError(t, err)
	assert.Equal(t, "existing-id", b.ID)
}
