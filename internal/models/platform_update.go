package models

type PlatformUpdateTargetConfig struct {
	ImageRepository string `json:"image_repository" binding:"required"`
	Namespace       string `json:"namespace" binding:"required"`
	DeploymentName  string `json:"deployment_name" binding:"required"`
	ContainerName   string `json:"container_name" binding:"required"`
}

type PlatformUpdateConfig struct {
	API PlatformUpdateTargetConfig `json:"api" binding:"required"`
	UI  PlatformUpdateTargetConfig `json:"ui" binding:"required"`
}

type PlatformUpdateComponentStatus struct {
	CurrentVersion    string   `json:"current_version"`
	AvailableVersions []string `json:"available_versions"`
	LatestVersion     string   `json:"latest_version"`
	UpdateAvailable   bool     `json:"update_available"`
	VersionSource     string   `json:"version_source"`
	RolloutPhase      string   `json:"rollout_phase"`
	RolloutMessage    string   `json:"rollout_message"`
	Error             string   `json:"error,omitempty"`
}

type PlatformUpdateStatus struct {
	LocalPlatformVersion     string                        `json:"local_platform_version"`
	RunningInKubernetes      bool                          `json:"running_in_kubernetes"`
	CanRollout               bool                          `json:"can_rollout"`
	RolloutBlockers          []string                      `json:"rollout_blockers"`
	Config                   PlatformUpdateConfig          `json:"config"`
	API                      PlatformUpdateComponentStatus `json:"api"`
	UI                       PlatformUpdateComponentStatus `json:"ui"`
	RecommendedSharedVersion string                        `json:"recommended_shared_version"`
}

type CheckPlatformUpdateRequest struct {
	Mode string `json:"mode" binding:"required,oneof=auto manual"`
}

type TriggerPlatformRolloutRequest struct {
	SharedVersion string `json:"shared_version,omitempty"`
	APIVersion    string `json:"api_version,omitempty"`
	UIVersion     string `json:"ui_version,omitempty"`
}

type PlatformUpdateRolloutTarget struct {
	Namespace      string `json:"namespace"`
	DeploymentName string `json:"deployment_name"`
	ContainerName  string `json:"container_name"`
	PreviousImage  string `json:"previous_image"`
	TargetImage    string `json:"target_image"`
}

type PlatformUpdateRolloutResult struct {
	API               PlatformUpdateRolloutTarget `json:"api"`
	UI                PlatformUpdateRolloutTarget `json:"ui"`
	RollbackAttempted bool                        `json:"rollback_attempted"`
	RollbackSucceeded bool                        `json:"rollback_succeeded"`
}
