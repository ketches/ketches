package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBuilderWorkerStartupRejectsLegacyExecutingRuns(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	projectID := "project-preflight"

	insertBuilderSessionSeed(t, now, projectID, "session-queued", "env-queued", entities.BuilderSessionStatusReady, "Queued legacy session")
	insertBuilderMessageSeed(t, now, "session-queued", "message-queued", "Queued prompt", "user-queued")
	insertLegacyBuilderRunSeed(t, now, "run-queued-legacy", "session-queued", "message-queued", entities.BuilderRunStatusQueued, nil)

	insertBuilderSessionSeed(t, now.Add(time.Minute), projectID, "session-terminal", "env-terminal", entities.BuilderSessionStatusReady, "Terminal legacy session")
	insertBuilderMessageSeed(t, now.Add(time.Minute), "session-terminal", "message-terminal", "Terminal prompt", "user-terminal")
	insertLegacyBuilderRunSeed(t, now.Add(time.Minute), "run-terminal-legacy", "session-terminal", "message-terminal", entities.BuilderRunStatusFailed, nil)

	insertBuilderSessionSeed(t, now.Add(2*time.Minute), projectID, "session-executing-legacy", "env-executing", entities.BuilderSessionStatusRunning, "Executing legacy session")
	insertBuilderMessageSeed(t, now.Add(2*time.Minute), "session-executing-legacy", "message-executing-legacy", "Executing prompt", "user-executing")
	insertLegacyBuilderRunSeed(t, now.Add(2*time.Minute), "run-executing-legacy", "session-executing-legacy", "message-executing-legacy", entities.BuilderRunStatusExecuting, nil)

	claimToken := "claim-token"
	claimedAt := now.Add(3 * time.Minute)
	heartbeatAt := claimedAt.Add(10 * time.Second)
	timeoutAt := claimedAt.Add(time.Minute)
	insertBuilderSessionSeed(t, now.Add(3*time.Minute), projectID, "session-executing-modern", "env-modern", entities.BuilderSessionStatusRunning, "Executing modern session")
	insertBuilderMessageSeed(t, now.Add(3*time.Minute), "session-executing-modern", "message-executing-modern", "Modern executing prompt", "user-modern")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-executing-modern",
		CreatedAt:          now.Add(3 * time.Minute),
		UpdatedAt:          now.Add(3 * time.Minute),
		SessionID:          "session-executing-modern",
		TriggerMessageID:   "message-executing-modern",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseClaiming),
		ClaimToken:         &claimToken,
		ClaimedAt:          &claimedAt,
		HeartbeatAt:        &heartbeatAt,
		TimeoutAt:          &timeoutAt,
		RequestedBy:        "user-modern",
		InstructionSummary: "Modern executing prompt",
		StartedAt:          &claimedAt,
	}).Error)

	err := PreflightBuilderWorkerStartup(context.Background())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBuilderWorkerStartupPreflightBlocked)

	var preflightErr *BuilderWorkerStartupPreflightError
	require.ErrorAs(t, err, &preflightErr)
	assert.Equal(t, int64(1), preflightErr.LegacyExecutingRunCount)
	assert.Equal(t, []string{"run-executing-legacy"}, preflightErr.LegacyExecutingRunIDs)

	var queuedRun entities.BuilderRun
	require.NoError(t, db.DB.First(&queuedRun, "id = ?", "run-queued-legacy").Error)
	require.NotNil(t, queuedRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseQueued, *queuedRun.Phase)

	var terminalRun entities.BuilderRun
	require.NoError(t, db.DB.First(&terminalRun, "id = ?", "run-terminal-legacy").Error)
	require.NotNil(t, terminalRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseFinalizing, *terminalRun.Phase)

	var legacyExecutingRun entities.BuilderRun
	require.NoError(t, db.DB.First(&legacyExecutingRun, "id = ?", "run-executing-legacy").Error)
	assert.Nil(t, legacyExecutingRun.Phase)

	var modernExecutingRun entities.BuilderRun
	require.NoError(t, db.DB.First(&modernExecutingRun, "id = ?", "run-executing-modern").Error)
	require.NotNil(t, modernExecutingRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseClaiming, *modernExecutingRun.Phase)
}

func TestBuilderWorkerStartupAllowsQueueOnlyAppendRuns(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	session := entities.BuilderSession{
		Base:           entities.Base{ID: "session-inline-era"},
		ProjectID:      "project-inline-era",
		BuildEnvID:     "env-inline-era",
		Title:          "Inline era session",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "user-inline-era",
		LastActivityAt: time.Now().UTC().Add(-time.Minute),
	}
	require.NoError(t, db.DB.Create(&session).Error)

	resp, err := AppendBuilderSessionMessage(context.Background(), session.ProjectID, session.ID, "user-inline-era", &models.AppendBuilderSessionMessageRequest{
		Content: "Run inline before worker enablement.",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Len(t, resp.Runs, 1)
	assert.Equal(t, string(entities.BuilderRunStatusQueued), resp.Runs[0].Status)

	preflightErr := PreflightBuilderWorkerStartup(context.Background())
	require.NoError(t, preflightErr)

	var executingRun entities.BuilderRun
	require.NoError(t, db.DB.First(&executingRun, "id = ?", resp.Runs[0].ID).Error)
	assert.Equal(t, entities.BuilderRunStatusQueued, executingRun.Status)
	require.NotNil(t, executingRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseQueued, *executingRun.Phase)
	assert.Nil(t, executingRun.StartedAt)
	assert.Nil(t, executingRun.ClaimToken)
	assert.Nil(t, executingRun.ClaimedAt)
	assert.Nil(t, executingRun.HeartbeatAt)
	assert.Nil(t, executingRun.TimeoutAt)
}

func TestBuilderWorkerClaimsQueuedRuns(t *testing.T) {
	t.Run("queue scan claims queued runs in oldest-first order", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)

		now := time.Now().UTC().Truncate(time.Second)
		insertQueuedBuilderRunSeed(t, now.Add(-2*time.Minute), "project-worker-queue", "session-worker-oldest", "message-worker-oldest", "run-worker-oldest", "Oldest queued prompt")
		insertQueuedBuilderRunSeed(t, now.Add(-time.Minute), "project-worker-queue", "session-worker-newer", "message-worker-newer", "run-worker-newer", "Newer queued prompt")

		worker := NewBuilderWorker()
		worker.pollInterval = 10 * time.Millisecond
		worker.leaseDuration = time.Minute

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		worker.SetParentContext(ctx)

		var mu sync.Mutex
		claimedOrder := make([]string, 0, 2)
		worker.handleClaimedRunFn = func(ctx context.Context, run *entities.BuilderRun) error {
			if run == nil || run.ClaimToken == nil || *run.ClaimToken == "" {
				return errors.New("claimed builder run is missing a claim token")
			}

			mu.Lock()
			claimedOrder = append(claimedOrder, run.ID)
			mu.Unlock()

			_, err := FinalizeBuilderRun(ctx, BuilderRunFinalizeInput{
				RunID:           run.ID,
				ClaimToken:      *run.ClaimToken,
				Status:          entities.BuilderRunStatusSucceeded,
				WorkspaceUsable: true,
			})
			return err
		}

		worker.Start()
		defer func() {
			worker.Stop()
			worker.Wait()
		}()

		require.Eventually(t, func() bool {
			mu.Lock()
			defer mu.Unlock()
			return len(claimedOrder) == 2
		}, time.Second, 20*time.Millisecond)

		mu.Lock()
		capturedOrder := append([]string(nil), claimedOrder...)
		mu.Unlock()
		assert.Equal(t, []string{"run-worker-oldest", "run-worker-newer"}, capturedOrder)
	})

	t.Run("polling claims queued runs without any nudge", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)

		worker := NewBuilderWorker()
		worker.pollInterval = 10 * time.Millisecond
		worker.leaseDuration = time.Minute

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		worker.SetParentContext(ctx)

		claimedRunIDs := make(chan string, 1)
		worker.handleClaimedRunFn = func(ctx context.Context, run *entities.BuilderRun) error {
			if run == nil || run.ClaimToken == nil || *run.ClaimToken == "" {
				return errors.New("claimed builder run is missing a claim token")
			}

			select {
			case claimedRunIDs <- run.ID:
			default:
			}

			_, err := FinalizeBuilderRun(ctx, BuilderRunFinalizeInput{
				RunID:           run.ID,
				ClaimToken:      *run.ClaimToken,
				Status:          entities.BuilderRunStatusSucceeded,
				WorkspaceUsable: true,
			})
			return err
		}

		worker.Start()
		defer func() {
			worker.Stop()
			worker.Wait()
		}()

		now := time.Now().UTC().Truncate(time.Second)
		insertQueuedBuilderRunSeed(t, now, "project-worker-poll", "session-worker-poll", "message-worker-poll", "run-worker-poll", "Polling should claim me")

		select {
		case runID := <-claimedRunIDs:
			assert.Equal(t, "run-worker-poll", runID)
		case <-time.After(time.Second):
			t.Fatal("expected worker polling loop to claim queued run without a nudge")
		}
	})
}

