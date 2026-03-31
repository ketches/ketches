package services

import "testing"

import "github.com/stretchr/testify/assert"

func TestAnalyzeBuilderProjectIntent(t *testing.T) {
	t.Run("detects node ssr requests", func(t *testing.T) {
		intent := AnalyzeBuilderProjectIntent("Build a Next.js SSR app with authentication")
		assert.Equal(t, "node_ssr_app", intent.ProjectKind)
		assert.Equal(t, "workspace-node-ssr", intent.SuggestedExecutorPolicyKey)
		assert.Equal(t, "node-ssr", intent.SuggestedImageProfileKey)
	})

	t.Run("detects go api requests", func(t *testing.T) {
		intent := AnalyzeBuilderProjectIntent("Create a Golang API using Gin")
		assert.Equal(t, "go_api_service", intent.ProjectKind)
		assert.Equal(t, "workspace-go-api", intent.SuggestedExecutorPolicyKey)
		assert.Equal(t, "go-api", intent.SuggestedImageProfileKey)
	})

	t.Run("defaults to static frontend", func(t *testing.T) {
		intent := AnalyzeBuilderProjectIntent("Build a marketing landing page")
		assert.Equal(t, "static_frontend_app", intent.ProjectKind)
		assert.Equal(t, "workspace-node-static", intent.SuggestedExecutorPolicyKey)
		assert.Equal(t, "node-static", intent.SuggestedImageProfileKey)
	})
}
