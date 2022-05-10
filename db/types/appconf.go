package types

type AppConf struct {
	Base
	AppID     uint32 `json:"appID" gorm:"column:app_id"`
	FileName  string `json:"fileName" gorm:"column:file_name;type:varchar(255);not null"`
	MountPath string `json:"mountPath" gorm:"column:mount_path;type:varchar(255);not null"`
	Content   string `json:"content" gorm:"column:content;type:longtext"`
}
