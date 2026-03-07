package model

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/dysodeng/ai-adp/internal/domain/app/valueobject"
)

// AppVersion AI 应用版本实体
type AppVersion struct {
	id          uuid.UUID
	appID       uuid.UUID
	version     int
	status      valueobject.VersionStatus
	config      *valueobject.AppConfig
	publishedAt *time.Time
}

// NewAppVersion 创建新版本（初始状态为 Draft）
func NewAppVersion(appID uuid.UUID, version int, config *valueobject.AppConfig) (*AppVersion, error) {
	if version < 1 {
		return nil, fmt.Errorf("app_version: version must be >= 1")
	}
	if config == nil {
		return nil, fmt.Errorf("app_version: config cannot be nil")
	}
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("app_version: failed to generate ID: %w", err)
	}
	return &AppVersion{
		id:      id,
		appID:   appID,
		version: version,
		status:  valueobject.VersionStatusDraft,
		config:  config,
	}, nil
}

// ReconstituteVersion 从持久化数据重建版本
func ReconstituteVersion(
	id, appID uuid.UUID,
	version int,
	status valueobject.VersionStatus,
	config *valueobject.AppConfig,
	publishedAt *time.Time,
) *AppVersion {
	return &AppVersion{
		id:          id,
		appID:       appID,
		version:     version,
		status:      status,
		config:      config,
		publishedAt: publishedAt,
	}
}

func (v *AppVersion) ID() uuid.UUID                     { return v.id }
func (v *AppVersion) AppID() uuid.UUID                  { return v.appID }
func (v *AppVersion) Version() int                      { return v.version }
func (v *AppVersion) Status() valueobject.VersionStatus { return v.status }
func (v *AppVersion) Config() *valueobject.AppConfig    { return v.config }
func (v *AppVersion) PublishedAt() *time.Time           { return v.publishedAt }

func (v *AppVersion) Publish() {
	v.status = valueobject.VersionStatusPublished
	now := time.Now()
	v.publishedAt = &now
}

func (v *AppVersion) Archive() {
	v.status = valueobject.VersionStatusArchived
}

func (v *AppVersion) UpdateConfig(config *valueobject.AppConfig) error {
	if v.status != valueobject.VersionStatusDraft {
		return fmt.Errorf("app_version: can only update config in draft status")
	}
	v.config = config
	return nil
}
