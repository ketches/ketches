package models

import "time"

type PluginEnvVar struct {
	Key   string `json:"key" binding:"required"`
	Value string `json:"value"`
}

type CreatePluginRequest struct {
	ProjectID        string         `json:"project_id" binding:"required"`
	Slug             string         `json:"slug" binding:"required"`
	Name             string         `json:"name" binding:"required"`
	Description      string         `json:"description"`
	Image            string         `json:"image" binding:"required"`
	RegistryUsername string         `json:"registry_username"`
	RegistryPassword string         `json:"registry_password"`
	Command          string         `json:"command"`
	EnvVars          []PluginEnvVar `json:"env_vars"`
	PluginType       string         `json:"plugin_type" binding:"required,oneof=init sidecar"`
}

type UpdatePluginRequest struct {
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	Image            string         `json:"image"`
	RegistryUsername string         `json:"registry_username"`
	RegistryPassword string         `json:"registry_password"`
	Command          string         `json:"command"`
	EnvVars          []PluginEnvVar `json:"env_vars"`
	PluginType       string         `json:"plugin_type" binding:"omitempty,oneof=init sidecar"`
}

type PluginResponse struct {
	ID               string         `json:"id"`
	Slug             string         `json:"slug"`
	Name             string         `json:"name"`
	Description      string         `json:"description"`
	Image            string         `json:"image"`
	RegistryUsername string         `json:"registry_username"`
	Command          string         `json:"command"`
	EnvVars          []PluginEnvVar `json:"env_vars"`
	PluginType       string         `json:"plugin_type"`
	InstallCount     int            `json:"install_count"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

type InstallPluginRequest struct {
	PluginID string         `json:"plugin_id" binding:"required"`
	EnvVars  []PluginEnvVar `json:"env_vars"`
}

type UpdateAppPluginEnvRequest struct {
	EnvVars []PluginEnvVar `json:"env_vars" binding:"required"`
}

type UpdateAppPluginResourcesRequest struct {
	RequestCPU    *int `json:"request_cpu"`
	LimitCPU      *int `json:"limit_cpu"`
	RequestMemory *int `json:"request_memory"`
	LimitMemory   *int `json:"limit_memory"`
}

type AppPluginResponse struct {
	ID        string         `json:"id"`
	AppID     string         `json:"app_id"`
	PluginID  string         `json:"plugin_id"`
	Enabled   bool           `json:"enabled"`
	EnvVars   []PluginEnvVar `json:"env_vars"`
	Plugin    PluginResponse `json:"plugin"`
	CreatedAt time.Time      `json:"created_at"`
}
