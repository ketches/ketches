package core

import (
	"context"
	"log"

	"github.com/ketches/ketches/internal/app"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func DeleteResource(ctx context.Context, cli client.Client, obj client.Object) app.Error {
	if err := cli.Delete(ctx, obj); err != nil {
		if k8serrors.IsNotFound(err) {
			// Resource already deleted
			return nil
		}
		log.Println("failed to delete resource:", err)
		return app.ErrClusterOperationFailed
	}
	return nil
}
