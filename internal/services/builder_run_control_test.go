package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestNormalizeLegacyBuilderRunsForControlPlane(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	projectID := "project-cutover"

	insertBuilderSessionSeed(t, now, projectID, "session-queued", "env-queued", entities.BuilderSessionStatusReady, "Queued legacy session")
	insertBuilderMessageSeed(t, now, "session-queued", "message-queued", "Queued legacy prompt", "user-queued")
	insertLegacyBuilderRunSeed(t, now, "run-queued", "session-queued", "message-queued", entities.BuilderRunStatusQueued, nil)

	workspaceID := "workspace-terminal"
	insertBuilderSessionSeed(t, now.Add(time.Minute), projectID, "session-terminal", "env-terminal", entities.BuilderSessionStatusReady, "Terminal legacy session")
	insertBuilderMessageSeed(t, now.Add(time.Minute), "session-terminal", "message-terminal", "Terminal legacy prompt", "user-terminal")
	insertLegacyBuilderRunSeed(t, now.Add(time.Minute), "run-terminal", "session-terminal", "message-terminal", entities.BuilderRunStatusSucceeded, &workspaceID)
	insertBuilderWorkspaceSeed(t, now.Add(time.Minute), "session-terminal", workspaceID, "env-terminal")

	insertBuilderSessionSeed(t, now.Add(90*time.Second), projectID, "session-terminal-queued-phase", "env-terminal-queued-phase", entities.BuilderSessionStatusReady, "Terminal queued-phase legacy session")
	insertBuilderMessageSeed(t, now.Add(90*time.Second), "session-terminal-queued-phase", "message-terminal-queued-phase", "Terminal queued phase prompt", "user-terminal-queued-phase")
	insertLegacyBuilderRunSeed(t, now.Add(90*time.Second), "run-terminal-queued-phase", "session-terminal-queued-phase", "message-terminal-queued-phase", entities.BuilderRunStatusFailed, nil)
	setBuilderRunPhaseSeed(t, "run-terminal-queued-phase", builderRunPhasePtr(entities.BuilderRunPhaseQueued))

	insertBuilderSessionSeed(t, now.Add(2*time.Minute), projectID, "session-modern", "env-modern", entities.BuilderSessionStatusReady, "Modern session")
	insertBuilderMessageSeed(t, now.Add(2*time.Minute), "session-modern", "message-modern", "Modern prompt", "user-modern")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-modern",
		CreatedAt:          now.Add(2 * time.Minute),
		UpdatedAt:          now.Add(2 * time.Minute),
		SessionID:          "session-modern",
		TriggerMessageID:   "message-modern",
		Status:             entities.BuilderRunStatusQueued,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseQueued),
		RequestedBy:        "user-modern",
		InstructionSummary: "Modern prompt",
	}).Error)

	err := NormalizeLegacyBuilderRunsForControlPlane(context.Background())
	require.NoError(t, err)

	var queuedRun entities.BuilderRun
	require.NoError(t, db.DB.First(&queuedRun, "id = ?", "run-queued").Error)
	require.NotNil(t, queuedRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseQueued, *queuedRun.Phase)
	assert.Equal(t, entities.BuilderRunStatusQueued, queuedRun.Status)

	var terminalRun entities.BuilderRun
	require.NoError(t, db.DB.First(&terminalRun, "id = ?", "run-terminal").Error)
	require.NotNil(t, terminalRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseFinalizing, *terminalRun.Phase)
	assert.Equal(t, entities.BuilderRunStatusSucceeded, terminalRun.Status)

	var terminalQueuedPhaseRun entities.BuilderRun
	require.NoError(t, db.DB.First(&terminalQueuedPhaseRun, "id = ?", "run-terminal-queued-phase").Error)
	require.NotNil(t, terminalQueuedPhaseRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseFinalizing, *terminalQueuedPhaseRun.Phase)
	assert.Equal(t, entities.BuilderRunStatusFailed, terminalQueuedPhaseRun.Status)

	var modernRun entities.BuilderRun
	require.NoError(t, db.DB.First(&modernRun, "id = ?", "run-modern").Error)
	require.NotNil(t, modernRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseQueued, *modernRun.Phase)

	detail, err := GetBuilderSessionDetail(context.Background(), projectID, "session-terminal")
	require.NoError(t, err)
	require.NotNil(t, detail)
	assert.Equal(t, "session-terminal", detail.Session.ID)
	assert.Equal(t, string(entities.BuilderRunStatusSucceeded), detail.Session.LatestRunStatus)
	require.Len(t, detail.Runs, 1)
	assert.Equal(t, "run-terminal", detail.Runs[0].ID)
	require.NotNil(t, detail.Workspace)
	assert.Equal(t, workspaceID, detail.Workspace.ID)

	items, err := ListBuilderSessions(context.Background(), projectID)
	require.NoError(t, err)
	require.Len(t, items, 4)

	latestStatuses := make(map[string]string, len(items))
	for i := range items {
		latestStatuses[items[i].ID] = items[i].LatestRunStatus
	}

	assert.Equal(t, string(entities.BuilderRunStatusQueued), latestStatuses["session-queued"])
	assert.Equal(t, string(entities.BuilderRunStatusSucceeded), latestStatuses["session-terminal"])
	assert.Equal(t, string(entities.BuilderRunStatusFailed), latestStatuses["session-terminal-queued-phase"])
	assert.Equal(t, string(entities.BuilderRunStatusQueued), latestStatuses["session-modern"])
}

