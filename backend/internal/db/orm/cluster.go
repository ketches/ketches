package orm

import (
	"context"
	"net/http"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
)

func GetClusterByID(ctx context.Context, clusterID string) (*entities.Cluster, app.Error) {
	cluster := &entities.Cluster{}
	if err := db.Instance().First(cluster, "id = ?", clusterID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Cluster not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	return cluster, nil
}

func GetClusterSlugByID(ctx context.Context, clusterID string) (string, app.Error) {
	cluster := &entities.Cluster{}
	if err := db.Instance().Select("slug").First(cluster, "id = ?", clusterID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return "", app.NewError(http.StatusNotFound, "Cluster not found")
		}
		return "", app.ErrDatabaseOperationFailed
	}

	return cluster.Slug, nil
}
