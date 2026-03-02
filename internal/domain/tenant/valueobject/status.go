package valueobject

type TenantStatus string

const (
	StatusActive    TenantStatus = "active"
	StatusInactive  TenantStatus = "inactive"
	StatusSuspended TenantStatus = "suspended"
)
