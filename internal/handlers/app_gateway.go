package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/kube"
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

// ProxyGatewayHTTP reverse-proxies HTTP requests to the application via the
// Kubernetes API Server service proxy sub-resource.
// Route: GET|HEAD /api/v1/gateways/:gatewayID/proxy/*path
// Route: GET|HEAD /forward/:gatewayID/*path
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

	// 3. Get the cached HTTP client for this cluster from the global store.
	// The client reuses TCP/TLS connections to the K8s apiserver across requests.
	httpClient, err := kube.GlobalClusterStore.GetHTTPProxyClient(application.Env.Cluster.ID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	// 4. Construct target URL using the stored ApiServer address.
	k8sHost := strings.TrimSuffix(application.Env.Cluster.ApiServer, "/")
	ns := application.Env.ClusterNamespace
	svcName := application.Slug
	port := gateway.Port
	// Ensure path starts with /.
	if !strings.HasPrefix(proxyPath, "/") {
		proxyPath = "/" + proxyPath
	}
	rawQuery := c.Request.URL.RawQuery
	// Remove the auth token from the forwarded query string.
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

	// Forward safe request headers (skip hop-by-hop and Ketches-internal headers)
	// Forward safe request headers (skip hop-by-hop and Ketches-internal headers).
	// Drop Accept-Encoding so the upstream returns a plain (non-compressed) body that
	// we can inspect and rewrite before forwarding to the browser.
	hopByHop := map[string]bool{
		"connection": true, "keep-alive": true, "proxy-authenticate": true,
		"proxy-authorization": true, "te": true, "trailers": true,
		"transfer-encoding": true, "upgrade": true, "authorization": true,
		"accept-encoding": true, // prevent compressed response we cannot rewrite
	}
	for key, vals := range c.Request.Header {
		if hopByHop[strings.ToLower(key)] {
			continue
		}
		// Strip the Ketches auth cookie so it is never forwarded to the upstream app.
		if strings.ToLower(key) == "cookie" {
			filtered := filterCookie(vals, "X-Ketches-Token")
			if len(filtered) > 0 {
				req.Header[key] = filtered
			}
			continue
		}
		for _, v := range vals {
			req.Header.Add(key, v)
		}
	}

	// 8. Execute and stream response
	resp, err := httpClient.Do(req)
	if err != nil {
		api.Error(c, http.StatusBadGateway, fmt.Errorf("upstream request failed: %w", err))
		return
	}
	defer resp.Body.Close()

	// The /forward prefix for this gateway — used when rewriting Location headers
	// and injecting <base href> into HTML responses.
	forwardPrefix := "/forward/" + gatewayID
	contentType := resp.Header.Get("Content-Type")
	isHTML := strings.Contains(strings.ToLower(contentType), "text/html")
	willRewriteHTML := isHTML && c.Request.Method != http.MethodHead

	// Forward response headers, rewriting Location for redirect responses.
	for key, vals := range resp.Header {
		klower := strings.ToLower(key)
		if hopByHop[klower] {
			continue
		}
		if willRewriteHTML && klower == "content-length" {
			continue
		}
		if klower == "location" {
			for _, v := range vals {
				c.Header(key, rewriteLocation(v, forwardPrefix))
			}
			continue
		}
		for _, v := range vals {
			c.Header(key, v)
		}
	}
	c.Status(resp.StatusCode)

	// For HTML responses, buffer the body and inject <base href> so that
	// root-relative asset/link paths resolve correctly through the proxy.
	if willRewriteHTML {
		body, _ := io.ReadAll(resp.Body)
		// Rewrite K8s apiserver proxy prefix → /forward/{gatewayID} so that
		// absolute links already rewritten by kube-apiserver resolve correctly.
		k8sProxyPrefix := fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy", ns, svcName, port)
		body = bytes.ReplaceAll(body, []byte(k8sProxyPrefix), []byte(forwardPrefix))
		body = injectBaseHref(body, forwardPrefix+"/")
		c.Header("Content-Length", fmt.Sprintf("%d", len(body)))
		c.Writer.Write(body) //nolint:errcheck
	} else {
		io.Copy(c.Writer, resp.Body) //nolint:errcheck
	}
}

// filterCookie removes a named cookie from a slice of raw Cookie header values.
// Each element in vals is a full Cookie header line (may contain multiple name=value pairs).
func filterCookie(vals []string, name string) []string {
	var out []string
	for _, line := range vals {
		var kept []string
		for _, part := range strings.Split(line, ";") {
			part = strings.TrimSpace(part)
			if kv := strings.SplitN(part, "=", 2); len(kv) > 0 && strings.TrimSpace(kv[0]) == name {
				continue // drop this cookie
			}
			if part != "" {
				kept = append(kept, part)
			}
		}
		if len(kept) > 0 {
			out = append(out, strings.Join(kept, "; "))
		}
	}
	return out
}

// rewriteLocation rewrites an absolute-path Location header to include the
// /forward/{gatewayID} prefix so the browser stays within the proxy scope.
func rewriteLocation(location, forwardPrefix string) string {
	if strings.HasPrefix(location, "/") && !strings.HasPrefix(location, forwardPrefix) {
		return forwardPrefix + location
	}
	return location
}

// injectBaseHref inserts a <base href="prefix"> tag immediately after <head>
// in an HTML document so relative and root-relative paths resolve correctly.
func injectBaseHref(body []byte, prefix string) []byte {
	baseTag := []byte(`<base href="` + prefix + `">`)
	// Try to inject after <head> (case-insensitive match).
	lower := bytes.ToLower(body)
	if idx := bytes.Index(lower, []byte("<head>")); idx != -1 {
		insertAt := idx + len("<head>")
		return append(body[:insertAt:insertAt], append(baseTag, body[insertAt:]...)...)
	}
	// Fallback: inject after <html ...> opening tag if no explicit <head>.
	if idx := bytes.Index(lower, []byte("<html")); idx != -1 {
		end := bytes.IndexByte(body[idx:], '>')
		if end != -1 {
			insertAt := idx + end + 1
			return append(body[:insertAt:insertAt], append(baseTag, body[insertAt:]...)...)
		}
	}
	return body
}
