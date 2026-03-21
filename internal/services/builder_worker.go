package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	builderWorkerDefaultPollInterval      = 2 * time.Second
	builderWorkerDefaultLeaseDuration     = 2 * time.Minute
	builderWorkerDefaultHeartbeatInterval = 30 * time.Second

	builderWorkerLeaseExpiredErrorCode = "lease_expired"
	builderWorkerCancelRequestedCode   = "cancel_requested"
	builderWorkerTimeoutErrorClass     = "timeout"
	builderWorkerCancelledErrorClass   = "cancelled"
	builderWorkerUnknownErrorClass     = "unknown"
	builderWorkerProvisioningErrorCode = "workspace_provision_failed"
	builderWorkerProvisioningClass     = "executor_provisioning"
	builderWorkerGenerationErrorCode   = "generation_failed"
	builderWorkerFileWriteErrorCode    = "workspace_write_failed"
	builderWorkerWorkspaceIOClass      = "workspace_io"
)

var (
	ErrBuilderWorkerStartupPreflightBlocked = errors.New("builder worker startup preflight blocked")
	GlobalBuilderWorker                     = NewBuilderWorker()
)

type BuilderWorkerStartupPreflightError struct {
	LegacyExecutingRunCount int64
	LegacyExecutingRunIDs   []string
}

type BuilderWorker struct {
	mu sync.Mutex

	parentCtx context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	pollInterval       time.Duration
	leaseDuration      time.Duration
	heartbeatInterval  time.Duration
	nowFn              func() time.Time
	claimTokenFn       func() string
	handleClaimedRunFn func(ctx context.Context, run *entities.BuilderRun) error
}

func NewBuilderWorker() *BuilderWorker {
	worker := &BuilderWorker{
		pollInterval:      builderWorkerDefaultPollInterval,
		leaseDuration:     builderWorkerDefaultLeaseDuration,
		heartbeatInterval: builderWorkerDefaultHeartbeatInterval,
		nowFn: func() time.Time {
			return time.Now().UTC()
		},
		claimTokenFn: uuid.New,
	}
	worker.handleClaimedRunFn = worker.executeClaimedRun
	return worker
}

func (e *BuilderWorkerStartupPreflightError) Error() string {
	return fmt.Sprintf("builder worker startup blocked by %d legacy executing runs", e.LegacyExecutingRunCount)
}

func (e *BuilderWorkerStartupPreflightError) Unwrap() error {
	return ErrBuilderWorkerStartupPreflightBlocked
}

func (w *BuilderWorker) SetParentContext(ctx context.Context) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.parentCtx = ctx
}

func (w *BuilderWorker) RecoverActiveRuns(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if err := PreflightBuilderWorkerStartup(ctx); err != nil {
		return err
	}

	return w.recoverExpiredExecutingRuns(ctx)
}

func (w *BuilderWorker) Start() {
	w.mu.Lock()
	if w.cancel != nil {
		w.mu.Unlock()
		return
	}

	parentCtx := w.parentCtx
	if parentCtx == nil {
		parentCtx = context.Background()
	}

	ctx, cancel := context.WithCancel(parentCtx)
	w.cancel = cancel
	w.wg.Add(1)
	w.mu.Unlock()

	go func() {
		defer w.wg.Done()
		defer func() {
			w.mu.Lock()
			w.cancel = nil
			w.mu.Unlock()
		}()

		w.run(ctx)
	}()
}

func (w *BuilderWorker) Stop() {
	w.mu.Lock()
	cancel := w.cancel
	w.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

func (w *BuilderWorker) Wait() {
	w.wg.Wait()
}

func (w *BuilderWorker) run(ctx context.Context) {
	if err := w.processAvailableWork(ctx); err != nil && ctx.Err() == nil {
		log.Printf("builder worker initial scan failed: %v", err)
	}

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processAvailableWork(ctx); err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("builder worker scan failed: %v", err)
			}
		}
	}
}

