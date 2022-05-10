package gateway

func CreateGateway(req CreateGatewayRequest) (CreateGatewayResponse, error) {
	// todo
	// if istio extension is installed, then use istio-ingressGateway
	// or use nginx-ingress

	gateway := &entities.Gateway{
		Name:           req.Name,
		Namespace:      istio.SystemNamespace,
		Description:    req.Description,
		LastOperatorId: req.LastOperatorId,
		ClusterId:      req.ClusterId,
	}
	repos.NsoKube.Create(gateway)

	istioClient, _ := istio.Client(req.ClusterId)

	gatewayClient := istio.NewGatewayClient(istioClient)
	gatewayClient.ApplyGateway(kube.ApplyGatewayModel{
		Name:      gateway.Name,
		Namespace: gateway.Namespace,
		Labels: map[string]string{
			"app": gateway.Name,
		},
		Annotations: map[string]string{},
		Hosts: []kube.GatewayHostModel{
			{
				Host: "example.com",
				Services: []kube.HostedService{
					{
						Name: "application-name",
						Path: "/api",
						Port: "80",
					},
				},
			},
		},
	})
}
