package services

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupBuilderSessionServiceTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	t.Cleanup(func() {
		db.DB = originalDB
	})

	testDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	db.DB = testDB
	require.NoError(t, db.Migrate())
}

func TestCreateBuilderSessionUsesProjectScopedContractAndStartsQueuedRun(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	resp, err := CreateBuilderSession(context.Background(), "project-1", "user-1", &models.CreateBuilderSessionRequest{
		BuildEnvID: "env-1",
		Prompt:     "Create a service layer for builder sessions.",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, "project-1", resp.Session.ProjectID)
	assert.Equal(t, "env-1", resp.Session.BuildEnvID)
	assert.Equal(t, "", resp.Session.Title)
	assert.Equal(t, string(entities.BuilderSessionStatusProvisioning), resp.Session.Status)
	assert.Equal(t, string(entities.BuilderRunStatusQueued), resp.Session.LatestRunStatus)
	assert.NotZero(t, resp.Session.LastActivityAt)
	assert.Nil(t, resp.Session.ExpiresAt)

	var session entities.BuilderSession
	require.NoError(t, db.DB.First(&session, "id = ?", resp.Session.ID).Error)
	assert.Equal(t, "project-1", session.ProjectID)
	assert.Equal(t, "env-1", session.BuildEnvID)
	assert.Equal(t, entities.BuilderSessionStatusProvisioning, session.Status)
	assert.Equal(t, "", session.Summary)
	assert.False(t, session.LastActivityAt.IsZero())
	assert.Nil(t, session.ExpiresAt)

	var messages []entities.BuilderMessage
	require.NoError(t, db.DB.Where("session_id = ?", resp.Session.ID).Order("created_at ASC, id ASC").Find(&messages).Error)
	require.Len(t, messages, 1)
	assert.Equal(t, entities.BuilderMessageRoleUser, messages[0].Role)
	assert.Equal(t, "Create a service layer for builder sessions.", messages[0].Content)
	assert.Equal(t, "user-1", messages[0].CreatedBy)
	assert.Nil(t, messages[0].RunID)
	assert.Equal(t, "", messages[0].MetadataJSON)

	var runs []entities.BuilderRun
	require.NoError(t, db.DB.Where("session_id = ?", resp.Session.ID).Order("created_at ASC, id ASC").Find(&runs).Error)
	require.Len(t, runs, 1)
	assert.Equal(t, messages[0].ID, runs[0].TriggerMessageID)
	assert.Equal(t, entities.BuilderRunStatusQueued, runs[0].Status)
	assert.Equal(t, "user-1", runs[0].RequestedBy)
	assert.Nil(t, runs[0].StartedAt)
	assert.Nil(t, runs[0].WorkspaceID)
	assert.Equal(t, "Create a service layer for builder sessions.", runs[0].InstructionSummary)
	assert.Equal(t, "", runs[0].ExecutionLog)

	require.Len(t, resp.Messages, 1)
	assert.Equal(t, "", resp.Messages[0].RunID)
	assert.Equal(t, "", resp.Messages[0].MetadataJSON)
	require.Len(t, resp.Runs, 1)
	assert.Equal(t, string(entities.BuilderRunStatusQueued), resp.Runs[0].Status)
	assert.Equal(t, "Create a service layer for builder sessions.", resp.Runs[0].InstructionSummary)
	assert.Equal(t, "", resp.Runs[0].WorkspaceID)
	assert.Nil(t, resp.Workspace)
	assert.Empty(t, resp.Artifacts)
}

func TestAppendBuilderMessageQueuesRunWhenSessionIsProvisioning(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC()
	lastActivityAt := now.Add(-10 * time.Minute)
	session := entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-1",
			CreatedAt: now.Add(-30 * time.Minute),
			UpdatedAt: now.Add(-10 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Bootstrap API",
		Summary:        "Waiting for workspace.",
		Status:         entities.BuilderSessionStatusProvisioning,
		CreatedBy:      "user-1",
		LastActivityAt: lastActivityAt,
	}
	require.NoError(t, db.DB.Create(&session).Error)

	resp, err := AppendBuilderSessionMessage(context.Background(), "project-1", session.ID, "user-2", &models.AppendBuilderSessionMessageRequest{
		Content: "Queue this until provisioning is finished.",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	var runs []entities.BuilderRun
	require.NoError(t, db.DB.Where("session_id = ?", session.ID).Order("created_at ASC, id ASC").Find(&runs).Error)
	require.Len(t, runs, 1)
	assert.Equal(t, entities.BuilderRunStatusQueued, runs[0].Status)
	assert.Nil(t, runs[0].StartedAt)
	assert.Equal(t, "Queue this until provisioning is finished.", runs[0].InstructionSummary)

	var updatedSession entities.BuilderSession
	require.NoError(t, db.DB.First(&updatedSession, "id = ?", session.ID).Error)
	assert.Equal(t, entities.BuilderSessionStatusProvisioning, updatedSession.Status)
	assert.True(t, updatedSession.LastActivityAt.After(lastActivityAt))

	assert.Equal(t, string(entities.BuilderSessionStatusProvisioning), resp.Session.Status)
	assert.Equal(t, string(entities.BuilderRunStatusQueued), resp.Session.LatestRunStatus)
	require.Len(t, resp.Runs, 1)
	assert.Equal(t, string(entities.BuilderRunStatusQueued), resp.Runs[0].Status)
}

func TestAppendBuilderMessageClaimsExecutionOnlyOnceFromReadySession(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	session := entities.BuilderSession{
		Base:           entities.Base{ID: "session-1"},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Bootstrap API",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "user-1",
		LastActivityAt: time.Now().UTC().Add(-time.Minute),
	}
	require.NoError(t, db.DB.Create(&session).Error)

	claimed, startedAt, err := claimBuilderSessionExecution(db.DB, session.ID)
	require.NoError(t, err)
	assert.True(t, claimed)
	assert.NotNil(t, startedAt)

	claimed, startedAt, err = claimBuilderSessionExecution(db.DB, session.ID)
	require.NoError(t, err)
	assert.False(t, claimed)
	assert.Nil(t, startedAt)

	var updatedSession entities.BuilderSession
	require.NoError(t, db.DB.First(&updatedSession, "id = ?", session.ID).Error)
	assert.Equal(t, entities.BuilderSessionStatusRunning, updatedSession.Status)
}

func TestAppendBuilderMessageQueuesRunWhenReadySessionAlreadyHasExecutingRun(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC()
	session := entities.BuilderSession{
		Base:           entities.Base{ID: "session-1"},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Bootstrap API",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "user-1",
		LastActivityAt: now.Add(-time.Minute),
	}
	require.NoError(t, db.DB.Create(&session).Error)

	originalMessage := entities.BuilderMessage{
		ID:        "message-1",
		SessionID: session.ID,
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Initial prompt",
		CreatedBy: "user-1",
	}
	require.NoError(t, db.DB.Create(&originalMessage).Error)

	executingRun := entities.BuilderRun{
		ID:               "run-1",
		SessionID:        session.ID,
		TriggerMessageID: originalMessage.ID,
		Status:           entities.BuilderRunStatusExecuting,
		RequestedBy:      "user-1",
	}
	require.NoError(t, db.DB.Create(&executingRun).Error)

	resp, err := AppendBuilderSessionMessage(context.Background(), "project-1", session.ID, "user-2", &models.AppendBuilderSessionMessageRequest{
		Content: "Queue behind the active run.",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	var runs []entities.BuilderRun
	require.NoError(t, db.DB.Where("session_id = ?", session.ID).Order("created_at ASC, id ASC").Find(&runs).Error)
	require.Len(t, runs, 2)
	assert.Equal(t, entities.BuilderRunStatusExecuting, runs[0].Status)
	assert.Equal(t, entities.BuilderRunStatusQueued, runs[1].Status)
	assert.Nil(t, runs[1].StartedAt)

	var executingCount int64
	require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("session_id = ? AND status = ?", session.ID, entities.BuilderRunStatusExecuting).Count(&executingCount).Error)
	assert.Equal(t, int64(1), executingCount)
	assert.Equal(t, string(entities.BuilderRunStatusQueued), resp.Session.LatestRunStatus)
}

func TestTouchAppendableBuilderSessionRejectsClosedStateInsideTransaction(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	session := entities.BuilderSession{
		Base:           entities.Base{ID: "session-1"},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Bootstrap API",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "user-1",
		LastActivityAt: time.Now().UTC().Add(-time.Minute),
	}
	require.NoError(t, db.DB.Create(&session).Error)

	tx := db.DB.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() {
		if tx != nil {
			_ = tx.Rollback().Error
		}
	})

	require.NoError(t, tx.Model(&entities.BuilderSession{}).Where("id = ?", session.ID).Update("status", entities.BuilderSessionStatusFailed).Error)

	err := touchAppendableBuilderSession(tx, session.ID, time.Now().UTC())
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBuilderSessionNotAppendable)
	var appendableErr *BuilderSessionNotAppendableError
	require.ErrorAs(t, err, &appendableErr)
	assert.Equal(t, entities.BuilderSessionStatusFailed, appendableErr.Status)

	require.NoError(t, tx.Rollback().Error)
	tx = nil
}

func TestAppendBuilderMessageRejectsClosedSessionStates(t *testing.T) {
	statuses := []entities.BuilderSessionStatus{
		entities.BuilderSessionStatusFailed,
		entities.BuilderSessionStatusArchived,
		entities.BuilderSessionStatusExpired,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			setupBuilderSessionServiceTestDB(t)

			session := entities.BuilderSession{
				Base:           entities.Base{ID: "session-1"},
				ProjectID:      "project-1",
				BuildEnvID:     "env-1",
				Title:          "Bootstrap API",
				Status:         status,
				CreatedBy:      "user-1",
				LastActivityAt: time.Now().UTC().Add(-time.Minute),
			}
			require.NoError(t, db.DB.Create(&session).Error)

			resp, err := AppendBuilderSessionMessage(context.Background(), "project-1", session.ID, "user-2", &models.AppendBuilderSessionMessageRequest{
				Content: "Should be rejected.",
			})
			require.Error(t, err)
			assert.Nil(t, resp)
			assert.ErrorIs(t, err, ErrBuilderSessionNotAppendable)
			var appendableErr *BuilderSessionNotAppendableError
			require.ErrorAs(t, err, &appendableErr)
			assert.Equal(t, status, appendableErr.Status)

			var messageCount int64
			require.NoError(t, db.DB.Model(&entities.BuilderMessage{}).Where("session_id = ?", session.ID).Count(&messageCount).Error)
			assert.Equal(t, int64(0), messageCount)

			var runCount int64
			require.NoError(t, db.DB.Model(&entities.BuilderRun{}).Where("session_id = ?", session.ID).Count(&runCount).Error)
			assert.Equal(t, int64(0), runCount)
		})
	}
}

