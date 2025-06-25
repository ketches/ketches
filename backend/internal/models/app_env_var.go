package models

type AppEnvVarModel struct {
	EnvVarID string `json:"envVarID" gorm:"column:id"`
	Key      string `json:"key" gorm:"column:key"`
	Value    string `json:"value" gorm:"column:value"`
	AppID    string `json:"appID" gorm:"column:app_id"`
}

type ListAppEnvVarsRequest struct {
	AppID string `uri:"appID" binding:"required"`
}

type ListAppEnvVarsResponse struct {
	Total   int64             `json:"total"`
	Records []*AppEnvVarModel `json:"records"`
}

type CreateAppEnvVarRequest struct {
	AppID string `json:"-" uri:"appID"`
	Key   string `json:"key" binding:"required"`
	Value string `json:"value" binding:"required"`
}

type UpdateAppEnvVarRequest struct {
	AppID    string `json:"-" uri:"appID"`
	EnvVarID string `json:"-" uri:"envVarID"`
	Value    string `json:"value" binding:"required"`
}

type DeleteAppEnvVarsRequest struct {
	AppID     string   `json:"-" uri:"appID"`
	EnvVarIDs []string `json:"envVarIDs" binding:"required,dive,required"` // List of env var IDs to delete
}
