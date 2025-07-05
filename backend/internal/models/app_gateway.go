package models

type AppGatewayModel struct {
	AppID       string `json:"appID"`
	GatewayID   string `json:"gatewayID"`
	Port        int32  `json:"port"`
	Protocol    string `json:"protocol"`
	Domain      string `json:"domain,omitempty"`
	Path        string `json:"path,omitempty"`
	CertID      string `json:"certID,omitempty"`
	GatewayPort int32  `json:"gatewayPort" binding:"required,min=1,max=65535"`
	Exposed     bool   `json:"exposed"`
}

type ListAppGatewaysRequest struct {
	AppID string `json:"-" uri:"appID" binding:"required"`
}

type CreateAppGatewayRequest struct {
	AppID       string `json:"-" uri:"appID"`
	Port        int32  `json:"port" binding:"required,min=1,max=65535"`
	Protocol    string `json:"protocol" binding:"required,oneof=http https tcp udp"`
	Domain      string `json:"domain,omitempty"`
	Path        string `json:"path,omitempty"`
	CertID      string `json:"certID,omitempty"`
	GatewayPort int32  `json:"gatewayPort"`
	Exposed     bool   `json:"exposed"`
}

type UpdateAppGatewayRequest struct {
	AppID       string `json:"-" uri:"appID"`
	GatewayID   string `json:"-" uri:"gatewayID"`
	Port        int32  `json:"port" binding:"required,min=1,max=65535"`
	Protocol    string `json:"protocol" binding:"required,oneof=http https tcp udp"`
	Domain      string `json:"domain,omitempty"`
	Path        string `json:"path,omitempty"`
	CertID      string `json:"certID"`
	GatewayPort int32  `json:"gatewayPort"`
	Exposed     bool   `json:"exposed"`
}

type ToggleAppGatewayExposedRequest struct {
	AppID     string `json:"-" uri:"appID"`
	GatewayID string `json:"-" uri:"gatewayID"`
	Exposed   bool   `json:"exposed"`
}

type DeleteAppGatewaysRequest struct {
	AppID      string   `json:"-" uri:"appID"`
	GatewayIDs []string `json:"gatewayIDs" binding:"required"`
}
