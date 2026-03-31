package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	builderWorkerDefaultPollInterval       = 2 * time.Second
	builderWorkerDefaultLeaseDuration      = 2 * time.Minute
	builderWorkerDefaultHeartbeatInterval  = 30 * time.Second
	builderWorkerDefaultCancelPollInterval = 250 * time.Millisecond

	builderWorkerLeaseExpiredErrorCode  = "lease_expired"
	builderWorkerCancelRequestedCode    = "cancel_requested"
	builderWorkerTimeoutErrorClass      = "timeout"
	builderWorkerCancelledErrorClass    = "cancelled"
	builderWorkerUnknownErrorClass      = "unknown"
	builderWorkerProvisioningErrorCode  = "workspace_provision_failed"
	builderWorkerProvisioningClass      = "executor_provisioning"
	builderWorkerGenerationErrorCode    = "generation_failed"
	builderWorkerFileWriteErrorCode     = "workspace_write_failed"
	builderWorkerWorkspaceIOClass       = "workspace_io"
	builderWorkerExecutionPlanErrorCode = "frontend_execution_plan_failed"
	builderWorkerExecutionErrorCode     = "frontend_execution_failed"
	builderWorkerValidationErrorCode    = "output_validation_failed"
	builderWorkerArtifactErrorCode      = "build_artifact_collection_failed"
	builderWorkerExecutionPlaneClass    = "execution_plane"
	builderWorkerCommandExitCodeMarker  = "__KETCHES_BUILDER_EXIT_CODE__:"
)

const builderWorkerPhaseInstallingDependencies entities.BuilderRunPhase = "installing_dependencies"