func TestPreflightBuilderWorkerStartupRepairsMissingPhaseColumn(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	require.NoError(t, db.DB.Migrator().DropColumn(&entities.BuilderRun{}, "Phase"))
	assert.False(t, db.DB.Migrator().HasColumn(&entities.BuilderRun{}, "Phase"))

	require.NoError(t, db.DB.Exec(`
		INSERT INTO builder_runs (
			id, created_at, updated_at, session_id, trigger_message_id, status, requested_by, instruction_summary, execution_log, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "run-missing-phase", now, now, "session-missing-phase", "message-missing-phase", entities.BuilderRunStatusQueued, "test-user", "legacy prompt", "", "").Error)

	err := PreflightBuilderWorkerStartup(context.Background())
	require.NoError(t, err)
	assert.True(t, db.DB.Migrator().HasColumn(&entities.BuilderRun{}, "Phase"))

	var run entities.BuilderRun
	require.NoError(t, db.DB.First(&run, "id = ?", "run-missing-phase").Error)
	require.NotNil(t, run.Phase)
	assert.Equal(t, entities.BuilderRunPhaseQueued, *run.Phase)
}

func TestPreflightBuilderWorkerStartupRepairsMissingControlPlaneColumns(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	insertBuilderSessionSeed(t, now, "project-missing-control-plane-columns", "session-missing-control-plane-columns", "env-missing-control-plane-columns", entities.BuilderSessionStatusRunning, "Legacy executing session")
	insertBuilderMessageSeed(t, now, "session-missing-control-plane-columns", "message-missing-control-plane-columns", "Legacy executing prompt", "test-user")

	missingFields := []string{
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
	for _, field := range missingFields {
		require.NoError(t, db.DB.Migrator().DropColumn(&entities.BuilderRun{}, field))
		assert.False(t, db.DB.Migrator().HasColumn(&entities.BuilderRun{}, field))
	}

	require.NoError(t, db.DB.Exec(`
		INSERT INTO builder_runs (
			id, created_at, updated_at, session_id, trigger_message_id, status, requested_by, instruction_summary, execution_log, started_at, completed_at, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "run-missing-control-plane-columns", now, now, "session-missing-control-plane-columns", "message-missing-control-plane-columns", entities.BuilderRunStatusExecuting, "test-user", "legacy prompt", "", now, nil, "").Error)

	err := PreflightBuilderWorkerStartup(context.Background())
	var startupErr *BuilderWorkerStartupPreflightError
	require.ErrorAs(t, err, &startupErr)
	require.NotNil(t, startupErr)
	assert.Equal(t, int64(1), startupErr.LegacyExecutingRunCount)
	assert.Equal(t, []string{"run-missing-control-plane-columns"}, startupErr.LegacyExecutingRunIDs)

	for _, field := range missingFields {
		assert.True(t, db.DB.Migrator().HasColumn(&entities.BuilderRun{}, field))
	}

	var run entities.BuilderRun
	require.NoError(t, db.DB.First(&run, "id = ?", "run-missing-control-plane-columns").Error)
	assert.Equal(t, entities.BuilderRunStatusExecuting, run.Status)
	assert.Nil(t, run.ClaimToken)
	assert.Nil(t, run.ClaimedAt)
	assert.Nil(t, run.HeartbeatAt)
	assert.Nil(t, run.TimeoutAt)
}

func TestClaimNextQueuedBuilderRun(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	insertBuilderSessionSeed(t, now, "project-claim", "session-claim", "env-claim", entities.BuilderSessionStatusReady, "Queued claim session")
	insertBuilderMessageSeed(t, now, "session-claim", "message-claim", "Queued claim prompt", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-claim",
		CreatedAt:          now,
		UpdatedAt:          now,
		SessionID:          "session-claim",
		TriggerMessageID:   "message-claim",
		Status:             entities.BuilderRunStatusQueued,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseQueued),
		RequestedBy:        "worker-user",
		InstructionSummary: "Queued claim prompt",
	}).Error)

	claimedRun, err := ClaimNextQueuedBuilderRun(context.Background(), "worker-1", 5*time.Minute)
	require.NoError(t, err)
	require.NotNil(t, claimedRun)
	assert.Equal(t, "run-claim", claimedRun.ID)
	assert.Equal(t, entities.BuilderRunStatusExecuting, claimedRun.Status)
	require.NotNil(t, claimedRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseClaiming, *claimedRun.Phase)
	require.NotNil(t, claimedRun.ClaimToken)
	assert.Equal(t, "worker-1", *claimedRun.ClaimToken)
	assert.NotNil(t, claimedRun.ClaimedAt)
	assert.NotNil(t, claimedRun.HeartbeatAt)
	assert.NotNil(t, claimedRun.TimeoutAt)
	assert.NotNil(t, claimedRun.StartedAt)

	secondClaim, err := ClaimNextQueuedBuilderRun(context.Background(), "worker-2", 5*time.Minute)
	require.NoError(t, err)
	assert.Nil(t, secondClaim)

	var persistedRun entities.BuilderRun
	require.NoError(t, db.DB.First(&persistedRun, "id = ?", "run-claim").Error)
	assert.Equal(t, entities.BuilderRunStatusExecuting, persistedRun.Status)
	require.NotNil(t, persistedRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseClaiming, *persistedRun.Phase)
	require.NotNil(t, persistedRun.ClaimToken)
	assert.Equal(t, "worker-1", *persistedRun.ClaimToken)

	var session entities.BuilderSession
	require.NoError(t, db.DB.First(&session, "id = ?", "session-claim").Error)
	assert.Equal(t, entities.BuilderSessionStatusRunning, session.Status)
}

