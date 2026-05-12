package services

import (
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBuilderProviderRegistry(t *testing.T) {
	t.Run("loads default provider and model profile from registry config", func(t *testing.T) {
		registry, err := loadBuilderProviderRegistry(app.AppConfig{
			BuilderProviderRegistryJSON:     `[{"key":"openai-compatible-primary","base_url":"https://registry.example.com","api_key":"registry-secret"}]`,
			BuilderModelProfileRegistryJSON: `[{"key":"builder-fast","model":"gpt-5.4-mini"}]`,
			BuilderDefaultProviderKey:       "openai-compatible-primary",
			BuilderDefaultModelProfileKey:   "builder-fast",
		})
		require.NoError(t, err)

		resolved, err := registry.resolveBuilderProviderProfile("", "")
		require.NoError(t, err)
		assert.Equal(t, "openai-compatible-primary", resolved.Provider.Key)
		assert.Equal(t, "https://registry.example.com", resolved.Provider.BaseURL)
		assert.Equal(t, "registry-secret", resolved.Provider.APIKey)
		assert.Equal(t, "builder-fast", resolved.ModelProfile.Key)
		assert.Equal(t, "gpt-5.4-mini", resolved.ModelProfile.Model)
	})

	t.Run("loads single-provider env config as the default registry path", func(t *testing.T) {
		registry, err := loadBuilderProviderRegistry(app.AppConfig{
			BuilderDefaultProviderKey:     "default",
			BuilderDefaultModelProfileKey: "builder-default",
			BuilderAgentBaseURL:           "https://builder.example.com",
			BuilderAgentAPIKey:            "builder-secret",
			BuilderAgentModel:             "gpt-4.1",
		})
		require.NoError(t, err)

		resolved, err := registry.resolveBuilderProviderProfile("", "")
		require.NoError(t, err)
		assert.Equal(t, "default", resolved.Provider.Key)
		assert.Equal(t, "https://builder.example.com", resolved.Provider.BaseURL)
		assert.Equal(t, "builder-secret", resolved.Provider.APIKey)
		assert.Equal(t, "builder-default", resolved.ModelProfile.Key)
		assert.Equal(t, "gpt-4.1", resolved.ModelProfile.Model)
	})

	t.Run("rejects unknown default provider alias", func(t *testing.T) {
		_, err := loadBuilderProviderRegistry(app.AppConfig{
			BuilderProviderRegistryJSON:     `[{"key":"known-provider","base_url":"https://registry.example.com","api_key":"registry-secret"}]`,
			BuilderModelProfileRegistryJSON: `[{"key":"builder-default","model":"gpt-5.4"}]`,
			BuilderDefaultProviderKey:       "missing-provider",
			BuilderDefaultModelProfileKey:   "builder-default",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown builder provider alias")
	})

	t.Run("rejects unknown default model profile alias", func(t *testing.T) {
		_, err := loadBuilderProviderRegistry(app.AppConfig{
			BuilderProviderRegistryJSON:     `[{"key":"known-provider","base_url":"https://registry.example.com","api_key":"registry-secret"}]`,
			BuilderModelProfileRegistryJSON: `[{"key":"builder-default","model":"gpt-5.4"}]`,
			BuilderDefaultProviderKey:       "known-provider",
			BuilderDefaultModelProfileKey:   "missing-model",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown builder model profile alias")
	})
}

func TestResolveBuilderProviderProfile(t *testing.T) {
	registry, err := loadBuilderProviderRegistry(app.AppConfig{
		BuilderProviderRegistryJSON: `[
			{"key":"openai-compatible-primary","base_url":"https://primary.example.com","api_key":"primary-secret"},
			{"key":"openai-compatible-secondary","base_url":"https://secondary.example.com","api_key":"secondary-secret"}
		]`,
		BuilderModelProfileRegistryJSON: `[
			{"key":"builder-default","model":"gpt-5.4"},
			{"key":"builder-fast","model":"gpt-5.4-mini"}
		]`,
		BuilderDefaultProviderKey:     "openai-compatible-primary",
		BuilderDefaultModelProfileKey: "builder-default",
	})
	require.NoError(t, err)

	t.Run("resolves exactly one provider alias and one model profile alias per run", func(t *testing.T) {
		resolved, resolveErr := registry.resolveBuilderProviderProfile("openai-compatible-secondary", "builder-fast")
		require.NoError(t, resolveErr)
		assert.Equal(t, "openai-compatible-secondary", resolved.Provider.Key)
		assert.Equal(t, "https://secondary.example.com", resolved.Provider.BaseURL)
		assert.Equal(t, "secondary-secret", resolved.Provider.APIKey)
		assert.Equal(t, "builder-fast", resolved.ModelProfile.Key)
		assert.Equal(t, "gpt-5.4-mini", resolved.ModelProfile.Model)
	})

	t.Run("rejects unknown provider alias", func(t *testing.T) {
		_, resolveErr := registry.resolveBuilderProviderProfile("missing-provider", "builder-default")
		require.Error(t, resolveErr)
		assert.Contains(t, resolveErr.Error(), "unknown builder provider alias")
	})

	t.Run("rejects unknown model profile alias", func(t *testing.T) {
		_, resolveErr := registry.resolveBuilderProviderProfile("openai-compatible-primary", "missing-model")
		require.Error(t, resolveErr)
		assert.Contains(t, resolveErr.Error(), "unknown builder model profile alias")
	})
}