func TestBuilderWorkerRecoversExpiredExecutingRuns(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	claimToken := "worker-expired-claim"
	claimedAt := now.Add(-8 * time.Minute)
	heartbeatAt := now.Add(-6 * time.Minute)
	timeoutAt := now.Add(-3 * time.Minute)

	insertBuilderSessionSeed(t, now.Add(-9*time.Minute), "project-worker-recovery", "session-worker-recovery", "env-worker-recovery", entities.BuilderSessionStatusRunning, "Worker recovery session")
	insertBuilderMessageSeed(t, now.Add(-9*time.Minute), "session-worker-recovery", "message-worker-recovery", "Recover the expired run", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-worker-recovery",
		CreatedAt:          now.Add(-8 * time.Minute),
		UpdatedAt:          now.Add(-8 * time.Minute),
		SessionID:          "session-worker-recovery",
		TriggerMessageID:   "message-worker-recovery",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseGenerating),
		AttemptCount:       0,
		MaxAttempts:        3,
		ClaimToken:         &claimToken,
		ClaimedAt:          &claimedAt,
		HeartbeatAt:        &heartbeatAt,
		TimeoutAt:          &timeoutAt,
		RequestedBy:        "worker-user",
		InstructionSummary: "Recover the expired run",
		StartedAt:          &claimedAt,
	}).Error)

	worker := NewBuilderWorker()
	worker.nowFn = func() time.Time {
		return now
	}

	err := worker.RecoverActiveRuns(context.Background())
	require.NoError(t, err)

	var recoveredRun entities.BuilderRun
	require.NoError(t, db.DB.First(&recoveredRun, "id = ?", "run-worker-recovery").Error)
	assert.Equal(t, entities.BuilderRunStatusQueued, recoveredRun.Status)
	require.NotNil(t, recoveredRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseQueued, *recoveredRun.Phase)
	assert.Equal(t, 1, recoveredRun.AttemptCount)
	assert.Nil(t, recoveredRun.ClaimToken)
	assert.Nil(t, recoveredRun.ClaimedAt)
	assert.Nil(t, recoveredRun.HeartbeatAt)
	assert.Nil(t, recoveredRun.TimeoutAt)
	require.NotNil(t, recoveredRun.ErrorCode)
	require.NotNil(t, recoveredRun.ErrorClass)
	assert.Equal(t, "lease_expired", *recoveredRun.ErrorCode)
	assert.Equal(t, "timeout", *recoveredRun.ErrorClass)
	assert.Contains(t, recoveredRun.ErrorMessage, "lease expired")

	events, err := ReplayBuilderRunEventsAfterCursor(context.Background(), recoveredRun.ID, 0)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	assert.Equal(t, entities.BuilderRunEventKindStatus, events[len(events)-1].Kind)
	assert.Contains(t, events[len(events)-1].Message, "lease expired")
}

func TestBuilderWorkerRecoveryHonorsCancellationRequest(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	claimToken := "worker-expired-cancel-claim"
	claimedAt := now.Add(-8 * time.Minute)
	heartbeatAt := now.Add(-6 * time.Minute)
	timeoutAt := now.Add(-3 * time.Minute)
	cancelRequestedAt := now.Add(-2 * time.Minute)

	insertBuilderSessionSeed(t, now.Add(-9*time.Minute), "project-worker-recovery-cancel", "session-worker-recovery-cancel", "env-worker-recovery-cancel", entities.BuilderSessionStatusRunning, "Worker recovery cancelled session")
	insertBuilderMessageSeed(t, now.Add(-9*time.Minute), "session-worker-recovery-cancel", "message-worker-recovery-cancel", "Recover the cancelled expired run", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-worker-recovery-cancel",
		CreatedAt:          now.Add(-8 * time.Minute),
		UpdatedAt:          now.Add(-8 * time.Minute),
		SessionID:          "session-worker-recovery-cancel",
		TriggerMessageID:   "message-worker-recovery-cancel",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseGenerating),
		AttemptCount:       3,
		MaxAttempts:        3,
		ClaimToken:         &claimToken,
		ClaimedAt:          &claimedAt,
		HeartbeatAt:        &heartbeatAt,
		TimeoutAt:          &timeoutAt,
		CancelRequestedAt:  &cancelRequestedAt,
		RequestedBy:        "worker-user",
		InstructionSummary: "Recover the cancelled expired run",
		StartedAt:          &claimedAt,
	}).Error)

	worker := NewBuilderWorker()
	worker.nowFn = func() time.Time {
		return now
	}

	err := worker.RecoverActiveRuns(context.Background())
	require.NoError(t, err)

	var recoveredRun entities.BuilderRun
	require.NoError(t, db.DB.First(&recoveredRun, "id = ?", "run-worker-recovery-cancel").Error)
	assert.Equal(t, entities.BuilderRunStatusCancelled, recoveredRun.Status)
	require.NotNil(t, recoveredRun.ErrorCode)
	require.NotNil(t, recoveredRun.ErrorClass)
	assert.Equal(t, builderWorkerCancelRequestedCode, *recoveredRun.ErrorCode)
	assert.Equal(t, builderWorkerCancelledErrorClass, *recoveredRun.ErrorClass)
	assert.Equal(t, "The builder run was cancelled.", recoveredRun.ErrorMessage)

	events, err := ReplayBuilderRunEventsAfterCursor(context.Background(), recoveredRun.ID, 0)
	require.NoError(t, err)
	require.NotEmpty(t, events)
	assert.Equal(t, entities.BuilderRunEventKindStatus, events[len(events)-1].Kind)
	assert.Contains(t, events[len(events)-1].Message, "cancelled")
	assert.NotContains(t, events[len(events)-1].Message, "lease expired")
}

func TestBuilderWorkerStopsCleanlyOnShutdown(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	insertQueuedBuilderRunSeed(t, now, "project-worker-stop", "session-worker-stop", "message-worker-stop", "run-worker-stop", "Stop the worker cleanly")

	worker := NewBuilderWorker()
	worker.pollInterval = 10 * time.Millisecond
	worker.leaseDuration = time.Minute

	ctx, cancel := context.WithCancel(context.Background())
	worker.SetParentContext(ctx)

	started := make(chan struct{})
	stopped := make(chan struct{})
	worker.handleClaimedRunFn = func(ctx context.Context, run *entities.BuilderRun) error {
		select {
		case <-started:
		default:
			close(started)
		}

		<-ctx.Done()

		select {
		case <-stopped:
		default:
			close(stopped)
		}
		return ctx.Err()
	}

	worker.Start()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("expected builder worker to start processing a queued run")
	}

	worker.Stop()
	cancel()

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("expected builder worker run handler to stop when shutdown begins")
	}

	waitDone := make(chan struct{})
	go func() {
		defer close(waitDone)
		worker.Wait()
	}()

	select {
	case <-waitDone:
	case <-time.After(time.Second):
		t.Fatal("expected builder worker goroutines to stop cleanly")
	}
}

func TestFinalizeClaimedBuilderRunSuccessHonorsLateCancellation(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	claimToken := "worker-success-cancel-claim"
	claimedAt := now.Add(-time.Minute)
	heartbeatAt := now.Add(-30 * time.Second)
	timeoutAt := now.Add(time.Minute)
	cancelRequestedAt := now.Add(-time.Second)

	insertBuilderSessionSeed(t, now.Add(-2*time.Minute), "project-worker-success-cancel", "session-worker-success-cancel", "env-worker-success-cancel", entities.BuilderSessionStatusRunning, "Worker success cancel session")
	insertBuilderMessageSeed(t, now.Add(-2*time.Minute), "session-worker-success-cancel", "message-worker-success-cancel", "Finish only if not cancelled", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-worker-success-cancel",
		CreatedAt:          now.Add(-time.Minute),
		UpdatedAt:          now.Add(-time.Minute),
		SessionID:          "session-worker-success-cancel",
		TriggerMessageID:   "message-worker-success-cancel",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseMaterializingFiles),
		ClaimToken:         &claimToken,
		ClaimedAt:          &claimedAt,
		HeartbeatAt:        &heartbeatAt,
		TimeoutAt:          &timeoutAt,
		CancelRequestedAt:  &cancelRequestedAt,
		RequestedBy:        "worker-user",
		InstructionSummary: "Finish only if not cancelled",
		StartedAt:          &claimedAt,
	}).Error)

	err := finalizeClaimedBuilderRunSuccess(
		context.Background(),
		&entities.BuilderRun{ID: "run-worker-success-cancel", SessionID: "session-worker-success-cancel"},
		claimToken,
		"Assistant output that should not persist after cancellation.",
		"run completed: 1 files generated",
	)
	require.NoError(t, err)

	var persistedRun entities.BuilderRun
	require.NoError(t, db.DB.First(&persistedRun, "id = ?", "run-worker-success-cancel").Error)
	assert.Equal(t, entities.BuilderRunStatusCancelled, persistedRun.Status)
	require.NotNil(t, persistedRun.ErrorCode)
	require.NotNil(t, persistedRun.ErrorClass)
	assert.Equal(t, builderWorkerCancelRequestedCode, *persistedRun.ErrorCode)
	assert.Equal(t, builderWorkerCancelledErrorClass, *persistedRun.ErrorClass)
	assert.Equal(t, "The builder run was cancelled.", persistedRun.ErrorMessage)

	var assistantMessages []entities.BuilderMessage
	require.NoError(t, db.DB.Where("session_id = ? AND role = ?", "session-worker-success-cancel", entities.BuilderMessageRoleAssistant).Find(&assistantMessages).Error)
	assert.Empty(t, assistantMessages)

	var systemMessages []entities.BuilderMessage
	require.NoError(t, db.DB.Where("session_id = ? AND role = ?", "session-worker-success-cancel", entities.BuilderMessageRoleSystem).Order("created_at ASC, id ASC").Find(&systemMessages).Error)
	require.NotEmpty(t, systemMessages)
	assert.Equal(t, "run cancelled", systemMessages[len(systemMessages)-1].Content)
}

func TestAppendOwnedBuilderRunLogEventRejectsLostOwnership(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	claimToken := "worker-lost-ownership-claim"
	claimedAt := now.Add(-time.Minute)
	heartbeatAt := now.Add(-30 * time.Second)
	timeoutAt := now.Add(time.Minute)

	insertBuilderSessionSeed(t, now.Add(-2*time.Minute), "project-worker-owned-log", "session-worker-owned-log", "env-worker-owned-log", entities.BuilderSessionStatusRunning, "Worker owned log session")
	insertBuilderMessageSeed(t, now.Add(-2*time.Minute), "session-worker-owned-log", "message-worker-owned-log", "Only active owners may log", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-worker-owned-log",
		CreatedAt:          now.Add(-time.Minute),
		UpdatedAt:          now.Add(-time.Minute),
		SessionID:          "session-worker-owned-log",
		TriggerMessageID:   "message-worker-owned-log",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseGenerating),
		ClaimToken:         &claimToken,
		ClaimedAt:          &claimedAt,
		HeartbeatAt:        &heartbeatAt,
		TimeoutAt:          &timeoutAt,
		RequestedBy:        "worker-user",
		InstructionSummary: "Only active owners may log",
		StartedAt:          &claimedAt,
	}).Error)

	require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("id = ?", "run-worker-owned-log").Updates(map[string]any{
		"status":       entities.BuilderRunStatusQueued,
		"phase":        entities.BuilderRunPhaseQueued,
		"claim_token":  nil,
		"claimed_at":   nil,
		"heartbeat_at": nil,
		"timeout_at":   nil,
	}).Error)

	err := appendOwnedBuilderRunLogEvent(context.Background(), "run-worker-owned-log", claimToken, "[system] stale worker event\n")
	require.Error(t, err)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

	var eventCount int64
	require.NoError(t, db.DB.Model(&entities.BuilderRunEvent{}).Where("run_id = ?", "run-worker-owned-log").Count(&eventCount).Error)
	assert.Equal(t, int64(0), eventCount)
}

