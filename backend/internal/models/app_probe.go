package models

type Probe struct {
	ProbeMode           string `json:"probeMode,omitempty" validate:"required,oneof=httpGet tcpSocket exec"`
	HTTPGetPath         string `json:"httpGetPath,omitempty"`
	HTTPGetPort         int    `json:"httpGetPort,omitempty"`
	TCPSocketPort       int    `json:"tcpSocketPort,omitempty"`
	ExecCommand         string `json:"execCommand,omitempty"`
	InitialDelaySeconds int32  `json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       int32  `json:"periodSeconds,omitempty"`
	TimeoutSeconds      int32  `json:"timeoutSeconds,omitempty"`
	SuccessThreshold    int32  `json:"successThreshold,omitempty"`
	FailureThreshold    int32  `json:"failureThreshold,omitempty"`
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
	AppID string `json:"-" uri:"appID"`
	Type  string `json:"type" validate:"required,oneof=liveness readiness startup"`
	Probe
	Enabled bool `json:"enabled"`
}

type UpdateAppProbeRequest struct {
	AppID   string `json:"-" uri:"appID"`
	ProbeID string `json:"-" uri:"probeID"`
	Type    string `json:"type" validate:"required,oneof=liveness readiness startup"`
	Probe
	Enabled bool `json:"enabled"`
}

type ToggleAppProbeRequest struct {
	AppID   string `json:"-" uri:"appID"`
	ProbeID string `json:"-" uri:"probeID"`
	Enabled bool   `json:"enabled"`
}

type DeleteAppProbeRequest struct {
	AppID   string `json:"-" uri:"appID" binding:"required"`
	ProbeID string `json:"-" uri:"probeID" binding:"required"`
}
