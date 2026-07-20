package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// ListAppGateways lists all gateways for an app
func ListAppGateways(c *gin.Context) {
	appID := c.Param("appID")
	gateways, err := services.ListAppGateways(appID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, gateways)
}

// CreateAppGateway creates a new gateway for an app
func CreateAppGateway(c *gin.Context) {
	appID := c.Param("appID")
	var req models.CreateGatewayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	// Validate gateway fields based on protocol and exposed state
	if err := validateGatewayRequest(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	gateway, err := services.CreateAppGateway(c.Request.Context(), appID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") || errors.Is(err, services.ErrInvalidGatewayCertificate) {
			api.Error(c, http.StatusBadRequest, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, gateway)
}

// UpdateAppGateway updates a gateway
func UpdateAppGateway(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateGatewayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	// Validate gateway fields based on protocol and exposed state
	if err := validateGatewayRequest(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	gateway, err := services.UpdateAppGateway(c.Request.Context(), id, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.Error(c, http.StatusNotFound, err)
			return
		}
		if errors.Is(err, services.ErrInvalidGatewayCertificate) {
			api.Error(c, http.StatusBadRequest, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, gateway)
}

// DeleteAppGateway deletes a gateway
func DeleteAppGateway(c *gin.Context) {
	id := c.Param("id")
	if err := services.DeleteAppGateway(c.Request.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.Error(c, http.StatusNotFound, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

// validateGatewayRequest validates gateway configuration based on protocol and exposed state
func validateGatewayRequest(gateway any) error {
	var port *int
	var protocol *string
	var serviceType *string
	var nodePort *int
	var routes *[]models.GatewayRouteSpec

	switch g := gateway.(type) {
	case *models.CreateGatewayRequest:
		port = &g.Port
		protocol = &g.Protocol
		serviceType = &g.ServiceType
		nodePort = &g.NodePort
		routes = &g.Routes
	case *models.UpdateGatewayRequest:
		port = &g.Port
		protocol = &g.Protocol
		serviceType = &g.ServiceType
		nodePort = &g.NodePort
		routes = &g.Routes
	default:
		return errors.New("invalid gateway request type")
	}

	// Port validation
	if *port < 1 || *port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}

	// Protocol validation
	proto := strings.ToLower(*protocol)
	if proto != "http" && proto != "tcp" && proto != "udp" {
		return errors.New("protocol must be one of: http, tcp, udp")
	}
	*protocol = proto

	// ServiceType validation and NodePort range check
	if *serviceType == "" {
		*serviceType = "ClusterIP"
	}
	if *serviceType != "ClusterIP" && *serviceType != "NodePort" {
		return errors.New("service_type must be ClusterIP or NodePort")
	}
	if *serviceType == "NodePort" && *nodePort != 0 {
		if *nodePort < 30000 || *nodePort > 32767 {
			return errors.New("node_port must be between 30000 and 32767")
		}
	}
	if *serviceType != "NodePort" {
		*nodePort = 0
	}

	if proto != "http" && len(*routes) > 0 {
		return errors.New("HTTP routes are only supported when gateway protocol is http")
	}
	if proto == "http" {
		for i := range *routes {
			route := &(*routes)[i]
			route.ListenerProtocol = strings.ToLower(strings.TrimSpace(route.ListenerProtocol))
			if route.ListenerProtocol == "" {
				route.ListenerProtocol = "http"
			}
			if route.ListenerProtocol != "http" && route.ListenerProtocol != "https" {
				return errors.New("route listener_protocol must be http or https")
			}
			if route.Enabled && strings.TrimSpace(route.Host) == "" {
				return errors.New("route host is required when enabled")
			}
			if route.Path == "" {
				route.Path = "/"
			}
			if !strings.HasPrefix(route.Path, "/") {
				return errors.New("route path must start with /")
			}
			if route.PathMatchType == "" {
				route.PathMatchType = "PathPrefix"
			}
			if route.PathMatchType != "PathPrefix" && route.PathMatchType != "Exact" {
				return errors.New("route path_match_type must be PathPrefix or Exact")
			}
			if route.Enabled && route.ListenerProtocol == "https" && strings.TrimSpace(route.CertID) == "" {
				return errors.New("certificate is required for HTTPS routes")
			}
		}
	}

	return nil
}
