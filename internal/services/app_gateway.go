package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

var ErrInvalidGatewayCertificate = errors.New("invalid gateway certificate")

var (
	syncGatewaysToK8s    = core.SyncGatewaysToK8s
	readNodePortsFromK8s = core.ReadNodePortsFromK8s
	deleteGatewayFromK8s = core.DeleteGatewayFromK8s
)

type gatewayCertificateError struct {
	message string
}

func (e gatewayCertificateError) Error() string {
	return ErrInvalidGatewayCertificate.Error() + ": " + e.message
}

func (e gatewayCertificateError) Unwrap() error {
	return ErrInvalidGatewayCertificate
}

func newGatewayCertificateError(message string) error {
	return gatewayCertificateError{message: message}
}

// ListAppGateways returns all gateways for a given app.
func ListAppGateways(appID string) ([]models.AppGatewayResponse, error) {
	appCtx, err := GetAppContext(context.Background(), appID)
	if err != nil {
		return nil, err
	}
	return buildGatewayResponses(appCtx), nil
}

// CreateAppGateway creates a new gateway for an app.
func CreateAppGateway(ctx context.Context, appID string, req *models.CreateGatewayRequest) (*models.AppGatewayResponse, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}

	normalized, err := normalizeCreateGatewayRequest(appCtx, req, "")
	if err != nil {
		return nil, err
	}

	var gateway entities.AppGateway
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		var existing entities.AppGateway
		err := tx.Where("app_id = ? AND port = ? AND protocol = ?", appID, normalized.Port, normalized.Protocol).First(&existing).Error
		if err == nil {
			return errors.New("gateway with this port and protocol already exists for this app")
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		gateway = entities.AppGateway{
			ID:          uuid.New(),
			AppID:       appID,
			Port:        normalized.Port,
			Protocol:    normalized.Protocol,
			GatewayPort: normalized.GatewayPort,
			ServiceType: normalized.ServiceType,
			NodePort:    normalized.NodePort,
		}
		if err := tx.Create(&gateway).Error; err != nil {
			return err
		}
		return replaceGatewayRoutes(tx, appCtx, &gateway, normalized.Routes)
	}); err != nil {
		return nil, err
	}

	return syncAndLoadGatewayResponse(ctx, appID, gateway.ID)
}

// UpdateAppGateway updates an existing gateway and replaces its nested HTTP routes.
func UpdateAppGateway(ctx context.Context, id string, req *models.UpdateGatewayRequest) (*models.AppGatewayResponse, error) {
	var gateway entities.AppGateway
	err := db.DB.First(&gateway, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("gateway not found")
		}
		return nil, err
	}

	appCtx, err := GetAppContext(ctx, gateway.AppID)
	if err != nil {
		return nil, err
	}

	normalized, err := normalizeUpdateGatewayRequest(appCtx, req, gateway.ID)
	if err != nil {
		return nil, err
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if normalized.Port != gateway.Port || normalized.Protocol != gateway.Protocol {
			var existing entities.AppGateway
			err := tx.Where("app_id = ? AND port = ? AND protocol = ? AND id != ?", gateway.AppID, normalized.Port, normalized.Protocol, id).First(&existing).Error
			if err == nil {
				return errors.New("gateway with this port and protocol already exists for this app")
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}

		gateway.Port = normalized.Port
		gateway.Protocol = normalized.Protocol
		gateway.GatewayPort = normalized.GatewayPort
		gateway.ServiceType = normalized.ServiceType
		gateway.NodePort = normalized.NodePort
		if err := tx.Save(&gateway).Error; err != nil {
			return err
		}
		return replaceGatewayRoutes(tx, appCtx, &gateway, normalized.Routes)
	}); err != nil {
		return nil, err
	}

	return syncAndLoadGatewayResponse(ctx, gateway.AppID, gateway.ID)
}

