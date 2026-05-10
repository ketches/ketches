package handlers

import (
	"testing"

	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateGatewayRequestRejectsTCPRoutes(t *testing.T) {
	req := &models.CreateGatewayRequest{
		Port:        8080,
		Protocol:    "tcp",
		GatewayPort: 30080,
		ServiceType: "ClusterIP",
		Routes: []models.GatewayRouteSpec{
			{Host: "example.com", ListenerProtocol: "http", Path: "/", Enabled: true},
		},
	}

	err := validateGatewayRequest(req)

	require.Error(t, err)
	assert.Equal(t, "HTTP routes are only supported when gateway protocol is http", err.Error())
}

func TestValidateGatewayRequestRejectsUDPRoutes(t *testing.T) {
	req := &models.CreateGatewayRequest{
		Port:        8080,
		Protocol:    "udp",
		GatewayPort: 30080,
		ServiceType: "ClusterIP",
		Routes: []models.GatewayRouteSpec{
			{Host: "example.com", ListenerProtocol: "http", Path: "/", Enabled: true},
		},
	}

	err := validateGatewayRequest(req)

	require.Error(t, err)
	assert.Equal(t, "HTTP routes are only supported when gateway protocol is http", err.Error())
}

func TestValidateGatewayRequestAllowsHTTPRoutes(t *testing.T) {
	req := &models.CreateGatewayRequest{
		Port:        8080,
		Protocol:    "http",
		ServiceType: "ClusterIP",
		Routes: []models.GatewayRouteSpec{
			{Host: "example.com", ListenerProtocol: "http", Path: "", Enabled: true},
		},
	}

	err := validateGatewayRequest(req)

	require.NoError(t, err)
	assert.Equal(t, "http", req.Protocol)
	assert.Equal(t, "/", req.Routes[0].Path)
}

func TestValidateGatewayRequestAllowsHTTPSRoutes(t *testing.T) {
	req := &models.UpdateGatewayRequest{
		Port:        8443,
		Protocol:    "http",
		ServiceType: "ClusterIP",
		Routes: []models.GatewayRouteSpec{
			{Host: "secure.example.com", ListenerProtocol: "https", Path: "/", CertID: "cert-1", Enabled: true},
		},
	}

	err := validateGatewayRequest(req)

	require.NoError(t, err)
	assert.Equal(t, "http", req.Protocol)
	assert.Equal(t, "https", req.Routes[0].ListenerProtocol)
}

func TestValidateGatewayRequestAllowsTCPWithoutRoutes(t *testing.T) {
	req := &models.CreateGatewayRequest{
		Port:        8080,
		Protocol:    "tcp",
		GatewayPort: 30080,
		ServiceType: "ClusterIP",
	}

	err := validateGatewayRequest(req)

	require.NoError(t, err)
	assert.Equal(t, 30080, req.GatewayPort)
}

func TestValidateGatewayRequestRejectsHTTPSRouteWithoutCertificate(t *testing.T) {
	req := &models.CreateGatewayRequest{
		Port:        8443,
		Protocol:    "http",
		ServiceType: "ClusterIP",
		Routes: []models.GatewayRouteSpec{
			{Host: "secure.example.com", ListenerProtocol: "https", Path: "/", Enabled: true},
		},
	}

	err := validateGatewayRequest(req)

	require.Error(t, err)
	assert.Equal(t, "certificate is required for HTTPS routes", err.Error())
}
