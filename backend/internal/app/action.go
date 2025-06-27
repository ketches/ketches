package app

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