func TestClaimNextQueuedBuilderRunUsesMySQLSafeUpdateShape(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	insertBuilderSessionSeed(t, now, "project-claim-sql", "session-claim-sql", "env-claim-sql", entities.BuilderSessionStatusReady, "Queued claim SQL session")
	insertBuilderMessageSeed(t, now, "session-claim-sql", "message-claim-sql", "Queued claim SQL prompt", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-claim-sql",
		CreatedAt:          now,
		UpdatedAt:          now,
		SessionID:          "session-claim-sql",
		TriggerMessageID:   "message-claim-sql",
		Status:             entities.BuilderRunStatusQueued,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseQueued),
		RequestedBy:        "worker-user",
		InstructionSummary: "Queued claim SQL prompt",
	}).Error)

	callbackName := "test:capture-builder-run-claim-update-sql"
	var capturedSQL string
	require.NoError(t, db.DB.Callback().Update().After("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Schema == nil || tx.Statement.Schema.Table != "builder_runs" {
			return
		}
		capturedSQL = tx.Statement.SQL.String()
	}))
	t.Cleanup(func() {
		db.DB.Callback().Update().Remove(callbackName)
	})

	claimedRun, err := ClaimNextQueuedBuilderRun(context.Background(), "worker-sql", 5*time.Minute)
	require.NoError(t, err)
	require.NotNil(t, claimedRun)
	require.NotEmpty(t, capturedSQL)
	assert.False(t, strings.Contains(capturedSQL, "FROM builder_runs AS active"), capturedSQL)
	assert.False(t, strings.Contains(capturedSQL, "NOT EXISTS (SELECT 1 FROM builder_runs AS active"), capturedSQL)
}

func TestSessionHasExecutingBuilderRunUsesLockingRead(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	claimToken := "worker-lock-check"
	claimedAt := now.Add(-time.Minute)
	timeoutAt := now.Add(time.Minute)
	insertBuilderSessionSeed(t, now, "project-lock-check", "session-lock-check", "env-lock-check", entities.BuilderSessionStatusRunning, "Lock check session")
	insertBuilderMessageSeed(t, now, "session-lock-check", "message-lock-check", "Lock check prompt", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-lock-check",
		CreatedAt:          now,
		UpdatedAt:          now,
		SessionID:          "session-lock-check",
		TriggerMessageID:   "message-lock-check",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseGenerating),
		ClaimToken:         &claimToken,
		ClaimedAt:          &claimedAt,
		HeartbeatAt:        &claimedAt,
		TimeoutAt:          &timeoutAt,
		RequestedBy:        "worker-user",
		InstructionSummary: "Lock check prompt",
	}).Error)

	callbackName := "test:capture-executing-run-locking-read"
	lockingReadSeen := false
	require.NoError(t, db.DB.Callback().Query().Before("gorm:query").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement == nil || tx.Statement.Table != "builder_runs" {
			return
		}
		lockingClause, ok := tx.Statement.Clauses["FOR"]
		if !ok {
			return
		}
		locking, ok := lockingClause.Expression.(clause.Locking)
		if !ok {
			return
		}
		if locking.Strength == "UPDATE" {
			lockingReadSeen = true
		}
	}))
	t.Cleanup(func() {
		db.DB.Callback().Query().Remove(callbackName)
	})

	hasExecutingRun, err := sessionHasExecutingBuilderRun(db.DB, "session-lock-check")
	require.NoError(t, err)
	assert.True(t, hasExecutingRun)
	assert.True(t, lockingReadSeen)
}