// DeleteAppGateway deletes a gateway and its nested route graph.
func DeleteAppGateway(ctx context.Context, id string) error {
	var gateway entities.AppGateway
	err := db.DB.First(&gateway, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("gateway not found")
		}
		return err
	}

	appCtx, err := GetAppContext(ctx, gateway.AppID)
	if err != nil {
		return err
	}
	if err := deleteGatewayFromK8s(ctx, appCtx, &gateway); err != nil {
		return err
	}

	return db.DB.Transaction(func(tx *gorm.DB) error {
		var routeIDs []string
		if err := tx.Model(&entities.AppGatewayHTTPRoute{}).Where("app_gateway_id = ?", gateway.ID).Pluck("id", &routeIDs).Error; err != nil {
			return err
		}
		if len(routeIDs) > 0 {
			if err := tx.Where("route_id IN ?", routeIDs).Delete(&entities.AppGatewayHTTPRouteBackend{}).Error; err != nil {
				return err
			}
		}
		if err := tx.Where("app_gateway_id = ?", gateway.ID).Delete(&entities.AppGatewayHTTPRoute{}).Error; err != nil {
			return err
		}
		return tx.Delete(&gateway).Error
	})
}

type normalizedGatewayRequest struct {
	Port        int
	Protocol    string
	GatewayPort int
	ServiceType string
	NodePort    int
	Routes      []models.GatewayRouteSpec
}

func normalizeCreateGatewayRequest(appCtx *models.AppContext, req *models.CreateGatewayRequest, currentGatewayID string) (*normalizedGatewayRequest, error) {
	return normalizeGatewayRequest(appCtx, req.Port, req.Protocol, req.GatewayPort, req.ServiceType, req.NodePort, req.Routes, currentGatewayID)
}

func normalizeUpdateGatewayRequest(appCtx *models.AppContext, req *models.UpdateGatewayRequest, currentGatewayID string) (*normalizedGatewayRequest, error) {
	return normalizeGatewayRequest(appCtx, req.Port, req.Protocol, req.GatewayPort, req.ServiceType, req.NodePort, req.Routes, currentGatewayID)
}

func normalizeGatewayRequest(appCtx *models.AppContext, port int, protocol string, gatewayPort int, serviceType string, nodePort int, routes []models.GatewayRouteSpec, currentGatewayID string) (*normalizedGatewayRequest, error) {
	if port < 1 || port > 65535 {
		return nil, errors.New("port must be between 1 and 65535")
	}

	protocol = strings.ToLower(strings.TrimSpace(protocol))
	if protocol != "http" && protocol != "tcp" && protocol != "udp" {
		return nil, errors.New("protocol must be one of: http, tcp, udp")
	}

	serviceType = strings.TrimSpace(serviceType)
	if serviceType == "" {
		serviceType = "ClusterIP"
	}
	if serviceType != "ClusterIP" && serviceType != "NodePort" {
		return nil, errors.New("service_type must be ClusterIP or NodePort")
	}
	if serviceType == "NodePort" && nodePort != 0 && (nodePort < 30000 || nodePort > 32767) {
		return nil, errors.New("node_port must be between 30000 and 32767")
	}
	if serviceType != "NodePort" {
		nodePort = 0
	}

	if protocol != "http" && len(routes) > 0 {
		return nil, errors.New("HTTP routes are only supported when gateway protocol is http")
	}

	normalizedRoutes := make([]models.GatewayRouteSpec, 0, len(routes))
	if protocol == "http" {
		seenRoutes := make(map[string]struct{}, len(routes))
		for _, route := range routes {
			normalized, err := normalizeGatewayRoute(appCtx, port, route, currentGatewayID)
			if err != nil {
				return nil, err
			}
			key := strings.Join([]string{
				strings.ToLower(normalized.Host),
				normalized.ListenerProtocol,
				normalized.PathMatchType,
				normalized.Path,
			}, "\x00")
			if _, ok := seenRoutes[key]; ok {
				return nil, errors.New("duplicate HTTP route for host, protocol, path match type, and path")
			}
			seenRoutes[key] = struct{}{}
			normalizedRoutes = append(normalizedRoutes, normalized)
		}
	}

	return &normalizedGatewayRequest{
		Port:        port,
		Protocol:    protocol,
		GatewayPort: gatewayPort,
		ServiceType: serviceType,
		NodePort:    nodePort,
		Routes:      normalizedRoutes,
	}, nil
}