var (
	ErrBuilderWorkerStartupPreflightBlocked = errors.New("builder worker startup preflight blocked")
	GlobalBuilderWorker                     = NewBuilderWorker()
	builderWorkerProvisionWorkspace         = ProvisionBuilderWorkspace
	builderWorkerLoadConversationMessages   = loadBuilderConversationMessages
	builderWorkerGenerateFiles              = GenerateBuilderFilesWithSelection
	builderWorkerWriteAgentFiles            = writeBuilderAgentFiles
	builderWorkerListWorkspaceFiles         = defaultBuilderWorkerListWorkspaceFiles
	builderWorkerRunFrontendCommand         = defaultBuilderWorkerRunFrontendCommand
	builderWorkerExecCommandWithContext     = execCommandWithContext
	builderWorkerPublishOutputSnapshot      = PublishBuilderOutputSnapshot
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

	messages, err := builderWorkerLoadConversationMessages(ctx, run.SessionID)
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

	projectID, err := loadBuilderSessionProjectID(ctx, run.SessionID)
	if err != nil {
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerGenerationErrorCode,
			builderWorkerUnknownErrorClass,
			"failed to load builder session context: "+err.Error(),
			true,
		)
	}

	result, err := builderWorkerGenerateFiles(
		withBuilderRunGenerationContext(ctx, projectID, run),
		messages,
		stringPointerValue(run.ProviderKey),
		stringPointerValue(run.ModelProfileKey),
	)
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

	if result.Action == BuilderAgentActionReplyOnly {
		return finalizeClaimedBuilderRunSuccess(
			ctx,
			run,
			claimToken,
			result.AssistantMessage,
			"run completed: replied without workspace changes",
		)
	}

	if err := updateClaimedBuilderRunState(ctx, run.ID, claimToken, entities.BuilderRunPhasePreparingExecutor, nil, nil); err != nil {
		return err
	}
	if err := appendOwnedBuilderRunStatusEvent(ctx, run.ID, claimToken, entities.BuilderRunEventLevelInfo, "[system] preparing workspace\n"); err != nil {
		return err
	}

	executionSelection, err := ResolveBuilderExecutionSelection(ctx, run)
	if err != nil {
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerProvisioningErrorCode,
			builderWorkerProvisioningClass,
			"failed to resolve execution selection: "+err.Error(),
			false,
		)
	}
	if err := PersistBuilderRunExecutionSelection(ctx, run.ID, executionSelection); err != nil {
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerProvisioningErrorCode,
			builderWorkerProvisioningClass,
			"failed to persist execution selection: "+err.Error(),
			false,
		)
	}
	run.ExecutorPolicyKey = builderStringPtr(executionSelection.ExecutorPolicyKey)
	run.ExecutionImageProfileKey = builderStringPtr(executionSelection.ExecutionImageProfileKey)
	run.ExecutionImageRef = builderStringPtr(executionSelection.ExecutionImageRef)

	workspace, err := builderWorkerProvisionWorkspace(ctx, run.SessionID)
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

	if err := updateClaimedBuilderRunState(ctx, run.ID, claimToken, entities.BuilderRunPhaseMaterializingFiles, run.WorkspaceID, run.ExecutorHandleID); err != nil {
		return err
	}
	for _, file := range result.Files {
		if err := appendOwnedBuilderRunLogEvent(ctx, run.ID, claimToken, "[agent] writing "+file.Path+"\n"); err != nil {
			return err
		}
	}

	if err := builderWorkerWriteAgentFiles(ctx, workspace, run, result.Files); err != nil {
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

	if cancelled, err := finalizeClaimedBuilderRunCancellation(ctx, run, claimToken, "[system] run cancelled before frontend execution started\n"); cancelled || err != nil {
		return err
	}

	workspaceListing, err := builderWorkerListWorkspaceFiles(ctx, workspace, workspace.WorkspaceRoot)
	if err != nil {
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerExecutionPlanErrorCode,
			builderWorkerExecutionPlaneClass,
			"failed to inspect workspace for frontend execution: "+err.Error(),
			true,
		)
	}

	frontendPlan, err := DetectBuilderFrontendExecutionPlan(workspaceListing)
	if err != nil {
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerExecutionPlanErrorCode,
			builderWorkerExecutionPlaneClass,
			err.Error(),
			true,
		)
	}

	frontendRunner := BuilderFrontendCommandRunnerFunc(func(ctx context.Context, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
		phase := builderWorkerPhaseInstallingDependencies
		if step == BuilderFrontendExecutionStepBuild {
			phase = entities.BuilderRunPhaseBuilding
		}
		if err := updateClaimedBuilderRunState(ctx, run.ID, claimToken, phase, run.WorkspaceID, run.ExecutorHandleID); err != nil {
			return BuilderFrontendCommandResult{}, err
		}
		return builderWorkerRunFrontendCommand(ctx, workspace, step, command, appendLog)
	})

	frontendExecutionCtx, stopFrontendExecutionWatcher := w.newClaimedRunFrontendExecutionContext(ctx, run.ID, claimToken)
	defer stopFrontendExecutionWatcher()

	if _, err := executeBuilderFrontendPlanWithEventWriter(frontendExecutionCtx, run.ID, frontendPlan, frontendRunner, builderFrontendExecutionEventWriter{
		appendLog: func(ctx context.Context, message string) error {
			return appendOwnedBuilderRunLogEvent(ctx, run.ID, claimToken, message)
		},
		appendStatus: func(ctx context.Context, level entities.BuilderRunEventLevel, message string) error {
			return appendOwnedBuilderRunStatusEvent(ctx, run.ID, claimToken, level, message)
		},
	}); err != nil {
		if errors.Is(err, context.Canceled) {
			if cancelled, cancelErr := finalizeClaimedBuilderRunCancellation(ctx, run, claimToken, "[system] run cancelled during frontend execution\n"); cancelled || cancelErr != nil {
				return cancelErr
			}
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerExecutionErrorCode,
			builderWorkerExecutionPlaneClass,
			err.Error(),
			true,
		)
	}

	if cancelled, err := finalizeClaimedBuilderRunCancellation(ctx, run, claimToken, "[system] run cancelled after frontend execution completed\n"); cancelled || err != nil {
		return err
	}

	buildArtifactCount, err := collectAndPersistBuilderBuildArtifacts(ctx, workspace, run, claimToken)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			if cancelled, cancelErr := finalizeClaimedBuilderRunCancellation(ctx, run, claimToken, "[system] run cancelled before build artifacts were persisted\n"); cancelled || cancelErr != nil {
				return cancelErr
			}
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerArtifactErrorCode,
			builderWorkerExecutionPlaneClass,
			err.Error(),
			true,
		)
	}
	if err := ensureClaimedBuilderRunOwnership(ctx, run.ID, claimToken); err != nil {
		return err
	}

	if err := updateClaimedBuilderRunState(ctx, run.ID, claimToken, entities.BuilderRunPhaseTesting, run.WorkspaceID, run.ExecutorHandleID); err != nil {
		return err
	}
	if err := appendOwnedBuilderRunStatusEvent(ctx, run.ID, claimToken, entities.BuilderRunEventLevelInfo, "[system] validating build outputs\n"); err != nil {
		return err
	}
	buildArtifacts, err := listBuilderOutputArtifacts(ctx, run.ID)
	if err != nil {
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerValidationErrorCode,
			builderWorkerExecutionPlaneClass,
			"failed to load build artifacts for validation: "+err.Error(),
			true,
		)
	}
	if err := ValidateBuilderExecutionOutputs(run, buildArtifacts); err != nil {
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerValidationErrorCode,
			builderWorkerExecutionPlaneClass,
			err.Error(),
			true,
		)
	}
	if err := appendOwnedBuilderRunStatusEvent(ctx, run.ID, claimToken, entities.BuilderRunEventLevelInfo, "[system] build output validation completed\n"); err != nil {
		return err
	}
	if err := ensureClaimedBuilderRunOwnership(ctx, run.ID, claimToken); err != nil {
		return err
	}

	runtimeValidationCommand, err := DetectBuilderRuntimeValidationCommand(run, buildArtifacts)
	if err != nil {
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerValidationErrorCode,
			builderWorkerExecutionPlaneClass,
			err.Error(),
			true,
		)
	}
	if len(runtimeValidationCommand) > 0 {
		if err := updateClaimedBuilderRunState(ctx, run.ID, claimToken, entities.BuilderRunPhaseTesting, run.WorkspaceID, run.ExecutorHandleID); err != nil {
			return err
		}
		if err := appendOwnedBuilderRunStatusEvent(ctx, run.ID, claimToken, entities.BuilderRunEventLevelInfo, "[system] running runtime validation\n"); err != nil {
			return err
		}
		commandResult, err := builderWorkerRunFrontendCommand(ctx, workspace, BuilderFrontendExecutionStepValidate, runtimeValidationCommand, func(message string) error {
			return appendOwnedBuilderRunLogEvent(ctx, run.ID, claimToken, message)
		})
		if err != nil {
			return finalizeClaimedBuilderRunFailure(
				ctx,
				run,
				claimToken,
				builderWorkerValidationErrorCode,
				builderWorkerExecutionPlaneClass,
				err.Error(),
				true,
			)
		}
		if commandResult.ExitCode != 0 {
			return finalizeClaimedBuilderRunFailure(
				ctx,
				run,
				claimToken,
				builderWorkerValidationErrorCode,
				builderWorkerExecutionPlaneClass,
				fmt.Sprintf("validate command exited with status %d", commandResult.ExitCode),
				true,
			)
		}
		if err := appendOwnedBuilderRunStatusEvent(ctx, run.ID, claimToken, entities.BuilderRunEventLevelInfo, "[system] runtime validation completed\n"); err != nil {
			return err
		}
		if err := ensureClaimedBuilderRunOwnership(ctx, run.ID, claimToken); err != nil {
			return err
		}
	}

	snapshotRun := *run
	snapshotRun.Status = entities.BuilderRunStatusSucceeded
	snapshot, err := builderWorkerPublishOutputSnapshot(ctx, workspace, &snapshotRun)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			if cancelled, cancelErr := finalizeClaimedBuilderRunCancellation(ctx, run, claimToken, "[system] run cancelled before preview snapshot was published\n"); cancelled || cancelErr != nil {
				return cancelErr
			}
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		return finalizeClaimedBuilderRunFailure(
			ctx,
			run,
			claimToken,
			builderWorkerArtifactErrorCode,
			builderWorkerExecutionPlaneClass,
			"failed to publish preview snapshot: "+err.Error(),
			true,
		)
	}
	if err := ensureClaimedBuilderRunOwnership(ctx, run.ID, claimToken); err != nil {
		if cleanupErr := DeleteBuilderOutputSnapshotsByRunID(ctx, run.ID); cleanupErr != nil {
			return errors.Join(err, cleanupErr)
		}
		return err
	}
	if snapshot != nil {
		payloadJSON := fmt.Sprintf(`{"snapshot_id":%q,"status":%q}`, snapshot.ID, snapshot.Status)
		if err := appendOwnedBuilderRunEvent(ctx, run.ID, claimToken, BuilderRunEventInput{
			Kind:        entities.BuilderRunEventKindPreview,
			Level:       entities.BuilderRunEventLevelInfo,
			Message:     "[system] preview snapshot published\n",
			PayloadJSON: payloadJSON,
		}); err != nil {
			return err
		}
	}

	return finalizeClaimedBuilderRunSuccess(
		ctx,
		run,
		claimToken,
		result.AssistantMessage,
		fmt.Sprintf("run completed: %d files generated and %d build artifacts collected", len(result.Files), buildArtifactCount),
	)
}

