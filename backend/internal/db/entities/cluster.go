package entities

type Cluster struct {
	UUIDBase
	Slug        string `json:"slug" gorm:"not null;uniqueIndex;size:36"` // Cluster slug, typically a URL-friendly name
	DisplayName string `json:"displayName" gorm:"not null;size:255"`     // Human-readable name for the cluster
	Description string `json:"description" gorm:"size:255"`              // Optional description of the cluster
	KubeConfig  string `json:"kubeConfig" gorm:"not null;type:text"`     // Kubernetes configuration in YAML format
	Enabled     bool   `json:"enabled" gorm:"not null;default:false"`    // Whether the cluster is enabled
	AuditBase
}
