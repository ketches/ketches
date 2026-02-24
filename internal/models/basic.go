package models

type UpdateBasicInfoRequest struct {
	Name             string `json:"name" binding:"required"`
	Description      string `json:"description"`
	ContainerImage   string `json:"container_image"`
	RegistryUsername string `json:"registry_username"`
	RegistryPassword string `json:"registry_password"`
}
