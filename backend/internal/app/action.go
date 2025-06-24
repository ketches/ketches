package app

type AppAction = string

const (
	AppActionDeploy        AppAction = "deploy"
	AppActionStart         AppAction = "start"
	AppActionStop          AppAction = "stop"
	AppActionRollback      AppAction = "rollback"
	AppActionRollingUpdate AppAction = "rollingUpdate"
	AppActionRedeploy      AppAction = "redeploy"
)
