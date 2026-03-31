package services

import (
	"testing"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBuilderExecutionCatalog(t *testing.T) {
	t.Run("loads explicit execution catalog", func(t *testing.T) {
		catalog, err := loadBuilderExecutionCatalog(app.AppConfig{
			BuilderExecutionCatalogJSON: `{
				"image_profiles":[
					{"key":"node-default","image":"node:22-bookworm","description":"Node workspace","capabilities":["node","npm"]}
				],
				"executor_policies":[
					{"key":"workspace-only","executor_kind":"workspace_pod","image_profile_key":"node-default"}
				],
				"default_image_profile_key":"node-default",
				"default_executor_policy_key":"workspace-only"
			}`,
		})
		require.NoError(t, err)
		require.NotNil(t, catalog)
		assert.Equal(t, "node-default", catalog.DefaultImageProfileKey)
		assert.Equal(t, "workspace-only", catalog.DefaultExecutorPolicyKey)
		assert.Equal(t, "node:22-bookworm", catalog.ImageProfiles["node-default"].Image)
		assert.Equal(t, "workspace_pod", catalog.ExecutorPolicies["workspace-only"].ExecutorKind)
	})

	t.Run("falls back to current workspace image config", func(t *testing.T) {
		catalog, err := loadBuilderExecutionCatalog(app.AppConfig{
			BuilderWorkspaceImage:           "node:22-bookworm",
			BuilderDefaultExecutorPolicyKey: "workspace-only",
		})
		require.NoError(t, err)
		require.NotNil(t, catalog)
		assert.Equal(t, "workspace-default-image", catalog.DefaultImageProfileKey)
		assert.Equal(t, "workspace-only", catalog.DefaultExecutorPolicyKey)
		assert.Equal(t, "node:22-bookworm", catalog.ImageProfiles["workspace-default-image"].Image)
		assert.Equal(t, string(entities.BuilderExecutorHandleKindWorkspacePod), catalog.ExecutorPolicies["workspace-only"].ExecutorKind)
		assert.Equal(t, "node:22-bookworm", catalog.ImageProfiles["go-api"].Image)
		assert.Equal(t, "go-api", catalog.ExecutorPolicies["workspace-go-api"].ImageProfileKey)
	})

	t.Run("rejects duplicate image profile key", func(t *testing.T) {
		_, err := loadBuilderExecutionCatalog(app.AppConfig{
			BuilderExecutionCatalogJSON: `{
				"image_profiles":[
					{"key":"node-default","image":"node:22-bookworm"},
					{"key":"node-default","image":"node:20-bookworm"}
				],
				"executor_policies":[
					{"key":"workspace-only","executor_kind":"workspace_pod","image_profile_key":"node-default"}
				],
				"default_image_profile_key":"node-default",
				"default_executor_policy_key":"workspace-only"
			}`,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate builder image profile key")
	})

	t.Run("rejects duplicate executor policy key", func(t *testing.T) {
		_, err := loadBuilderExecutionCatalog(app.AppConfig{
			BuilderExecutionCatalogJSON: `{
				"image_profiles":[
					{"key":"node-default","image":"node:22-bookworm"}
				],
				"executor_policies":[
					{"key":"workspace-only","executor_kind":"workspace_pod","image_profile_key":"node-default"},
					{"key":"workspace-only","executor_kind":"build_job","image_profile_key":"node-default"}
				],
				"default_image_profile_key":"node-default",
				"default_executor_policy_key":"workspace-only"
			}`,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate builder executor policy key")
	})

	t.Run("rejects unsupported executor kind", func(t *testing.T) {
		_, err := loadBuilderExecutionCatalog(app.AppConfig{
			BuilderExecutionCatalogJSON: `{
				"image_profiles":[
					{"key":"node-default","image":"node:22-bookworm"}
				],
				"executor_policies":[
					{"key":"workspace-only","executor_kind":"unsupported_kind","image_profile_key":"node-default"}
				],
				"default_image_profile_key":"node-default",
				"default_executor_policy_key":"workspace-only"
			}`,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported builder executor kind")
	})

	t.Run("rejects missing referenced image profile", func(t *testing.T) {
		_, err := loadBuilderExecutionCatalog(app.AppConfig{
			BuilderExecutionCatalogJSON: `{
				"image_profiles":[
					{"key":"node-default","image":"node:22-bookworm"}
				],
				"executor_policies":[
					{"key":"workspace-only","executor_kind":"workspace_pod","image_profile_key":"missing-profile"}
				],
				"default_image_profile_key":"node-default",
				"default_executor_policy_key":"workspace-only"
			}`,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "references unknown image profile")
	})
}
