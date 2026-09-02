package services

import (
	"context"
	"log/slog"

	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
)

var ensureSharedGatewayForReconcile = core.EnsureSharedGateway

func ReconcilePublicGatewayResources(ctx context.Context) error {
	var clusters []entities.Cluster
	if err := db.DB.Select("id").Where("enabled = ?", true).Find(&clusters).Error; err != nil {
		return err
	}

	for _, cluster := range clusters {
		if cluster.ID == "" {
			continue
		}
		if err := ensureSharedGatewayForReconcile(ctx, cluster.ID); err != nil {
			slog.Warn("failed to reconcile shared gateway resources", "cluster_id", cluster.ID, "error", err)
		}
	}

	return nil
}