func defaultBuilderWorkerListWorkspaceFiles(ctx context.Context, workspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
	if workspace == nil {
		return nil, errors.New("builder workspace is required")
	}

	appCtx, err := buildBuilderWorkspaceAppContext(workspace)
	if err != nil {
		return nil, err
	}

	return listBuilderWorkspaceFilesInContainer(appCtx, workspace.PodName, workspace.ContainerName, requestedPath)
}

func defaultBuilderWorkerRunFrontendCommand(ctx context.Context, workspace *entities.BuilderWorkspace, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
	if workspace == nil {
		return BuilderFrontendCommandResult{}, errors.New("builder workspace is required")
	}
	if appendLog == nil {
		return BuilderFrontendCommandResult{}, errors.New("builder frontend append log callback is required")
	}
	if err := ctx.Err(); err != nil {
		return BuilderFrontendCommandResult{}, err
	}

	appCtx, err := buildBuilderWorkspaceAppContext(workspace)
	if err != nil {
		return BuilderFrontendCommandResult{}, err
	}

	commandScript, err := buildBuilderWorkerFrontendCommandScript(workspace.WorkspaceRoot, command)
	if err != nil {
		return BuilderFrontendCommandResult{}, err
	}
	if builderWorkerExecCommandWithContext == nil {
		return BuilderFrontendCommandResult{}, errors.New("builder worker exec command helper is required")
	}

	stdout, stderr, err := builderWorkerExecCommandWithContext(ctx, appCtx, workspace.PodName, workspace.ContainerName, []string{"sh", "-lc", commandScript})
	if err != nil {
		stdoutToAppend := stdout
		if stdout != "" {
			if cleanStdout, _, parseErr := parseBuilderWorkerFrontendCommandOutput(stdout); parseErr == nil {
				stdoutToAppend = cleanStdout
			}
		}
		if stdoutToAppend != "" {
			if appendErr := appendLog(stdoutToAppend); appendErr != nil {
				return BuilderFrontendCommandResult{}, appendErr
			}
		}
		if stderr != "" {
			if appendErr := appendLog(stderr); appendErr != nil {
				return BuilderFrontendCommandResult{}, appendErr
			}
		}
		return BuilderFrontendCommandResult{}, err
	}

	cleanStdout, exitCode, err := parseBuilderWorkerFrontendCommandOutput(stdout)
	if err != nil {
		return BuilderFrontendCommandResult{}, err
	}

	if cleanStdout != "" {
		if err := appendLog(cleanStdout); err != nil {
			return BuilderFrontendCommandResult{}, err
		}
	}
	if stderr != "" {
		if err := appendLog(stderr); err != nil {
			return BuilderFrontendCommandResult{}, err
		}
	}
	if err := ctx.Err(); err != nil {
		return BuilderFrontendCommandResult{}, err
	}

	return BuilderFrontendCommandResult{ExitCode: exitCode}, nil
}

