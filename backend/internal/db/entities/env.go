package entities

type Env struct {
	UUIDBase
	Slug             string `json:"slug" gorm:"not null;uniqueIndex:idx_projectID_slug;size:36"`                               // Env slug, typically a URL-friendly name
	DisplayName      string `json:"displayName" gorm:"not null;size:255"`                                                      // Human-readable name for the environment
	Description      string `json:"description" gorm:"size:255"`                                                               // Optional description of the environment
	ProjectID        string `json:"projectID" gorm:"not null;uniqueIndex:idx_projectID_slug,idx_projectID_name;index;size:64"` // Project UUID this environment belongs to
	ProjectSlug      string `json:"projectSlug" gorm:"not null;size:36"`                                                       // Project slug this environment belongs to, typically a URL-friendly name
	ClusterID        string `json:"clusterID" gorm:"not null;index;size:64"`                                                   // Cluster UUID where this environment is deployed
	ClusterSlug      string `json:"clusterSlug" gorm:"not null;size:36"`                                                       // Cluster slug where this environment is deployed, typically a URL-friendly name
	ClusterNamespace string `json:"clusterNamespace" gorm:"not null;uniqueIndex:idx_projectID_slug;size:64"`                   // Cluster namespace for this environment
	AuditBase
}