func TestClaimNextQueuedBuilderRunRejectsClosedSessionBeforeCommit(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	insertBuilderSessionSeed(t, now, "project-claim-closed", "session-claim-closed", "env-claim-closed", entities.BuilderSessionStatusReady, "Queued claim closed session")
	insertBuilderMessageSeed(t, now, "session-claim-closed", "message-claim-closed", "Queued claim closed prompt", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-claim-closed",
		CreatedAt:          now,
		UpdatedAt:          now,
		SessionID:          "session-claim-closed",
		TriggerMessageID:   "message-claim-closed",
		Status:             entities.BuilderRunStatusQueued,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseQueued),
		RequestedBy:        "worker-user",
		InstructionSummary: "Queued claim closed prompt",
	}).Error)

	registerBuilderSessionClosureDuringRunClaim(t, "session-claim-closed", entities.BuilderSessionStatusArchived)

	claimedRun, err := ClaimNextQueuedBuilderRun(context.Background(), "worker-closed", 5*time.Minute)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBuilderSessionNotAppendable)
	assert.Nil(t, claimedRun)

	var persistedRun entities.BuilderRun
	require.NoError(t, db.DB.First(&persistedRun, "id = ?", "run-claim-closed").Error)
	assert.Equal(t, entities.BuilderRunStatusQueued, persistedRun.Status)
	require.NotNil(t, persistedRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseQueued, *persistedRun.Phase)
	assert.Nil(t, persistedRun.ClaimToken)
	assert.Nil(t, persistedRun.ClaimedAt)
	assert.Nil(t, persistedRun.HeartbeatAt)
	assert.Nil(t, persistedRun.TimeoutAt)
	assert.Nil(t, persistedRun.StartedAt)

	var session entities.BuilderSession
	require.NoError(t, db.DB.First(&session, "id = ?", "session-claim-closed").Error)
	assert.Equal(t, entities.BuilderSessionStatusReady, session.Status)
}

func TestHeartbeatBuilderRunLease(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	claimToken := "worker-lease"
	claimedAt := now.Add(-2 * time.Minute)
	heartbeatAt := now.Add(-90 * time.Second)
	timeoutAt := now.Add(-30 * time.Second)

	insertBuilderSessionSeed(t, now.Add(-3*time.Minute), "project-heartbeat", "session-heartbeat", "env-heartbeat", entities.BuilderSessionStatusRunning, "Heartbeat session")
	insertBuilderMessageSeed(t, now.Add(-3*time.Minute), "session-heartbeat", "message-heartbeat", "Heartbeat prompt", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-heartbeat",
		CreatedAt:          now.Add(-2 * time.Minute),
		UpdatedAt:          now.Add(-2 * time.Minute),
		SessionID:          "session-heartbeat",
		TriggerMessageID:   "message-heartbeat",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseGenerating),
		ClaimToken:         &claimToken,
		ClaimedAt:          &claimedAt,
		HeartbeatAt:        &heartbeatAt,
		TimeoutAt:          &timeoutAt,
		RequestedBy:        "worker-user",
		InstructionSummary: "Heartbeat prompt",
		StartedAt:          &claimedAt,
	}).Error)

	updatedRun, err := HeartbeatBuilderRunLease(context.Background(), "run-heartbeat", claimToken, 4*time.Minute)
	require.NoError(t, err)
	require.NotNil(t, updatedRun)
	require.NotNil(t, updatedRun.HeartbeatAt)
	require.NotNil(t, updatedRun.TimeoutAt)
	assert.True(t, updatedRun.HeartbeatAt.After(heartbeatAt))
	assert.True(t, updatedRun.TimeoutAt.After(timeoutAt))
	assert.Equal(t, entities.BuilderRunStatusExecuting, updatedRun.Status)
	require.NotNil(t, updatedRun.ClaimToken)
	assert.Equal(t, claimToken, *updatedRun.ClaimToken)
}

func TestListRecoverableBuilderRuns(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	expiredClaimToken := "worker-expired"
	expiredClaimedAt := now.Add(-10 * time.Minute)
	expiredHeartbeatAt := now.Add(-4 * time.Minute)
	expiredTimeoutAt := now.Add(-time.Minute)
	healthyClaimToken := "worker-healthy"
	healthyClaimedAt := now.Add(-time.Minute)
	healthyHeartbeatAt := now.Add(-15 * time.Second)
	healthyTimeoutAt := now.Add(2 * time.Minute)

	insertBuilderSessionSeed(t, now.Add(-11*time.Minute), "project-recovery", "session-expired", "env-recovery", entities.BuilderSessionStatusRunning, "Expired recovery session")
	insertBuilderMessageSeed(t, now.Add(-11*time.Minute), "session-expired", "message-expired", "Expired prompt", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-expired",
		CreatedAt:          now.Add(-10 * time.Minute),
		UpdatedAt:          now.Add(-10 * time.Minute),
		SessionID:          "session-expired",
		TriggerMessageID:   "message-expired",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseGenerating),
		ClaimToken:         &expiredClaimToken,
		ClaimedAt:          &expiredClaimedAt,
		HeartbeatAt:        &expiredHeartbeatAt,
		TimeoutAt:          &expiredTimeoutAt,
		RequestedBy:        "worker-user",
		InstructionSummary: "Expired prompt",
		StartedAt:          &expiredClaimedAt,
	}).Error)

	insertBuilderSessionSeed(t, now.Add(-2*time.Minute), "project-recovery", "session-healthy", "env-recovery", entities.BuilderSessionStatusRunning, "Healthy recovery session")
	insertBuilderMessageSeed(t, now.Add(-2*time.Minute), "session-healthy", "message-healthy", "Healthy prompt", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-healthy",
		CreatedAt:          now.Add(-time.Minute),
		UpdatedAt:          now.Add(-time.Minute),
		SessionID:          "session-healthy",
		TriggerMessageID:   "message-healthy",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseGenerating),
		ClaimToken:         &healthyClaimToken,
		ClaimedAt:          &healthyClaimedAt,
		HeartbeatAt:        &healthyHeartbeatAt,
		TimeoutAt:          &healthyTimeoutAt,
		RequestedBy:        "worker-user",
		InstructionSummary: "Healthy prompt",
		StartedAt:          &healthyClaimedAt,
	}).Error)

	recoverableRuns, err := ListRecoverableBuilderRuns(context.Background(), now)
	require.NoError(t, err)
	require.Len(t, recoverableRuns, 1)
	assert.Equal(t, "run-expired", recoverableRuns[0].ID)
	assert.Equal(t, entities.BuilderRunStatusExecuting, recoverableRuns[0].Status)
}

