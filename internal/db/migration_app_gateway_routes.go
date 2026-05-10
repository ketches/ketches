package db

import (
	"strings"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

type legacyAppGatewayRouteRow struct {
	ID       string
	AppID    string
	Port     int
	Protocol string
	Domain   string
	Path     string
	CertID   *string
}

func migrateAppGatewayHTTPRoutes() error {
	if DB == nil {
		return nil
	}
	if !DB.Migrator().HasTable(&entities.AppGateway{}) {
		return nil
	}
	if !DB.Migrator().HasTable(&entities.AppGatewayHTTPRoute{}) {
		return nil
	}
	if !DB.Migrator().HasTable(&entities.AppGatewayHTTPRouteBackend{}) {
		return nil
	}

	requiredColumns := []string{"domain", "path", "exposed", "cert_id"}
	for _, column := range requiredColumns {
		if !DB.Migrator().HasColumn(&entities.AppGateway{}, column) {
			return nil
		}
	}

	var rows []legacyAppGatewayRouteRow
	if err := DB.Table("app_gateways").
		Select("id, app_id, port, LOWER(protocol) AS protocol, domain, path, cert_id").
		Where("exposed = ? AND LOWER(protocol) IN ? AND TRIM(COALESCE(domain, '')) <> ''", true, []string{"http", "https"}).
		Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		if err := migrateLegacyAppGatewayRoute(row); err != nil {
			return err
		}
	}
	return nil
}

func migrateLegacyAppGatewayRoute(row legacyAppGatewayRouteRow) error {
	host := strings.TrimSpace(row.Domain)
	if host == "" {
		return nil
	}

	path := strings.TrimSpace(row.Path)
	if path == "" {
		path = "/"
	}

	var existing entities.AppGatewayHTTPRoute
	err := DB.Where(
		"app_gateway_id = ? AND host = ? AND listener_protocol = ? AND path_match_type = ? AND path = ?",
		row.ID,
		host,
		row.Protocol,
		"PathPrefix",
		path,
	).First(&existing).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	route := existing
	if err == gorm.ErrRecordNotFound {
		route = entities.AppGatewayHTTPRoute{
			ID:               uuid.New(),
			AppGatewayID:     row.ID,
			Host:             host,
			ListenerProtocol: row.Protocol,
			Path:             path,
			PathMatchType:    "PathPrefix",
			Enabled:          true,
		}
		if row.CertID != nil && strings.TrimSpace(*row.CertID) != "" {
			certID := strings.TrimSpace(*row.CertID)
			route.CertID = &certID
		}
		if err := DB.Create(&route).Error; err != nil {
			return err
		}
	}

	var backendCount int64
	if err := DB.Model(&entities.AppGatewayHTTPRouteBackend{}).Where("route_id = ?", route.ID).Count(&backendCount).Error; err != nil {
		return err
	}
	if backendCount > 0 {
		return nil
	}

	backend := entities.AppGatewayHTTPRouteBackend{
		ID:           uuid.New(),
		RouteID:      route.ID,
		BackendAppID: row.AppID,
		BackendPort:  row.Port,
		Weight:       1,
	}
	return DB.Create(&backend).Error
}