func normalizeGatewayRoute(appCtx *models.AppContext, gatewayPort int, route models.GatewayRouteSpec, currentGatewayID string) (models.GatewayRouteSpec, error) {
	route.Host = strings.TrimSpace(strings.ToLower(route.Host))
	route.ListenerProtocol = strings.ToLower(strings.TrimSpace(route.ListenerProtocol))
	if route.ListenerProtocol == "" {
		route.ListenerProtocol = "http"
	}
	if route.ListenerProtocol != "http" && route.ListenerProtocol != "https" {
		return route, errors.New("route listener_protocol must be http or https")
	}
	route.Path = strings.TrimSpace(route.Path)
	if route.Path == "" {
		route.Path = "/"
	}
	if !strings.HasPrefix(route.Path, "/") {
		return route, errors.New("route path must start with /")
	}
	route.PathMatchType = strings.TrimSpace(route.PathMatchType)
	if route.PathMatchType == "" {
		route.PathMatchType = "PathPrefix"
	}
	if route.PathMatchType != "PathPrefix" && route.PathMatchType != "Exact" {
		return route, errors.New("route path_match_type must be PathPrefix or Exact")
	}

	if route.Enabled && route.Host == "" {
		return route, errors.New("route host is required when enabled")
	}
	if route.Enabled && route.ListenerProtocol == "https" {
		if err := validateGatewayRouteCertificateReference(appCtx, route.CertID); err != nil {
			return route, err
		}
		if err := validateHTTPSRouteCertificateConflict(appCtx, route.Host, route.CertID, currentGatewayID); err != nil {
			return route, err
		}
	}
	if route.Enabled && route.Timeouts != nil {
		if err := validateGatewayRouteTimeouts(route.Timeouts); err != nil {
			return route, err
		}
	}

	backends, err := normalizeGatewayRouteBackends(appCtx, gatewayPort, route.Backends)
	if err != nil {
		return route, err
	}
	route.Backends = backends
	return route, nil
}

func normalizeGatewayRouteBackends(appCtx *models.AppContext, gatewayPort int, backends []models.GatewayRouteBackendSpec) ([]models.GatewayRouteBackendSpec, error) {
	if len(backends) == 0 {
		return []models.GatewayRouteBackendSpec{{
			BackendAppID: appCtx.App.ID,
			BackendPort:  gatewayPort,
			Weight:       1,
		}}, nil
	}
	if len(backends) > 16 {
		return nil, errors.New("route may have no more than 16 backends")
	}

	hasPositiveWeight := false
	normalized := make([]models.GatewayRouteBackendSpec, len(backends))
	for i, backend := range backends {
		if backend.BackendAppID == "" && backend.BackendAppSlug != "" {
			appID, err := resolveBackendAppID(appCtx, backend.BackendAppSlug)
			if err != nil {
				return nil, err
			}
			backend.BackendAppID = appID
		}
		if backend.BackendAppID == "" {
			backend.BackendAppID = appCtx.App.ID
		}
		if backend.BackendPort == 0 {
			backend.BackendPort = gatewayPort
		}
		if backend.BackendPort < 1 || backend.BackendPort > 65535 {
			return nil, errors.New("backend_port must be between 1 and 65535")
		}
		if backend.Weight < 0 || backend.Weight > 1000000 {
			return nil, errors.New("backend weight must be between 0 and 1000000")
		}
		if backend.Weight > 0 {
			hasPositiveWeight = true
		}
		normalized[i] = backend
	}
	if !hasPositiveWeight {
		return nil, errors.New("at least one backend weight must be positive")
	}
	return normalized, nil
}