func TestBuilderWorkerExecutesFrontendBuildPlan(t *testing.T) {
	t.Run("successful execution progresses through installing and building and persists build output artifacts", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		run, workspace, claimToken := seedClaimedBuilderWorkerExecutionFixture(t, "success")
		resetBuilderWorkerExecutionHooks(t)

		builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
			assert.Equal(t, run.SessionID, sessionID)
			return workspace, nil
		}
		builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
			assert.Equal(t, run.SessionID, sessionID)
			return []BuilderAgentMessage{{Role: "user", Content: "Build the generated app."}}, nil
		}
		run.ProviderKey = stringPtr("anthropic-project")
		run.ModelProfileKey = stringPtr("claude-sonnet-4")

		builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, providerKey, modelProfileKey string) (*BuilderAgentResult, error) {
			require.Len(t, messages, 1)
			assert.Equal(t, "anthropic-project", providerKey)
			assert.Equal(t, "claude-sonnet-4", modelProfileKey)
			return &BuilderAgentResult{
				AssistantMessage: "Build completed successfully.",
				Files: []BuilderAgentFileWrite{
					{Path: "package.json", Content: "{\"name\":\"demo\"}"},
					{Path: "package-lock.json", Content: "{}"},
				},
			}, nil
		}
		builderWorkerWriteAgentFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun, files []BuilderAgentFileWrite) error {
			assert.Equal(t, workspace.ID, actualWorkspace.ID)
			assert.Equal(t, run.ID, actualRun.ID)
			require.Len(t, files, 2)
			return nil
		}

		rootListingCalls := 0
		builderWorkerListWorkspaceFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
			assert.Equal(t, workspace.ID, actualWorkspace.ID)
			switch requestedPath {
			case workspace.WorkspaceRoot:
				rootListingCalls++
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot,
					Files: []models.FileInfo{
						{Name: "package.json", Type: "file", Size: 40},
						{Name: "package-lock.json", Type: "file", Size: 20},
						{Name: "dist", Type: "dir"},
					},
				}, nil
			case workspace.WorkspaceRoot + "/dist":
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot + "/dist",
					Files: []models.FileInfo{
						{Name: "index.html", Type: "file", Size: 512},
						{Name: "assets", Type: "dir"},
					},
				}, nil
			case workspace.WorkspaceRoot + "/dist/assets":
				return &models.ListFilesResponse{
					Path:  workspace.WorkspaceRoot + "/dist/assets",
					Files: []models.FileInfo{{Name: "app.js", Type: "file", Size: 2048}},
				}, nil
			default:
				t.Fatalf("unexpected workspace listing path %q", requestedPath)
				return nil, nil
			}
		}

		observedPhases := make([]entities.BuilderRunPhase, 0, 2)
		builderWorkerRunFrontendCommand = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			assert.Equal(t, workspace.ID, actualWorkspace.ID)
			var persistedRun entities.BuilderRun
			require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
			require.NotNil(t, persistedRun.Phase)
			observedPhases = append(observedPhases, *persistedRun.Phase)

			switch step {
			case BuilderFrontendExecutionStepInstall:
				assert.Equal(t, []string{"npm", "ci"}, command)
				require.NoError(t, appendLog("npm ci complete\n"))
				return BuilderFrontendCommandResult{ExitCode: 0}, nil
			case BuilderFrontendExecutionStepBuild:
				assert.Equal(t, []string{"npm", "run", "build"}, command)
				require.NoError(t, appendLog("npm run build complete\n"))
				return BuilderFrontendCommandResult{ExitCode: 0}, nil
			default:
				t.Fatalf("unexpected frontend execution step %q", step)
				return BuilderFrontendCommandResult{}, nil
			}
		}
		builderWorkerPublishOutputSnapshot = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun) (*entities.BuilderOutputSnapshot, error) {
			assert.Equal(t, workspace.ID, actualWorkspace.ID)
			assert.Equal(t, run.ID, actualRun.ID)
			return &entities.BuilderOutputSnapshot{ID: "snapshot-worker-success", RunID: run.ID, SessionID: run.SessionID}, nil
		}

		worker := NewBuilderWorker()
		err := worker.runClaimedExecution(context.Background(), run, claimToken)
		require.NoError(t, err)

		assert.Equal(t, []entities.BuilderRunPhase{builderWorkerPhaseInstallingDependencies, entities.BuilderRunPhaseBuilding}, observedPhases)
		assert.GreaterOrEqual(t, rootListingCalls, 2)

		var persistedRun entities.BuilderRun
		require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
		assert.Equal(t, entities.BuilderRunStatusSucceeded, persistedRun.Status)
		require.NotNil(t, persistedRun.Phase)
		assert.Equal(t, entities.BuilderRunPhaseFinalizing, *persistedRun.Phase)
		assert.Nil(t, persistedRun.ClaimToken)
		assert.NotNil(t, persistedRun.CompletedAt)

		var session entities.BuilderSession
		require.NoError(t, db.DB.First(&session, "id = ?", run.SessionID).Error)
		assert.Equal(t, entities.BuilderSessionStatusReady, session.Status)

		var artifacts []entities.BuilderArtifact
		require.NoError(t, db.DB.Where("session_id = ? AND workspace_id = ?", workspace.SessionID, workspace.ID).Order("path ASC").Find(&artifacts).Error)
		require.Len(t, artifacts, 4)
		assert.Equal(t, entities.BuilderArtifactKindBuildOutput, artifacts[0].Kind)
		assert.Equal(t, "dist/assets/app.js", artifacts[0].Path)
		assert.JSONEq(t, `{"size_bytes":2048,"output_root":"dist"}`, artifacts[0].MetadataJSON)
		assert.Equal(t, entities.BuilderArtifactKindBuildOutput, artifacts[1].Kind)
		assert.Equal(t, "dist/index.html", artifacts[1].Path)
		assert.JSONEq(t, `{"size_bytes":512,"output_root":"dist"}`, artifacts[1].MetadataJSON)
		assert.Equal(t, entities.BuilderArtifactKindWorkspaceFile, artifacts[2].Kind)
		assert.Equal(t, "package-lock.json", artifacts[2].Path)
		assert.JSONEq(t, `{"size_bytes":20}`, artifacts[2].MetadataJSON)
		assert.Equal(t, entities.BuilderArtifactKindWorkspaceFile, artifacts[3].Kind)
		assert.Equal(t, "package.json", artifacts[3].Path)
		assert.JSONEq(t, `{"size_bytes":40}`, artifacts[3].MetadataJSON)
	})

	t.Run("install failure finalizes as failed while leaving the workspace usable", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		run, workspace, claimToken := seedClaimedBuilderWorkerExecutionFixture(t, "install-failure")
		resetBuilderWorkerExecutionHooks(t)

		builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
			return workspace, nil
		}
		builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
			return []BuilderAgentMessage{{Role: "user", Content: "Build the generated app."}}, nil
		}
		builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, _ string, _ string) (*BuilderAgentResult, error) {
			return &BuilderAgentResult{
				AssistantMessage: "Build failed during install.",
				Files: []BuilderAgentFileWrite{
					{Path: "package.json", Content: "{\"name\":\"demo\"}"},
					{Path: "package-lock.json", Content: "{}"},
				},
			}, nil
		}
		builderWorkerWriteAgentFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun, files []BuilderAgentFileWrite) error {
			return nil
		}
		builderWorkerListWorkspaceFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
			assert.Equal(t, workspace.WorkspaceRoot, requestedPath)
			return &models.ListFilesResponse{
				Path: workspace.WorkspaceRoot,
				Files: []models.FileInfo{
					{Name: "package.json", Type: "file", Size: 40},
					{Name: "package-lock.json", Type: "file", Size: 20},
				},
			}, nil
		}

		observedPhases := make([]entities.BuilderRunPhase, 0, 1)
		builderWorkerRunFrontendCommand = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			var persistedRun entities.BuilderRun
			require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
			require.NotNil(t, persistedRun.Phase)
			observedPhases = append(observedPhases, *persistedRun.Phase)
			require.Equal(t, BuilderFrontendExecutionStepInstall, step)
			require.NoError(t, appendLog("npm ci failed\n"))
			return BuilderFrontendCommandResult{ExitCode: 17}, nil
		}

		worker := NewBuilderWorker()
		err := worker.runClaimedExecution(context.Background(), run, claimToken)
		require.NoError(t, err)

		assert.Equal(t, []entities.BuilderRunPhase{builderWorkerPhaseInstallingDependencies}, observedPhases)

		var persistedRun entities.BuilderRun
		require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
		assert.Equal(t, entities.BuilderRunStatusFailed, persistedRun.Status)
		require.NotNil(t, persistedRun.Phase)
		assert.Equal(t, entities.BuilderRunPhaseFinalizing, *persistedRun.Phase)
		assert.Contains(t, persistedRun.ErrorMessage, "install command exited with status 17")

		var session entities.BuilderSession
		require.NoError(t, db.DB.First(&session, "id = ?", run.SessionID).Error)
		assert.Equal(t, entities.BuilderSessionStatusReady, session.Status)
	})

	t.Run("build failure finalizes as failed while leaving the workspace usable", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		run, workspace, claimToken := seedClaimedBuilderWorkerExecutionFixture(t, "build-failure")
		resetBuilderWorkerExecutionHooks(t)

		builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
			return workspace, nil
		}
		builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
			return []BuilderAgentMessage{{Role: "user", Content: "Build the generated app."}}, nil
		}
		builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, _ string, _ string) (*BuilderAgentResult, error) {
			return &BuilderAgentResult{
				AssistantMessage: "Build failed during build.",
				Files: []BuilderAgentFileWrite{
					{Path: "package.json", Content: "{\"name\":\"demo\"}"},
					{Path: "package-lock.json", Content: "{}"},
				},
			}, nil
		}
		builderWorkerWriteAgentFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun, files []BuilderAgentFileWrite) error {
			return nil
		}
		builderWorkerListWorkspaceFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
			assert.Equal(t, workspace.WorkspaceRoot, requestedPath)
			return &models.ListFilesResponse{
				Path: workspace.WorkspaceRoot,
				Files: []models.FileInfo{
					{Name: "package.json", Type: "file", Size: 40},
					{Name: "package-lock.json", Type: "file", Size: 20},
				},
			}, nil
		}

		observedPhases := make([]entities.BuilderRunPhase, 0, 2)
		builderWorkerRunFrontendCommand = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			var persistedRun entities.BuilderRun
			require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
			require.NotNil(t, persistedRun.Phase)
			observedPhases = append(observedPhases, *persistedRun.Phase)

			switch step {
			case BuilderFrontendExecutionStepInstall:
				require.NoError(t, appendLog("npm ci complete\n"))
				return BuilderFrontendCommandResult{ExitCode: 0}, nil
			case BuilderFrontendExecutionStepBuild:
				require.NoError(t, appendLog("npm run build failed\n"))
				return BuilderFrontendCommandResult{ExitCode: 2}, nil
			default:
				t.Fatalf("unexpected frontend execution step %q", step)
				return BuilderFrontendCommandResult{}, nil
			}
		}

		worker := NewBuilderWorker()
		err := worker.runClaimedExecution(context.Background(), run, claimToken)
		require.NoError(t, err)

		assert.Equal(t, []entities.BuilderRunPhase{builderWorkerPhaseInstallingDependencies, entities.BuilderRunPhaseBuilding}, observedPhases)

		var persistedRun entities.BuilderRun
		require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
		assert.Equal(t, entities.BuilderRunStatusFailed, persistedRun.Status)
		require.NotNil(t, persistedRun.Phase)
		assert.Equal(t, entities.BuilderRunPhaseFinalizing, *persistedRun.Phase)
		assert.Contains(t, persistedRun.ErrorMessage, "build command exited with status 2")

		var session entities.BuilderSession
		require.NoError(t, db.DB.First(&session, "id = ?", run.SessionID).Error)
		assert.Equal(t, entities.BuilderSessionStatusReady, session.Status)
	})

	t.Run("successful execution publishes snapshot before succeeded finalization", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		setBuilderOutputSnapshotServiceConfigForTest(t)
		run, workspace, claimToken := seedClaimedBuilderWorkerExecutionFixture(t, "snapshot-success")
		resetBuilderWorkerExecutionHooks(t)

		sourceRoot := t.TempDir()
		writeBuilderOutputSnapshotSourceFiles(t, sourceRoot, map[string]string{
			"dist/index.html":    "<html><body>preview</body></html>",
			"dist/assets/app.js": "console.log('preview');\n",
		})
		setBuilderOutputSnapshotSourceForTest(t, sourceRoot)

		builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
			return workspace, nil
		}
		builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
			return []BuilderAgentMessage{{Role: "user", Content: "Build the generated app."}}, nil
		}
		builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, _ string, _ string) (*BuilderAgentResult, error) {
			return &BuilderAgentResult{
				AssistantMessage: "Build completed successfully.",
				Files: []BuilderAgentFileWrite{
					{Path: "package.json", Content: "{\"name\":\"demo\"}"},
					{Path: "package-lock.json", Content: "{}"},
				},
			}, nil
		}
		builderWorkerWriteAgentFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun, files []BuilderAgentFileWrite) error {
			return nil
		}
		builderWorkerListWorkspaceFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
			switch requestedPath {
			case workspace.WorkspaceRoot:
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot,
					Files: []models.FileInfo{
						{Name: "package.json", Type: "file", Size: 40},
						{Name: "package-lock.json", Type: "file", Size: 20},
						{Name: "dist", Type: "dir"},
					},
				}, nil
			case workspace.WorkspaceRoot + "/dist":
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot + "/dist",
					Files: []models.FileInfo{
						{Name: "index.html", Type: "file", Size: 512},
						{Name: "assets", Type: "dir"},
					},
				}, nil
			case workspace.WorkspaceRoot + "/dist/assets":
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot + "/dist/assets",
					Files: []models.FileInfo{
						{Name: "app.js", Type: "file", Size: 2048},
					},
				}, nil
			default:
				t.Fatalf("unexpected workspace listing path %q", requestedPath)
				return nil, nil
			}
		}
		builderWorkerRunFrontendCommand = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			return BuilderFrontendCommandResult{ExitCode: 0}, nil
		}
		builderWorkerPublishOutputSnapshot = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun) (*entities.BuilderOutputSnapshot, error) {
			var persistedRun entities.BuilderRun
			require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
			assert.Equal(t, entities.BuilderRunStatusExecuting, persistedRun.Status)
			require.NotNil(t, persistedRun.ClaimToken)

			var buildArtifacts []entities.BuilderArtifact
			require.NoError(t, db.DB.Where("run_id = ? AND kind = ?", run.ID, entities.BuilderArtifactKindBuildOutput).Order("path ASC").Find(&buildArtifacts).Error)
			require.Len(t, buildArtifacts, 2)
			assert.Equal(t, []string{"dist/assets/app.js", "dist/index.html"}, []string{buildArtifacts[0].Path, buildArtifacts[1].Path})

			assert.Equal(t, entities.BuilderRunStatusSucceeded, actualRun.Status)
			return PublishBuilderOutputSnapshot(ctx, actualWorkspace, actualRun)
		}

		worker := NewBuilderWorker()
		err := worker.runClaimedExecution(context.Background(), run, claimToken)
		require.NoError(t, err)

		var previewEvents []entities.BuilderRunEvent
		require.NoError(t, db.DB.Where("run_id = ? AND kind = ?", run.ID, entities.BuilderRunEventKindPreview).Find(&previewEvents).Error)
		require.Len(t, previewEvents, 1)
		assert.Contains(t, previewEvents[0].Message, "preview snapshot published")

		var snapshots []entities.BuilderOutputSnapshot
		require.NoError(t, db.DB.Where("run_id = ?", run.ID).Find(&snapshots).Error)
		require.Len(t, snapshots, 1)
		assert.Equal(t, entities.BuilderOutputSnapshotStatusPreviewable, snapshots[0].Status)

		var persistedRun entities.BuilderRun
		require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
		assert.Equal(t, entities.BuilderRunStatusSucceeded, persistedRun.Status)
	})

	t.Run("successful go api execution collects build directory artifacts", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		run, workspace, claimToken := seedClaimedBuilderWorkerExecutionFixture(t, "go-api-success")
		resetBuilderWorkerExecutionHooks(t)
		run.PlannedProjectKind = stringPtr("go_api_service")

		builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
			return workspace, nil
		}
		builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
			return []BuilderAgentMessage{{Role: "user", Content: "Build a Go API service."}}, nil
		}
		builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, _ string, _ string) (*BuilderAgentResult, error) {
			return &BuilderAgentResult{
				AssistantMessage: "Go API build completed successfully.",
				Files: []BuilderAgentFileWrite{
					{Path: "go.mod", Content: "module demo\n\ngo 1.25.0\n"},
					{Path: "main.go", Content: "package main\nfunc main() {}\n"},
				},
			}, nil
		}
		builderWorkerWriteAgentFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun, files []BuilderAgentFileWrite) error {
			return nil
		}

		rootListingCalls := 0
		builderWorkerListWorkspaceFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
			switch requestedPath {
			case workspace.WorkspaceRoot:
				rootListingCalls++
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot,
					Files: []models.FileInfo{
						{Name: "go.mod", Type: "file", Size: 24},
						{Name: "main.go", Type: "file", Size: 28},
						{Name: "build", Type: "dir"},
					},
				}, nil
			case workspace.WorkspaceRoot + "/build":
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot + "/build",
					Files: []models.FileInfo{
						{Name: "app", Type: "file", Size: 4096},
					},
				}, nil
			default:
				t.Fatalf("unexpected workspace listing path %q", requestedPath)
				return nil, nil
			}
		}

		observedCommands := make([]string, 0, 3)
		builderWorkerRunFrontendCommand = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			observedCommands = append(observedCommands, strings.Join(command, " "))
			switch step {
			case BuilderFrontendExecutionStepInstall:
				require.NoError(t, appendLog("go mod download complete\n"))
				return BuilderFrontendCommandResult{ExitCode: 0}, nil
			case BuilderFrontendExecutionStepBuild:
				require.NoError(t, appendLog("go build complete\n"))
				return BuilderFrontendCommandResult{ExitCode: 0}, nil
			case BuilderFrontendExecutionStepValidate:
				require.NoError(t, appendLog("go runtime validation complete\n"))
				return BuilderFrontendCommandResult{ExitCode: 0}, nil
			default:
				t.Fatalf("unexpected execution step %q", step)
				return BuilderFrontendCommandResult{}, nil
			}
		}
		builderWorkerPublishOutputSnapshot = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun) (*entities.BuilderOutputSnapshot, error) {
			return &entities.BuilderOutputSnapshot{ID: "snapshot-go-api", RunID: run.ID, SessionID: run.SessionID}, nil
		}

		worker := NewBuilderWorker()
		err := worker.runClaimedExecution(context.Background(), run, claimToken)
		require.NoError(t, err)

		assert.Equal(t, []string{
			"go mod download",
			"sh -lc mkdir -p build && go build -o build/app .",
			`sh -lc PORT=18080 ./build/app >/tmp/builder-runtime.log 2>&1 & pid=$!; sleep 2; if kill -0 "$pid" 2>/dev/null; then kill "$pid"; wait "$pid" || true; exit 0; fi; wait "$pid"; status=$?; if [ -f /tmp/builder-runtime.log ]; then cat /tmp/builder-runtime.log; fi; exit ${status:-1}`,
		}, observedCommands)
		assert.GreaterOrEqual(t, rootListingCalls, 2)

		var artifacts []entities.BuilderArtifact
		require.NoError(t, db.DB.Where("session_id = ? AND workspace_id = ?", workspace.SessionID, workspace.ID).Order("path ASC").Find(&artifacts).Error)
		require.Len(t, artifacts, 3)
		assert.Equal(t, entities.BuilderArtifactKindBuildOutput, artifacts[0].Kind)
		assert.Equal(t, "build/app", artifacts[0].Path)
		assert.JSONEq(t, `{"size_bytes":4096,"output_root":"build"}`, artifacts[0].MetadataJSON)
	})

	t.Run("go api runtime validation failure finalizes run as failed", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		run, workspace, claimToken := seedClaimedBuilderWorkerExecutionFixture(t, "go-api-runtime-failure")
		resetBuilderWorkerExecutionHooks(t)

		builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
			return workspace, nil
		}
		builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
			return []BuilderAgentMessage{{Role: "user", Content: "Build a Go API service."}}, nil
		}
		run.PlannedProjectKind = stringPtr("go_api_service")
		builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, _ string, _ string) (*BuilderAgentResult, error) {
			return &BuilderAgentResult{
				AssistantMessage: "Go API build completed successfully.",
				Files: []BuilderAgentFileWrite{
					{Path: "go.mod", Content: "module demo\n\ngo 1.25.0\n"},
					{Path: "main.go", Content: "package main\nfunc main() {}\n"},
				},
			}, nil
		}
		builderWorkerWriteAgentFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun, files []BuilderAgentFileWrite) error {
			return nil
		}
		builderWorkerListWorkspaceFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
			switch requestedPath {
			case workspace.WorkspaceRoot:
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot,
					Files: []models.FileInfo{
						{Name: "go.mod", Type: "file", Size: 24},
						{Name: "main.go", Type: "file", Size: 28},
						{Name: "build", Type: "dir"},
					},
				}, nil
			case workspace.WorkspaceRoot + "/build":
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot + "/build",
					Files: []models.FileInfo{
						{Name: "app", Type: "file", Size: 4096},
					},
				}, nil
			default:
				t.Fatalf("unexpected workspace listing path %q", requestedPath)
				return nil, nil
			}
		}
		builderWorkerRunFrontendCommand = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			switch step {
			case BuilderFrontendExecutionStepInstall, BuilderFrontendExecutionStepBuild:
				return BuilderFrontendCommandResult{ExitCode: 0}, nil
			case BuilderFrontendExecutionStepValidate:
				require.NoError(t, appendLog("runtime process exited immediately\n"))
				return BuilderFrontendCommandResult{ExitCode: 7}, nil
			default:
				t.Fatalf("unexpected execution step %q", step)
				return BuilderFrontendCommandResult{}, nil
			}
		}

		worker := NewBuilderWorker()
		err := worker.runClaimedExecution(context.Background(), run, claimToken)
		require.NoError(t, err)

		var persistedRun entities.BuilderRun
		require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
		assert.Equal(t, entities.BuilderRunStatusFailed, persistedRun.Status)
		assert.Contains(t, persistedRun.ErrorMessage, "validate command exited with status 7")
	})

	t.Run("successful python api execution collects build directory artifacts", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		run, workspace, claimToken := seedClaimedBuilderWorkerExecutionFixture(t, "python-api-success")
		resetBuilderWorkerExecutionHooks(t)

		builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
			return workspace, nil
		}
		builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
			return []BuilderAgentMessage{{Role: "user", Content: "Build a Python API service."}}, nil
		}
		run.PlannedProjectKind = stringPtr("python_api_service")
		builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, _ string, _ string) (*BuilderAgentResult, error) {
			return &BuilderAgentResult{
				AssistantMessage: "Python API build completed successfully.",
				Files: []BuilderAgentFileWrite{
					{Path: "requirements.txt", Content: "fastapi==0.116.0\n"},
					{Path: "app.py", Content: "from fastapi import FastAPI\napp = FastAPI()\n"},
				},
			}, nil
		}
		builderWorkerWriteAgentFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun, files []BuilderAgentFileWrite) error {
			return nil
		}

		rootListingCalls := 0
		builderWorkerListWorkspaceFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
			switch requestedPath {
			case workspace.WorkspaceRoot:
				rootListingCalls++
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot,
					Files: []models.FileInfo{
						{Name: "requirements.txt", Type: "file", Size: 16},
						{Name: "app.py", Type: "file", Size: 44},
						{Name: "build", Type: "dir"},
					},
				}, nil
			case workspace.WorkspaceRoot + "/build":
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot + "/build",
					Files: []models.FileInfo{
						{Name: "app.py", Type: "file", Size: 44},
					},
				}, nil
			default:
				t.Fatalf("unexpected workspace listing path %q", requestedPath)
				return nil, nil
			}
		}

		observedCommands := make([]string, 0, 3)
		builderWorkerRunFrontendCommand = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			observedCommands = append(observedCommands, strings.Join(command, " "))
			switch step {
			case BuilderFrontendExecutionStepInstall:
				require.NoError(t, appendLog("pip install complete\n"))
				return BuilderFrontendCommandResult{ExitCode: 0}, nil
			case BuilderFrontendExecutionStepBuild:
				require.NoError(t, appendLog("python build staging complete\n"))
				return BuilderFrontendCommandResult{ExitCode: 0}, nil
			case BuilderFrontendExecutionStepValidate:
				require.NoError(t, appendLog("python import validation complete\n"))
				return BuilderFrontendCommandResult{ExitCode: 0}, nil
			default:
				t.Fatalf("unexpected execution step %q", step)
				return BuilderFrontendCommandResult{}, nil
			}
		}
		builderWorkerPublishOutputSnapshot = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun) (*entities.BuilderOutputSnapshot, error) {
			return &entities.BuilderOutputSnapshot{ID: "snapshot-python-api", RunID: run.ID, SessionID: run.SessionID}, nil
		}

		worker := NewBuilderWorker()
		err := worker.runClaimedExecution(context.Background(), run, claimToken)
		require.NoError(t, err)

		assert.Equal(t, []string{
			"pip install -r requirements.txt",
			"sh -lc mkdir -p build && if [ -f app.py ]; then cp app.py build/app.py; else cp main.py build/main.py; fi",
			`sh -lc if [ -f build/app.py ]; then python -c "import importlib.util; path='build/app.py'; spec=importlib.util.spec_from_file_location('builder_app', path); mod=importlib.util.module_from_spec(spec); spec.loader.exec_module(mod)"; else python -c "import importlib.util; path='build/main.py'; spec=importlib.util.spec_from_file_location('builder_app', path); mod=importlib.util.module_from_spec(spec); spec.loader.exec_module(mod)"; fi`,
		}, observedCommands)
		assert.GreaterOrEqual(t, rootListingCalls, 2)

		var artifacts []entities.BuilderArtifact
		require.NoError(t, db.DB.Where("session_id = ? AND workspace_id = ?", workspace.SessionID, workspace.ID).Order("path ASC").Find(&artifacts).Error)
		require.Len(t, artifacts, 3)
		artifactsByPath := make(map[string]entities.BuilderArtifact, len(artifacts))
		for _, artifact := range artifacts {
			artifactsByPath[artifact.Path] = artifact
		}
		buildArtifact, ok := artifactsByPath["build/app.py"]
		require.True(t, ok)
		assert.Equal(t, entities.BuilderArtifactKindBuildOutput, buildArtifact.Kind)
		assert.JSONEq(t, `{"size_bytes":44,"output_root":"build"}`, buildArtifact.MetadataJSON)
	})

	t.Run("successful node ssr execution publishes delivery-only next snapshot", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		setBuilderOutputSnapshotServiceConfigForTest(t)
		run, workspace, claimToken := seedClaimedBuilderWorkerExecutionFixture(t, "node-ssr-success")
		resetBuilderWorkerExecutionHooks(t)

		sourceRoot := t.TempDir()
		writeBuilderOutputSnapshotSourceFiles(t, sourceRoot, map[string]string{
			".next/routes-manifest.json": "{\"version\":1}\n",
			".next/static/app.js":        "console.log('ssr');\n",
		})
		setBuilderOutputSnapshotSourceForTest(t, sourceRoot)

		builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
			return workspace, nil
		}
		builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
			return []BuilderAgentMessage{{Role: "user", Content: "Build a Next.js SSR app."}}, nil
		}
		run.PlannedProjectKind = stringPtr("node_ssr_app")
		builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, _ string, _ string) (*BuilderAgentResult, error) {
			return &BuilderAgentResult{
				AssistantMessage: "Node SSR build completed successfully.",
				Files: []BuilderAgentFileWrite{
					{Path: "package.json", Content: "{\"name\":\"demo\"}"},
					{Path: "package-lock.json", Content: "{}"},
				},
			}, nil
		}
		builderWorkerWriteAgentFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun, files []BuilderAgentFileWrite) error {
			return nil
		}
		builderWorkerListWorkspaceFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
			switch requestedPath {
			case workspace.WorkspaceRoot:
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot,
					Files: []models.FileInfo{
						{Name: "package.json", Type: "file", Size: 40},
						{Name: "package-lock.json", Type: "file", Size: 20},
						{Name: ".next", Type: "dir"},
					},
				}, nil
			case workspace.WorkspaceRoot + "/.next":
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot + "/.next",
					Files: []models.FileInfo{
						{Name: "routes-manifest.json", Type: "file", Size: 14},
						{Name: "static", Type: "dir"},
					},
				}, nil
			case workspace.WorkspaceRoot + "/.next/static":
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot + "/.next/static",
					Files: []models.FileInfo{
						{Name: "app.js", Type: "file", Size: 19},
					},
				}, nil
			default:
				t.Fatalf("unexpected workspace listing path %q", requestedPath)
				return nil, nil
			}
		}
		builderWorkerRunFrontendCommand = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			return BuilderFrontendCommandResult{ExitCode: 0}, nil
		}
		builderWorkerPublishOutputSnapshot = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun) (*entities.BuilderOutputSnapshot, error) {
			assert.Equal(t, entities.BuilderRunStatusSucceeded, actualRun.Status)
			return PublishBuilderOutputSnapshot(ctx, actualWorkspace, actualRun)
		}

		worker := NewBuilderWorker()
		err := worker.runClaimedExecution(context.Background(), run, claimToken)
		require.NoError(t, err)

		var snapshots []entities.BuilderOutputSnapshot
		require.NoError(t, db.DB.Where("run_id = ?", run.ID).Find(&snapshots).Error)
		require.Len(t, snapshots, 1)
		assert.Equal(t, entities.BuilderOutputSnapshotStatusDeliveryOnly, snapshots[0].Status)
		assert.Equal(t, ".next", snapshots[0].OutputRoot)
		assert.Empty(t, snapshots[0].DefaultEntryPath)
	})

	t.Run("successful node ssr standalone execution runs runtime validation before publishing snapshot", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		setBuilderOutputSnapshotServiceConfigForTest(t)
		run, workspace, claimToken := seedClaimedBuilderWorkerExecutionFixture(t, "node-ssr-standalone-success")
		resetBuilderWorkerExecutionHooks(t)

		sourceRoot := t.TempDir()
		writeBuilderOutputSnapshotSourceFiles(t, sourceRoot, map[string]string{
			".next/routes-manifest.json": "{\"version\":1}\n",
			".next/standalone/server.js": "console.log('server');\n",
			".next/static/app.js":        "console.log('ssr');\n",
		})
		setBuilderOutputSnapshotSourceForTest(t, sourceRoot)

		builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
			return workspace, nil
		}
		builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
			return []BuilderAgentMessage{{Role: "user", Content: "Build a Next.js SSR app with standalone output."}}, nil
		}
		run.PlannedProjectKind = stringPtr("node_ssr_app")
		builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, _ string, _ string) (*BuilderAgentResult, error) {
			return &BuilderAgentResult{
				AssistantMessage: "Node SSR build completed successfully.",
				Files: []BuilderAgentFileWrite{
					{Path: "package.json", Content: "{\"name\":\"demo\"}"},
					{Path: "package-lock.json", Content: "{}"},
				},
			}, nil
		}
		builderWorkerWriteAgentFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun, files []BuilderAgentFileWrite) error {
			return nil
		}
		builderWorkerListWorkspaceFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
			switch requestedPath {
			case workspace.WorkspaceRoot:
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot,
					Files: []models.FileInfo{
						{Name: "package.json", Type: "file", Size: 40},
						{Name: "package-lock.json", Type: "file", Size: 20},
						{Name: ".next", Type: "dir"},
					},
				}, nil
			case workspace.WorkspaceRoot + "/.next":
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot + "/.next",
					Files: []models.FileInfo{
						{Name: "routes-manifest.json", Type: "file", Size: 14},
						{Name: "standalone", Type: "dir"},
						{Name: "static", Type: "dir"},
					},
				}, nil
			case workspace.WorkspaceRoot + "/.next/standalone":
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot + "/.next/standalone",
					Files: []models.FileInfo{
						{Name: "server.js", Type: "file", Size: 24},
					},
				}, nil
			case workspace.WorkspaceRoot + "/.next/static":
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot + "/.next/static",
					Files: []models.FileInfo{
						{Name: "app.js", Type: "file", Size: 19},
					},
				}, nil
			default:
				t.Fatalf("unexpected workspace listing path %q", requestedPath)
				return nil, nil
			}
		}

		observedCommands := make([]string, 0, 3)
		builderWorkerRunFrontendCommand = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			observedCommands = append(observedCommands, strings.Join(command, " "))
			return BuilderFrontendCommandResult{ExitCode: 0}, nil
		}
		builderWorkerPublishOutputSnapshot = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun) (*entities.BuilderOutputSnapshot, error) {
			return PublishBuilderOutputSnapshot(ctx, actualWorkspace, actualRun)
		}

		worker := NewBuilderWorker()
		err := worker.runClaimedExecution(context.Background(), run, claimToken)
		require.NoError(t, err)

		assert.Equal(t, []string{
			"npm ci",
			"npm run build",
			`sh -lc PORT=18081 node .next/standalone/server.js >/tmp/builder-runtime.log 2>&1 & pid=$!; sleep 3; if kill -0 "$pid" 2>/dev/null; then kill "$pid"; wait "$pid" || true; exit 0; fi; wait "$pid"; status=$?; if [ -f /tmp/builder-runtime.log ]; then cat /tmp/builder-runtime.log; fi; exit ${status:-1}`,
		}, observedCommands)
	})

	t.Run("snapshot publication failure finalizes run as failed", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		run, workspace, claimToken := seedClaimedBuilderWorkerExecutionFixture(t, "snapshot-failure")
		resetBuilderWorkerExecutionHooks(t)

		builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
			return workspace, nil
		}
		builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
			return []BuilderAgentMessage{{Role: "user", Content: "Build the generated app."}}, nil
		}
		builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, _ string, _ string) (*BuilderAgentResult, error) {
			return &BuilderAgentResult{
				AssistantMessage: "Build completed successfully.",
				Files: []BuilderAgentFileWrite{
					{Path: "package.json", Content: "{\"name\":\"demo\"}"},
					{Path: "package-lock.json", Content: "{}"},
				},
			}, nil
		}
		builderWorkerWriteAgentFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun, files []BuilderAgentFileWrite) error {
			return nil
		}
		builderWorkerListWorkspaceFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
			switch requestedPath {
			case workspace.WorkspaceRoot:
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot,
					Files: []models.FileInfo{
						{Name: "package.json", Type: "file", Size: 40},
						{Name: "package-lock.json", Type: "file", Size: 20},
						{Name: "dist", Type: "dir"},
					},
				}, nil
			case workspace.WorkspaceRoot + "/dist":
				return &models.ListFilesResponse{
					Path:  workspace.WorkspaceRoot + "/dist",
					Files: []models.FileInfo{{Name: "index.html", Type: "file", Size: 512}},
				}, nil
			default:
				t.Fatalf("unexpected workspace listing path %q", requestedPath)
				return nil, nil
			}
		}
		builderWorkerRunFrontendCommand = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			return BuilderFrontendCommandResult{ExitCode: 0}, nil
		}

		builderWorkerPublishOutputSnapshot = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun) (*entities.BuilderOutputSnapshot, error) {
			assert.Equal(t, entities.BuilderRunStatusSucceeded, actualRun.Status)
			return nil, errors.New("snapshot publish failed")
		}

		worker := NewBuilderWorker()
		err := worker.runClaimedExecution(context.Background(), run, claimToken)
		require.NoError(t, err)

		var persistedRun entities.BuilderRun
		require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
		assert.Equal(t, entities.BuilderRunStatusFailed, persistedRun.Status)
		assert.Contains(t, persistedRun.ErrorMessage, "snapshot publish failed")

		var previewEventCount int64
		require.NoError(t, db.DB.Model(&entities.BuilderRunEvent{}).Where("run_id = ? AND kind = ?", run.ID, entities.BuilderRunEventKindPreview).Count(&previewEventCount).Error)
		assert.Equal(t, int64(0), previewEventCount)
	})

	t.Run("ownership loss during execution prevents later execution events from being persisted", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		run, workspace, claimToken := seedClaimedBuilderWorkerExecutionFixture(t, "ownership-loss")
		resetBuilderWorkerExecutionHooks(t)

		builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
			return workspace, nil
		}
		builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
			return []BuilderAgentMessage{{Role: "user", Content: "Build the generated app."}}, nil
		}
		builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, _ string, _ string) (*BuilderAgentResult, error) {
			return &BuilderAgentResult{
				AssistantMessage: "Build ownership loss test.",
				Files: []BuilderAgentFileWrite{
					{Path: "package.json", Content: "{\"name\":\"demo\"}"},
					{Path: "package-lock.json", Content: "{}"},
				},
			}, nil
		}
		builderWorkerWriteAgentFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun, files []BuilderAgentFileWrite) error {
			return nil
		}
		builderWorkerListWorkspaceFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
			assert.Equal(t, workspace.WorkspaceRoot, requestedPath)
			return &models.ListFilesResponse{
				Path: workspace.WorkspaceRoot,
				Files: []models.FileInfo{
					{Name: "package.json", Type: "file", Size: 40},
					{Name: "package-lock.json", Type: "file", Size: 20},
				},
			}, nil
		}

		builderWorkerRunFrontendCommand = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			require.Equal(t, BuilderFrontendExecutionStepInstall, step)
			require.NoError(t, appendLog("npm ci complete\n"))
			require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("id = ?", run.ID).Updates(map[string]any{
				"status":       entities.BuilderRunStatusQueued,
				"phase":        entities.BuilderRunPhaseQueued,
				"claim_token":  nil,
				"claimed_at":   nil,
				"heartbeat_at": nil,
				"timeout_at":   nil,
			}).Error)
			return BuilderFrontendCommandResult{ExitCode: 0}, nil
		}

		worker := NewBuilderWorker()
		err := worker.runClaimedExecution(context.Background(), run, claimToken)
		require.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)

		var persistedEvents []entities.BuilderRunEvent
		require.NoError(t, db.DB.Where("run_id = ?", run.ID).Order("sequence ASC").Find(&persistedEvents).Error)
		require.NotEmpty(t, persistedEvents)

		eventMessages := make([]string, 0, len(persistedEvents))
		for _, event := range persistedEvents {
			eventMessages = append(eventMessages, event.Message)
		}

		assert.Contains(t, eventMessages, "[system] installing frontend dependencies\n")
		assert.Contains(t, eventMessages, "npm ci complete\n")
		assert.NotContains(t, eventMessages, "[system] running frontend build\n")
	})

	t.Run("ownership loss after snapshot publication cleans up stale snapshot", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		setBuilderOutputSnapshotServiceConfigForTest(t)
		run, workspace, claimToken := seedClaimedBuilderWorkerExecutionFixture(t, "snapshot-ownership-loss")
		resetBuilderWorkerExecutionHooks(t)

		sourceRoot := t.TempDir()
		writeBuilderOutputSnapshotSourceFiles(t, sourceRoot, map[string]string{
			"dist/index.html":    "<html><body>preview</body></html>",
			"dist/assets/app.js": "console.log('preview');\n",
		})
		setBuilderOutputSnapshotSourceForTest(t, sourceRoot)

		builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
			return workspace, nil
		}
		builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
			return []BuilderAgentMessage{{Role: "user", Content: "Build the generated app."}}, nil
		}
		builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, _ string, _ string) (*BuilderAgentResult, error) {
			return &BuilderAgentResult{
				AssistantMessage: "Build completed successfully.",
				Files: []BuilderAgentFileWrite{
					{Path: "package.json", Content: "{\"name\":\"demo\"}"},
					{Path: "package-lock.json", Content: "{}"},
				},
			}, nil
		}
		builderWorkerWriteAgentFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun, files []BuilderAgentFileWrite) error {
			return nil
		}
		builderWorkerListWorkspaceFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
			switch requestedPath {
			case workspace.WorkspaceRoot:
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot,
					Files: []models.FileInfo{
						{Name: "package.json", Type: "file", Size: 40},
						{Name: "package-lock.json", Type: "file", Size: 20},
						{Name: "dist", Type: "dir"},
					},
				}, nil
			case workspace.WorkspaceRoot + "/dist":
				return &models.ListFilesResponse{
					Path: workspace.WorkspaceRoot + "/dist",
					Files: []models.FileInfo{
						{Name: "index.html", Type: "file", Size: 512},
						{Name: "assets", Type: "dir"},
					},
				}, nil
			case workspace.WorkspaceRoot + "/dist/assets":
				return &models.ListFilesResponse{
					Path:  workspace.WorkspaceRoot + "/dist/assets",
					Files: []models.FileInfo{{Name: "app.js", Type: "file", Size: 2048}},
				}, nil
			default:
				t.Fatalf("unexpected workspace listing path %q", requestedPath)
				return nil, nil
			}
		}
		builderWorkerRunFrontendCommand = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			return BuilderFrontendCommandResult{ExitCode: 0}, nil
		}

		published := make(chan struct{}, 1)
		builderWorkerPublishOutputSnapshot = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun) (*entities.BuilderOutputSnapshot, error) {
			snapshot, err := PublishBuilderOutputSnapshot(ctx, actualWorkspace, actualRun)
			if err != nil {
				return nil, err
			}
			require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("id = ?", run.ID).Updates(map[string]any{
				"status":       entities.BuilderRunStatusQueued,
				"phase":        entities.BuilderRunPhaseQueued,
				"claim_token":  nil,
				"claimed_at":   nil,
				"heartbeat_at": nil,
				"timeout_at":   nil,
			}).Error)
			published <- struct{}{}
			return snapshot, nil
		}

		worker := NewBuilderWorker()
		err := worker.runClaimedExecution(context.Background(), run, claimToken)
		require.Error(t, err)
		assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
		select {
		case <-published:
		default:
			t.Fatal("expected snapshot publication seam to run before ownership was lost")
		}

		var snapshotCount int64
		require.NoError(t, db.DB.Model(&entities.BuilderOutputSnapshot{}).Where("run_id = ?", run.ID).Count(&snapshotCount).Error)
		assert.Equal(t, int64(0), snapshotCount)

		var previewEventCount int64
		require.NoError(t, db.DB.Model(&entities.BuilderRunEvent{}).Where("run_id = ? AND kind = ?", run.ID, entities.BuilderRunEventKindPreview).Count(&previewEventCount).Error)
		assert.Equal(t, int64(0), previewEventCount)
	})
}

