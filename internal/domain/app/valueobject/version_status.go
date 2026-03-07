package valueobject

// VersionStatus 应用版本状态
type VersionStatus string

const (
	VersionStatusDraft     VersionStatus = "draft"
	VersionStatusPublished VersionStatus = "published"
	VersionStatusArchived  VersionStatus = "archived"
)

func (s VersionStatus) IsValid() bool {
	switch s {
	case VersionStatusDraft, VersionStatusPublished, VersionStatusArchived:
		return true
	}
	return false
}

func (s VersionStatus) String() string { return string(s) }
