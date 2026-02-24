package entities

// DeploymentHistory records each deployment event when an app's image is updated
type DeploymentHistory struct {
	Base
	AppID string `gorm:"type:varchar(36);index;not null"`

	// Deployment snapshot
	ImageBefore    string `gorm:"type:varchar(256)"` // Previous image
	ImageAfter     string `gorm:"type:varchar(256)"` // New image deployed
	ReplicasBefore int    `gorm:"type:int"`          // Previous replicas count
	ReplicasAfter  int    `gorm:"type:int"`          // New replicas count

	// Resource limits snapshot
	RequestCPUBefore    int `gorm:"type:int"`
	RequestCPUAfter     int `gorm:"type:int"`
	RequestMemoryBefore int `gorm:"type:int"`
	RequestMemoryAfter  int `gorm:"type:int"`
	LimitCPUBefore      int `gorm:"type:int"`
	LimitCPUAfter       int `gorm:"type:int"`
	LimitMemoryBefore   int `gorm:"type:int"`
	LimitMemoryAfter    int `gorm:"type:int"`

	// Deployment metadata
	DeployType string `gorm:"type:varchar(32);default:'manual'"`  // manual, auto, rollback
	DeployedBy string `gorm:"type:varchar(128)"`                  // User ID or system
	Reason     string `gorm:"type:text"`                          // Deployment reason or description
	Status     string `gorm:"type:varchar(32);default:'success'"` // success, failed, in_progress

	// Build reference (optional)
	BuildID *string `gorm:"type:varchar(36);index"`

	// Relations
	App   App    `gorm:"foreignKey:AppID"`
	Build *Build `gorm:"foreignKey:BuildID"`
}
