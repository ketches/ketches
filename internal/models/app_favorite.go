package models

// AppFavoriteResponse is the response for an app favorite record.
type AppFavoriteResponse struct {
	ID    string `json:"id"`
	EnvID string `json:"env_id"`
	AppID string `json:"app_id"`
}
