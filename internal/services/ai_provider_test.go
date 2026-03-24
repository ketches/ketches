package services

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupAIProviderServiceTestDB(t *testing.T) {
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

func TestCreateUserAIProvider(t *testing.T) {
	setupAIProviderServiceTestDB(t)

	provider := &entities.UserAIProvider{
		ID:                     "user-provider-1",
		UserID:                 "user-1",
		ProviderKey:            "openai-user",
		DisplayName:            "OpenAI Personal",
		BaseURL:                "https://api.openai.com",
		APIKey:                 "secret-key",
		DefaultModelProfileKey: "gpt-4.1",
		Enabled:                true,
	}

	require.NoError(t, db.DB.Create(provider).Error)

	var stored entities.UserAIProvider
	require.NoError(t, db.DB.First(&stored, "id = ?", provider.ID).Error)
	assert.Equal(t, provider.UserID, stored.UserID)
	assert.Equal(t, provider.ProviderKey, stored.ProviderKey)
	assert.Equal(t, provider.DefaultModelProfileKey, stored.DefaultModelProfileKey)
	assert.True(t, stored.Enabled)
}

func TestCreateProjectAIProvider(t *testing.T) {
	setupAIProviderServiceTestDB(t)

	provider := &entities.ProjectAIProvider{
		ID:                     "project-provider-1",
		ProjectID:              "project-1",
		ProviderKey:            "anthropic-project",
		DisplayName:            "Anthropic Shared",
		BaseURL:                "https://api.anthropic.com",
		APIKey:                 "shared-secret",
		DefaultModelProfileKey: "claude-sonnet-4",
		Enabled:                true,
	}

	require.NoError(t, db.DB.Create(provider).Error)

	var stored entities.ProjectAIProvider
	require.NoError(t, db.DB.First(&stored, "id = ?", provider.ID).Error)
	assert.Equal(t, provider.ProjectID, stored.ProjectID)
	assert.Equal(t, provider.ProviderKey, stored.ProviderKey)
	assert.Equal(t, provider.DefaultModelProfileKey, stored.DefaultModelProfileKey)
	assert.True(t, stored.Enabled)
}

func TestCreateUserAIProviderPersistsDefaultMarker(t *testing.T) {
	setupAIProviderServiceTestDB(t)

	provider := &entities.UserAIProvider{
		ID:                     "user-provider-default",
		UserID:                 "user-1",
		ProviderKey:            "openai-user",
		DisplayName:            "OpenAI Personal",
		BaseURL:                "https://api.openai.com",
		APIKey:                 "secret-key",
		DefaultModelProfileKey: "gpt-4.1",
		Enabled:                true,
		IsDefault:              true,
	}

	require.NoError(t, db.DB.Create(provider).Error)

	var stored entities.UserAIProvider
	require.NoError(t, db.DB.First(&stored, "id = ?", provider.ID).Error)
	assert.True(t, stored.IsDefault)
}

func TestCreateProjectAIProviderPersistsDefaultMarker(t *testing.T) {
	setupAIProviderServiceTestDB(t)

	provider := &entities.ProjectAIProvider{
		ID:                     "project-provider-default",
		ProjectID:              "project-1",
		ProviderKey:            "anthropic-project",
		DisplayName:            "Anthropic Shared",
		BaseURL:                "https://api.anthropic.com",
		APIKey:                 "shared-secret",
		DefaultModelProfileKey: "claude-sonnet-4",
		Enabled:                true,
		IsDefault:              true,
	}

	require.NoError(t, db.DB.Create(provider).Error)

	var stored entities.ProjectAIProvider
	require.NoError(t, db.DB.First(&stored, "id = ?", provider.ID).Error)
	assert.True(t, stored.IsDefault)
}

func TestListUserAIProviders(t *testing.T) {
	setupAIProviderServiceTestDB(t)
	require.NoError(t, db.DB.Create(&entities.UserAIProvider{
		ID:                     "user-provider-1",
		UserID:                 "user-1",
		ProviderKey:            "openai-user",
		DisplayName:            "OpenAI Personal",
		BaseURL:                "https://api.openai.com",
		APIKey:                 "secret-key",
		DefaultModelProfileKey: "gpt-4.1",
		Enabled:                true,
	}).Error)

	providers, err := ListUserAIProviders("user-1")
	require.NoError(t, err)
	require.Len(t, providers, 1)
	assert.Equal(t, "OpenAI Personal", providers[0].DisplayName)
}

func TestListProjectAIProviders(t *testing.T) {
	setupAIProviderServiceTestDB(t)
	require.NoError(t, db.DB.Create(&entities.ProjectAIProvider{
		ID:                     "project-provider-1",
		ProjectID:              "project-1",
		ProviderKey:            "anthropic-project",
		DisplayName:            "Anthropic Shared",
		BaseURL:                "https://api.anthropic.com",
		APIKey:                 "shared-secret",
		DefaultModelProfileKey: "claude-sonnet-4",
		Enabled:                true,
	}).Error)

	providers, err := ListProjectAIProviders("project-1")
	require.NoError(t, err)
	require.Len(t, providers, 1)
	assert.Equal(t, "Anthropic Shared", providers[0].DisplayName)
}

func TestListBuilderAvailableModelOptions(t *testing.T) {
	setupAIProviderServiceTestDB(t)

	originalConfig := app.Config
	app.Config = app.AppConfig{
		BuilderProviderRegistryJSON: `[{
			"key":"anthropic-project","base_url":"https://api.anthropic.com","api_key":"shared-secret"},
			{"key":"openai-user","base_url":"https://api.openai.com","api_key":"secret-key"},
			{"key":"disabled-project","base_url":"https://disabled.example.com","api_key":"disabled"}
		]`,
		BuilderModelProfileRegistryJSON: `[{
			"key":"claude-sonnet-4","model":"claude-4-sonnet"},
			{"key":"gpt-4.1","model":"gpt-4.1"}
		]`,
		BuilderDefaultProviderKey:     "anthropic-project",
		BuilderDefaultModelProfileKey: "claude-sonnet-4",
	}
	t.Cleanup(func() {
		app.Config = originalConfig
	})

	require.NoError(t, db.DB.Create(&entities.UserAIProvider{
		ID:                     "user-provider-1",
		UserID:                 "user-1",
		ProviderKey:            "openai-user",
		DisplayName:            "OpenAI Personal",
		BaseURL:                "https://api.openai.com",
		APIKey:                 "secret-key",
		DefaultModelProfileKey: "gpt-4.1",
		Enabled:                true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectAIProvider{
		ID:                     "project-provider-1",
		ProjectID:              "project-1",
		ProviderKey:            "anthropic-project",
		DisplayName:            "Anthropic Shared",
		BaseURL:                "https://api.anthropic.com",
		APIKey:                 "shared-secret",
		DefaultModelProfileKey: "claude-sonnet-4",
		Enabled:                true,
	}).Error)
	require.NoError(t, db.DB.Create(&entities.ProjectAIProvider{
		ID:                     "project-provider-disabled",
		ProjectID:              "project-1",
		ProviderKey:            "disabled-project",
		DisplayName:            "Disabled Provider",
		BaseURL:                "https://disabled.example.com",
		APIKey:                 "disabled",
		DefaultModelProfileKey: "gpt-4.1",
		Enabled:                false,
	}).Error)

	options, err := ListBuilderAvailableModelOptions("project-1", "user-1")
	require.NoError(t, err)
	require.Len(t, options, 2)

	assert.Equal(t, "project", options[0].Scope)
	assert.Equal(t, "Anthropic Shared", options[0].ProviderLabel)
	assert.Equal(t, "anthropic-project", options[0].ProviderKey)
	assert.Equal(t, "claude-sonnet-4", options[0].ModelProfileKey)

	assert.Equal(t, "user", options[1].Scope)
	assert.Equal(t, "OpenAI Personal", options[1].ProviderLabel)
	assert.Equal(t, "openai-user", options[1].ProviderKey)
	assert.Equal(t, "gpt-4.1", options[1].ModelProfileKey)
}

func TestResolveBuilderEffectiveDefault(t *testing.T) {
	t.Run("project default wins over user default", func(t *testing.T) {
		setupAIProviderServiceTestDB(t)

		originalConfig := app.Config
		app.Config = app.AppConfig{
			BuilderProviderRegistryJSON: `[
				{"key":"anthropic-project","base_url":"https://api.anthropic.com","api_key":"shared-secret"},
				{"key":"openai-user","base_url":"https://api.openai.com","api_key":"secret-key"}
			]`,
			BuilderModelProfileRegistryJSON: `[
				{"key":"claude-sonnet-4","model":"claude-4-sonnet"},
				{"key":"gpt-4.1","model":"gpt-4.1"}
			]`,
			BuilderDefaultProviderKey:     "anthropic-project",
			BuilderDefaultModelProfileKey: "claude-sonnet-4",
		}
		t.Cleanup(func() { app.Config = originalConfig })

		require.NoError(t, db.DB.Create(&entities.ProjectAIProvider{
			ID:                     "project-provider-default",
			ProjectID:              "project-1",
			ProviderKey:            "anthropic-project",
			DisplayName:            "Anthropic Shared",
			BaseURL:                "https://api.anthropic.com",
			APIKey:                 "shared-secret",
			DefaultModelProfileKey: "claude-sonnet-4",
			Enabled:                true,
			IsDefault:              true,
		}).Error)
		require.NoError(t, db.DB.Create(&entities.UserAIProvider{
			ID:                     "user-provider-default",
			UserID:                 "user-1",
			ProviderKey:            "openai-user",
			DisplayName:            "OpenAI Personal",
			BaseURL:                "https://api.openai.com",
			APIKey:                 "secret-key",
			DefaultModelProfileKey: "gpt-4.1",
			Enabled:                true,
			IsDefault:              true,
		}).Error)

		selection, err := ResolveBuilderEffectiveDefault("project-1", "user-1")
		require.NoError(t, err)
		require.NotNil(t, selection)
		assert.Equal(t, "project", selection.EffectiveDefaultSource)
		assert.Equal(t, "anthropic-project", selection.EffectiveDefaultOption.ProviderKey)
	})

	t.Run("user default is used when no project default exists", func(t *testing.T) {
		setupAIProviderServiceTestDB(t)

		originalConfig := app.Config
		app.Config = app.AppConfig{
			BuilderProviderRegistryJSON: `[
				{"key":"openai-user","base_url":"https://api.openai.com","api_key":"secret-key"}
			]`,
			BuilderModelProfileRegistryJSON: `[
				{"key":"gpt-4.1","model":"gpt-4.1"}
			]`,
			BuilderDefaultProviderKey:     "openai-user",
			BuilderDefaultModelProfileKey: "gpt-4.1",
		}
		t.Cleanup(func() { app.Config = originalConfig })

		require.NoError(t, db.DB.Create(&entities.UserAIProvider{
			ID:                     "user-provider-default",
			UserID:                 "user-1",
			ProviderKey:            "openai-user",
			DisplayName:            "OpenAI Personal",
			BaseURL:                "https://api.openai.com",
			APIKey:                 "secret-key",
			DefaultModelProfileKey: "gpt-4.1",
			Enabled:                true,
			IsDefault:              true,
		}).Error)

		selection, err := ResolveBuilderEffectiveDefault("project-1", "user-1")
		require.NoError(t, err)
		require.NotNil(t, selection)
		assert.Equal(t, "user", selection.EffectiveDefaultSource)
		assert.Equal(t, "openai-user", selection.EffectiveDefaultOption.ProviderKey)
	})

	t.Run("invalid defaults are ignored", func(t *testing.T) {
		setupAIProviderServiceTestDB(t)

		originalConfig := app.Config
		app.Config = app.AppConfig{
			BuilderProviderRegistryJSON: `[
				{"key":"valid-provider","base_url":"https://valid.example.com","api_key":"valid-key"}
			]`,
			BuilderModelProfileRegistryJSON: `[
				{"key":"valid-model","model":"valid-model"}
			]`,
			BuilderDefaultProviderKey:     "valid-provider",
			BuilderDefaultModelProfileKey: "valid-model",
		}
		t.Cleanup(func() { app.Config = originalConfig })

		require.NoError(t, db.DB.Create(&entities.ProjectAIProvider{
			ID:                     "project-provider-default",
			ProjectID:              "project-1",
			ProviderKey:            "missing-provider",
			DisplayName:            "Missing",
			BaseURL:                "https://missing.example.com",
			APIKey:                 "missing",
			DefaultModelProfileKey: "missing-model",
			Enabled:                true,
			IsDefault:              true,
		}).Error)

		selection, err := ResolveBuilderEffectiveDefault("project-1", "user-1")
		require.NoError(t, err)
		require.NotNil(t, selection)
		assert.Equal(t, "none", selection.EffectiveDefaultSource)
		assert.Nil(t, selection.EffectiveDefaultOption)
	})
}
