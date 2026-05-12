package services

import (
	"context"
	"log/slog"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
)

func ReconcilePublicGatewayResources(ctx context.Context) error {
	type clusterRow struct {
		ClusterID string
	}

	var rows []clusterRow
	err := db.DB.Table("app_gateway_http_routes r").
		Select("DISTINCT e.cluster_id AS cluster_id").
		Joins("JOIN app_gateways ag ON ag.id = r.app_gateway_id").
		Joins("JOIN apps a ON a.id = ag.app_id").
		Joins("JOIN envs e ON e.id = a.env_id").
		Where("r.enabled = ? AND LOWER(ag.protocol) = ? AND e.cluster_id <> ''", true, "http").
		Scan(&rows).Error
	if err != nil {
		return err
	}

	for _, row := range rows {
		if row.ClusterID == "" {
			continue
		}
		if err := core.EnsureSharedGateway(ctx, row.ClusterID); err != nil {
			slog.Warn("failed to reconcile shared gateway resources", "cluster_id", row.ClusterID, "error", err)
		}
	}

	return nil
}
