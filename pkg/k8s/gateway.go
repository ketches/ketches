package k8s

type ApplyGatewayModel struct {
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
	Hosts       []GatewayHostModel
}

type GatewayHostModel struct {
	Host     string
	Services []HostedService
}

type HostedService struct {
	Name string
	Port string
	Path string
}

type DeleteGatewayModel struct {
	Name      string
	Namespace string
}

type CreateGatewayHostModel struct {
	GatewayName string
	Namespace   string
	Host        string
}

type UpdateGatewayHostModel struct {
	GatewayName string
	Namespace   string
	Host        string
}

type DeleteGatewayHostModel struct {
	GatewayName string
	Namespace   string
	Host        string
}

type CreateGatewayHostRouteModel struct {
}

type UpdateGatewayHostRouteModel struct {
}

type DeleteGatewayHostRouteModel struct {
}

type GatewayClient interface {
	CreateGateway(gateway *ApplyGatewayModel) error
	UpdateGateway(gateway *ApplyGatewayModel) error
	DeleteGateway(gateway *DeleteGatewayModel) error
	CreateGatewayHost(host *CreateGatewayHostModel) error
	UpdateGatewayHost(host *UpdateGatewayHostModel) error
	DeleteGatewayHost(host *DeleteGatewayHostModel) error

	CreateGatewayHostRoute(route *CreateGatewayHostRouteModel) error
	UpdateGatewayHostRoute(route *UpdateGatewayHostRouteModel) error
	DeleteGatewayHostRoute(route *DeleteGatewayHostRouteModel) error
}