func buildBuilderWorkerFrontendCommandScript(workspaceRoot string, command []string) (string, error) {
	cleanWorkspaceRoot := strings.TrimSpace(workspaceRoot)
	if cleanWorkspaceRoot == "" {
		return "", errors.New("builder workspace root is required")
	}
	if len(command) == 0 {
		return "", errors.New("builder frontend command is required")
	}

	quotedCommand := make([]string, 0, len(command))
	for i := range command {
		quotedCommand = append(quotedCommand, quoteBuilderWorkerShellArg(command[i]))
	}

	return strings.Join([]string{
		"cd " + quoteBuilderWorkerShellArg(cleanWorkspaceRoot),
		"status=$?",
		"if [ \"$status\" -eq 0 ]; then",
		strings.Join(quotedCommand, " "),
		"status=$?",
		"fi",
		"printf '\\n" + builderWorkerCommandExitCodeMarker + "%s\\n' \"$status\"",
	}, "\n"), nil
}

func quoteBuilderWorkerShellArg(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `"'"'`) + "'"
}

func parseBuilderWorkerFrontendCommandOutput(stdout string) (string, int, error) {
	trimmedOutput := stdout
	markerIndex := strings.LastIndex(trimmedOutput, "\n"+builderWorkerCommandExitCodeMarker)
	markerStart := 0
	if markerIndex >= 0 {
		markerStart = markerIndex + 1
	} else if strings.HasPrefix(trimmedOutput, builderWorkerCommandExitCodeMarker) {
		markerStart = 0
	} else {
		return "", 0, errors.New("builder frontend command output missing exit code marker")
	}

	markerLine := strings.TrimSpace(trimmedOutput[markerStart:])
	exitCodeText := strings.TrimPrefix(markerLine, builderWorkerCommandExitCodeMarker)
	exitCode, err := strconv.Atoi(exitCodeText)
	if err != nil {
		return "", 0, fmt.Errorf("parse builder frontend command exit code: %w", err)
	}

	return trimmedOutput[:markerStart], exitCode, nil
}

