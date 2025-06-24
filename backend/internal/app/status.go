package app

type AppStatus = string

const (
	AppStatusUndeployed    AppStatus = "undeployed"
	AppStatusStarting      AppStatus = "starting"
	AppStatusRunning       AppStatus = "running"
	AppStatusStopped       AppStatus = "stopped"
	AppStatusStopping      AppStatus = "stopping"
	AppStatusRollingUpdate AppStatus = "rollingUpdate"
	AppStatusAbnormal      AppStatus = "abnormal"
	AppStatusCompleted     AppStatus = "completed"
	AppStatusUnknown       AppStatus = "unknown"
)
