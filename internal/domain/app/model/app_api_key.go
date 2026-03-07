package model

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AppApiKey 应用 API Key 实体
type AppApiKey struct {
	id          uuid.UUID
	appID       uuid.UUID
	apiKey      string
	description string
	isActive    bool
	lastUsedAt  *time.Time
	createdAt   time.Time
}

// NewAppApiKey 创建新的 API Key
func NewAppApiKey(appID uuid.UUID, description string) (*AppApiKey, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("app_api_key: failed to generate ID: %w", err)
	}
	key, err := generateApiKey()
	if err != nil {
		return nil, fmt.Errorf("app_api_key: failed to generate key: %w", err)
	}
	return &AppApiKey{
		id:          id,
		appID:       appID,
		apiKey:      key,
		description: description,
		isActive:    true,
		createdAt:   time.Now(),
	}, nil
}

// ReconstituteAppApiKey 从持久化数据重建 API Key
func ReconstituteAppApiKey(
	id, appID uuid.UUID,
	apiKey, description string,
	isActive bool,
	lastUsedAt *time.Time,
	createdAt time.Time,
) *AppApiKey {
	return &AppApiKey{
		id:          id,
		appID:       appID,
		apiKey:      apiKey,
		description: description,
		isActive:    isActive,
		lastUsedAt:  lastUsedAt,
		createdAt:   createdAt,
	}
}

func (k *AppApiKey) ID() uuid.UUID          { return k.id }
func (k *AppApiKey) AppID() uuid.UUID       { return k.appID }
func (k *AppApiKey) ApiKey() string         { return k.apiKey }
func (k *AppApiKey) Description() string    { return k.description }
func (k *AppApiKey) IsActive() bool         { return k.isActive }
func (k *AppApiKey) LastUsedAt() *time.Time { return k.lastUsedAt }
func (k *AppApiKey) CreatedAt() time.Time   { return k.createdAt }

// Revoke 作废 API Key
func (k *AppApiKey) Revoke() {
	k.isActive = false
}

// IsValid 检查 API Key 是否有效
func (k *AppApiKey) IsValid() bool {
	return k.isActive
}

// generateApiKey 生成带 app- 前缀的随机 Key
func generateApiKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "app-" + hex.EncodeToString(b), nil
}
