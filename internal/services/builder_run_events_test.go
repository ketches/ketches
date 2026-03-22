package services

import (
	"context"
	"testing"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedBuilderRunEventTestRun(t *testing.T, runID string) *entities.BuilderRun {
	t.Helper()

	now := time.Now().UTC().Truncate(time.Second)
	queuedPhase := entities.BuilderRunPhaseQueued

	session := &entities.BuilderSession{
		Base: entities.Base{
			ID:        "session-" + runID,
			CreatedAt: now.Add(-2 * time.Minute),
			UpdatedAt: now.Add(-2 * time.Minute),
		},
		ProjectID:      "project-1",
		BuildEnvID:     "env-1",
		Title:          "Event test session",
		Status:         entities.BuilderSessionStatusReady,
		CreatedBy:      "user-1",
		LastActivityAt: now.Add(-time.Minute),
	}
	require.NoError(t, db.DB.Create(session).Error)

	message := &entities.BuilderMessage{
		ID:        "message-" + runID,
		CreatedAt: now.Add(-90 * time.Second),
		UpdatedAt: now.Add(-90 * time.Second),
		SessionID: session.ID,
		Role:      entities.BuilderMessageRoleUser,
		Content:   "Create builder event coverage.",
		CreatedBy: "user-1",
	}
	require.NoError(t, db.DB.Create(message).Error)

	run := &entities.BuilderRun{
		ID:                 runID,
		CreatedAt:          now.Add(-time.Minute),
		UpdatedAt:          now.Add(-time.Minute),
		SessionID:          session.ID,
		TriggerMessageID:   message.ID,
		Status:             entities.BuilderRunStatusExecuting,
		Phase:              &queuedPhase,
		RequestedBy:        "user-1",
		InstructionSummary: message.Content,
		ExecutionLog:       "legacy independent execution log",
	}
	require.NoError(t, db.DB.Create(run).Error)

	return run
}

func TestAppendBuilderRunEventSequence(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)
	run := seedBuilderRunEventTestRun(t, "run-sequence-1")

	firstEvent, err := AppendBuilderRunEvent(context.Background(), run.ID, BuilderRunEventInput{
		Kind:    entities.BuilderRunEventKindLog,
		Message: "[system] run started\n",
	})
	require.NoError(t, err)
	require.NotNil(t, firstEvent)
	assert.Equal(t, int64(1), firstEvent.Sequence)

	secondEvent, err := AppendBuilderRunEvent(context.Background(), run.ID, BuilderRunEventInput{
		Kind:    entities.BuilderRunEventKindLog,
		Message: "[agent] generating files...\n",
	})
	require.NoError(t, err)
	require.NotNil(t, secondEvent)
	assert.Equal(t, int64(2), secondEvent.Sequence)

	var persistedEvents []entities.BuilderRunEvent
	require.NoError(t, db.DB.Where("run_id = ?", run.ID).Order("sequence ASC").Find(&persistedEvents).Error)
	require.Len(t, persistedEvents, 2)
	assert.Equal(t, []int64{1, 2}, []int64{persistedEvents[0].Sequence, persistedEvents[1].Sequence})
	assert.Equal(t, []string{"[system] run started\n", "[agent] generating files...\n"}, []string{persistedEvents[0].Message, persistedEvents[1].Message})

	var persistedRun entities.BuilderRun
	require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
	assert.Equal(t, "[system] run started\n[agent] generating files...\n", persistedRun.ExecutionLog)
	assert.NotContains(t, persistedRun.ExecutionLog, "legacy independent")
}

