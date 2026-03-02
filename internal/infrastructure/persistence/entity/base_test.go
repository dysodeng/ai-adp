package entity_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/dysodeng/ai-adp/internal/infrastructure/persistence/entity"
)

func TestBase_IDGeneratedOnCreate(t *testing.T) {
	b := &entity.Base{}
	assert.Equal(t, uuid.Nil, b.ID)

	err := b.GenerateID()
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, b.ID)
	// UUID v7 string representation is 36 characters
	assert.Len(t, b.ID.String(), 36)
}

func TestBase_IDNotOverwrittenIfSet(t *testing.T) {
	existing, err := uuid.NewV7()
	assert.NoError(t, err)

	b := &entity.Base{ID: existing}
	err = b.GenerateID()
	assert.NoError(t, err)
	assert.Equal(t, existing, b.ID)
}
