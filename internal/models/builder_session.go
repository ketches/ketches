package models

import "time"

type CreateBuilderSessionRequest struct {
	BuildEnvID       string `json:"build_env_id" binding:"required"`
	Title            string `json:"title"`
	Prompt           string `json:"prompt" binding:"required"`
	SelectedModelKey string `json:"selected_model_key"`
	ProviderKey      string `json:"provider_key"`
	ModelProfileKey  string `json:"model_profile_key"`
}

type AppendBuilderSessionMessageRequest struct {
	Content          string `json:"content" binding:"required"`
	SelectedModelKey string `json:"selected_model_key"`
	ProviderKey      string `json:"provider_key"`
	ModelProfileKey  string `json:"model_profile_key"`
}

type BuilderSessionResponse struct {
	ID                     string     `json:"id"`
	ProjectID              string     `json:"project_id"`
	BuildEnvID             string     `json:"build_env_id"`
	Title                  string     `json:"title"`
	Summary                string     `json:"summary"`
	Status                 string     `json:"status"`
	CreatedBy              string     `json:"created_by"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	LastActivityAt         time.Time  `json:"last_activity_at"`
	ExpiresAt              *time.Time `json:"expires_at"`
	LatestRunID            string     `json:"latest_run_id"`
	LatestRunStatus        string     `json:"latest_run_status"`
	CurrentWorkspaceID     string     `json:"current_workspace_id"`
	CurrentWorkspaceStatus string     `json:"current_workspace_status"`
	CurrentWorkspaceRoot   string     `json:"current_workspace_root"`
}

type BuilderSessionListItem struct {
	ID                     string     `json:"id"`
	ProjectID              string     `json:"project_id"`
	BuildEnvID             string     `json:"build_env_id"`
	Title                  string     `json:"title"`
	Summary                string     `json:"summary"`
	Status                 string     `json:"status"`
	CreatedBy              string     `json:"created_by"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
	LastActivityAt         time.Time  `json:"last_activity_at"`
	ExpiresAt              *time.Time `json:"expires_at"`
	LatestRunID            string     `json:"latest_run_id"`
	LatestRunStatus        string     `json:"latest_run_status"`
	CurrentWorkspaceID     string     `json:"current_workspace_id"`
	CurrentWorkspaceStatus string     `json:"current_workspace_status"`
	CurrentWorkspaceRoot   string     `json:"current_workspace_root"`
	ArtifactCount          int64      `json:"artifact_count"`
}