func TestRequeueBuilderRun(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	claimToken := "worker-retry"
	claimedAt := now.Add(-2 * time.Minute)
	heartbeatAt := now.Add(-90 * time.Second)
	timeoutAt := now.Add(2 * time.Minute)
	errorCode := "provider_503"
	errorClass := "provider_transient"

	insertBuilderSessionSeed(t, now.Add(-3*time.Minute), "project-requeue", "session-requeue", "env-requeue", entities.BuilderSessionStatusRunning, "Retry session")
	insertBuilderMessageSeed(t, now.Add(-3*time.Minute), "session-requeue", "message-requeue", "Retry prompt", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-requeue",
		CreatedAt:          now.Add(-2 * time.Minute),
		UpdatedAt:          now.Add(-2 * time.Minute),
		SessionID:          "session-requeue",
		TriggerMessageID:   "message-requeue",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseGenerating),
		AttemptCount:       1,
		ClaimToken:         &claimToken,
		ClaimedAt:          &claimedAt,
		HeartbeatAt:        &heartbeatAt,
		TimeoutAt:          &timeoutAt,
		RequestedBy:        "worker-user",
		InstructionSummary: "Retry prompt",
		StartedAt:          &claimedAt,
	}).Error)

	requeuedRun, err := RequeueBuilderRun(context.Background(), BuilderRunRequeueInput{
		RunID:        "run-requeue",
		ClaimToken:   claimToken,
		ErrorCode:    &errorCode,
		ErrorClass:   &errorClass,
		ErrorMessage: "Upstream provider asked for retry.",
	})
	require.NoError(t, err)
	require.NotNil(t, requeuedRun)
	assert.Equal(t, entities.BuilderRunStatusQueued, requeuedRun.Status)
	require.NotNil(t, requeuedRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseQueued, *requeuedRun.Phase)
	assert.Equal(t, 2, requeuedRun.AttemptCount)
	assert.Nil(t, requeuedRun.ClaimToken)
	assert.Nil(t, requeuedRun.ClaimedAt)
	assert.Nil(t, requeuedRun.HeartbeatAt)
	assert.Nil(t, requeuedRun.TimeoutAt)
	require.NotNil(t, requeuedRun.ErrorCode)
	require.NotNil(t, requeuedRun.ErrorClass)
	assert.Equal(t, errorCode, *requeuedRun.ErrorCode)
	assert.Equal(t, errorClass, *requeuedRun.ErrorClass)
	assert.Equal(t, "Upstream provider asked for retry.", requeuedRun.ErrorMessage)
}

func TestRequeueBuilderRunDeletesPublishedSnapshotsForTheRun(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	claimToken := "worker-retry-snapshot"
	claimedAt := now.Add(-2 * time.Minute)
	heartbeatAt := now.Add(-90 * time.Second)
	timeoutAt := now.Add(2 * time.Minute)

	insertBuilderSessionSeed(t, now.Add(-3*time.Minute), "project-requeue-snapshot", "session-requeue-snapshot", "env-requeue-snapshot", entities.BuilderSessionStatusRunning, "Retry snapshot session")
	insertBuilderMessageSeed(t, now.Add(-3*time.Minute), "session-requeue-snapshot", "message-requeue-snapshot", "Retry snapshot prompt", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-requeue-snapshot",
		CreatedAt:          now.Add(-2 * time.Minute),
		UpdatedAt:          now.Add(-2 * time.Minute),
		SessionID:          "session-requeue-snapshot",
		TriggerMessageID:   "message-requeue-snapshot",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseBuilding),
		AttemptCount:       1,
		ClaimToken:         &claimToken,
		ClaimedAt:          &claimedAt,
		HeartbeatAt:        &heartbeatAt,
		TimeoutAt:          &timeoutAt,
		RequestedBy:        "worker-user",
		InstructionSummary: "Retry snapshot prompt",
		StartedAt:          &claimedAt,
	}).Error)

	publishedAt := now.Add(-time.Minute)
	require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshot{
		ID:               "snapshot-requeue",
		CreatedAt:        publishedAt,
		UpdatedAt:        publishedAt,
		SessionID:        "session-requeue-snapshot",
		RunID:            "run-requeue-snapshot",
		WorkspaceID:      "workspace-requeue-snapshot",
		Status:           entities.BuilderOutputSnapshotStatusPreviewable,
		OutputRoot:       "dist",
		DefaultEntryPath: "dist/index.html",
		StoragePath:      "sessions/session-requeue-snapshot/runs/run-requeue-snapshot/snapshot-requeue",
		FileCount:        1,
		TotalSizeBytes:   512,
		PublishedAt:      publishedAt,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.BuilderOutputSnapshotFile{
		ID:             "snapshot-file-requeue",
		CreatedAt:      publishedAt,
		UpdatedAt:      publishedAt,
		SnapshotID:     "snapshot-requeue",
		RelativePath:   "dist/index.html",
		StoragePath:    "sessions/session-requeue-snapshot/runs/run-requeue-snapshot/snapshot-requeue/dist/index.html",
		SizeBytes:      512,
		ContentType:    "text/html; charset=utf-8",
		IsDefaultEntry: true,
	}).Error)

	requeuedRun, err := RequeueBuilderRun(context.Background(), BuilderRunRequeueInput{
		RunID:        "run-requeue-snapshot",
		ClaimToken:   claimToken,
		ErrorMessage: "Retry after pre-finalization ownership loss.",
	})
	require.NoError(t, err)
	require.NotNil(t, requeuedRun)

	var snapshotCount int64
	require.NoError(t, db.DB.Model(&entities.BuilderOutputSnapshot{}).Where("run_id = ?", "run-requeue-snapshot").Count(&snapshotCount).Error)
	assert.Equal(t, int64(0), snapshotCount)

	var snapshotFileCount int64
	require.NoError(t, db.DB.Model(&entities.BuilderOutputSnapshotFile{}).Where("snapshot_id = ?", "snapshot-requeue").Count(&snapshotFileCount).Error)
	assert.Equal(t, int64(0), snapshotFileCount)
}