func resolveBackendAppID(appCtx *models.AppContext, backendAppSlug string) (string, error) {
	var backend entities.App
	err := db.DB.Where("env_id = ? AND slug = ?", appCtx.App.EnvID, strings.TrimSpace(backendAppSlug)).First(&backend).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", app.NewErrorf("backend app %q was not found in this environment", backendAppSlug)
		}
		return "", err
	}
	return backend.ID, nil
}

func validateGatewayRouteTimeouts(timeouts *models.GatewayRouteTimeouts) error {
	requestTimeout, hasRequestTimeout, err := parseOptionalDuration(timeouts.Request, "request timeout")
	if err != nil {
		return err
	}
	backendTimeout, hasBackendTimeout, err := parseOptionalDuration(timeouts.BackendRequest, "backend_request timeout")
	if err != nil {
		return err
	}
	if hasRequestTimeout && hasBackendTimeout && backendTimeout > requestTimeout {
		return errors.New("backend_request timeout must not exceed request timeout")
	}
	return nil
}

func parseOptionalDuration(value, label string) (time.Duration, bool, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false, nil
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, false, app.NewErrorf("%s must be a valid duration", label)
	}
	return duration, true, nil
}

func validateGatewayRouteCertificateReference(appCtx *models.AppContext, certID string) error {
	if appCtx == nil {
		return nil
	}

	trimmedCertID := strings.TrimSpace(certID)
	if trimmedCertID == "" {
		return newGatewayCertificateError("certificate is required for HTTPS public access")
	}

	certificate, err := GetCertificate(trimmedCertID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return newGatewayCertificateError("selected certificate was not found")
		}
		return err
	}
	if certificate.ClusterID != appCtx.EnvContext.Env.ClusterID {
		return newGatewayCertificateError("selected certificate does not belong to the current cluster")
	}
	if certificate.Scope == "env" {
		if certificate.EnvID == nil || strings.TrimSpace(*certificate.EnvID) != appCtx.EnvContext.Env.ID {
			return newGatewayCertificateError("selected certificate is not available in this environment")
		}
		return nil
	}
	if certificate.Scope != "cluster" {
		return newGatewayCertificateError("selected certificate has an unsupported scope")
	}
	return nil
}

func validateHTTPSRouteCertificateConflict(appCtx *models.AppContext, host, certID, currentGatewayID string) error {
	if appCtx == nil || host == "" || certID == "" {
		return nil
	}

	query := db.DB.Table("app_gateway_http_routes r").
		Select("r.cert_id").
		Joins("JOIN app_gateways ag ON ag.id = r.app_gateway_id").
		Joins("JOIN apps a ON a.id = ag.app_id").
		Joins("JOIN envs e ON e.id = a.env_id").
		Where("e.cluster_id = ? AND LOWER(r.host) = ? AND LOWER(r.listener_protocol) = ? AND r.enabled = ? AND r.cert_id IS NOT NULL",
			appCtx.EnvContext.Env.ClusterID,
			strings.ToLower(host),
			"https",
			true,
		)
	if currentGatewayID != "" {
		query = query.Where("ag.id <> ?", currentGatewayID)
	}

	var certIDs []string
	if err := query.Pluck("r.cert_id", &certIDs).Error; err != nil {
		return err
	}
	for _, existingCertID := range certIDs {
		if strings.TrimSpace(existingCertID) != strings.TrimSpace(certID) {
			return app.NewErrorf("HTTPS host %q is already configured with a different certificate", host)
		}
	}
	return nil
}

