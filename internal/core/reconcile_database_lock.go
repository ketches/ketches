package core

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
)

const (
	databaseReconcileLockPollInterval = 100 * time.Millisecond
	databaseReconcileLockNamePrefix   = "ketches-reconcile-"
	databaseReconcileUnlockTimeout    = 5 * time.Second
)

// withDatabaseReconcileLock holds a database advisory lock for the complete
// Kubernetes reconciliation callback. PostgreSQL and MySQL bind advisory
// locks to a database session, so a dedicated pooled connection is retained
// until the callback finishes. Other dialects (including SQLite test
// databases) use the caller's process-local lock only.
//
// The application database pool must permit at least one additional connection
// while a reconciliation is running (the default pool size is 50). A session
// advisory lock cannot safely share the callback's connection because most
// existing reconciliation code queries db.DB directly.
func withDatabaseReconcileLock(ctx context.Context, key string, fn func() error) error {
	if fn == nil {
		return errors.New("reconcile lock callback is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	database := db.DB
	if database == nil {
		return fn()
	}

	dialect := normalizeReconcileLockDialect(database.Dialector.Name())
	if dialect == "" {
		return fn()
	}

	sqlDB, err := database.DB()
	if err != nil {
		return app.WrapErrorf(err, "open database connection for reconcile lock: %w", err)
	}
	if maxOpen := sqlDB.Stats().MaxOpenConnections; maxOpen > 0 && maxOpen < 2 {
		return app.NewErrorf("database advisory reconcile lock requires DB_MAX_OPEN_CONNS >= 2")
	}

	connection, err := sqlDB.Conn(ctx)
	if err != nil {
		return app.WrapErrorf(err, "reserve database connection for reconcile lock: %w", err)
	}
	defer connection.Close()

	release, err := acquireDatabaseReconcileLock(ctx, connection, dialect, key)
	if err != nil {
		return err
	}

	callbackErr := fn()
	releaseErr := release()
	if callbackErr != nil && releaseErr != nil {
		return errors.Join(callbackErr, releaseErr)
	}
	if callbackErr != nil {
		return callbackErr
	}
	return releaseErr
}

func normalizeReconcileLockDialect(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "postgres", "postgresql":
		return "postgres"
	case "mysql":
		return "mysql"
	default:
		return ""
	}
}

func acquireDatabaseReconcileLock(
	ctx context.Context,
	connection *sql.Conn,
	dialect, key string,
) (func() error, error) {
	switch dialect {
	case "postgres":
		lockID := postgresReconcileLockID(key)
		for {
			var acquired bool
			if err := connection.QueryRowContext(ctx, "SELECT pg_try_advisory_lock($1)", lockID).Scan(&acquired); err != nil {
				return nil, app.WrapErrorf(err, "acquire PostgreSQL reconcile lock: %w", err)
			}
			if acquired {
				return func() error {
					unlockCtx, cancel := context.WithTimeout(context.Background(), databaseReconcileUnlockTimeout)
					defer cancel()
					var released bool
					if err := connection.QueryRowContext(unlockCtx, "SELECT pg_advisory_unlock($1)", lockID).Scan(&released); err != nil {
						return app.WrapErrorf(err, "release PostgreSQL reconcile lock: %w", err)
					}
					if !released {
						return app.NewErrorf("release PostgreSQL reconcile lock: lock was not held")
					}
					return nil
				}, nil
			}
			if err := waitForReconcileLock(ctx); err != nil {
				return nil, app.WrapErrorf(err, "wait for PostgreSQL reconcile lock: %w", err)
			}
		}

	case "mysql":
		lockName := mysqlReconcileLockName(key)
		for {
			var result sql.NullInt64
			if err := connection.QueryRowContext(ctx, "SELECT GET_LOCK(?, 0)", lockName).Scan(&result); err != nil {
				return nil, app.WrapErrorf(err, "acquire MySQL reconcile lock: %w", err)
			}
			if !result.Valid {
				return nil, errors.New("acquire MySQL reconcile lock: GET_LOCK returned NULL")
			}
			if result.Int64 == 1 {
				return func() error {
					unlockCtx, cancel := context.WithTimeout(context.Background(), databaseReconcileUnlockTimeout)
					defer cancel()
					var released sql.NullInt64
					if err := connection.QueryRowContext(unlockCtx, "SELECT RELEASE_LOCK(?)", lockName).Scan(&released); err != nil {
						return app.WrapErrorf(err, "release MySQL reconcile lock: %w", err)
					}
					if !released.Valid || released.Int64 != 1 {
						return app.NewErrorf("release MySQL reconcile lock: lock was not held")
					}
					return nil
				}, nil
			}
			if result.Int64 != 0 {
				return nil, app.NewErrorf("acquire MySQL reconcile lock: GET_LOCK returned %d", result.Int64)
			}
			if err := waitForReconcileLock(ctx); err != nil {
				return nil, app.WrapErrorf(err, "wait for MySQL reconcile lock: %w", err)
			}
		}
	default:
		return func() error { return nil }, nil
	}
}

func waitForReconcileLock(ctx context.Context) error {
	timer := time.NewTimer(databaseReconcileLockPollInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func postgresReconcileLockID(key string) int64 {
	sum := sha256.Sum256([]byte(key))
	// Keep the value positive while retaining 63 bits for PostgreSQL BIGINT.
	return int64(binary.BigEndian.Uint64(sum[:8]) &^ (uint64(1) << 63))
}

func mysqlReconcileLockName(key string) string {
	sum := sha256.Sum256([]byte(key))
	// GET_LOCK names are limited to 64 characters on supported MySQL versions.
	return databaseReconcileLockNamePrefix + fmt.Sprintf("%x", sum[:16])
}