func TestBuilderWorkerRunClaimedExecutionSkipsWorkspaceProvisioningForReplyOnlyTurns(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)
	run, workspace, claimToken := seedClaimedBuilderWorkerExecutionFixture(t, "provision-hook")
	resetBuilderWorkerExecutionHooks(t)

	provisionCalled := false
	builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
		provisionCalled = true
		return workspace, nil
	}
	builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
		return []BuilderAgentMessage{{Role: "user", Content: "Build the generated app."}}, nil
	}
	builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, _ string, _ string) (*BuilderAgentResult, error) {
		return &BuilderAgentResult{
			Action:           BuilderAgentActionReplyOnly,
			AssistantMessage: "No files.",
			Files:            nil,
		}, nil
	}

	worker := NewBuilderWorker()
	err := worker.runClaimedExecution(context.Background(), run, claimToken)
	require.NoError(t, err)
	assert.False(t, provisionCalled)

	var persistedRun entities.BuilderRun
	require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
	assert.Equal(t, entities.BuilderRunStatusSucceeded, persistedRun.Status)
	assert.NotContains(t, persistedRun.ExecutionLog, "run completed: no files generated")

	var messages []entities.BuilderMessage
	require.NoError(t, db.DB.Where("session_id = ?", run.SessionID).Order("created_at ASC, id ASC").Find(&messages).Error)
	require.Len(t, messages, 3)
	assert.Equal(t, entities.BuilderMessageRoleAssistant, messages[1].Role)
	assert.Equal(t, "No files.", messages[1].Content)
	assert.Equal(t, entities.BuilderMessageRoleSystem, messages[2].Role)
	assert.Contains(t, messages[2].Content, "replied without workspace changes")
}