func replaceGatewayRoutes(tx *gorm.DB, appCtx *models.AppContext, gateway *entities.AppGateway, routes []models.GatewayRouteSpec) error {
	var existingRouteIDs []string
	if err := tx.Model(&entities.AppGatewayHTTPRoute{}).Where("app_gateway_id = ?", gateway.ID).Pluck("id", &existingRouteIDs).Error; err != nil {
		return err
	}
	if len(existingRouteIDs) > 0 {
		if err := tx.Where("route_id IN ?", existingRouteIDs).Delete(&entities.AppGatewayHTTPRouteBackend{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Where("app_gateway_id = ?", gateway.ID).Delete(&entities.AppGatewayHTTPRoute{}).Error; err != nil {
		return err
	}

	for index, routeSpec := range routes {
		routeID := strings.TrimSpace(routeSpec.ID)
		if routeID == "" {
			routeID = uuid.New()
		}
		route := entities.AppGatewayHTTPRoute{
			ID:                     routeID,
			AppGatewayID:           gateway.ID,
			Host:                   routeSpec.Host,
			ListenerProtocol:       routeSpec.ListenerProtocol,
			Path:                   routeSpec.Path,
			PathMatchType:          routeSpec.PathMatchType,
			Enabled:                routeSpec.Enabled,
			MatchesJSON:            marshalGatewayJSON(routeSpec.Matches),
			FiltersJSON:            marshalGatewayJSON(routeSpec.Filters),
			TimeoutsJSON:           marshalGatewayJSON(routeSpec.Timeouts),
			RetryJSON:              marshalGatewayJSON(routeSpec.Retry),
			SessionPersistenceJSON: marshalGatewayJSON(routeSpec.SessionPersistence),
			ExtensionJSON:          marshalGatewayJSON(routeSpec.Extension),
			SortOrder:              firstNonZero(routeSpec.SortOrder, index),
		}
		if routeSpec.CertID != "" {
			certID := routeSpec.CertID
			route.CertID = &certID
		}
		if err := tx.Create(&route).Error; err != nil {
			return err
		}
		for _, backendSpec := range routeSpec.Backends {
			backendID := strings.TrimSpace(backendSpec.ID)
			if backendID == "" {
				backendID = uuid.New()
			}
			backend := entities.AppGatewayHTTPRouteBackend{
				ID:           backendID,
				RouteID:      route.ID,
				BackendAppID: backendSpec.BackendAppID,
				BackendPort:  backendSpec.BackendPort,
				Weight:       backendSpec.Weight,
			}
			if err := tx.Create(&backend).Error; err != nil {
				return err
			}
		}
	}
	_ = appCtx
	return nil
}

func firstNonZero(value, fallback int) int {
	if value != 0 {
		return value
	}
	return fallback
}

func marshalGatewayJSON(value any) entities.JSONBlob {
	if value == nil {
		return nil
	}
	encoded, err := json.Marshal(value)
	if err != nil || string(encoded) == "null" {
		return nil
	}
	return entities.JSONBlob(encoded)
}

func syncAndLoadGatewayResponse(ctx context.Context, appID, gatewayID string) (*models.AppGatewayResponse, error) {
	appCtx, err := GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}
	if err := syncGatewaysToK8s(ctx, appCtx); err != nil {
		return nil, err
	}

	if nodePorts, err := readNodePortsFromK8s(ctx, appCtx); err == nil {
		for _, gateway := range appCtx.Gateways {
			if gateway.ID != gatewayID {
				continue
			}
			if np, ok := nodePorts[gateway.Port]; ok && gateway.NodePort == 0 {
				_ = db.DB.Model(&entities.AppGateway{}).Where("id = ?", gateway.ID).Update("node_port", np).Error
			}
		}
	}

	appCtx, err = GetAppContext(ctx, appID)
	if err != nil {
		return nil, err
	}
	for _, response := range buildGatewayResponses(appCtx) {
		if response.ID == gatewayID {
			return &response, nil
		}
	}
	return nil, errors.New("gateway not found")
}

func buildGatewayResponses(appCtx *models.AppContext) []models.AppGatewayResponse {
	routesByGateway := groupGatewayRoutes(appCtx.GatewayRoutes)
	backendsByRoute := groupGatewayBackends(appCtx.GatewayBackends)

	result := make([]models.AppGatewayResponse, 0, len(appCtx.Gateways))
	for _, gw := range appCtx.Gateways {
		resp := toAppGatewayResponse(&gw)
		resp.GatewayHost = appCtx.EnvContext.Cluster.GatewayHost
		resp.InternalAddress = fmt.Sprintf("%s.%s:%d", appCtx.App.Slug, appCtx.EnvContext.Env.ClusterNamespace, gw.Port)
		for _, route := range routesByGateway[gw.ID] {
			resp.Routes = append(resp.Routes, routeEntityToSpec(&route, backendsByRoute[route.ID]))
		}
		result = append(result, resp)
	}
	return result
}

func groupGatewayRoutes(routes []entities.AppGatewayHTTPRoute) map[string][]entities.AppGatewayHTTPRoute {
	grouped := make(map[string][]entities.AppGatewayHTTPRoute)
	for _, route := range routes {
		grouped[route.AppGatewayID] = append(grouped[route.AppGatewayID], route)
	}
	return grouped
}

func groupGatewayBackends(backends []entities.AppGatewayHTTPRouteBackend) map[string][]entities.AppGatewayHTTPRouteBackend {
	grouped := make(map[string][]entities.AppGatewayHTTPRouteBackend)
	for _, backend := range backends {
		grouped[backend.RouteID] = append(grouped[backend.RouteID], backend)
	}
	return grouped
}

// toAppGatewayResponse converts an AppGateway entity to a response model with snake_case JSON fields.
func toAppGatewayResponse(gw *entities.AppGateway) models.AppGatewayResponse {
	return models.AppGatewayResponse{
		ID:          gw.ID,
		AppID:       gw.AppID,
		Port:        gw.Port,
		Protocol:    gw.Protocol,
		GatewayPort: gw.GatewayPort,
		ServiceType: gw.ServiceType,
		NodePort:    gw.NodePort,
		CreatedAt:   gw.CreatedAt,
		UpdatedAt:   gw.UpdatedAt,
	}
}

func routeEntityToSpec(route *entities.AppGatewayHTTPRoute, backends []entities.AppGatewayHTTPRouteBackend) models.GatewayRouteSpec {
	spec := models.GatewayRouteSpec{
		ID:               route.ID,
		GatewayID:        route.AppGatewayID,
		Host:             route.Host,
		ListenerProtocol: route.ListenerProtocol,
		Path:             route.Path,
		PathMatchType:    route.PathMatchType,
		Enabled:          route.Enabled,
		SortOrder:        route.SortOrder,
	}
	if route.CertID != nil {
		spec.CertID = *route.CertID
	}
	decodeGatewayJSON(route.MatchesJSON, &spec.Matches)
	decodeGatewayJSON(route.FiltersJSON, &spec.Filters)
	decodeGatewayJSON(route.TimeoutsJSON, &spec.Timeouts)
	decodeGatewayJSON(route.RetryJSON, &spec.Retry)
	decodeGatewayJSON(route.SessionPersistenceJSON, &spec.SessionPersistence)
	decodeGatewayJSON(route.ExtensionJSON, &spec.Extension)
	for _, backend := range backends {
		spec.Backends = append(spec.Backends, models.GatewayRouteBackendSpec{
			ID:           backend.ID,
			RouteID:      backend.RouteID,
			BackendAppID: backend.BackendAppID,
			BackendPort:  backend.BackendPort,
			Weight:       backend.Weight,
		})
	}
	return spec
}

func decodeGatewayJSON[T any](blob entities.JSONBlob, target **T) {
	if len(blob) == 0 {
		return
	}
	var decoded T
	if err := json.Unmarshal(blob, &decoded); err == nil {
		*target = &decoded
	}
}

// GetGatewayWithApp loads a gateway along with its parent App, Env, and Cluster.
func GetGatewayWithApp(ctx context.Context, gatewayID string) (*entities.AppGateway, *models.AppContext, error) {
	var gateway entities.AppGateway
	err := db.DB.First(&gateway, "id = ?", gatewayID).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, errors.New("gateway not found")
		}
		return nil, nil, err
	}

	appCtx, err := GetAppContext(ctx, gateway.AppID)
	if err != nil {
		return nil, nil, err
	}
	return &gateway, appCtx, nil
}