func (w *BuilderWorker) processAvailableWork(ctx context.Context) error {
	if err := w.recoverExpiredExecutingRuns(ctx); err != nil {
		return err
	}

	for {
		claimedRun, err := ClaimNextQueuedBuilderRun(ctx, w.claimTokenFn(), w.leaseDuration)
		if err != nil {
			return err
		}
		if claimedRun == nil {
			return nil
		}

		if err := w.handleClaimedRunFn(ctx, claimedRun); err != nil {
			if ctx.Err() != nil || errors.Is(err, context.Canceled) {
				return ctx.Err()
			}
			log.Printf("builder worker failed processing run %s: %v", claimedRun.ID, err)
		}
	}
}

func (w *BuilderWorker) recoverExpiredExecutingRuns(ctx context.Context) error {
	recoverableRuns, err := ListRecoverableBuilderRuns(ctx, w.nowFn())
	if err != nil {
		return err
	}

	for i := range recoverableRuns {
		if err := w.recoverExpiredExecutingRun(ctx, &recoverableRuns[i]); err != nil {
			return err
		}
	}

	return nil
}

func (w *BuilderWorker) recoverExpiredExecutingRun(ctx context.Context, run *entities.BuilderRun) error {
	if run == nil || run.ClaimToken == nil || strings.TrimSpace(*run.ClaimToken) == "" {
		return nil
	}

	if cancelled, err := finalizeClaimedBuilderRunCancellation(ctx, run, *run.ClaimToken, "[system] run cancelled after lease expiry\n"); cancelled || err != nil {
		return err
	}

	if run.AttemptCount < run.MaxAttempts {
		message := "builder worker lease expired before completion; requeued for recovery"
		if _, err := AppendBuilderRunStatusEvent(ctx, run.ID, entities.BuilderRunEventLevelWarn, "[system] "+message+"\n"); err != nil {
			return err
		}

		errorCode := builderWorkerLeaseExpiredErrorCode
		errorClass := builderWorkerTimeoutErrorClass
		_, err := RequeueBuilderRun(ctx, BuilderRunRequeueInput{
			RunID:        run.ID,
			ClaimToken:   *run.ClaimToken,
			ErrorCode:    &errorCode,
			ErrorClass:   &errorClass,
			ErrorMessage: message,
		})
		return err
	}

	message := "builder worker lease expired before completion and no retry attempts remain"
	if _, err := AppendBuilderRunStatusEvent(ctx, run.ID, entities.BuilderRunEventLevelError, "[system] "+message+"\n"); err != nil {
		return err
	}

	errorCode := builderWorkerLeaseExpiredErrorCode
	errorClass := builderWorkerTimeoutErrorClass
	_, err := FinalizeBuilderRun(ctx, BuilderRunFinalizeInput{
		RunID:           run.ID,
		ClaimToken:      *run.ClaimToken,
		Status:          entities.BuilderRunStatusTimedOut,
		ErrorCode:       &errorCode,
		ErrorClass:      &errorClass,
		ErrorMessage:    message,
		WorkspaceUsable: true,
	})
	return err
}

func (w *BuilderWorker) executeClaimedRun(ctx context.Context, run *entities.BuilderRun) error {
	if run == nil {
		return errors.New("builder run is required")
	}
	if run.ClaimToken == nil || strings.TrimSpace(*run.ClaimToken) == "" {
		return errors.New("builder run claim token is required")
	}

	claimToken := *run.ClaimToken
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		heartbeatErr   error
		heartbeatErrMu sync.Mutex
		heartbeatWG    sync.WaitGroup
	)
	heartbeatWG.Add(1)
	go func() {
		defer heartbeatWG.Done()
		if err := w.heartbeatRunLease(runCtx, run.ID, claimToken); err != nil {
			heartbeatErrMu.Lock()
			heartbeatErr = err
			heartbeatErrMu.Unlock()
			cancel()
		}
	}()

	execErr := w.runClaimedExecution(runCtx, run, claimToken)
	cancel()
	heartbeatWG.Wait()
	heartbeatErrMu.Lock()
	deferredHeartbeatErr := heartbeatErr
	heartbeatErrMu.Unlock()

	if execErr == nil && deferredHeartbeatErr != nil {
		execErr = deferredHeartbeatErr
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return execErr
}

func (w *BuilderWorker) heartbeatRunLease(ctx context.Context, runID, claimToken string) error {
	interval := w.heartbeatInterval
	if interval <= 0 {
		return nil
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if _, err := HeartbeatBuilderRunLease(ctx, runID, claimToken, w.leaseDuration); err != nil {
				if ctx.Err() != nil {
					return nil
				}
				return err
			}
		}
	}
}

