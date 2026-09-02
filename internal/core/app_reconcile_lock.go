package core

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/models"
)

// appReconcileLocks serialize resource reconciliation for one application in
// an API process. PostgreSQL/MySQL advisory locks add cross-replica ordering,
// while the process lock also protects SQLite tests and unsupported dialects.
// Kubernetes resource-version retries continue to protect individual objects.
var appReconcileLocks sync.Map

type appContextLoader func(context.Context, string) (*models.AppContext, error)

func withAppReconcileLock(appCtx *models.AppContext, fn func(*models.AppContext) error) error {
	return withAppReconcileLockContext(context.Background(), appCtx, fn)
}

func withAppReconcileLockContext(ctx context.Context, appCtx *models.AppContext, fn func(*models.AppContext) error) error {
	if appCtx == nil {
		return errors.New("app reconcile context is nil")
	}
	if fn == nil {
		return errors.New("app reconcile callback is nil")
	}
	key := appReconcileLockKey(appCtx)
	return withAppReconcileFenceKey(ctx, key, func() error {
		var loader appContextLoader
		if db.DB != nil && appCtx.App.ID != "" {
			loader = LoadAppContext
		}
		return reconcileLatestAppContext(ctx, appCtx, loader, fn)
	})
}

// WithAppReconcileFence serializes a database mutation and its Kubernetes side
// effects with every reconcile for the same application. The callback is
// responsible for loading desired state after the fence has been acquired.
func WithAppReconcileFence(ctx context.Context, appID string, fn func() error) error {
	if appID == "" {
		return errors.New("app reconcile ID is empty")
	}
	return withAppReconcileFenceKey(ctx, "app-id:"+appID, fn)
}

func withAppReconcileFenceKey(ctx context.Context, key string, fn func() error) error {
	if fn == nil {
		return errors.New("app reconcile fence callback is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	value, _ := appReconcileLocks.LoadOrStore(key, &sync.Mutex{})
	mutex := value.(*sync.Mutex)
	mutex.Lock()
	defer mutex.Unlock()

	return withDatabaseReconcileLock(ctx, "app:"+key, fn)
}

// reconcileLatestAppContext reloads desired state only after both reconcile
// locks are held. A second read after the callback detects writes that raced
// with the Kubernetes operation and repeats it with the newer state. This
// prevents a request that loaded its context before taking the lock from being
// the last writer of stale Kubernetes resources.
func reconcileLatestAppContext(
	ctx context.Context,
	original *models.AppContext,
	load appContextLoader,
	fn func(*models.AppContext) error,
) error {
	if original == nil {
		return errors.New("app reconcile context is nil")
	}
	if fn == nil {
		return errors.New("app reconcile callback is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if load == nil || original.App.ID == "" {
		return fn(original)
	}

	podAccessPolicy := original.PodAccessPolicy
	current, err := load(ctx, original.App.ID)
	if err != nil {
		return app.WrapErrorf(err, "reload app context before reconcile: %w", err)
	}
	for {
		if current == nil {
			return errors.New("reload app context before reconcile returned nil")
		}
		current.PodAccessPolicy = podAccessPolicy
		currentRevision, err := appContextReconcileRevision(current)
		if err != nil {
			return err
		}

		if err := fn(current); err != nil {
			return err
		}

		observed, err := load(ctx, original.App.ID)
		if err != nil {
			return app.WrapErrorf(err, "reload app context after reconcile: %w", err)
		}
		if observed == nil {
			return errors.New("reload app context after reconcile returned nil")
		}
		observed.PodAccessPolicy = podAccessPolicy
		observedRevision, err := appContextReconcileRevision(observed)
		if err != nil {
			return err
		}
		if currentRevision == observedRevision {
			*original = *observed
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			current = observed
		}
	}
}

func appContextReconcileRevision(appCtx *models.AppContext) ([sha256.Size]byte, error) {
	if appCtx == nil {
		return [sha256.Size]byte{}, errors.New("calculate reconcile revision for nil app context")
	}
	copy := *appCtx
	copy.PodAccessPolicy = nil
	encoded, err := json.Marshal(copy)
	if err != nil {
		return [sha256.Size]byte{}, app.WrapErrorf(err, "encode app reconcile revision: %w", err)
	}
	return sha256.Sum256(encoded), nil
}

func appReconcileLockKey(appCtx *models.AppContext) string {
	if appCtx == nil {
		return "<nil>"
	}
	if appCtx.App.ID != "" {
		return "app-id:" + appCtx.App.ID
	}
	return fmt.Sprintf(
		"app:%s/%s/%s",
		appCtx.EnvContext.Env.ClusterID,
		appCtx.EnvContext.Env.ClusterNamespace,
		appCtx.App.Slug,
	)
}