func TestBuilderWorkerCancelsDuringExecution(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)
	run, workspace, claimToken := seedClaimedBuilderWorkerExecutionFixture(t, "cancelled")
	resetBuilderWorkerExecutionHooks(t)

	builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
		return workspace, nil
	}
	builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
		return []BuilderAgentMessage{{Role: "user", Content: "Build the generated app."}}, nil
	}
	builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, _ string, _ string) (*BuilderAgentResult, error) {
		return &BuilderAgentResult{
			AssistantMessage: "Build cancelled.",
			Files: []BuilderAgentFileWrite{
				{Path: "package.json", Content: "{\"name\":\"demo\"}"},
				{Path: "package-lock.json", Content: "{}"},
			},
		}, nil
	}
	builderWorkerWriteAgentFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun, files []BuilderAgentFileWrite) error {
		return nil
	}
	builderWorkerListWorkspaceFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
		assert.Equal(t, workspace.WorkspaceRoot, requestedPath)
		return &models.ListFilesResponse{
			Path: workspace.WorkspaceRoot,
			Files: []models.FileInfo{
				{Name: "package.json", Type: "file", Size: 40},
				{Name: "package-lock.json", Type: "file", Size: 20},
			},
		}, nil
	}

	observedPhases := make([]entities.BuilderRunPhase, 0, 2)
	builderWorkerRunFrontendCommand = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
		var persistedRun entities.BuilderRun
		require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
		require.NotNil(t, persistedRun.Phase)
		observedPhases = append(observedPhases, *persistedRun.Phase)

		switch step {
		case BuilderFrontendExecutionStepInstall:
			require.NoError(t, appendLog("npm ci complete\n"))
			return BuilderFrontendCommandResult{ExitCode: 0}, nil
		case BuilderFrontendExecutionStepBuild:
			require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("id = ?", run.ID).Update("cancel_requested_at", time.Now().UTC()).Error)
			require.NoError(t, appendLog("npm run build interrupted\n"))
			return BuilderFrontendCommandResult{}, context.Canceled
		default:
			t.Fatalf("unexpected frontend execution step %q", step)
			return BuilderFrontendCommandResult{}, nil
		}
	}

	worker := NewBuilderWorker()
	err := worker.runClaimedExecution(context.Background(), run, claimToken)
	require.NoError(t, err)

	assert.Equal(t, []entities.BuilderRunPhase{builderWorkerPhaseInstallingDependencies, entities.BuilderRunPhaseBuilding}, observedPhases)

	var persistedRun entities.BuilderRun
	require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
	assert.Equal(t, entities.BuilderRunStatusCancelled, persistedRun.Status)
	require.NotNil(t, persistedRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseFinalizing, *persistedRun.Phase)
	require.NotNil(t, persistedRun.ErrorCode)
	require.NotNil(t, persistedRun.ErrorClass)
	assert.Equal(t, builderWorkerCancelRequestedCode, *persistedRun.ErrorCode)
	assert.Equal(t, builderWorkerCancelledErrorClass, *persistedRun.ErrorClass)

	var session entities.BuilderSession
	require.NoError(t, db.DB.First(&session, "id = ?", run.SessionID).Error)
	assert.Equal(t, entities.BuilderSessionStatusReady, session.Status)
}

