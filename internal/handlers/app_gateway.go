package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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


// ProxyGatewayHTTP reverse-proxies HTTP requests to the application via the
// Kubernetes API Server service proxy sub-resource.
// Route: GET|HEAD /api/v1/gateways/:gatewayID/proxy/*path
func ProxyGatewayHTTP(c *gin.Context) {
	gatewayID := c.Param("gatewayID")
	proxyPath := c.Param("path") // e.g. "/", "/healthz"

	// 1. Load gateway + app + cluster
	gateway, application, err := services.GetGatewayWithApp(gatewayID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.Error(c, http.StatusNotFound, err)
			return
		}
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	// 2. Validate protocol
	proto := strings.ToLower(gateway.Protocol)
	if proto != "http" && proto != "https" {
		api.Error(c, http.StatusBadRequest, fmt.Errorf("quick access is only available for HTTP/HTTPS gateways"))
		return
	}

	// 3. Validate app status
	status := services.GetAppStatus(context.Background(), application)
	if status != "running" && status != "updating" {
		api.Error(c, http.StatusBadRequest, fmt.Errorf("quick access is only available when the app is running or updating (current: %s)", status))
		return
	}

	// 4. Build K8s REST config from cluster KubeConfig
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(application.Env.Cluster.KubeConfig))
	if err != nil {
		api.Error(c, http.StatusInternalServerError, fmt.Errorf("failed to build cluster config: %w", err))
		return
	}

	// 5. Build TLS-aware HTTP client
	tlsCfg, err := rest.TLSConfigFor(restConfig)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, fmt.Errorf("failed to build TLS config: %w", err))
		return
	}
	httpClient := &http.Client{
		Transport: &http.Transport{TLSClientConfig: tlsCfg},
	}

	// 6. Construct target URL
	k8sHost := strings.TrimSuffix(restConfig.Host, "/")
	ns := application.Env.ClusterNamespace
	svcName := application.Slug
	port := gateway.Port
	// Ensure path starts with /
	if !strings.HasPrefix(proxyPath, "/") {
		proxyPath = "/" + proxyPath
	}
	rawQuery := c.Request.URL.RawQuery
	// Remove the auth token from the forwarded query string
	if rawQuery != "" {
		qv, _ := url.ParseQuery(rawQuery)
		qv.Del("token")
		rawQuery = qv.Encode()
	}
	targetURL := fmt.Sprintf("%s/api/v1/namespaces/%s/services/%s:%d/proxy%s", k8sHost, ns, svcName, port, proxyPath)
	if rawQuery != "" {
		targetURL += "?" + rawQuery
	}

	// 7. Forward request
	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, fmt.Errorf("failed to build proxy request: %w", err))
		return
	}

	// Forward safe request headers (skip hop-by-hop)
	hopByHop := map[string]bool{
		"connection": true, "keep-alive": true, "proxy-authenticate": true,
		"proxy-authorization": true, "te": true, "trailers": true,
		"transfer-encoding": true, "upgrade": true, "authorization": true,
	}
	for key, vals := range c.Request.Header {
		if !hopByHop[strings.ToLower(key)] {
			for _, v := range vals {
				req.Header.Add(key, v)
			}
		}
	}

	// 8. Execute and stream response
	resp, err := httpClient.Do(req)
	if err != nil {
		api.Error(c, http.StatusBadGateway, fmt.Errorf("upstream request failed: %w", err))
		return
	}
	defer resp.Body.Close()

	// Forward response headers
	for key, vals := range resp.Header {
		if !hopByHop[strings.ToLower(key)] {
			for _, v := range vals {
				c.Header(key, v)
			}
		}
	}
	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}