func TestReplayBuilderRunEventsAfterCursor(t *testing.T) {
	t.Run("replays from the start when no cursor is supplied", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		run := seedBuilderRunEventTestRun(t, "run-replay-start")

		require.NoError(t, db.DB.Create(&[]entities.BuilderRunEvent{
			{
				ID:        "event-start-1",
				CreatedAt: time.Now().UTC().Add(-3 * time.Second),
				RunID:     run.ID,
				Sequence:  1,
				Level:     entities.BuilderRunEventLevelInfo,
				Kind:      entities.BuilderRunEventKindLog,
				Message:   "[system] start\n",
			},
			{
				ID:        "event-start-2",
				CreatedAt: time.Now().UTC().Add(-2 * time.Second),
				RunID:     run.ID,
				Sequence:  2,
				Level:     entities.BuilderRunEventLevelInfo,
				Kind:      entities.BuilderRunEventKindLog,
				Message:   "[agent] continue\n",
			},
		}).Error)

		events, err := ReplayBuilderRunEventsAfterCursor(context.Background(), run.ID, 0)
		require.NoError(t, err)
		require.Len(t, events, 2)
		assert.Equal(t, []int64{1, 2}, []int64{events[0].Sequence, events[1].Sequence})
	})

	t.Run("replays only events after the supplied cursor", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		run := seedBuilderRunEventTestRun(t, "run-replay-after")

		require.NoError(t, db.DB.Create(&[]entities.BuilderRunEvent{
			{
				ID:        "event-after-1",
				CreatedAt: time.Now().UTC().Add(-3 * time.Second),
				RunID:     run.ID,
				Sequence:  1,
				Level:     entities.BuilderRunEventLevelInfo,
				Kind:      entities.BuilderRunEventKindLog,
				Message:   "[system] start\n",
			},
			{
				ID:        "event-after-2",
				CreatedAt: time.Now().UTC().Add(-2 * time.Second),
				RunID:     run.ID,
				Sequence:  2,
				Level:     entities.BuilderRunEventLevelInfo,
				Kind:      entities.BuilderRunEventKindLog,
				Message:   "[agent] continue\n",
			},
			{
				ID:        "event-after-3",
				CreatedAt: time.Now().UTC().Add(-time.Second),
				RunID:     run.ID,
				Sequence:  3,
				Level:     entities.BuilderRunEventLevelInfo,
				Kind:      entities.BuilderRunEventKindStatus,
				Message:   "[system] completed\n",
			},
		}).Error)

		events, err := ReplayBuilderRunEventsAfterCursor(context.Background(), run.ID, 1)
		require.NoError(t, err)
		require.Len(t, events, 2)
		assert.Equal(t, []int64{2, 3}, []int64{events[0].Sequence, events[1].Sequence})
		assert.Equal(t, []string{"[agent] continue\n", "[system] completed\n"}, []string{events[0].Message, events[1].Message})
	})
}

func TestSubscribeBuilderRunEventsDropsOverflowedSubscriber(t *testing.T) {
	events, unsubscribe := SubscribeBuilderRunEvents("run-overflow-1")

	for sequence := int64(1); sequence <= 257; sequence++ {
		publishBuilderRunEvent(&entities.BuilderRunEvent{RunID: "run-overflow-1", Sequence: sequence, Message: "[system] overflow\n"})
	}

	for expectedSequence := int64(1); expectedSequence <= 256; expectedSequence++ {
		event, ok := <-events
		require.True(t, ok)
		assert.Equal(t, expectedSequence, event.Sequence)
	}

	select {
	case _, ok := <-events:
		assert.False(t, ok)
	case <-time.After(50 * time.Millisecond):
		t.Fatal("expected overflowed subscriber channel to close")
	}

	_, exists := builderRunEventBroadcasters.Load("run-overflow-1")
	assert.False(t, exists)
	assert.NotPanics(t, unsubscribe)
}

func TestAppendBuilderRunExecutionEventHelpers(t *testing.T) {
	setupBuilderSessionServiceTestDB(t)
	run := seedBuilderRunEventTestRun(t, "run-execution-event-helper")
	buildingPhase := entities.BuilderRunPhaseBuilding

	logEvent, err := AppendBuilderRunExecutionLogEvent(context.Background(), run.ID, &buildingPhase, "[build] vite build\n")
	require.NoError(t, err)
	require.NotNil(t, logEvent)
	assert.Equal(t, entities.BuilderRunEventKindLog, logEvent.Kind)
	require.NotNil(t, logEvent.Phase)
	assert.Equal(t, buildingPhase, *logEvent.Phase)

	statusEvent, err := AppendBuilderRunExecutionStatusEvent(context.Background(), run.ID, &buildingPhase, entities.BuilderRunEventLevelInfo, "[system] frontend build completed\n")
	require.NoError(t, err)
	require.NotNil(t, statusEvent)
	assert.Equal(t, entities.BuilderRunEventKindStatus, statusEvent.Kind)
	require.NotNil(t, statusEvent.Phase)
	assert.Equal(t, buildingPhase, *statusEvent.Phase)

	var persistedRun entities.BuilderRun
	require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
	assert.Equal(t, "[build] vite build\n[system] frontend build completed\n", persistedRun.ExecutionLog)
}
