package services

import (
	"testing"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectBuilderRuntimeValidationCommand(t *testing.T) {
	t.Run("returns go api runtime smoke command", func(t *testing.T) {
		command, err := DetectBuilderRuntimeValidationCommand(&entities.BuilderRun{
			PlannedProjectKind: builderStringPtr("go_api_service"),
		}, []entities.BuilderArtifact{
			{Path: "build/app"},
		})
		require.NoError(t, err)
		require.NotNil(t, command)
		assert.Equal(t, []string{
			"sh",
			"-lc",
			`PORT=18080 ./build/app >/tmp/builder-runtime.log 2>&1 & pid=$!; sleep 2; if kill -0 "$pid" 2>/dev/null; then kill "$pid"; wait "$pid" || true; exit 0; fi; wait "$pid"; status=$?; if [ -f /tmp/builder-runtime.log ]; then cat /tmp/builder-runtime.log; fi; exit ${status:-1}`,
		}, command)
	})

	t.Run("returns nil for unsupported runtime validation kinds", func(t *testing.T) {
		command, err := DetectBuilderRuntimeValidationCommand(&entities.BuilderRun{
			PlannedProjectKind: builderStringPtr("static_frontend_app"),
		}, []entities.BuilderArtifact{
			{Path: "dist/index.html"},
		})
		require.NoError(t, err)
		assert.Nil(t, command)
	})

	t.Run("returns python import validation command", func(t *testing.T) {
		command, err := DetectBuilderRuntimeValidationCommand(&entities.BuilderRun{
			PlannedProjectKind: builderStringPtr("python_api_service"),
		}, []entities.BuilderArtifact{
			{Path: "build/app.py"},
		})
		require.NoError(t, err)
		require.NotNil(t, command)
		assert.Equal(t, "sh", command[0])
		assert.Equal(t, "-lc", command[1])
		assert.Contains(t, command[2], "build/app.py")
		assert.Contains(t, command[2], "importlib.util")
	})

	t.Run("returns node ssr smoke command only for standalone server output", func(t *testing.T) {
		command, err := DetectBuilderRuntimeValidationCommand(&entities.BuilderRun{
			PlannedProjectKind: builderStringPtr("node_ssr_app"),
		}, []entities.BuilderArtifact{
			{Path: ".next/standalone/server.js"},
		})
		require.NoError(t, err)
		require.NotNil(t, command)
		assert.Equal(t, "sh", command[0])
		assert.Equal(t, "-lc", command[1])
		assert.Contains(t, command[2], "node .next/standalone/server.js")
	})
}
