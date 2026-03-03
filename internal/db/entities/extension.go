package entities

// Extension represents a platform-level OCI-based Helm chart extension in the catalog.
// It is admin-managed and can be installed on any cluster.
type Extension struct {
	Base
	Name        string `gorm:"type:varchar(128);uniqueIndex;not null"`
	DisplayName string `gorm:"type:varchar(256)"`
	Description string `gorm:"type:text"`
	OCIUrl      string `gorm:"type:varchar(512);not null"`
	IconURL     string `gorm:"type:varchar(512)"`
	Builtin     bool   `gorm:"type:bool;default:false"`
	CreatedBy   string `gorm:"type:varchar(36)"`
}