func TestRequestBuilderRunCancel(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	insertBuilderSessionSeed(t, now.Add(-2*time.Minute), "project-cancel", "session-cancel", "env-cancel", entities.BuilderSessionStatusRunning, "Cancel session")
	insertBuilderMessageSeed(t, now.Add(-2*time.Minute), "session-cancel", "message-cancel", "Cancel prompt", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-cancel",
		CreatedAt:          now.Add(-time.Minute),
		UpdatedAt:          now.Add(-time.Minute),
		SessionID:          "session-cancel",
		TriggerMessageID:   "message-cancel",
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseGenerating),
		RequestedBy:        "worker-user",
		InstructionSummary: "Cancel prompt",
	}).Error)

	cancelledRun, err := RequestBuilderRunCancel(context.Background(), "run-cancel")
	require.NoError(t, err)
	require.NotNil(t, cancelledRun)
	assert.NotNil(t, cancelledRun.CancelRequestedAt)
	assert.Equal(t, entities.BuilderRunStatusExecuting, cancelledRun.Status)
}

func TestRequestBuilderRunCancelLeavesTerminalRunUntouched(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC().Truncate(time.Second)
	startedAt := now.Add(-2 * time.Minute)
	completedAt := now.Add(-time.Minute)
	updatedAt := now.Add(-90 * time.Second)
	insertBuilderSessionSeed(t, now.Add(-3*time.Minute), "project-cancel-terminal", "session-cancel-terminal", "env-cancel-terminal", entities.BuilderSessionStatusReady, "Terminal cancel session")
	insertBuilderMessageSeed(t, now.Add(-3*time.Minute), "session-cancel-terminal", "message-cancel-terminal", "Terminal cancel prompt", "worker-user")
	require.NoError(t, db.DB.Create(&entities.BuilderRun{
		ID:                 "run-cancel-terminal",
		CreatedAt:          now.Add(-2 * time.Minute),
		UpdatedAt:          updatedAt,
		SessionID:          "session-cancel-terminal",
		TriggerMessageID:   "message-cancel-terminal",
		Status:             entities.BuilderRunStatusSucceeded,
		Phase:              builderRunPhasePtr(entities.BuilderRunPhaseFinalizing),
		RequestedBy:        "worker-user",
		InstructionSummary: "Terminal cancel prompt",
		StartedAt:          &startedAt,
		CompletedAt:        &completedAt,
	}).Error)

	untouchedRun, err := RequestBuilderRunCancel(context.Background(), "run-cancel-terminal")
	require.NoError(t, err)
	require.NotNil(t, untouchedRun)
	assert.Equal(t, entities.BuilderRunStatusSucceeded, untouchedRun.Status)
	require.NotNil(t, untouchedRun.Phase)
	assert.Equal(t, entities.BuilderRunPhaseFinalizing, *untouchedRun.Phase)
	assert.Nil(t, untouchedRun.CancelRequestedAt)
	assert.Equal(t, updatedAt, untouchedRun.UpdatedAt)

	var persistedRun entities.BuilderRun
	require.NoError(t, db.DB.First(&persistedRun, "id = ?", "run-cancel-terminal").Error)
	assert.Equal(t, entities.BuilderRunStatusSucceeded, persistedRun.Status)
	assert.Nil(t, persistedRun.CancelRequestedAt)
	assert.Equal(t, updatedAt, persistedRun.UpdatedAt)
}