func (w *BuilderWorker) runClaimedExecution(ctx context.Context, run *entities.BuilderRun, claimToken string) error {
	if cancelled, err := finalizeClaimedBuilderRunCancellation(ctx, run, claimToken, "[system] run cancelled before execution started\n"); cancelled || err != nil {
		return err
	}

	if err := appendOwnedBuilderRunLogEvent(ctx, run.ID, claimToken, "[system] run started\n"); err != nil {
		return err
	}

	if err := updateClaimedBuilderRunState(ctx, run.ID, claimToken, entities.BuilderRunPhasePreparingExecutor, nil, nil); err != nil {
		return err
	}
	if err := appendOwnedBuilderRunStatusEvent(ctx, run.ID, claimToken, entities.BuilderRunEventLevelInfo, "[system] preparing workspace\n"); err != nil {
		return err
	}

	workspace, err := ProvisionBuilderWorkspace(ctx, run.SessionID)
	if err != nil {
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerProvisioningErrorCode,
			builderWorkerProvisioningClass,
			"workspace provisioning failed: "+err.Error(),
			false,
		)
	}

	run.WorkspaceID = builderStringPtr(workspace.ID)
	run.ExecutorHandleID = workspace.ExecutorHandleID
	if err := updateClaimedBuilderRunState(ctx, run.ID, claimToken, entities.BuilderRunPhasePreparingExecutor, run.WorkspaceID, run.ExecutorHandleID); err != nil {
		return err
	}

	if cancelled, err := finalizeClaimedBuilderRunCancellation(ctx, run, claimToken, "[system] run cancelled after workspace preparation\n"); cancelled || err != nil {
		return err
	}

	messages, err := loadBuilderConversationMessages(ctx, run.SessionID)
	if err != nil {
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerGenerationErrorCode,
			builderWorkerUnknownErrorClass,
			"failed to load conversation: "+err.Error(),
			true,
		)
	}

	if err := updateClaimedBuilderRunState(ctx, run.ID, claimToken, entities.BuilderRunPhaseGenerating, run.WorkspaceID, run.ExecutorHandleID); err != nil {
		return err
	}
	if err := appendOwnedBuilderRunLogEvent(ctx, run.ID, claimToken, "[agent] generating files...\n"); err != nil {
		return err
	}

	result, err := GenerateBuilderFiles(ctx, messages)
	if err != nil {
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerGenerationErrorCode,
			builderWorkerUnknownErrorClass,
			err.Error(),
			true,
		)
	}

	if cancelled, err := finalizeClaimedBuilderRunCancellation(ctx, run, claimToken, "[system] run cancelled after generation\n"); cancelled || err != nil {
		return err
	}

	if len(result.Files) == 0 {
		return finalizeClaimedBuilderRunSuccess(
			ctx,
			run,
			claimToken,
			result.AssistantMessage,
			"run completed: no files generated",
		)
	}

	if err := updateClaimedBuilderRunState(ctx, run.ID, claimToken, entities.BuilderRunPhaseMaterializingFiles, run.WorkspaceID, run.ExecutorHandleID); err != nil {
		return err
	}
	for _, file := range result.Files {
		if err := appendOwnedBuilderRunLogEvent(ctx, run.ID, claimToken, "[agent] writing "+file.Path+"\n"); err != nil {
			return err
		}
	}

	if err := writeBuilderAgentFiles(ctx, workspace, run, result.Files); err != nil {
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerFileWriteErrorCode,
			builderWorkerWorkspaceIOClass,
			err.Error(),
			true,
		)
	}

	if err := appendOwnedBuilderRunLogEvent(ctx, run.ID, claimToken, fmt.Sprintf("[system] %d files written\n", len(result.Files))); err != nil {
		return err
	}

	return finalizeClaimedBuilderRunSuccess(
		ctx,
		run,
		claimToken,
		result.AssistantMessage,
		fmt.Sprintf("run completed: %d files generated", len(result.Files)),
	)
}