func TestDefaultBuilderWorkerRunFrontendCommandHonorsCancellation(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	_, workspace, _ := seedBuilderWorkspaceServiceFixture(t)
	resetBuilderWorkerExecutionHooks(t)

	started := make(chan struct{})
	builderWorkerExecCommandWithContext = func(ctx context.Context, appCtx *models.AppContext, instanceName, containerName string, command []string) (string, string, error) {
		assert.Equal(t, workspace.PodName, instanceName)
		assert.Equal(t, workspace.ContainerName, containerName)
		require.Len(t, command, 3)
		assert.Equal(t, []string{"sh", "-lc"}, command[:2])
		close(started)
		<-ctx.Done()
		return "", "", ctx.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	resultCh := make(chan error, 1)
	go func() {
		_, err := defaultBuilderWorkerRunFrontendCommand(ctx, workspace, BuilderFrontendExecutionStepInstall, []string{"npm", "ci"}, func(string) error {
			return nil
		})
		resultCh <- err
	}()

	<-started
	cancel()

	require.ErrorIs(t, <-resultCh, context.Canceled)
}

func TestDefaultBuilderWorkerRunFrontendCommandPersistsPartialOutputOnCancellation(t *testing.T) {
	setupBuilderWorkspaceServiceTestDB(t)
	setBuilderWorkspaceServiceConfigForTest(t)
	_, workspace, _ := seedBuilderWorkspaceServiceFixture(t)
	resetBuilderWorkerExecutionHooks(t)

	builderWorkerExecCommandWithContext = func(ctx context.Context, appCtx *models.AppContext, instanceName, containerName string, command []string) (string, string, error) {
		assert.Equal(t, workspace.PodName, instanceName)
		assert.Equal(t, workspace.ContainerName, containerName)
		require.Len(t, command, 3)
		return "stdout before cancel\n", "stderr before cancel\n", context.Canceled
	}

	var appended []string
	_, err := defaultBuilderWorkerRunFrontendCommand(context.Background(), workspace, BuilderFrontendExecutionStepInstall, []string{"npm", "ci"}, func(message string) error {
		appended = append(appended, message)
		return nil
	})

	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, []string{"stdout before cancel\n", "stderr before cancel\n"}, appended)
}

