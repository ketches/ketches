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