func finalizeClaimedBuilderRunSuccess(ctx context.Context, run *entities.BuilderRun, claimToken, assistantMessage, summaryMessage string) error {
	if cancelled, err := finalizeClaimedBuilderRunCancellation(ctx, run, claimToken, "[system] run cancelled before completion was finalized\n"); cancelled || err != nil {
		return err
	}

	if err := appendOwnedBuilderSessionAssistantMessage(ctx, run.SessionID, run.ID, claimToken, assistantMessage); err != nil {
		return err
	}
	if err := appendOwnedBuilderSessionSystemMessage(ctx, run.SessionID, run.ID, claimToken, summaryMessage); err != nil {
		return err
	}
	if err := appendOwnedBuilderRunStatusEvent(ctx, run.ID, claimToken, entities.BuilderRunEventLevelInfo, "[system] "+summaryMessage+"\n"); err != nil {
		return err
	}

	_, err := FinalizeBuilderRun(ctx, BuilderRunFinalizeInput{
		RunID:           run.ID,
		ClaimToken:      claimToken,
		Status:          entities.BuilderRunStatusSucceeded,
		WorkspaceUsable: true,
	})
	return err
}

func finalizeClaimedBuilderRunCancellation(ctx context.Context, run *entities.BuilderRun, claimToken, eventMessage string) (bool, error) {
	ownedRun, err := loadOwnedExecutingBuilderRun(db.DB.WithContext(ctx), run.ID, claimToken)
	if err != nil {
		return false, err
	}
	if ownedRun.CancelRequestedAt == nil {
		return false, nil
	}

	if err := appendOwnedBuilderRunStatusEvent(ctx, run.ID, claimToken, entities.BuilderRunEventLevelWarn, eventMessage); err != nil {
		return false, err
	}
	if err := appendOwnedBuilderSessionSystemMessage(ctx, run.SessionID, run.ID, claimToken, "run cancelled"); err != nil {
		return false, err
	}

	errorCode := builderWorkerCancelRequestedCode
	errorClass := builderWorkerCancelledErrorClass
	_, err = FinalizeBuilderRun(ctx, BuilderRunFinalizeInput{
		RunID:           run.ID,
		ClaimToken:      claimToken,
		Status:          entities.BuilderRunStatusCancelled,
		ErrorCode:       &errorCode,
		ErrorClass:      &errorClass,
		ErrorMessage:    "The builder run was cancelled.",
		WorkspaceUsable: true,
	})
	return true, err
}

func finalizeClaimedBuilderRunFailure(ctx context.Context, run *entities.BuilderRun, claimToken, errorCode, errorClass, errorMessage string, workspaceUsable bool) error {
	if err := appendOwnedBuilderRunLogEvent(ctx, run.ID, claimToken, "[system] run failed: "+errorMessage+"\n"); err != nil {
		return err
	}
	if err := appendOwnedBuilderRunStatusEvent(ctx, run.ID, claimToken, entities.BuilderRunEventLevelError, "[system] run failed\n"); err != nil {
		return err
	}
	if err := appendOwnedBuilderSessionSystemMessage(ctx, run.SessionID, run.ID, claimToken, "run failed: "+errorMessage); err != nil {
		return err
	}

	resolvedErrorCode := errorCode
	resolvedErrorClass := errorClass
	_, err := FinalizeBuilderRun(ctx, BuilderRunFinalizeInput{
		RunID:           run.ID,
		ClaimToken:      claimToken,
		Status:          entities.BuilderRunStatusFailed,
		ErrorCode:       &resolvedErrorCode,
		ErrorClass:      &resolvedErrorClass,
		ErrorMessage:    errorMessage,
		WorkspaceUsable: workspaceUsable,
	})
	return err
}

func ensureClaimedBuilderRunOwnership(ctx context.Context, runID, claimToken string) error {
	_, err := loadOwnedExecutingBuilderRun(db.DB.WithContext(ctx), runID, claimToken)
	return err
}

func appendOwnedBuilderRunLogEvent(ctx context.Context, runID, claimToken, message string) error {
	return appendOwnedBuilderRunEvent(ctx, runID, claimToken, BuilderRunEventInput{
		Kind:    entities.BuilderRunEventKindLog,
		Message: message,
	})
}