func collectAndPersistBuilderBuildArtifacts(ctx context.Context, workspace *entities.BuilderWorkspace, run *entities.BuilderRun, claimToken string) (int, error) {
	if workspace == nil {
		return 0, errors.New("builder workspace is required")
	}
	if run == nil {
		return 0, errors.New("builder run is required")
	}

	workspaceListing, err := builderWorkerListWorkspaceFiles(ctx, workspace, workspace.WorkspaceRoot)
	if err != nil {
		return 0, err
	}

	workspaceArtifacts, err := collectBuilderWorkspaceFileArtifacts(workspace, run, workspaceListing)
	if err != nil {
		return 0, err
	}

	outputRoot, err := detectBuilderWorkerOutputRootFromWorkspaceListing(workspaceListing)
	if err != nil {
		return 0, err
	}

	buildOutputListing, err := builderWorkerListWorkspaceFiles(ctx, workspace, path.Join(workspace.WorkspaceRoot, outputRoot))
	if err != nil {
		return 0, err
	}

	buildArtifacts, err := collectBuilderBuildArtifactsWithListFn(ctx, workspace, run, buildOutputListing, builderWorkerListWorkspaceFiles)
	if err != nil {
		return 0, err
	}

	artifacts := append(workspaceArtifacts, buildArtifacts...)
	if err := withOwnedExecutingBuilderRunTx(ctx, run.ID, claimToken, func(tx *gorm.DB, _ *entities.BuilderRun) error {
		return replaceBuilderWorkspaceArtifactsTx(tx, workspace, artifacts)
	}); err != nil {
		return 0, err
	}

	return len(buildArtifacts), nil
}

func (w *BuilderWorker) newClaimedRunFrontendExecutionContext(ctx context.Context, runID, claimToken string) (context.Context, context.CancelFunc) {
	frontendExecutionCtx, cancel := context.WithCancel(ctx)
	if strings.TrimSpace(runID) == "" || strings.TrimSpace(claimToken) == "" {
		return frontendExecutionCtx, cancel
	}

	pollInterval := w.frontendExecutionCancelPollInterval()
	go w.watchClaimedRunFrontendExecutionCancellation(frontendExecutionCtx, runID, claimToken, pollInterval, cancel)
	return frontendExecutionCtx, cancel
}

func (w *BuilderWorker) frontendExecutionCancelPollInterval() time.Duration {
	if w != nil && w.heartbeatInterval > 0 && w.heartbeatInterval < builderWorkerDefaultCancelPollInterval {
		return w.heartbeatInterval
	}
	return builderWorkerDefaultCancelPollInterval
}

func (w *BuilderWorker) watchClaimedRunFrontendExecutionCancellation(ctx context.Context, runID, claimToken string, pollInterval time.Duration, cancel context.CancelFunc) {
	if cancel == nil {
		return
	}
	if pollInterval <= 0 {
		pollInterval = builderWorkerDefaultCancelPollInterval
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			run, err := loadOwnedExecutingBuilderRun(db.DB.WithContext(ctx), runID, claimToken)
			if err != nil {
				if ctx.Err() != nil || errors.Is(err, gorm.ErrRecordNotFound) {
					return
				}
				continue
			}
			if run.CancelRequestedAt != nil {
				cancel()
				return
			}
		}
	}
}

func detectBuilderWorkerOutputRootFromWorkspaceListing(listing *models.ListFilesResponse) (string, error) {
	if listing == nil {
		return "", errors.New("builder workspace listing is required")
	}

	for _, candidate := range []string{builderBuildOutputRootDist, builderBuildOutputRootBuild, builderBuildOutputRootNext} {
		for _, file := range listing.Files {
			if file.Type == "dir" && file.Name == candidate {
				return candidate, nil
			}
		}
	}

	return "", errors.New("build succeeded but no supported output directory was found")
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
