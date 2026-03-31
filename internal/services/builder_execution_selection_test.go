package services

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupBuilderExecutionSelectionTestDB(t *testing.T) {
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

func TestResolveBuilderExecutionSelection(t *testing.T) {
	originalConfig := app.Config
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	app.Config = app.AppConfig{
		BuilderWorkspaceImage: "node:22-bookworm",
		BuilderExecutionCatalogJSON: `{
			"image_profiles":[
				{"key":"node-static","image":"ghcr.io/ketches/builder-node-static:2026-03-29","capabilities":["node","npm","static-frontend"]},
				{"key":"go-api","image":"ghcr.io/ketches/builder-go-api:2026-03-29","capabilities":["go","api"]}
			],
			"executor_policies":[
				{"key":"workspace-node-static","executor_kind":"workspace_pod","image_profile_key":"node-static"},
				{"key":"workspace-go-api","executor_kind":"workspace_pod","image_profile_key":"go-api"}
			],
			"default_image_profile_key":"node-static",
			"default_executor_policy_key":"workspace-node-static"
		}`,
	}

	t.Run("uses explicit run policy and image profile", func(t *testing.T) {
		selection, err := ResolveBuilderExecutionSelection(context.Background(), &entities.BuilderRun{
			ID:                       "run-explicit",
			ExecutorPolicyKey:        builderStringPtr("workspace-go-api"),
			ExecutionImageProfileKey: builderStringPtr("go-api"),
		})
		require.NoError(t, err)
		require.NotNil(t, selection)
		assert.Equal(t, "workspace-go-api", selection.ExecutorPolicyKey)
		assert.Equal(t, entities.BuilderExecutorHandleKindWorkspacePod, selection.ExecutorKind)
		assert.Equal(t, "go-api", selection.ExecutionImageProfileKey)
		assert.Equal(t, "ghcr.io/ketches/builder-go-api:2026-03-29", selection.ExecutionImageRef)
	})

	t.Run("falls back to catalog defaults", func(t *testing.T) {
		selection, err := ResolveBuilderExecutionSelection(context.Background(), &entities.BuilderRun{
			ID: "run-default",
		})
		require.NoError(t, err)
		require.NotNil(t, selection)
		assert.Equal(t, "workspace-node-static", selection.ExecutorPolicyKey)
		assert.Equal(t, "node-static", selection.ExecutionImageProfileKey)
		assert.Equal(t, "ghcr.io/ketches/builder-node-static:2026-03-29", selection.ExecutionImageRef)
	})

	t.Run("uses planned policy and image profile when explicit selection is absent", func(t *testing.T) {
		selection, err := ResolveBuilderExecutionSelection(context.Background(), &entities.BuilderRun{
			ID:                       "run-planned",
			PlannedProjectKind:       builderStringPtr("go_api_service"),
			PlannedExecutorPolicyKey: builderStringPtr("workspace-go-api"),
			PlannedImageProfileKey:   builderStringPtr("go-api"),
		})
		require.NoError(t, err)
		require.NotNil(t, selection)
		assert.Equal(t, "workspace-go-api", selection.ExecutorPolicyKey)
		assert.Equal(t, "go-api", selection.ExecutionImageProfileKey)
		assert.Equal(t, "ghcr.io/ketches/builder-go-api:2026-03-29", selection.ExecutionImageRef)
	})

	t.Run("rejects unknown policy", func(t *testing.T) {
		selection, err := ResolveBuilderExecutionSelection(context.Background(), &entities.BuilderRun{
			ID:                "run-unknown-policy",
			ExecutorPolicyKey: builderStringPtr("missing-policy"),
		})
		require.Error(t, err)
		assert.Nil(t, selection)
		assert.Contains(t, err.Error(), "unknown builder executor policy key")
	})

	t.Run("rejects unknown image profile", func(t *testing.T) {
		selection, err := ResolveBuilderExecutionSelection(context.Background(), &entities.BuilderRun{
			ID:                       "run-unknown-image",
			ExecutionImageProfileKey: builderStringPtr("missing-image"),
		})
		require.Error(t, err)
		assert.Nil(t, selection)
		assert.Contains(t, err.Error(), "unknown builder image profile key")
	})
}

func TestPersistBuilderRunExecutionSelection(t *testing.T) {
	setupBuilderExecutionSelectionTestDB(t)

	run := &entities.BuilderRun{
		ID:               "run-selection-persist",
		SessionID:        "session-1",
		TriggerMessageID: "message-1",
		Status:           entities.BuilderRunStatusQueued,
		RequestedBy:      "user-1",
	}
	require.NoError(t, db.DB.Create(run).Error)

	err := PersistBuilderRunExecutionSelection(context.Background(), run.ID, &builderResolvedExecutionSelection{
		ExecutorPolicyKey:        "workspace-node-static",
		ExecutorKind:             entities.BuilderExecutorHandleKindWorkspacePod,
		ExecutionImageProfileKey: "node-static",
		ExecutionImageRef:        "ghcr.io/ketches/builder-node-static:2026-03-29",
	})
	require.NoError(t, err)

	var stored entities.BuilderRun
	require.NoError(t, db.DB.First(&stored, "id = ?", run.ID).Error)
	require.NotNil(t, stored.ExecutorPolicyKey)
	require.NotNil(t, stored.ExecutionImageProfileKey)
	require.NotNil(t, stored.ExecutionImageRef)
	assert.Equal(t, "workspace-node-static", *stored.ExecutorPolicyKey)
	assert.Equal(t, "node-static", *stored.ExecutionImageProfileKey)
	assert.Equal(t, "ghcr.io/ketches/builder-node-static:2026-03-29", *stored.ExecutionImageRef)
}
