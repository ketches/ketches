package models

type Probe struct {
	InitialDelaySeconds int64  `json:"initialDelaySeconds"`
	PeriodSeconds       int64  `json:"periodSeconds"`
	TimeoutSeconds      int64  `json:"timeoutSeconds"`
	SuccessThreshold    int64  `json:"successThreshold"`
	FailureThreshold    int64  `json:"failureThreshold"`
	ProbeMode           string `json:"probeMode" validate:"required,oneof=httpGet tcpSocket exec"`
	HTTPGetPath         string `json:"httpGetPath,omitempty"`
	HTTPGetPort         int    `json:"httpGetPort,omitempty"`
	TCPSocketPort       int    `json:"tcpSocketPort,omitempty"`
	ExecCommand         string `json:"exec,omitempty"`
}

type HTTPGetAction struct {
	Path    string            `json:"path"`
	Port    int               `json:"port"`
	Host    string            `json:"host,omitempty"`
	Scheme  string            `json:"scheme,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type TCPSocketAction struct {
	Port int    `json:"port"`
	Host string `json:"host,omitempty"`
}

type AppProbeModel struct {
	ProbeID string `json:"probeID" gorm:"column:id"`
	AppID   string `json:"appID"`
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
	Probe
}

type ListAppProbesRequest struct {
	AppID string `uri:"appID" binding:"required"`
}

type CreateAppProbeRequest struct {
	AppID   string `json:"-" uri:"appID"`
	Type    string `json:"type" validate:"required,oneof=liveness readiness startup"`
	Enabled bool   `json:"enabled"`
	Probe
}

type UpdateAppProbeRequest struct {
	AppID   string `json:"-" uri:"appID"`
	ProbeID string `json:"-" uri:"probeID"`
	Enabled bool   `json:"enabled"`
	Probe
}

type DeleteAppProbeRequest struct {
	AppID   string `json:"-" uri:"appID" binding:"required"`
	ProbeID string `json:"-" uri:"probeID" binding:"required"`
}
