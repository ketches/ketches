package app

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
