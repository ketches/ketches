package orm

import (
	"context"
	"net/http"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
)

func GetAppGatewayByIDs(ctx context.Context, gatewayID string) (*entities.AppGateway, app.Error) {
	gateway := &entities.AppGateway{}
	if err := db.Instance().First(gateway, "id = ?", gatewayID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "App gateway not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}
	return gateway, nil
}