func TestFinalizeBuilderRunPersistsTerminalMetadataAndSessionOutcome(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	t.Run("ordinary run failure keeps session ready when workspace is still usable", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		claimToken := "worker-finalize-failed"
		claimedAt := now.Add(-2 * time.Minute)
		heartbeatAt := now.Add(-time.Minute)
		timeoutAt := now.Add(4 * time.Minute)
		errorCode := "command_failed"
		errorClass := "executor_runtime"

		insertBuilderSessionSeed(t, now.Add(-3*time.Minute), "project-finalize", "session-finalize-failed", "env-finalize", entities.BuilderSessionStatusRunning, "Finalize failed session")
		insertBuilderMessageSeed(t, now.Add(-3*time.Minute), "session-finalize-failed", "message-finalize-failed", "Finalize failed prompt", "worker-user")
		insertBuilderWorkspaceSeed(t, now.Add(-2*time.Minute), "session-finalize-failed", "workspace-finalize-failed", "env-finalize")
		require.NoError(t, db.DB.Create(&entities.BuilderRun{
			ID:                 "run-finalize-failed",
			CreatedAt:          now.Add(-2 * time.Minute),
			UpdatedAt:          now.Add(-2 * time.Minute),
			SessionID:          "session-finalize-failed",
			TriggerMessageID:   "message-finalize-failed",
			Status:             entities.BuilderRunStatusExecuting,
			Phase:              builderRunPhasePtr(entities.BuilderRunPhaseGenerating),
			WorkspaceID:        stringPtr("workspace-finalize-failed"),
			ClaimToken:         &claimToken,
			ClaimedAt:          &claimedAt,
			HeartbeatAt:        &heartbeatAt,
			TimeoutAt:          &timeoutAt,
			RequestedBy:        "worker-user",
			InstructionSummary: "Finalize failed prompt",
			StartedAt:          &claimedAt,
		}).Error)

		finalizedRun, err := FinalizeBuilderRun(context.Background(), BuilderRunFinalizeInput{
			RunID:           "run-finalize-failed",
			ClaimToken:      claimToken,
			Status:          entities.BuilderRunStatusFailed,
			ErrorCode:       &errorCode,
			ErrorClass:      &errorClass,
			ErrorMessage:    "The workspace command failed.",
			WorkspaceUsable: true,
		})
		require.NoError(t, err)
		require.NotNil(t, finalizedRun)
		assert.Equal(t, entities.BuilderRunStatusFailed, finalizedRun.Status)
		require.NotNil(t, finalizedRun.Phase)
		assert.Equal(t, entities.BuilderRunPhaseFinalizing, *finalizedRun.Phase)
		assert.NotNil(t, finalizedRun.CompletedAt)
		assert.Nil(t, finalizedRun.ClaimToken)
		assert.Nil(t, finalizedRun.ClaimedAt)
		assert.Nil(t, finalizedRun.HeartbeatAt)
		assert.Nil(t, finalizedRun.TimeoutAt)
		require.NotNil(t, finalizedRun.ErrorCode)
		require.NotNil(t, finalizedRun.ErrorClass)
		assert.Equal(t, errorCode, *finalizedRun.ErrorCode)
		assert.Equal(t, errorClass, *finalizedRun.ErrorClass)
		assert.Equal(t, "The workspace command failed.", finalizedRun.ErrorMessage)

		var session entities.BuilderSession
		require.NoError(t, db.DB.First(&session, "id = ?", "session-finalize-failed").Error)
		assert.Equal(t, entities.BuilderSessionStatusReady, session.Status)
	})

	t.Run("timed out status persists timeout classification metadata", func(t *testing.T) {
		now := time.Now().UTC().Truncate(time.Second)
		claimToken := "worker-finalize-timeout"
		claimedAt := now.Add(-6 * time.Minute)
		heartbeatAt := now.Add(-5 * time.Minute)
		timeoutAt := now.Add(-time.Minute)
		errorCode := "lease_expired"
		errorClass := "timeout"

		insertBuilderSessionSeed(t, now.Add(-7*time.Minute), "project-finalize-timeout", "session-finalize-timeout", "env-finalize", entities.BuilderSessionStatusRunning, "Finalize timeout session")
		insertBuilderMessageSeed(t, now.Add(-7*time.Minute), "session-finalize-timeout", "message-finalize-timeout", "Finalize timeout prompt", "worker-user")
		insertBuilderWorkspaceSeed(t, now.Add(-6*time.Minute), "session-finalize-timeout", "workspace-finalize-timeout", "env-finalize")
		require.NoError(t, db.DB.Create(&entities.BuilderRun{
			ID:                 "run-finalize-timeout",
			CreatedAt:          now.Add(-6 * time.Minute),
			UpdatedAt:          now.Add(-6 * time.Minute),
			SessionID:          "session-finalize-timeout",
			TriggerMessageID:   "message-finalize-timeout",
			Status:             entities.BuilderRunStatusExecuting,
			Phase:              builderRunPhasePtr(entities.BuilderRunPhaseGenerating),
			WorkspaceID:        stringPtr("workspace-finalize-timeout"),
			ClaimToken:         &claimToken,
			ClaimedAt:          &claimedAt,
			HeartbeatAt:        &heartbeatAt,
			TimeoutAt:          &timeoutAt,
			RequestedBy:        "worker-user",
			InstructionSummary: "Finalize timeout prompt",
			StartedAt:          &claimedAt,
		}).Error)

		finalizedRun, err := FinalizeBuilderRun(context.Background(), BuilderRunFinalizeInput{
			RunID:           "run-finalize-timeout",
			ClaimToken:      claimToken,
			Status:          entities.BuilderRunStatusTimedOut,
			ErrorCode:       &errorCode,
			ErrorClass:      &errorClass,
			ErrorMessage:    "The run lease expired before completion.",
			WorkspaceUsable: true,
		})
		require.NoError(t, err)
		require.NotNil(t, finalizedRun)
		assert.Equal(t, entities.BuilderRunStatusTimedOut, finalizedRun.Status)
		require.NotNil(t, finalizedRun.ErrorCode)
		require.NotNil(t, finalizedRun.ErrorClass)
		assert.Equal(t, errorCode, *finalizedRun.ErrorCode)
		assert.Equal(t, errorClass, *finalizedRun.ErrorClass)
		assert.Equal(t, "The run lease expired before completion.", finalizedRun.ErrorMessage)

		var session entities.BuilderSession
		require.NoError(t, db.DB.First(&session, "id = ?", "session-finalize-timeout").Error)
		assert.Equal(t, entities.BuilderSessionStatusReady, session.Status)
	})
}