type BuilderMessageResponse struct {
	ID           string    `json:"id"`
	SessionID    string    `json:"session_id"`
	RunID        string    `json:"run_id"`
	Role         string    `json:"role"`
	Content      string    `json:"content"`
	MetadataJSON string    `json:"metadata_json"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type BuilderRunResponse struct {
	ID                 string     `json:"id"`
	SessionID          string     `json:"session_id"`
	TriggerMessageID   string     `json:"trigger_message_id"`
	WorkspaceID        string     `json:"workspace_id"`
	Status             string     `json:"status"`
	RequestedBy        string     `json:"requested_by"`
	InstructionSummary string     `json:"instruction_summary"`
	ExecutionLog       string     `json:"execution_log"`
	StartedAt          *time.Time `json:"started_at"`
	CompletedAt        *time.Time `json:"completed_at"`
	ErrorMessage       string     `json:"error_message"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

type BuilderWorkspaceSummaryResponse struct {
	ID            string     `json:"id"`
	SessionID     string     `json:"session_id"`
	BuildEnvID    string     `json:"build_env_id"`
	ClusterID     string     `json:"cluster_id"`
	Namespace     string     `json:"namespace"`
	PodName       string     `json:"pod_name"`
	ContainerName string     `json:"container_name"`
	Status        string     `json:"status"`
	WorkspaceRoot string     `json:"workspace_root"`
	TerminatedAt  *time.Time `json:"terminated_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type BuilderArtifactSummaryResponse struct {
	ID           string    `json:"id"`
	SessionID    string    `json:"session_id"`
	WorkspaceID  string    `json:"workspace_id"`
	RunID        string    `json:"run_id"`
	Kind         string    `json:"kind"`
	Path         string    `json:"path"`
	MetadataJSON string    `json:"metadata_json"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type BuilderPreviewSummaryResponse struct {
	Available         bool       `json:"available"`
	Status            string     `json:"status"`
	ResolvedRunID     string     `json:"resolved_run_id"`
	PublishedAt       *time.Time `json:"published_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	OutputRoot        string     `json:"output_root"`
	DefaultEntryPath  string     `json:"default_entry_path"`
	DownloadAvailable bool       `json:"download_available"`
	PreviewAvailable  bool       `json:"preview_available"`
	IsStale           bool       `json:"is_stale"`
	NewerRunID        string     `json:"newer_run_id"`
	NewerRunStatus    string     `json:"newer_run_status"`
}

type BuilderSessionPreviewResponse struct {
	Available         bool       `json:"available"`
	Status            string     `json:"status"`
	ResolvedRunID     string     `json:"resolved_run_id"`
	PublishedAt       *time.Time `json:"published_at,omitempty"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	OutputRoot        string     `json:"output_root"`
	DefaultEntryPath  string     `json:"default_entry_path"`
	DownloadAvailable bool       `json:"download_available"`
	PreviewAvailable  bool       `json:"preview_available"`
	IsStale           bool       `json:"is_stale"`
	NewerRunID        string     `json:"newer_run_id"`
	NewerRunStatus    string     `json:"newer_run_status"`
	DownloadURL       string     `json:"download_url"`
	PreviewLaunchURL  string     `json:"preview_launch_url"`
}

type BuilderPreviewLaunchResponse struct {
	FrameURL string `json:"frame_url"`
}

type BuilderSessionDetailResponse struct {
	Session   BuilderSessionResponse           `json:"session"`
	Messages  []BuilderMessageResponse         `json:"messages"`
	Runs      []BuilderRunResponse             `json:"runs"`
	Workspace *BuilderWorkspaceSummaryResponse `json:"workspace,omitempty"`
	Preview   *BuilderPreviewSummaryResponse   `json:"preview,omitempty"`
	Artifacts []BuilderArtifactSummaryResponse `json:"artifacts"`
}

type BuilderSessionDetailRow struct {
	ID                            string     `gorm:"column:id"`
	ProjectID                     string     `gorm:"column:project_id"`
	BuildEnvID                    string     `gorm:"column:build_env_id"`
	Title                         string     `gorm:"column:title"`
	Summary                       string     `gorm:"column:summary"`
	Status                        string     `gorm:"column:status"`
	CreatedBy                     string     `gorm:"column:created_by"`
	CreatedAt                     time.Time  `gorm:"column:created_at"`
	UpdatedAt                     time.Time  `gorm:"column:updated_at"`
	LastActivityAt                time.Time  `gorm:"column:last_activity_at"`
	ExpiresAt                     *time.Time `gorm:"column:expires_at"`
	LatestRunID                   string     `gorm:"column:latest_run_id"`
	LatestRunStatus               string     `gorm:"column:latest_run_status"`
	CurrentWorkspaceID            string     `gorm:"column:current_workspace_id"`
	CurrentWorkspaceBuildEnvID    string     `gorm:"column:current_workspace_build_env_id"`
	CurrentWorkspaceClusterID     string     `gorm:"column:current_workspace_cluster_id"`
	CurrentWorkspaceNamespace     string     `gorm:"column:current_workspace_namespace"`
	CurrentWorkspacePodName       string     `gorm:"column:current_workspace_pod_name"`
	CurrentWorkspaceContainerName string     `gorm:"column:current_workspace_container_name"`
	CurrentWorkspaceStatus        string     `gorm:"column:current_workspace_status"`
	CurrentWorkspaceRoot          string     `gorm:"column:current_workspace_root"`
	CurrentWorkspaceTerminatedAt  *time.Time `gorm:"column:current_workspace_terminated_at"`
	CurrentWorkspaceCreatedAt     *time.Time `gorm:"column:current_workspace_created_at"`
	CurrentWorkspaceUpdatedAt     *time.Time `gorm:"column:current_workspace_updated_at"`
}

type BuilderSessionListRow struct {
	ID                     string     `gorm:"column:id"`
	ProjectID              string     `gorm:"column:project_id"`
	BuildEnvID             string     `gorm:"column:build_env_id"`
	Title                  string     `gorm:"column:title"`
	Summary                string     `gorm:"column:summary"`
	Status                 string     `gorm:"column:status"`
	CreatedBy              string     `gorm:"column:created_by"`
	CreatedAt              time.Time  `gorm:"column:created_at"`
	UpdatedAt              time.Time  `gorm:"column:updated_at"`
	LastActivityAt         time.Time  `gorm:"column:last_activity_at"`
	ExpiresAt              *time.Time `gorm:"column:expires_at"`
	LatestRunID            string     `gorm:"column:latest_run_id"`
	LatestRunStatus        string     `gorm:"column:latest_run_status"`
	CurrentWorkspaceID     string     `gorm:"column:current_workspace_id"`
	CurrentWorkspaceStatus string     `gorm:"column:current_workspace_status"`
	CurrentWorkspaceRoot   string     `gorm:"column:current_workspace_root"`
	ArtifactCount          int64      `gorm:"column:artifact_count"`
}

type BuilderArtifactSummaryRow struct {
	ID           string    `gorm:"column:id"`
	SessionID    string    `gorm:"column:session_id"`
	WorkspaceID  string    `gorm:"column:workspace_id"`
	RunID        string    `gorm:"column:run_id"`
	Kind         string    `gorm:"column:kind"`
	Path         string    `gorm:"column:path"`
	MetadataJSON string    `gorm:"column:metadata_json"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	UpdatedAt    time.Time `gorm:"column:updated_at"`
}
