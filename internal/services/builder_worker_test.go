package services

import (
	"context"
	"errors"
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
