package app

// AppStatus represents the current state of an application
type AppStatus = string

const (
	AppStatusUndeployed AppStatus = "undeployed"
	AppStatusStarting   AppStatus = "starting"
	AppStatusRunning    AppStatus = "running"
	AppStatusStopped    AppStatus = "stopped"
	AppStatusStopping   AppStatus = "stopping"
	AppStatusUpdating   AppStatus = "updating"
	AppStatusAbnormal   AppStatus = "abnormal"
	AppStatusCompleted  AppStatus = "completed"
	AppStatusDebugging  AppStatus = "debugging"
	AppStatusUnknown    AppStatus = "unknown"
)

// AppAction represents an operation that can be performed on an application
type AppAction = string

const (
	AppActionDeploy   AppAction = "deploy"
	AppActionStart    AppAction = "start"
	AppActionStop     AppAction = "stop"
	AppActionRollback AppAction = "rollback"
	AppActionUpdate   AppAction = "update"
	AppActionRedeploy AppAction = "redeploy"
	AppActionDebug    AppAction = "debug"
	AppActionDebugOff AppAction = "debugOff"
	AppActionDelete   AppAction = "delete"
)

// UserRole represents the system-level role of a user
type UserRole = string

const (
	UserRoleAdmin UserRole = "admin"
	UserRoleUser  UserRole = "user"
)

// ProjectRole represents the role of a user within a project
type ProjectRole = string

const (
	ProjectRoleOwner     ProjectRole = "owner"
	ProjectRoleDeveloper ProjectRole = "developer"
	ProjectRoleViewer    ProjectRole = "viewer"
)

// AppType represents the type of application in Kubernetes (e.g., Deployment, StatefulSet)
type AppType = string

const (
	AppTypeDeployment  AppType = "Deployment"
	AppTypeStatefulSet AppType = "StatefulSet"
)

// VolumeType represents the type of volume used by an application
type VolumeType = string

const (
	VolumeTypePVC      VolumeType = "pvc"
	VolumeTypeHostPath VolumeType = "hostPath"
	VolumeTypeEmptyDir VolumeType = "emptyDir"
)