func appendOwnedBuilderRunEvent(ctx context.Context, runID, claimToken string, input BuilderRunEventInput) error {
	var event *entities.BuilderRunEvent
	err := withOwnedExecutingBuilderRunTx(ctx, runID, claimToken, func(tx *gorm.DB, _ *entities.BuilderRun) error {
		var err error
		event, err = appendBuilderRunEventTx(tx, runID, input)
		return err
	})
	if err != nil {
		return err
	}
	publishBuilderRunEvent(event)
	return nil
}

func withOwnedExecutingBuilderRunTx(ctx context.Context, runID, claimToken string, fn func(tx *gorm.DB, run *entities.BuilderRun) error) error {
	if fn == nil {
		return errors.New("owned builder run transaction callback is required")
	}

	return db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		run, err := loadOwnedExecutingBuilderRunForUpdate(tx, runID, claimToken)
		if err != nil {
			return err
		}
		return fn(tx, run)
	})
}

func loadOwnedExecutingBuilderRunForUpdate(tx *gorm.DB, runID, claimToken string) (*entities.BuilderRun, error) {
	var run entities.BuilderRun
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND status = ? AND claim_token = ?", runID, entities.BuilderRunStatusExecuting, claimToken).First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		return nil, err
	}
	return &run, nil
}

func appendOwnedBuilderRunStatusEvent(ctx context.Context, runID, claimToken string, level entities.BuilderRunEventLevel, message string) error {
	return appendOwnedBuilderRunEvent(ctx, runID, claimToken, BuilderRunEventInput{
		Kind:    entities.BuilderRunEventKindStatus,
		Level:   level,
		Message: message,
	})
}

func appendOwnedBuilderSessionAssistantMessage(ctx context.Context, sessionID, runID, claimToken, content string) error {
	return withOwnedExecutingBuilderRunTx(ctx, runID, claimToken, func(tx *gorm.DB, _ *entities.BuilderRun) error {
		return appendBuilderSessionMessageTx(tx, sessionID, runID, entities.BuilderMessageRoleAssistant, content)
	})
}

func appendOwnedBuilderSessionSystemMessage(ctx context.Context, sessionID, runID, claimToken, content string) error {
	return withOwnedExecutingBuilderRunTx(ctx, runID, claimToken, func(tx *gorm.DB, _ *entities.BuilderRun) error {
		return appendBuilderSessionMessageTx(tx, sessionID, runID, entities.BuilderMessageRoleSystem, content)
	})
}

func updateClaimedBuilderRunState(ctx context.Context, runID, claimToken string, phase entities.BuilderRunPhase, workspaceID, executorHandleID *string) error {
	now := time.Now().UTC()
	updates := map[string]any{
		"phase":      phase,
		"updated_at": now,
	}
	if workspaceID != nil {
		updates["workspace_id"] = *workspaceID
	}
	if executorHandleID != nil {
		updates["executor_handle_id"] = *executorHandleID
	}

	result := db.DB.WithContext(ctx).
		Model(&entities.BuilderRun{}).
		Where("id = ? AND status = ? AND claim_token = ?", runID, entities.BuilderRunStatusExecuting, claimToken).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func PreflightBuilderWorkerStartup(ctx context.Context) error {
	if err := NormalizeLegacyBuilderRunsForControlPlane(ctx); err != nil {
		return err
	}

	runIDs, err := listLegacyExecutingBuilderRunIDs(db.DB.WithContext(ctx))
	if err != nil {
		return err
	}
	if len(runIDs) == 0 {
		return nil
	}

	return &BuilderWorkerStartupPreflightError{
		LegacyExecutingRunCount: int64(len(runIDs)),
		LegacyExecutingRunIDs:   runIDs,
	}
}

func listLegacyExecutingBuilderRunIDs(tx *gorm.DB) ([]string, error) {
	runIDs := []string{}
	err := tx.Model(&entities.BuilderRun{}).
		Where(
			"status = ? AND (phase IS NULL OR claim_token IS NULL OR claimed_at IS NULL OR heartbeat_at IS NULL OR timeout_at IS NULL)",
			entities.BuilderRunStatusExecuting,
		).
		Order("created_at ASC, id ASC").
		Pluck("id", &runIDs).Error
	if err != nil {
		return nil, err
	}
	return runIDs, nil
}