func TestAppendBuilderMessageReturnsBuilderSessionNotFound(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	resp, err := AppendBuilderSessionMessage(context.Background(), "project-1", "missing-session", "user-1", &models.AppendBuilderSessionMessageRequest{
		Content: "Should not be written.",
	})
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrBuilderSessionNotFound)
}

func TestAppendBuilderMessageUsesContentContractAndStartsRunExecutingWhenIdle(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC()
	lastActivityAt := now.Add(-10 * time.Minute)
	session := entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-1",
			CreatedAt: now.Add(-30 * time.Minute),
			UpdatedAt: now.Add(-10 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Bootstrap API",
		Summary:        "Create the first version.",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "user-1",
		LastActivityAt: lastActivityAt,
	}
	require.NoError(t, db.DB.Create(&session).Error)

	originalMessage := entities.BuilderMessage{
		ID:        "message-1",
		CreatedAt: now.Add(-20 * time.Minute),
		UpdatedAt: now.Add(-20 * time.Minute),
		SessionID: session.ID,
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Create the first version.",
		CreatedBy: "user-1",
	}
	require.NoError(t, db.DB.Create(&originalMessage).Error)

	succeededRun := entities.BuilderRun{
		ID:                 "run-1",
		CreatedAt:          now.Add(-19 * time.Minute),
		UpdatedAt:          now.Add(-18 * time.Minute),
		SessionID:          session.ID,
		TriggerMessageID:   originalMessage.ID,
		Status:             entities.BuilderRunStatusSucceeded,
		RequestedBy:        "user-1",
		InstructionSummary: "Create the first version.",
		ExecutionLog:       "Done.",
	}
	require.NoError(t, db.DB.Create(&succeededRun).Error)

	resp, err := AppendBuilderSessionMessage(context.Background(), "project-1", session.ID, "user-2", &models.AppendBuilderSessionMessageRequest{
		Content: "Add pagination to the service list.",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	var messages []entities.BuilderMessage
	require.NoError(t, db.DB.Where("session_id = ?", session.ID).Order("created_at ASC, id ASC").Find(&messages).Error)
	require.Len(t, messages, 2)
	assert.Equal(t, entities.BuilderMessageRoleUser, messages[1].Role)
	assert.Equal(t, "Add pagination to the service list.", messages[1].Content)
	assert.Equal(t, "user-2", messages[1].CreatedBy)
	assert.Equal(t, "", messages[1].MetadataJSON)

	var runs []entities.BuilderRun
	require.NoError(t, db.DB.Where("session_id = ?", session.ID).Order("created_at ASC, id ASC").Find(&runs).Error)
	require.Len(t, runs, 2)
	assert.Equal(t, entities.BuilderRunStatusSucceeded, runs[0].Status)
	assert.Equal(t, messages[1].ID, runs[1].TriggerMessageID)
	assert.Equal(t, entities.BuilderRunStatusExecuting, runs[1].Status)
	assert.Equal(t, "user-2", runs[1].RequestedBy)
	assert.NotNil(t, runs[1].StartedAt)
	assert.Equal(t, "Add pagination to the service list.", runs[1].InstructionSummary)
	assert.Equal(t, "", runs[1].ExecutionLog)

	var updatedSession entities.BuilderSession
	require.NoError(t, db.DB.First(&updatedSession, "id = ?", session.ID).Error)
	assert.Equal(t, entities.BuilderSessionStatusRunning, updatedSession.Status)
	assert.True(t, updatedSession.LastActivityAt.After(lastActivityAt))

	assert.Equal(t, string(entities.BuilderSessionStatusRunning), resp.Session.Status)
	assert.Equal(t, runs[1].ID, resp.Session.LatestRunID)
	assert.Equal(t, string(entities.BuilderRunStatusExecuting), resp.Session.LatestRunStatus)
	require.Len(t, resp.Messages, 2)
	require.Len(t, resp.Runs, 2)
}

func TestAppendBuilderMessageQueuesRunWhenAnotherRunIsExecuting(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC()
	lastActivityAt := now.Add(-10 * time.Minute)
	session := entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-1",
			CreatedAt: now.Add(-30 * time.Minute),
			UpdatedAt: now.Add(-10 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Bootstrap API",
		Summary:        "Create the first version.",
		Status:         entities.BuilderSessionStatusRunning,
		CreatedBy:      "user-1",
		LastActivityAt: lastActivityAt,
	}
	require.NoError(t, db.DB.Create(&session).Error)

	originalMessage := entities.BuilderMessage{
		ID:        "message-1",
		CreatedAt: now.Add(-20 * time.Minute),
		UpdatedAt: now.Add(-20 * time.Minute),
		SessionID: session.ID,
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Create the first version.",
		CreatedBy: "user-1",
	}
	require.NoError(t, db.DB.Create(&originalMessage).Error)

	executingRun := entities.BuilderRun{
		ID:                 "run-1",
		CreatedAt:          now.Add(-19 * time.Minute),
		UpdatedAt:          now.Add(-18 * time.Minute),
		SessionID:          session.ID,
		TriggerMessageID:   originalMessage.ID,
		Status:             entities.BuilderRunStatusExecuting,
		RequestedBy:        "user-1",
		InstructionSummary: "Create the first version.",
		ExecutionLog:       "Still working.",
	}
	require.NoError(t, db.DB.Create(&executingRun).Error)

	resp, err := AppendBuilderSessionMessage(context.Background(), "project-1", session.ID, "user-2", &models.AppendBuilderSessionMessageRequest{
		Content: "Add pagination to the service list.",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)

	var messages []entities.BuilderMessage
	require.NoError(t, db.DB.Where("session_id = ?", session.ID).Order("created_at ASC, id ASC").Find(&messages).Error)
	require.Len(t, messages, 2)
	assert.Equal(t, entities.BuilderMessageRoleUser, messages[1].Role)
	assert.Equal(t, "Add pagination to the service list.", messages[1].Content)
	assert.Equal(t, "user-2", messages[1].CreatedBy)

	var runs []entities.BuilderRun
	require.NoError(t, db.DB.Where("session_id = ?", session.ID).Order("created_at ASC, id ASC").Find(&runs).Error)
	require.Len(t, runs, 2)
	assert.Equal(t, entities.BuilderRunStatusExecuting, runs[0].Status)
	assert.Equal(t, messages[1].ID, runs[1].TriggerMessageID)
	assert.Equal(t, entities.BuilderRunStatusQueued, runs[1].Status)
	assert.Equal(t, "user-2", runs[1].RequestedBy)
	assert.Nil(t, runs[1].StartedAt)
	assert.Equal(t, "Add pagination to the service list.", runs[1].InstructionSummary)

	var updatedSession entities.BuilderSession
	require.NoError(t, db.DB.First(&updatedSession, "id = ?", session.ID).Error)
	assert.Equal(t, entities.BuilderSessionStatusRunning, updatedSession.Status)
	assert.True(t, updatedSession.LastActivityAt.After(lastActivityAt))

	assert.Equal(t, string(entities.BuilderSessionStatusRunning), resp.Session.Status)
	assert.Equal(t, runs[1].ID, resp.Session.LatestRunID)
	assert.Equal(t, string(entities.BuilderRunStatusQueued), resp.Session.LatestRunStatus)
	require.Len(t, resp.Messages, 2)
	require.Len(t, resp.Runs, 2)
}

func TestGetBuilderSessionDetailDerivesWorkspaceLatestRunAndArtifactSummary(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour)
	terminatedAt := now.Add(-2 * time.Minute)
	session := entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-1",
			CreatedAt: now.Add(-10 * time.Minute),
			UpdatedAt: now.Add(-5 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Bootstrap API",
		Summary:        "Builder session summary",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "user-1",
		LastActivityAt: now.Add(-1 * time.Minute),
		ExpiresAt:      &expiresAt,
	}
	require.NoError(t, db.DB.Create(&session).Error)

	messageOne := entities.BuilderMessage{
		ID:        "message-1",
		CreatedAt: now.Add(-9 * time.Minute),
		UpdatedAt: now.Add(-9 * time.Minute),
		SessionID: session.ID,
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Create the API layout.",
		CreatedBy: "user-1",
	}
	messageTwo := entities.BuilderMessage{
		ID:        "message-2",
		CreatedAt: now.Add(-8 * time.Minute),
		UpdatedAt: now.Add(-8 * time.Minute),
		SessionID: session.ID,
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Add query models.",
		CreatedBy: "user-1",
	}
	require.NoError(t, db.DB.Create(&messageOne).Error)
	require.NoError(t, db.DB.Create(&messageTwo).Error)

	workspaceOneID := "workspace-1"
	workspaceTwoID := "workspace-2"
	runOne := entities.BuilderRun{
		ID:                 "run-1",
		CreatedAt:          now.Add(-7 * time.Minute),
		UpdatedAt:          now.Add(-7 * time.Minute),
		SessionID:          session.ID,
		TriggerMessageID:   messageOne.ID,
		WorkspaceID:        &workspaceOneID,
		Status:             entities.BuilderRunStatusExecuting,
		RequestedBy:        "user-1",
		InstructionSummary: "Create the API layout.",
		ExecutionLog:       "Cloning repository",
	}
	runTwo := entities.BuilderRun{
		ID:                 "run-2",
		CreatedAt:          now.Add(-6 * time.Minute),
		UpdatedAt:          now.Add(-6 * time.Minute),
		SessionID:          session.ID,
		TriggerMessageID:   messageTwo.ID,
		WorkspaceID:        &workspaceTwoID,
		Status:             entities.BuilderRunStatusSucceeded,
		RequestedBy:        "user-1",
		InstructionSummary: "Add query models.",
		ExecutionLog:       "Applied builder diff",
	}
	require.NoError(t, db.DB.Create(&runOne).Error)
	require.NoError(t, db.DB.Create(&runTwo).Error)

	assistantMessageRunID := runTwo.ID
	assistantMessage := entities.BuilderMessage{
		ID:           "message-3",
		CreatedAt:    now.Add(-5 * time.Minute),
		UpdatedAt:    now.Add(-5 * time.Minute),
		SessionID:    session.ID,
		RunID:        &assistantMessageRunID,
		Role:         entities.BuilderMessageRoleAssistant,
		Content:      "Query models added.",
		MetadataJSON: `{"source":"builder-agent"}`,
		CreatedBy:    "user-1",
	}
	require.NoError(t, db.DB.Create(&assistantMessage).Error)

	workspaceOne := entities.BuilderWorkspace{
		ID:            workspaceOneID,
		CreatedAt:     now.Add(-3 * time.Minute),
		UpdatedAt:     now.Add(-3 * time.Minute),
		SessionID:     session.ID,
		BuildEnvID:    "env-1",
		ClusterID:     "cluster-1",
		Namespace:     "builder-session-1-v1",
		PodName:       "builder-session-1-v1-0",
		ContainerName: "workspace",
		Status:        entities.BuilderWorkspaceStatusExpired,
		WorkspaceRoot: "/workspace/session-1-v1",
		TerminatedAt:  &terminatedAt,
	}
	workspaceTwo := entities.BuilderWorkspace{
		ID:            workspaceTwoID,
		CreatedAt:     now.Add(-4 * time.Minute),
		UpdatedAt:     now.Add(-4 * time.Minute),
		SessionID:     session.ID,
		BuildEnvID:    "env-1",
		ClusterID:     "cluster-1",
		Namespace:     "builder-session-1-v2",
		PodName:       "builder-session-1-v2-0",
		ContainerName: "workspace",
		Status:        entities.BuilderWorkspaceStatusActive,
		WorkspaceRoot: "/workspace/session-1-v2",
	}
	require.NoError(t, db.DB.Create(&workspaceOne).Error)
	require.NoError(t, db.DB.Create(&workspaceTwo).Error)

	artifactOne := entities.BuilderArtifact{
		ID:           "artifact-1",
		CreatedAt:    now.Add(-3 * time.Minute),
		UpdatedAt:    now.Add(-3 * time.Minute),
		SessionID:    session.ID,
		WorkspaceID:  workspaceOne.ID,
		RunID:        runOne.ID,
		Kind:         "file",
		Path:         "plans/old-plan.md",
		MetadataJSON: `{"size_bytes":120}`,
	}
	artifactTwo := entities.BuilderArtifact{
		ID:           "artifact-2",
		CreatedAt:    now.Add(-2 * time.Minute),
		UpdatedAt:    now.Add(-2 * time.Minute),
		SessionID:    session.ID,
		WorkspaceID:  workspaceTwo.ID,
		RunID:        runTwo.ID,
		Kind:         "file",
		Path:         "plans/new-plan.md",
		MetadataJSON: `{"size_bytes":256}`,
	}
	require.NoError(t, db.DB.Create(&artifactOne).Error)
	require.NoError(t, db.DB.Create(&artifactTwo).Error)

	resp, err := GetBuilderSessionDetail(context.Background(), "project-1", session.ID)
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, session.ID, resp.Session.ID)
	assert.Equal(t, "env-1", resp.Session.BuildEnvID)
	assert.Equal(t, "Builder session summary", resp.Session.Summary)
	assert.Equal(t, string(entities.BuilderSessionStatusReady), resp.Session.Status)
	assert.WithinDuration(t, session.LastActivityAt, resp.Session.LastActivityAt, time.Second)
	if assert.NotNil(t, resp.Session.ExpiresAt) {
		assert.WithinDuration(t, expiresAt, *resp.Session.ExpiresAt, time.Second)
	}
	assert.Equal(t, workspaceTwo.ID, resp.Session.CurrentWorkspaceID)
	assert.Equal(t, string(entities.BuilderWorkspaceStatusActive), resp.Session.CurrentWorkspaceStatus)
	assert.Equal(t, workspaceTwo.WorkspaceRoot, resp.Session.CurrentWorkspaceRoot)
	assert.Equal(t, runTwo.ID, resp.Session.LatestRunID)
	assert.Equal(t, string(entities.BuilderRunStatusSucceeded), resp.Session.LatestRunStatus)

	require.Len(t, resp.Messages, 3)
	assert.Equal(t, []string{"message-1", "message-2", "message-3"}, []string{resp.Messages[0].ID, resp.Messages[1].ID, resp.Messages[2].ID})
	assert.Equal(t, "", resp.Messages[0].RunID)
	assert.Equal(t, runTwo.ID, resp.Messages[2].RunID)
	assert.Equal(t, `{"source":"builder-agent"}`, resp.Messages[2].MetadataJSON)

	require.Len(t, resp.Runs, 2)
	assert.Equal(t, []string{"run-1", "run-2"}, []string{resp.Runs[0].ID, resp.Runs[1].ID})
	assert.Equal(t, workspaceOne.ID, resp.Runs[0].WorkspaceID)
	assert.Equal(t, workspaceTwo.ID, resp.Runs[1].WorkspaceID)
	assert.Equal(t, "Add query models.", resp.Runs[1].InstructionSummary)
	assert.Equal(t, "Applied builder diff", resp.Runs[1].ExecutionLog)

	require.NotNil(t, resp.Workspace)
	assert.Equal(t, workspaceTwo.ID, resp.Workspace.ID)
	assert.Equal(t, workspaceTwo.BuildEnvID, resp.Workspace.BuildEnvID)
	assert.Equal(t, string(entities.BuilderWorkspaceStatusActive), resp.Workspace.Status)
	assert.Equal(t, workspaceTwo.ClusterID, resp.Workspace.ClusterID)
	assert.Equal(t, workspaceTwo.Namespace, resp.Workspace.Namespace)
	assert.Equal(t, workspaceTwo.PodName, resp.Workspace.PodName)
	assert.Equal(t, workspaceTwo.ContainerName, resp.Workspace.ContainerName)
	assert.Equal(t, workspaceTwo.WorkspaceRoot, resp.Workspace.WorkspaceRoot)
	assert.Nil(t, resp.Workspace.TerminatedAt)

	require.Len(t, resp.Artifacts, 1)
	assert.Equal(t, artifactTwo.ID, resp.Artifacts[0].ID)
	assert.Equal(t, workspaceTwo.ID, resp.Artifacts[0].WorkspaceID)
	assert.Equal(t, runTwo.ID, resp.Artifacts[0].RunID)
	assert.Equal(t, "file", resp.Artifacts[0].Kind)
	assert.Equal(t, "plans/new-plan.md", resp.Artifacts[0].Path)
	assert.Equal(t, `{"size_bytes":256}`, resp.Artifacts[0].MetadataJSON)
}

func TestGetBuilderSessionDetailReturnsBuilderSessionNotFound(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	resp, err := GetBuilderSessionDetail(context.Background(), "project-1", "missing-session")
	require.Error(t, err)
	assert.Nil(t, resp)
	assert.ErrorIs(t, err, ErrBuilderSessionNotFound)
}

func TestToBuilderWorkspaceSummaryFromDetailRowBuildsWorkspaceSummary(t *testing.T) {
	now := time.Now().UTC()
	terminatedAt := now.Add(-time.Minute)
	createdAt := now.Add(-2 * time.Minute)
	updatedAt := now.Add(-30 * time.Second)
	row := models.BuilderSessionDetailRow{
		ID:                            "session-1",
		CurrentWorkspaceID:            "workspace-1",
		CurrentWorkspaceBuildEnvID:    "env-1",
		CurrentWorkspaceClusterID:     "cluster-1",
		CurrentWorkspaceNamespace:     "builder-session-1",
		CurrentWorkspacePodName:       "builder-session-1-0",
		CurrentWorkspaceContainerName: "workspace",
		CurrentWorkspaceStatus:        string(entities.BuilderWorkspaceStatusActive),
		CurrentWorkspaceRoot:          "/workspace/session-1",
		CurrentWorkspaceTerminatedAt:  &terminatedAt,
		CurrentWorkspaceCreatedAt:     &createdAt,
		CurrentWorkspaceUpdatedAt:     &updatedAt,
	}

	workspace := toBuilderWorkspaceSummaryFromDetailRow(&row)
	require.NotNil(t, workspace)
	assert.Equal(t, "workspace-1", workspace.ID)
	assert.Equal(t, "session-1", workspace.SessionID)
	assert.Equal(t, "env-1", workspace.BuildEnvID)
	assert.Equal(t, "cluster-1", workspace.ClusterID)
	assert.Equal(t, "builder-session-1", workspace.Namespace)
	assert.Equal(t, "builder-session-1-0", workspace.PodName)
	assert.Equal(t, "workspace", workspace.ContainerName)
	assert.Equal(t, string(entities.BuilderWorkspaceStatusActive), workspace.Status)
	assert.Equal(t, "/workspace/session-1", workspace.WorkspaceRoot)
	require.NotNil(t, workspace.TerminatedAt)
	assert.WithinDuration(t, terminatedAt, *workspace.TerminatedAt, time.Second)
	assert.WithinDuration(t, createdAt, workspace.CreatedAt, time.Second)
	assert.WithinDuration(t, updatedAt, workspace.UpdatedAt, time.Second)
}

func TestListBuilderSessionsScopesByProjectAndIncludesArtifactCount(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)

	now := time.Now().UTC()
	lastActivityOne := now.Add(-6 * time.Minute)
	lastActivityTwo := now.Add(-2 * time.Minute)
	lastActivityThree := now.Add(-1 * time.Minute)
	sessions := []entities.BuilderSession{
		{
			Base: entities.Base{
				ID:        "session-1",
				CreatedAt: now.Add(-8 * time.Minute),
				UpdatedAt: now.Add(-6 * time.Minute),
			},
			ProjectID:      "project-1",
			BuildEnvID:     "env-1",
			Title:          "Older session",
			Summary:        "Older summary",
			Status:         entities.BuilderSessionStatusRunning,
			CreatedBy:      "user-1",
			LastActivityAt: lastActivityOne,
		},
		{
			Base: entities.Base{
				ID:        "session-2",
				CreatedAt: now.Add(-3 * time.Minute),
				UpdatedAt: now.Add(-2 * time.Minute),
			},
			ProjectID:      "project-1",
			BuildEnvID:     "env-2",
			Title:          "Newer session",
			Summary:        "Newer summary",
			Status:         entities.BuilderSessionStatusReady,
			CreatedBy:      "user-2",
			LastActivityAt: lastActivityTwo,
		},
		{
			Base: entities.Base{
				ID:        "session-3",
				CreatedAt: now.Add(-1 * time.Minute),
				UpdatedAt: now.Add(-1 * time.Minute),
			},
			ProjectID:      "project-2",
			BuildEnvID:     "env-3",
			Title:          "Other project session",
			Summary:        "Other project summary",
			Status:         entities.BuilderSessionStatusArchived,
			CreatedBy:      "user-3",
			LastActivityAt: lastActivityThree,
		},
	}
	require.NoError(t, db.DB.Create(&sessions).Error)

	workspaceTwoID := "workspace-2"
	terminatedWorkspaceTwoID := "workspace-2-terminated"
	runs := []entities.BuilderRun{
		{
			ID:                 "run-1",
			CreatedAt:          now.Add(-7 * time.Minute),
			UpdatedAt:          now.Add(-7 * time.Minute),
			SessionID:          "session-1",
			TriggerMessageID:   "message-1",
			Status:             entities.BuilderRunStatusExecuting,
			RequestedBy:        "user-1",
			InstructionSummary: "Older summary",
		},
		{
			ID:                 "run-2",
			CreatedAt:          now.Add(-2 * time.Minute),
			UpdatedAt:          now.Add(-2 * time.Minute),
			SessionID:          "session-2",
			TriggerMessageID:   "message-2",
			WorkspaceID:        &workspaceTwoID,
			Status:             entities.BuilderRunStatusSucceeded,
			RequestedBy:        "user-2",
			InstructionSummary: "Newer summary",
		},
		{
			ID:                 "run-3",
			CreatedAt:          now.Add(-1 * time.Minute),
			UpdatedAt:          now.Add(-1 * time.Minute),
			SessionID:          "session-3",
			TriggerMessageID:   "message-3",
			Status:             entities.BuilderRunStatusExecuting,
			RequestedBy:        "user-3",
			InstructionSummary: "Other project summary",
		},
	}
	require.NoError(t, db.DB.Create(&runs).Error)

	workspaces := []entities.BuilderWorkspace{
		{
			ID:            workspaceTwoID,
			CreatedAt:     now.Add(-2 * time.Minute),
			UpdatedAt:     now.Add(-2 * time.Minute),
			SessionID:     "session-2",
			BuildEnvID:    "env-2",
			ClusterID:     "cluster-1",
			Namespace:     "builder-session-2",
			PodName:       "builder-session-2-0",
			ContainerName: "workspace",
			Status:        entities.BuilderWorkspaceStatusActive,
			WorkspaceRoot: "/workspace/session-2",
		},
		{
			ID:            terminatedWorkspaceTwoID,
			CreatedAt:     now.Add(-90 * time.Second),
			UpdatedAt:     now.Add(-90 * time.Second),
			SessionID:     "session-2",
			BuildEnvID:    "env-2",
			ClusterID:     "cluster-1",
			Namespace:     "builder-session-2-old",
			PodName:       "builder-session-2-old-0",
			ContainerName: "workspace",
			Status:        entities.BuilderWorkspaceStatusExpired,
			WorkspaceRoot: "/workspace/session-2-old",
			TerminatedAt:  ptrTime(now.Add(-80 * time.Second)),
		},
		{
			ID:            "workspace-3",
			CreatedAt:     now.Add(-1 * time.Minute),
			UpdatedAt:     now.Add(-1 * time.Minute),
			SessionID:     "session-3",
			BuildEnvID:    "env-3",
			ClusterID:     "cluster-2",
			Namespace:     "builder-session-3",
			PodName:       "builder-session-3-0",
			ContainerName: "workspace",
			Status:        entities.BuilderWorkspaceStatusActive,
			WorkspaceRoot: "/workspace/session-3",
		},
	}
	require.NoError(t, db.DB.Create(&workspaces).Error)

	artifacts := []entities.BuilderArtifact{
		{
			ID:           "artifact-1",
			CreatedAt:    now.Add(-90 * time.Second),
			UpdatedAt:    now.Add(-90 * time.Second),
			SessionID:    "session-2",
			WorkspaceID:  workspaceTwoID,
			RunID:        "run-2",
			Kind:         "file",
			Path:         "notes/summary.md",
			MetadataJSON: `{"size_bytes":64}`,
		},
		{
			ID:           "artifact-2",
			CreatedAt:    now.Add(-80 * time.Second),
			UpdatedAt:    now.Add(-80 * time.Second),
			SessionID:    "session-2",
			WorkspaceID:  workspaceTwoID,
			RunID:        "run-2",
			Kind:         "diff",
			Path:         "notes/patch.diff",
			MetadataJSON: `{"size_bytes":128}`,
		},
		{
			ID:           "artifact-3",
			CreatedAt:    now.Add(-70 * time.Second),
			UpdatedAt:    now.Add(-70 * time.Second),
			SessionID:    "session-3",
			WorkspaceID:  "workspace-3",
			RunID:        "run-3",
			Kind:         "file",
			Path:         "notes/other.md",
			MetadataJSON: `{"size_bytes":42}`,
		},
	}
	require.NoError(t, db.DB.Create(&artifacts).Error)

	resp, err := ListBuilderSessions(context.Background(), "project-1")
	require.NoError(t, err)
	require.Len(t, resp, 2)

	assert.Equal(t, []string{"session-2", "session-1"}, []string{resp[0].ID, resp[1].ID})
	assert.Equal(t, "env-2", resp[0].BuildEnvID)
	assert.Equal(t, "Newer summary", resp[0].Summary)
	assert.Equal(t, string(entities.BuilderSessionStatusReady), resp[0].Status)
	assert.Equal(t, string(entities.BuilderRunStatusSucceeded), resp[0].LatestRunStatus)
	assert.Equal(t, string(entities.BuilderWorkspaceStatusActive), resp[0].CurrentWorkspaceStatus)
	assert.Equal(t, "/workspace/session-2", resp[0].CurrentWorkspaceRoot)
	assert.Equal(t, int64(2), resp[0].ArtifactCount)
	assert.WithinDuration(t, lastActivityTwo, resp[0].LastActivityAt, time.Second)

	assert.Equal(t, "env-1", resp[1].BuildEnvID)
	assert.Equal(t, "Older summary", resp[1].Summary)
	assert.Equal(t, string(entities.BuilderSessionStatusRunning), resp[1].Status)
	assert.Equal(t, string(entities.BuilderRunStatusExecuting), resp[1].LatestRunStatus)
	assert.Equal(t, int64(0), resp[1].ArtifactCount)
	assert.NotContains(t, []string{resp[0].ID, resp[1].ID}, "session-3")
}

func ptrTime(v time.Time) *time.Time {
	return &v
}
