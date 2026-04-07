package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

type BuilderRunEventInput struct {
	Level       entities.BuilderRunEventLevel
	Kind        entities.BuilderRunEventKind
	Phase       *entities.BuilderRunPhase
	Message     string
	PayloadJSON string
}

type builderRunEventBroadcaster struct {
	mu          sync.Mutex
	subscribers map[*builderRunEventSubscriber]struct{}
}

type builderRunEventSubscriber struct {
	ch        chan entities.BuilderRunEvent
	closeOnce sync.Once
}

var builderRunEventBroadcasters sync.Map

func AppendBuilderRunEvent(ctx context.Context, runID string, input BuilderRunEventInput) (*entities.BuilderRunEvent, error) {
	tx := db.DB.WithContext(ctx)
	var event *entities.BuilderRunEvent

	err := tx.Transaction(func(tx *gorm.DB) error {
		var err error
		event, err = appendBuilderRunEventTx(tx, runID, input)
		return err
	})
	if err != nil {
		return nil, err
	}

	publishBuilderRunEvent(event)
	return event, nil
}

func appendBuilderRunEventTx(tx *gorm.DB, runID string, input BuilderRunEventInput) (*entities.BuilderRunEvent, error) {
	if _, err := loadBuilderRunForEvents(tx, runID); err != nil {
		return nil, err
	}

	nextSequence, err := nextBuilderRunEventSequence(tx, runID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	level := input.Level
	if level == "" {
		level = entities.BuilderRunEventLevelInfo
	}
	kind := input.Kind
	if kind == "" {
		kind = entities.BuilderRunEventKindLog
	}

	event := &entities.BuilderRunEvent{
		ID:          uuid.New(),
		CreatedAt:   now,
		RunID:       runID,
		Sequence:    nextSequence,
		Level:       level,
		Kind:        kind,
		Phase:       input.Phase,
		Message:     input.Message,
		PayloadJSON: input.PayloadJSON,
	}
	if err := tx.Create(event).Error; err != nil {
		return nil, err
	}

	executionLog, err := synthesizeBuilderRunExecutionLogTx(tx, runID)
	if err != nil {
		return nil, err
	}

	if err := tx.Model(&entities.BuilderRun{}).
		Where("id = ?", runID).
		Updates(map[string]any{
			"execution_log": executionLog,
			"updated_at":    now,
		}).Error; err != nil {
		return nil, err
	}

	return event, nil
}

func ReplayBuilderRunEventsAfterCursor(ctx context.Context, runID string, afterSequence int64) ([]entities.BuilderRunEvent, error) {
	return replayBuilderRunEventsAfterCursorTx(db.DB.WithContext(ctx), runID, afterSequence)
}

func SubscribeBuilderRunEvents(runID string) (<-chan entities.BuilderRunEvent, func()) {
	subscriber := &builderRunEventSubscriber{
		ch: make(chan entities.BuilderRunEvent, 256),
	}
	rawBroadcaster, _ := builderRunEventBroadcasters.LoadOrStore(runID, &builderRunEventBroadcaster{
		subscribers: map[*builderRunEventSubscriber]struct{}{},
	})
	broadcaster := rawBroadcaster.(*builderRunEventBroadcaster)

	broadcaster.mu.Lock()
	broadcaster.subscribers[subscriber] = struct{}{}
	broadcaster.mu.Unlock()

	return subscriber.ch, func() {
		unsubscribeBuilderRunEventSubscriber(runID, broadcaster, subscriber)
	}
}

func AppendBuilderRunStatusEvent(ctx context.Context, runID string, level entities.BuilderRunEventLevel, message string) (*entities.BuilderRunEvent, error) {
	return AppendBuilderRunExecutionStatusEvent(ctx, runID, nil, level, message)
}

func AppendBuilderRunExecutionLogEvent(ctx context.Context, runID string, phase *entities.BuilderRunPhase, message string) (*entities.BuilderRunEvent, error) {
	return AppendBuilderRunEvent(ctx, runID, BuilderRunEventInput{
		Kind:    entities.BuilderRunEventKindLog,
		Phase:   phase,
		Message: message,
	})
}

func AppendBuilderRunExecutionStatusEvent(ctx context.Context, runID string, phase *entities.BuilderRunPhase, level entities.BuilderRunEventLevel, message string) (*entities.BuilderRunEvent, error) {
	return AppendBuilderRunEvent(ctx, runID, BuilderRunEventInput{
		Kind:    entities.BuilderRunEventKindStatus,
		Phase:   phase,
		Level:   level,
		Message: message,
	})
}

func loadBuilderRunForEvents(tx *gorm.DB, runID string) (*entities.BuilderRun, error) {
	var run entities.BuilderRun
	if err := tx.Where("id = ?", runID).First(&run).Error; err != nil {
		return nil, err
	}
	return &run, nil
}

func nextBuilderRunEventSequence(tx *gorm.DB, runID string) (int64, error) {
	var latest entities.BuilderRunEvent
	if err := tx.Where("run_id = ?", runID).Order("sequence DESC").Take(&latest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 1, nil
		}
		return 0, err
	}
	return latest.Sequence + 1, nil
}

func replayBuilderRunEventsAfterCursorTx(tx *gorm.DB, runID string, afterSequence int64) ([]entities.BuilderRunEvent, error) {
	events := make([]entities.BuilderRunEvent, 0)
	query := tx.Where("run_id = ?", runID)
	if afterSequence > 0 {
		query = query.Where("sequence > ?", afterSequence)
	}
	if err := query.Order("sequence ASC").Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func synthesizeBuilderRunExecutionLogTx(tx *gorm.DB, runID string) (string, error) {
	events, err := replayBuilderRunEventsAfterCursorTx(tx, runID, 0)
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	for i := range events {
		if events[i].Message == "" {
			continue
		}
		builder.WriteString(events[i].Message)
	}

	return builder.String(), nil
}

func publishBuilderRunEvent(event *entities.BuilderRunEvent) {
	if event == nil {
		return
	}

	rawBroadcaster, ok := builderRunEventBroadcasters.Load(event.RunID)
	if !ok {
		return
	}
	broadcaster := rawBroadcaster.(*builderRunEventBroadcaster)

	broadcaster.mu.Lock()
	overflowedSubscribers := make([]*builderRunEventSubscriber, 0)
	for subscriber := range broadcaster.subscribers {
		select {
		case subscriber.ch <- *event:
		default:
			delete(broadcaster.subscribers, subscriber)
			overflowedSubscribers = append(overflowedSubscribers, subscriber)
		}
	}
	empty := len(broadcaster.subscribers) == 0
	broadcaster.mu.Unlock()

	if empty {
		builderRunEventBroadcasters.Delete(event.RunID)
	}
	for i := range overflowedSubscribers {
		overflowedSubscribers[i].close()
	}
}

func unsubscribeBuilderRunEventSubscriber(runID string, broadcaster *builderRunEventBroadcaster, subscriber *builderRunEventSubscriber) {
	if broadcaster == nil || subscriber == nil {
		return
	}

	broadcaster.mu.Lock()
	delete(broadcaster.subscribers, subscriber)
	empty := len(broadcaster.subscribers) == 0
	broadcaster.mu.Unlock()

	if empty {
		builderRunEventBroadcasters.Delete(runID)
	}
	subscriber.close()
}

func (subscriber *builderRunEventSubscriber) close() {
	if subscriber == nil {
		return
	}
	subscriber.closeOnce.Do(func() {
		close(subscriber.ch)
	})
}
