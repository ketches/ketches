package app

type AppType = string

const (
	AppTypeDeployment  AppType = "Deployment"
	AppTypeStatefulSet AppType = "StatefulSet"
)

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

type AppGatewayProtocol = string

const (
	AppGatewayProtocolHTTP  AppGatewayProtocol = "http"
	AppGatewayProtocolHTTPS AppGatewayProtocol = "https"
	AppGatewayProtocolTCP   AppGatewayProtocol = "tcp"
	AppGatewayProtocolUDP   AppGatewayProtocol = "udp"
)

type AppProbeType = string

const (
	AppProbeTypeLiveness  AppProbeType = "liveness"
	AppProbeTypeReadiness AppProbeType = "readiness"
	AppProbeTypeStartup   AppProbeType = "startup"
)

type AppProbeMode = string

const (
	AppProbeModeHTTPGet   AppProbeMode = "httpGet"
	AppProbeModeTCPSocket AppProbeMode = "tcpSocket"
	AppProbeModeExec      AppProbeMode = "exec"
)

const (
	SchedulingRuleTypeNodeName     = "nodeName"
	SchedulingRuleTypeNodeSelector = "nodeSelector"
	SchedulingRuleTypeNodeAffinity = "nodeAffinity"
)

const (
	SchedulingRuleTolerationOperatorEqual    = "Equal"
	SchedulingRuleTolerationOperatorNotEqual = "NotEqual"
	SchedulingRuleTolerationOperatorExists   = "Exists"
)

const (
	SchedulingRuleTolerationEffectNoSchedule       = "NoSchedule"
	SchedulingRuleTolerationEffectNoExecute        = "NoExecute"
	SchedulingRuleTolerationEffectPreferNoSchedule = "PreferNoSchedule"
)
