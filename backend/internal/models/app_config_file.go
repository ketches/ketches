package models

type AppConfigFileModel struct {
	ConfigFileID string `json:"configFileID" gorm:"column:id"`
	AppID        string `json:"appID" gorm:"column:app_id"`
	Slug         string `json:"slug" gorm:"column:slug"`
	Content      string `json:"content" gorm:"column:content"`
	MountPath    string `json:"mountPath" gorm:"column:mount_path"`
	FileMode     string `json:"fileMode" gorm:"column:file_mode"`
}

type ListAppConfigFilesRequest struct {
	AppID string `uri:"appID" binding:"required"`
}

type ListAppConfigFilesResponse struct {
	Total   int64                 `json:"total"`
	Records []*AppConfigFileModel `json:"records"`
}

type CreateAppConfigFileRequest struct {
	AppID     string `json:"-" uri:"appID"`
	Slug      string `json:"slug" binding:"required"`
	Content   string `json:"content" binding:"required,max=972800"` // 950KB = 950*1024 bytes
	MountPath string `json:"mountPath" binding:"required"`
	FileMode  string `json:"fileMode" binding:"required"`
}

type UpdateAppConfigFileRequest struct {
	AppID        string `json:"-" uri:"appID"`
	ConfigFileID string `json:"-" uri:"configFileID"`
	Content      string `json:"content" binding:"required,max=972800"` // 950KB = 950*1024 bytes
	MountPath    string `json:"mountPath" binding:"required"`
	FileMode     string `json:"fileMode" binding:"required"`
}

type DeleteAppConfigFilesRequest struct {
	AppID         string   `json:"-" uri:"appID"`
	ConfigFileIDs []string `json:"configFileIDs" binding:"required,dive,required"` // List of config file IDs to delete
}
