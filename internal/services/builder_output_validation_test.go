package services

import (
	"testing"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateBuilderExecutionOutputs(t *testing.T) {
	t.Run("accepts static frontend outputs", func(t *testing.T) {
		err := ValidateBuilderExecutionOutputs(&entities.BuilderRun{
			PlannedProjectKind: builderStringPtr("static_frontend_app"),
		}, []entities.BuilderArtifact{
			{Path: "dist/index.html"},
		})
		require.NoError(t, err)
	})

	t.Run("rejects static frontend outputs without index", func(t *testing.T) {
		err := ValidateBuilderExecutionOutputs(&entities.BuilderRun{
			PlannedProjectKind: builderStringPtr("static_frontend_app"),
		}, []entities.BuilderArtifact{
			{Path: "dist/assets/app.js"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "index.html")
	})

	t.Run("accepts go api outputs", func(t *testing.T) {
		err := ValidateBuilderExecutionOutputs(&entities.BuilderRun{
			PlannedProjectKind: builderStringPtr("go_api_service"),
		}, []entities.BuilderArtifact{
			{Path: "build/app"},
		})
		require.NoError(t, err)
	})

	t.Run("rejects go api outputs without build app", func(t *testing.T) {
		err := ValidateBuilderExecutionOutputs(&entities.BuilderRun{
			PlannedProjectKind: builderStringPtr("go_api_service"),
		}, []entities.BuilderArtifact{
			{Path: "build/README.txt"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "build/app")
	})

	t.Run("accepts node ssr outputs", func(t *testing.T) {
		err := ValidateBuilderExecutionOutputs(&entities.BuilderRun{
			PlannedProjectKind: builderStringPtr("node_ssr_app"),
		}, []entities.BuilderArtifact{
			{Path: ".next/routes-manifest.json"},
		})
		require.NoError(t, err)
	})

	t.Run("accepts python api outputs", func(t *testing.T) {
		err := ValidateBuilderExecutionOutputs(&entities.BuilderRun{
			PlannedProjectKind: builderStringPtr("python_api_service"),
		}, []entities.BuilderArtifact{
			{Path: "build/app.py"},
		})
		require.NoError(t, err)
	})
}