func TestBuilderWorkerCancelsInFlightDefaultFrontendExecutionOnUserRequest(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)
	session, workspace, run, _ := seedClaimedBuilderWorkerDefaultExecutionFixture(t)
	resetBuilderWorkerExecutionHooks(t)

	builderWorkerProvisionWorkspace = func(ctx context.Context, sessionID string) (*entities.BuilderWorkspace, error) {
		assert.Equal(t, session.ID, sessionID)
		return workspace, nil
	}
	builderWorkerLoadConversationMessages = func(ctx context.Context, sessionID string) ([]BuilderAgentMessage, error) {
		return []BuilderAgentMessage{{Role: "user", Content: "Build the generated app."}}, nil
	}
	builderWorkerGenerateFiles = func(ctx context.Context, messages []BuilderAgentMessage, _ string, _ string) (*BuilderAgentResult, error) {
		return &BuilderAgentResult{
			AssistantMessage: "Cancelled during install.",
			Files: []BuilderAgentFileWrite{
				{Path: "package.json", Content: "{\"name\":\"demo\"}"},
				{Path: "package-lock.json", Content: "{}"},
			},
		}, nil
	}
	builderWorkerWriteAgentFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, actualRun *entities.BuilderRun, files []BuilderAgentFileWrite) error {
		return nil
	}
	builderWorkerListWorkspaceFiles = func(ctx context.Context, actualWorkspace *entities.BuilderWorkspace, requestedPath string) (*models.ListFilesResponse, error) {
		assert.Equal(t, workspace.WorkspaceRoot, requestedPath)
		return &models.ListFilesResponse{
			Path: workspace.WorkspaceRoot,
			Files: []models.FileInfo{
				{Name: "package.json", Type: "file", Size: 40},
				{Name: "package-lock.json", Type: "file", Size: 20},
			},
		}, nil
	}

	started := make(chan struct{})
	builderWorkerExecCommandWithContext = func(ctx context.Context, appCtx *models.AppContext, instanceName, containerName string, command []string) (string, string, error) {
		assert.Equal(t, workspace.PodName, instanceName)
		assert.Equal(t, workspace.ContainerName, containerName)
		require.Len(t, command, 3)
		close(started)
		<-ctx.Done()
		return "", "", ctx.Err()
	}

	worker := NewBuilderWorker()
	worker.heartbeatInterval = 10 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- worker.executeClaimedRun(ctx, run)
	}()

	<-started
	_, err := RequestBuilderRunCancel(context.Background(), run.ID)
	require.NoError(t, err)

	require.NoError(t, <-errCh)

	var persistedRun entities.BuilderRun
	require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
	assert.Equal(t, entities.BuilderRunStatusCancelled, persistedRun.Status)
	require.NotNil(t, persistedRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseFinalizing, *persistedRun.Phase)

	var persistedSession entities.BuilderSession
	require.NoError(t, db.DB.First(&persistedSession, "id = ?", session.ID).Error)
	assert.Equal(t, entities.BuilderSessionStatusReady, persistedSession.Status)
}

