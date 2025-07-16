package entities

type AppConfigFile struct {
	UUIDBase
	AppID     string `json:"appID" gorm:"not null;uniqueIndex:idx_appID_slug;uniqueIndex:idx_appID_mountPath;index;size:36"`
	Slug      string `json:"slug" gorm:"not null;uniqueIndex:idx_appID_slug;size:64"`
	Content   string `json:"content" gorm:"type:text"`
	MountPath string `json:"mountPath" gorm:"not null;uniqueIndex:idx_appID_mountPath;size:255"`
	FileMode  string `json:"fileMode" gorm:"size:4;default:0644"`
	AuditBase
}