func stringPtr(v string) *string {
	return &v
}

func registerBuilderSessionClosureDuringRunClaim(t *testing.T, sessionID string, closedStatus entities.BuilderSessionStatus) {
	t.Helper()

	callbackName := "test:close-session-during-run-claim:" + sessionID
	triggered := false
	require.NoError(t, db.DB.Callback().Update().After("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if triggered || tx.Statement == nil || tx.Statement.Schema == nil || tx.Statement.Schema.Table != "builder_runs" {
			return
		}
		triggered = true
		err := tx.Session(&gorm.Session{NewDB: true}).Model(&entities.BuilderSession{}).
			Where("id = ?", sessionID).
			Updates(map[string]any{
				"status":     closedStatus,
				"updated_at": time.Now().UTC(),
			}).Error
		if err != nil {
			tx.AddError(err)
		}
	}))
}

func builderRunPhasePtr(phase entities.BuilderRunPhase) *entities.BuilderRunPhase {
	return &phase
}

func insertBuilderSessionSeed(t *testing.T, now time.Time, projectID, sessionID, buildEnvID string, status entities.BuilderSessionStatus, title string) {
	t.Helper()

	require.NoError(t, db.DB.Exec(`
		INSERT INTO builder_sessions (
			id, created_at, updated_at, project_id, build_env_id, title, summary, status, created_by, last_activity_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, sessionID, now, now, projectID, buildEnvID, title, "", status, "test-user", now).Error)
}

func insertBuilderMessageSeed(t *testing.T, now time.Time, sessionID, messageID, content, createdBy string) {
	t.Helper()

	require.NoError(t, db.DB.Exec(`
		INSERT INTO builder_messages (
			id, created_at, updated_at, session_id, run_id, role, content, metadata_json, created_by
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, messageID, now, now, sessionID, nil, entities.BuilderMessageRoleUser, content, "", createdBy).Error)
}

func insertLegacyBuilderRunSeed(t *testing.T, now time.Time, runID, sessionID, messageID string, status entities.BuilderRunStatus, workspaceID *string) {
	t.Helper()

	completedAt := any(nil)
	if status != entities.BuilderRunStatusQueued && status != entities.BuilderRunStatusExecuting {
		completedAt = now.Add(time.Minute)
	}

	startedAt := any(nil)
	if status != entities.BuilderRunStatusQueued {
		startedAt = now
	}

	require.NoError(t, db.DB.Exec(`
		INSERT INTO builder_runs (
			id, created_at, updated_at, session_id, trigger_message_id, workspace_id, status, requested_by, instruction_summary, execution_log, started_at, completed_at, error_message
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, runID, now, now, sessionID, messageID, workspaceID, status, "test-user", "legacy prompt", "", startedAt, completedAt, "").Error)

	require.NoError(t, db.DB.Exec(`
		UPDATE builder_runs
		SET phase = NULL,
			claim_token = NULL,
			claimed_at = NULL,
			heartbeat_at = NULL,
			timeout_at = NULL,
			cancel_requested_at = NULL,
			provider_key = NULL,
			model_profile_key = NULL,
			executor_policy_key = NULL,
			executor_handle_id = NULL,
			error_code = NULL,
			error_class = NULL
		WHERE id = ?
	`, runID).Error)
}

func insertBuilderWorkspaceSeed(t *testing.T, now time.Time, sessionID, workspaceID, buildEnvID string) {
	t.Helper()

	require.NoError(t, db.DB.Exec(`
		INSERT INTO builder_workspaces (
			id, created_at, updated_at, session_id, build_env_id, cluster_id, namespace, pod_name, container_name, status, workspace_root, terminated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, workspaceID, now, now, sessionID, buildEnvID, "cluster-test", "builder-test", "builder-pod", "workspace", entities.BuilderWorkspaceStatusActive, "/workspace", nil).Error)
}

func setBuilderRunPhaseSeed(t *testing.T, runID string, phase *entities.BuilderRunPhase) {
	t.Helper()

	require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("id = ?", runID).Update("phase", phase).Error)
}
