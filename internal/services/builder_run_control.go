package services

import (
	"context"
	"errors"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var builderRunTerminalStatuses = []entities.BuilderRunStatus{
	entities.BuilderRunStatusSucceeded,
	entities.BuilderRunStatusFailed,
	entities.BuilderRunStatusCancelled,
	entities.BuilderRunStatusTimedOut,
}

type BuilderRunRequeueInput struct {
	RunID        string
	ClaimToken   string
	ErrorCode    *string
	ErrorClass   *string
	ErrorMessage string
}

type BuilderRunFinalizeInput struct {
	RunID           string
	ClaimToken      string
	Status          entities.BuilderRunStatus
	ErrorCode       *string
	ErrorClass      *string
	ErrorMessage    string
	WorkspaceUsable bool
}

func NormalizeLegacyBuilderRunsForControlPlane(ctx context.Context) error {
	return normalizeLegacyBuilderRunsForControlPlane(db.DB.WithContext(ctx))
}

var builderRunControlPlaneFields = []string{
	"Phase",
	"ClaimToken",
	"ClaimedAt",
	"HeartbeatAt",
	"TimeoutAt",
	"CancelRequestedAt",
	"ProviderKey",
	"ModelProfileKey",
	"ExecutorPolicyKey",
	"ExecutorHandleID",
	"ErrorCode",
	"ErrorClass",
}

func ensureBuilderRunControlPlaneColumns(tx *gorm.DB) error {
	for _, field := range builderRunControlPlaneFields {
		if tx.Migrator().HasColumn(&entities.BuilderRun{}, field) {
			continue
		}
		if err := tx.Migrator().AddColumn(&entities.BuilderRun{}, field); err != nil {
			return err
		}
	}

	return nil
}

func ClaimNextQueuedBuilderRun(ctx context.Context, claimToken string, leaseDuration time.Duration) (*entities.BuilderRun, error) {
	tx := db.DB.WithContext(ctx)
	var claimedRun *entities.BuilderRun

	err := tx.Transaction(func(tx *gorm.DB) error {
		candidateID, err := nextQueuedBuilderRunID(tx)
		if err != nil {
			return err
		}
		if candidateID == "" {
			return nil
		}

		candidateRun, err := loadBuilderRunByID(tx, candidateID)
		if err != nil {
			return err
		}

		session, err := lockBuilderSessionForRunClaim(tx, candidateRun.SessionID)
		if err != nil {
			return err
		}
		if err := validateBuilderSessionAppendable(session); err != nil {
			return err
		}

		hasExecutingRun, err := sessionHasExecutingBuilderRun(tx, candidateRun.SessionID)
		if err != nil {
			return err
		}
		if hasExecutingRun {
			return nil
		}

		now := time.Now().UTC()
		updates := map[string]any{
			"status":        entities.BuilderRunStatusExecuting,
			"phase":         entities.BuilderRunPhaseClaiming,
			"claim_token":   claimToken,
			"claimed_at":    now,
			"heartbeat_at":  now,
			"timeout_at":    now.Add(leaseDuration),
			"started_at":    now,
			"completed_at":  nil,
			"error_code":    nil,
			"error_class":   nil,
			"error_message": "",
			"updated_at":    now,
		}

		result := tx.Model(&entities.BuilderRun{}).
			Where("id = ? AND status = ?", candidateID, entities.BuilderRunStatusQueued).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		claimedRun, err = loadBuilderRunByID(tx, candidateID)
		if err != nil {
			return err
		}

		sessionUpdate := tx.Model(&entities.BuilderSession{}).
			Where("id = ? AND status IN ?", claimedRun.SessionID, appendableBuilderSessionStatuses()).
			Updates(map[string]any{
				"status":     entities.BuilderSessionStatusRunning,
				"updated_at": now,
			})
		if sessionUpdate.Error != nil {
			return sessionUpdate.Error
		}
		if sessionUpdate.RowsAffected == 0 {
			session, err := loadBuilderSession(tx, "", claimedRun.SessionID)
			if err != nil {
				return err
			}
			return validateBuilderSessionAppendable(session)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return claimedRun, nil
}

func HeartbeatBuilderRunLease(ctx context.Context, runID, claimToken string, leaseDuration time.Duration) (*entities.BuilderRun, error) {
	tx := db.DB.WithContext(ctx)
	var run *entities.BuilderRun

	err := tx.Transaction(func(tx *gorm.DB) error {
		now := time.Now().UTC()
		result := tx.Model(&entities.BuilderRun{}).
			Where("id = ? AND status = ? AND claim_token = ?", runID, entities.BuilderRunStatusExecuting, claimToken).
			Updates(map[string]any{
				"heartbeat_at": now,
				"timeout_at":   now.Add(leaseDuration),
				"updated_at":   now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		var err error
		run, err = loadBuilderRunByID(tx, runID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return run, nil
}

func ListRecoverableBuilderRuns(ctx context.Context, now time.Time) ([]entities.BuilderRun, error) {
	var runs []entities.BuilderRun
	err := db.DB.WithContext(ctx).
		Where("status = ? AND claim_token IS NOT NULL AND claimed_at IS NOT NULL AND heartbeat_at IS NOT NULL AND timeout_at IS NOT NULL AND timeout_at <= ?", entities.BuilderRunStatusExecuting, now).
		Order("timeout_at ASC, heartbeat_at ASC, created_at ASC, id ASC").
		Find(&runs).Error
	if err != nil {
		return nil, err
	}
	return runs, nil
}

func RequeueBuilderRun(ctx context.Context, input BuilderRunRequeueInput) (*entities.BuilderRun, error) {
	tx := db.DB.WithContext(ctx)
	var run *entities.BuilderRun

	err := tx.Transaction(func(tx *gorm.DB) error {
		ownedRun, err := loadOwnedExecutingBuilderRun(tx, input.RunID, input.ClaimToken)
		if err != nil {
			return err
		}
		if err := deleteBuilderOutputSnapshotsByRunIDTx(tx, input.RunID); err != nil {
			return err
		}

		now := time.Now().UTC()
		result := tx.Model(&entities.BuilderRun{}).
			Where("id = ? AND status = ? AND claim_token = ?", input.RunID, entities.BuilderRunStatusExecuting, input.ClaimToken).
			Updates(map[string]any{
				"status":        entities.BuilderRunStatusQueued,
				"phase":         entities.BuilderRunPhaseQueued,
				"attempt_count": ownedRun.AttemptCount + 1,
				"claim_token":   nil,
				"claimed_at":    nil,
				"heartbeat_at":  nil,
				"timeout_at":    nil,
				"started_at":    nil,
				"completed_at":  nil,
				"error_code":    nullableStringValue(input.ErrorCode),
				"error_class":   nullableStringValue(input.ErrorClass),
				"error_message": input.ErrorMessage,
				"updated_at":    now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		run, err = loadBuilderRunByID(tx, input.RunID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return run, nil
}

func RequestBuilderRunCancel(ctx context.Context, runID string) (*entities.BuilderRun, error) {
	tx := db.DB.WithContext(ctx)
	var run *entities.BuilderRun

	err := tx.Transaction(func(tx *gorm.DB) error {
		var err error
		run, err = loadBuilderRunByID(tx, runID)
		if err != nil {
			return err
		}
		if isBuilderRunTerminalStatus(run.Status) {
			return nil
		}

		now := time.Now().UTC()
		result := tx.Model(&entities.BuilderRun{}).
			Where("id = ? AND status NOT IN ?", runID, builderRunTerminalStatuses).
			Updates(map[string]any{
				"cancel_requested_at": now,
				"updated_at":          now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			run, err = loadBuilderRunByID(tx, runID)
			if err != nil {
				return err
			}
			if isBuilderRunTerminalStatus(run.Status) {
				return nil
			}
			return gorm.ErrRecordNotFound
		}

		run, err = loadBuilderRunByID(tx, runID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return run, nil
}

func FinalizeBuilderRun(ctx context.Context, input BuilderRunFinalizeInput) (*entities.BuilderRun, error) {
	tx := db.DB.WithContext(ctx)
	var run *entities.BuilderRun

	err := tx.Transaction(func(tx *gorm.DB) error {
		ownedRun, err := loadOwnedExecutingBuilderRun(tx, input.RunID, input.ClaimToken)
		if err != nil {
			return err
		}

		now := time.Now().UTC()
		result := tx.Model(&entities.BuilderRun{}).
			Where("id = ? AND status = ? AND claim_token = ?", input.RunID, entities.BuilderRunStatusExecuting, input.ClaimToken).
			Updates(map[string]any{
				"status":        input.Status,
				"phase":         entities.BuilderRunPhaseFinalizing,
				"claim_token":   nil,
				"claimed_at":    nil,
				"heartbeat_at":  nil,
				"timeout_at":    nil,
				"completed_at":  now,
				"error_code":    nullableStringValue(input.ErrorCode),
				"error_class":   nullableStringValue(input.ErrorClass),
				"error_message": input.ErrorMessage,
				"updated_at":    now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		nextStatus, err := resolveBuilderSessionStatusAfterTerminalRun(tx, ownedRun.SessionID, input.WorkspaceUsable)
		if err != nil {
			return err
		}

		if err := tx.Model(&entities.BuilderSession{}).
			Where("id = ?", ownedRun.SessionID).
			Updates(map[string]any{
				"status":     nextStatus,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}

		run, err = loadBuilderRunByID(tx, input.RunID)
		return err
	})
	if err != nil {
		return nil, err
	}

	return run, nil
}

func normalizeLegacyBuilderRunsForControlPlane(tx *gorm.DB) error {
	if err := ensureBuilderRunControlPlaneColumns(tx); err != nil {
		return err
	}

	if err := tx.Model(&entities.BuilderRun{}).
		Where("phase IS NULL AND status = ?", entities.BuilderRunStatusQueued).
		Update("phase", entities.BuilderRunPhaseQueued).Error; err != nil {
		return err
	}

	if err := tx.Model(&entities.BuilderRun{}).
		Where("status IN ? AND (phase IS NULL OR phase = ?)", builderRunTerminalStatuses, entities.BuilderRunPhaseQueued).
		Update("phase", entities.BuilderRunPhaseFinalizing).Error; err != nil {
		return err
	}

	return nil
}

func nextQueuedBuilderRunID(tx *gorm.DB) (string, error) {
	type builderRunIDRow struct {
		ID string
	}

	var row builderRunIDRow
	err := tx.Table("builder_runs AS br").
		Select("br.id").
		Joins("JOIN builder_sessions AS bs ON bs.id = br.session_id").
		Where("br.status = ?", entities.BuilderRunStatusQueued).
		Where("COALESCE(br.phase, ?) = ?", entities.BuilderRunPhaseQueued, entities.BuilderRunPhaseQueued).
		Where("bs.status IN ?", []entities.BuilderSessionStatus{
			entities.BuilderSessionStatusProvisioning,
			entities.BuilderSessionStatusReady,
			entities.BuilderSessionStatusRunning,
		}).
		Where("NOT EXISTS (SELECT 1 FROM builder_runs AS active WHERE active.session_id = br.session_id AND active.status = ?)", entities.BuilderRunStatusExecuting).
		Order("br.created_at ASC, br.id ASC").
		Limit(1).
		Scan(&row).Error
	if err != nil {
		return "", err
	}
	return row.ID, nil
}

func resolveBuilderSessionStatusAfterTerminalRun(tx *gorm.DB, sessionID string, workspaceUsable bool) (entities.BuilderSessionStatus, error) {
	if !workspaceUsable {
		return entities.BuilderSessionStatusFailed, nil
	}

	var queuedRunCount int64
	if err := tx.Model(&entities.BuilderRun{}).
		Where("session_id = ? AND status = ?", sessionID, entities.BuilderRunStatusQueued).
		Count(&queuedRunCount).Error; err != nil {
		return "", err
	}
	if queuedRunCount > 0 {
		return entities.BuilderSessionStatusRunning, nil
	}

	return entities.BuilderSessionStatusReady, nil
}

func loadBuilderRunByID(tx *gorm.DB, runID string) (*entities.BuilderRun, error) {
	var run entities.BuilderRun
	if err := tx.Where("id = ?", runID).First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func loadOwnedExecutingBuilderRun(tx *gorm.DB, runID, claimToken string) (*entities.BuilderRun, error) {
	var run entities.BuilderRun
	if err := tx.Where("id = ? AND status = ? AND claim_token = ?", runID, entities.BuilderRunStatusExecuting, claimToken).First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &run, nil
}

func lockBuilderSessionForRunClaim(tx *gorm.DB, sessionID string) (*entities.BuilderSession, error) {
	var session entities.BuilderSession
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", sessionID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &BuilderSessionNotFoundError{SessionID: sessionID}
		}
		return nil, err
	}
	return &session, nil
}

func sessionHasExecutingBuilderRun(tx *gorm.DB, sessionID string) (bool, error) {
	type builderRunExistsRow struct {
		ID string
	}

	var row builderRunExistsRow
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Model(&entities.BuilderRun{}).
		Select("id").
		Where("session_id = ? AND status = ?", sessionID, entities.BuilderRunStatusExecuting).
		Order("created_at ASC, id ASC").
		Limit(1).
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func nullableStringValue(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func appendableBuilderSessionStatuses() []entities.BuilderSessionStatus {
	return []entities.BuilderSessionStatus{
		entities.BuilderSessionStatusProvisioning,
		entities.BuilderSessionStatusReady,
		entities.BuilderSessionStatusRunning,
	}
}

func isBuilderRunTerminalStatus(status entities.BuilderRunStatus) bool {
	switch status {
	case entities.BuilderRunStatusSucceeded,
		entities.BuilderRunStatusFailed,
		entities.BuilderRunStatusCancelled,
		entities.BuilderRunStatusTimedOut:
		return true
	default:
		return false
	}
}
