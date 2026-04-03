package handlers

import (
	"testing"

	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateGatewayRequestRejectsExposedTCP(t *testing.T) {
	req := &models.CreateGatewayRequest{
		Port:        8080,
		Protocol:    "tcp",
		Exposed:     true,
		GatewayPort: 30080,
		ServiceType: "ClusterIP",
	}

	err := validateGatewayRequest(req)

	require.Error(t, err)
	assert.Equal(t, "public access is currently supported only for HTTP/HTTPS gateways", err.Error())
}

func TestValidateGatewayRequestRejectsExposedUDP(t *testing.T) {
	req := &models.CreateGatewayRequest{
		Port:        8080,
		Protocol:    "udp",
		Exposed:     true,
		GatewayPort: 30080,
		ServiceType: "ClusterIP",
	}

	err := validateGatewayRequest(req)

	require.Error(t, err)
	assert.Equal(t, "public access is currently supported only for HTTP/HTTPS gateways", err.Error())
}

func TestValidateGatewayRequestAllowsExposedHTTP(t *testing.T) {
	req := &models.CreateGatewayRequest{
		Port:        8080,
		Protocol:    "http",
		Exposed:     true,
		Domain:      "example.com",
		Path:        "/",
		ServiceType: "ClusterIP",
	}

	err := validateGatewayRequest(req)

	require.NoError(t, err)
	assert.Equal(t, "http", req.Protocol)
	assert.Equal(t, "/", req.Path)
}

func TestValidateGatewayRequestAllowsExposedHTTPS(t *testing.T) {
	req := &models.UpdateGatewayRequest{
		Port:        8443,
		Protocol:    "https",
		Exposed:     true,
		Domain:      "secure.example.com",
		Path:        "/",
		ServiceType: "ClusterIP",
	}

	err := validateGatewayRequest(req)

	require.NoError(t, err)
	assert.Equal(t, "https", req.Protocol)
}

func TestValidateGatewayRequestAllowsNonExposedTCP(t *testing.T) {
	req := &models.CreateGatewayRequest{
		Port:        8080,
		Protocol:    "tcp",
		Exposed:     false,
		GatewayPort: 30080,
		ServiceType: "ClusterIP",
	}

	err := validateGatewayRequest(req)

	require.NoError(t, err)
	assert.Equal(t, 0, req.GatewayPort)
}