func insertQueuedBuilderRunSeed(t *testing.T, now time.Time, projectID, sessionID, messageID, runID, prompt string) {
	t.Helper()

	insertBuilderSessionSeed(t, now, projectID, sessionID, "env-"+sessionID, entities.BuilderSessionStatusReady, "Queued worker session")
	insertBuilderMessageSeed(t, now, sessionID, messageID, prompt, "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 runID,
		CreatedAt:          now,
		UpdatedAt:          now,
		SessionID:          sessionID,
		TriggerMessageID:   messageID,
		Status:             entities.BuilderRunStatusQueued,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseQueued),
		RequestedBy:        "worker-user",
		InstructionSummary: prompt,
	}).Error)
}

func resetBuilderWorkerExecutionHooks(t *testing.T) {
	t.Helper()

	originalProvisionWorkspace := builderWorkerProvisionWorkspace
	originalLoadConversationMessages := builderWorkerLoadConversationMessages
	originalGenerateFiles := builderWorkerGenerateFiles
	originalWriteAgentFiles := builderWorkerWriteAgentFiles
	originalListWorkspaceFiles := builderWorkerListWorkspaceFiles
	originalRunFrontendCommand := builderWorkerRunFrontendCommand
	originalExecCommandWithContext := builderWorkerExecCommandWithContext
	originalPublishOutputSnapshot := builderWorkerPublishOutputSnapshot

	t.Cleanup(func() {
		builderWorkerProvisionWorkspace = originalProvisionWorkspace
		builderWorkerLoadConversationMessages = originalLoadConversationMessages
		builderWorkerGenerateFiles = originalGenerateFiles
		builderWorkerWriteAgentFiles = originalWriteAgentFiles
		builderWorkerListWorkspaceFiles = originalListWorkspaceFiles
		builderWorkerRunFrontendCommand = originalRunFrontendCommand
		builderWorkerExecCommandWithContext = originalExecCommandWithContext
		builderWorkerPublishOutputSnapshot = originalPublishOutputSnapshot
	})
}

func seedClaimedBuilderWorkerExecutionFixture(t *testing.T, suffix string) (*entities.BuilderRun, *entities.BuilderWorkspace, string) {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Second)
	sessionID := "session-worker-exec-" + suffix
	messageID := "message-worker-exec-" + suffix
	runID := "run-worker-exec-" + suffix
	workspaceID := "workspace-worker-exec-" + suffix
	buildEnvID := "env-worker-exec-" + suffix
	claimToken := "claim-worker-exec-" + suffix
	claimedAt := now.Add(-2 * time.Minute)
	heartbeatAt := claimedAt.Add(15 * time.Second)
	timeoutAt := claimedAt.Add(2 * time.Minute)

	insertBuilderSessionSeed(t, now.Add(-3*time.Minute), "project-worker-exec", sessionID, buildEnvID, entities.BuilderSessionStatusRunning, "Worker execution session")
	insertBuilderMessageSeed(t, now.Add(-3*time.Minute), sessionID, messageID, "Build the generated app.", "worker-user")
	insertBuilderWorkspaceSeed(t, now.Add(-2*time.Minute), sessionID, workspaceID, buildEnvID)

	runWorkspaceID := workspaceID
	run := &entities.BuilderRun{
		ID:                 runID,
		CreatedAt:          now.Add(-2 * time.Minute),
		UpdatedAt:          now.Add(-2 * time.Minute),
		SessionID:          sessionID,
		TriggerMessageID:   messageID,
		WorkspaceID:        &runWorkspaceID,
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseClaiming),
		ClaimToken:         &claimToken,
		ClaimedAt:          &claimedAt,
		HeartbeatAt:        &heartbeatAt,
		TimeoutAt:          &timeoutAt,
		RequestedBy:        "worker-user",
		InstructionSummary: "Build the generated app.",
		StartedAt:          &claimedAt,
	}
	require.NoError(t, db.DB.Create(run).Error)

	workspace := &entities.BuilderWorkspace{
		ID:            workspaceID,
		SessionID:     sessionID,
		BuildEnvID:    buildEnvID,
		ClusterID:     "cluster-test",
		Namespace:     "builder-test",
		PodName:       "builder-pod",
		ContainerName: "workspace",
		Status:        entities.BuilderWorkspaceStatusActive,
		WorkspaceRoot: "/workspace",
	}

	return run, workspace, claimToken
}

func seedClaimedBuilderWorkerDefaultExecutionFixture(t *testing.T) (*entities.BuilderSession, *entities.BuilderWorkspace, *entities.BuilderRun, string) {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Second)
	project := &entities.Project{
		Base: entities.Base{ID: "project-worker-default-cancel"},
		Slug: "project-worker-default-cancel",
		Name: "Default Cancel Project",
	}
	require.NoError(t, db.DB.Create(project).Error)

	cluster := &entities.Cluster{
		Base:       entities.Base{ID: "cluster-worker-default-cancel"},
		Slug:       "cluster-worker-default-cancel",
		Name:       "Default Cancel Cluster",
		KubeConfig: "apiVersion: v1",
	}
	require.NoError(t, db.DB.Create(cluster).Error)

	env := &entities.Env{
		Base:             entities.Base{ID: "env-worker-default-cancel"},
		Slug:             "env-worker-default-cancel",
		Name:             "Default Cancel Env",
		ProjectID:        project.ID,
		ClusterID:        cluster.ID,
		ClusterNamespace: "builder-default-cancel",
		IsBuildEnv:       true,
	}
	require.NoError(t, db.DB.Create(env).Error)

	session := &entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-worker-default-cancel",
			CreatedAt: now.Add(-10 * time.Minute),
			UpdatedAt: now.Add(-5 * time.Minute),
		},
		ProjectID:      project.ID,
		BuildEnvID:     env.ID,
		Title:          "Default cancel session",
		Status:         entities.BuilderSessionStatusRunning,
		CreatedBy:      "user-1",
		LastActivityAt: now.Add(-time.Minute),
	}
	require.NoError(t, db.DB.Create(session).Error)

	workspace := &entities.BuilderWorkspace{
		ID:            "workspace-worker-default-cancel",
		CreatedAt:     now.Add(-4 * time.Minute),
		UpdatedAt:     now.Add(-4 * time.Minute),
		SessionID:     session.ID,
		BuildEnvID:    env.ID,
		ClusterID:     cluster.ID,
		Namespace:     env.ClusterNamespace,
		PodName:       "builder-pod",
		ContainerName: "workspace",
		Status:        entities.BuilderWorkspaceStatusActive,
		WorkspaceRoot: "/workspace",
	}
	require.NoError(t, db.DB.Create(workspace).Error)

	runWorkspaceID := workspace.ID
	run := &entities.BuilderRun{
		ID:                 "run-worker-default-cancel",
		CreatedAt:          now.Add(-3 * time.Minute),
		UpdatedAt:          now.Add(-3 * time.Minute),
		SessionID:          session.ID,
		TriggerMessageID:   "message-worker-default-cancel",
		WorkspaceID:        &runWorkspaceID,
		Status:             entities.BuilderRunStatusExecuting,
		RequestedBy:        "user-1",
		InstructionSummary: "Build the generated app.",
	}
	require.NoError(t, db.DB.Create(run).Error)

	claimToken := "claim-worker-default-cancel"
	claimedAt := now.Add(-2 * time.Minute)
	heartbeatAt := claimedAt.Add(15 * time.Second)
	timeoutAt := claimedAt.Add(2 * time.Minute)

	require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("id = ?", run.ID).Updates(map[string]any{
		"phase":        entities.BuilderRunPhaseClaiming,
		"claim_token":  claimToken,
		"claimed_at":   claimedAt,
		"heartbeat_at": heartbeatAt,
		"timeout_at":   timeoutAt,
		"updated_at":   time.Now().UTC(),
	}).Error)

	phase := entities.BuilderRunPhaseClaiming
	run.Phase = &phase
	run.ClaimToken = &claimToken
	run.ClaimedAt = &claimedAt
	run.HeartbeatAt = &heartbeatAt
	run.TimeoutAt = &timeoutAt
	workspace.Status = entities.BuilderWorkspaceStatusActive

	return session, workspace, run, claimToken
}
