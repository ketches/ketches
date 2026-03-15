package core

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
)

const buildLogMaintenanceInterval = 24 * time.Hour

func RecoverTerminalBuildLogArchives(ctx context.Context) error {
	var builds []entities.Build
	if err := db.DB.
		Where("status IN ?", []entities.BuildStatus{
			entities.BuildStatusSucceeded,
			entities.BuildStatusFailed,
			entities.BuildStatusCancelled,
		}).
		Where("log_persist_status <> ?", entities.BuildLogPersistSucceeded).
		Where("log_persist_status <> ?", entities.BuildLogPersistExpired).
		Find(&builds).Error; err != nil {
		return err
	}

	for i := range builds {
		if err := PersistBuildLogs(ctx, builds[i].ID); err != nil {
			log.Printf("Build log maintenance: failed to recover archive for build %s: %v", builds[i].ID, err)
		}
	}

	return nil
}

func DeleteExpiredBuildLogs(ctx context.Context, now time.Time) error {
	var builds []entities.Build
	if err := db.DB.
		Where("log_persist_status = ?", entities.BuildLogPersistSucceeded).
		Where("log_expire_at IS NOT NULL").
		Where("log_expire_at <= ?", now).
		Find(&builds).Error; err != nil {
		return err
	}

	for i := range builds {
		if err := deleteExpiredBuildLog(&builds[i]); err != nil {
			return err
		}
	}

	return nil
}

func StartBuildLogMaintenance(ctx context.Context) {
	ticker := time.NewTicker(buildLogMaintenanceInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case tickTime := <-ticker.C:
			if err := DeleteExpiredBuildLogs(ctx, tickTime); err != nil {
				log.Printf("Build log maintenance: failed to delete expired archives: %v", err)
			}
		}
	}
}

func deleteExpiredBuildLog(build *entities.Build) error {
	if build == nil {
		return nil
	}

	if stringsTrimmed := build.LogPath; stringsTrimmed != "" {
		err := os.Remove(buildLogArchiveAbsPath(build.LogPath))
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	build.LogPath = ""
	build.LogSize = 0
	build.LogPersistStatus = entities.BuildLogPersistExpired
	build.LogPersistError = ""

	return db.DB.Save(build).Error
}
