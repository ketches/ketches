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

	gateway, err := services.CreateAppGateway(appID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
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

	gateway, err := services.UpdateAppGateway(id, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.Error(c, http.StatusNotFound, err)
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
	if err := services.DeleteAppGateway(id); err != nil {
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
	var domain *string
	var path *string
	var gatewayPort *int
	var certID *string
	var exposed *bool

	switch g := gateway.(type) {
	case *models.CreateGatewayRequest:
		port = &g.Port
		protocol = &g.Protocol
		domain = &g.Domain
		path = &g.Path
		gatewayPort = &g.GatewayPort
		certID = &g.CertID
		exposed = &g.Exposed
	case *models.UpdateGatewayRequest:
		port = &g.Port
		protocol = &g.Protocol
		domain = &g.Domain
		path = &g.Path
		gatewayPort = &g.GatewayPort
		certID = &g.CertID
		exposed = &g.Exposed
	default:
		return errors.New("invalid gateway request type")
	}

	// Port validation
	if *port < 1 || *port > 65535 {
		return errors.New("port must be between 1 and 65535")
	}

	// Protocol validation
	proto := strings.ToLower(*protocol)
	if proto != "http" && proto != "https" && proto != "tcp" && proto != "udp" {
		return errors.New("protocol must be one of: http, https, tcp, udp")
	}
	*protocol = proto

	// Only validate routing fields when exposed is true
	if *exposed {
		isHTTPProtocol := proto == "http" || proto == "https"

		if isHTTPProtocol {
			// HTTP/HTTPS requires domain and path
			if *domain == "" {
				return errors.New("domain is required for HTTP/HTTPS protocols when exposed")
			}
			if *path == "" {
				*path = "/"
			}
			if !strings.HasPrefix(*path, "/") {
				return errors.New("path must start with /")
			}
			// Clear TCP/UDP fields
			*gatewayPort = 0
		} else {
			// TCP/UDP requires gateway port
			if *gatewayPort < 1 || *gatewayPort > 65535 {
				return errors.New("gateway_port is required and must be between 1 and 65535 for TCP/UDP when exposed")
			}
			// Clear HTTP/HTTPS fields
			*domain = ""
			*path = ""
			*certID = ""
		}
	} else {
		// When not exposed, clear all routing fields
		*domain = ""
		*path = ""
		*gatewayPort = 0
		*certID = ""
	}

	return nil
}
