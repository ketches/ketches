package services

import (
	"context"
	"strings"
	"testing"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectBuilderFrontendExecutionPlan(t *testing.T) {
	t.Run("selects pnpm commands from pnpm lockfile", func(t *testing.T) {
		plan, err := DetectBuilderFrontendExecutionPlan(&models.ListFilesResponse{
			Path: "/workspace",
			Files: []models.FileInfo{
				{Name: "package.json", Type: "file"},
				{Name: "pnpm-lock.yaml", Type: "file"},
				{Name: "src", Type: "dir"},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		assert.Equal(t, BuilderFrontendPackageManagerPNPM, plan.PackageManager)
		assert.Equal(t, []string{"pnpm", "install", "--frozen-lockfile"}, plan.InstallCommand)
		assert.Equal(t, []string{"pnpm", "build"}, plan.BuildCommand)
	})

	t.Run("selects npm commands from package lockfile", func(t *testing.T) {
		plan, err := DetectBuilderFrontendExecutionPlan(&models.ListFilesResponse{
			Path: "/workspace",
			Files: []models.FileInfo{
				{Name: "package.json", Type: "file"},
				{Name: "package-lock.json", Type: "file"},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		assert.Equal(t, BuilderFrontendPackageManagerNPM, plan.PackageManager)
		assert.Equal(t, []string{"npm", "ci"}, plan.InstallCommand)
		assert.Equal(t, []string{"npm", "run", "build"}, plan.BuildCommand)
	})

	t.Run("selects yarn commands from yarn lockfile", func(t *testing.T) {
		plan, err := DetectBuilderFrontendExecutionPlan(&models.ListFilesResponse{
			Path: "/workspace",
			Files: []models.FileInfo{
				{Name: "package.json", Type: "file"},
				{Name: "yarn.lock", Type: "file"},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		assert.Equal(t, BuilderFrontendPackageManagerYarn, plan.PackageManager)
		assert.Equal(t, []string{"yarn", "install", "--frozen-lockfile"}, plan.InstallCommand)
		assert.Equal(t, []string{"yarn", "build"}, plan.BuildCommand)
	})

	t.Run("selects go commands from go module project", func(t *testing.T) {
		plan, err := DetectBuilderFrontendExecutionPlan(&models.ListFilesResponse{
			Path: "/workspace",
			Files: []models.FileInfo{
				{Name: "go.mod", Type: "file"},
				{Name: "main.go", Type: "file"},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		assert.Equal(t, BuilderFrontendPackageManagerGo, plan.PackageManager)
		assert.Equal(t, []string{"go", "mod", "download"}, plan.InstallCommand)
		assert.Equal(t, []string{"sh", "-lc", "mkdir -p build && go build -o build/app ."}, plan.BuildCommand)
	})

	t.Run("selects python commands from requirements and app entrypoint", func(t *testing.T) {
		plan, err := DetectBuilderFrontendExecutionPlan(&models.ListFilesResponse{
			Path: "/workspace",
			Files: []models.FileInfo{
				{Name: "requirements.txt", Type: "file"},
				{Name: "app.py", Type: "file"},
			},
		})

		require.NoError(t, err)
		require.NotNil(t, plan)
		assert.Equal(t, BuilderFrontendPackageManagerPython, plan.PackageManager)
		assert.Equal(t, []string{"pip", "install", "-r", "requirements.txt"}, plan.InstallCommand)
		assert.Equal(t, []string{"sh", "-lc", "mkdir -p build && if [ -f app.py ]; then cp app.py build/app.py; else cp main.py build/main.py; fi"}, plan.BuildCommand)
	})

	t.Run("fails when supported lockfile exists without package json", func(t *testing.T) {
		plan, err := DetectBuilderFrontendExecutionPlan(&models.ListFilesResponse{
			Path: "/workspace",
			Files: []models.FileInfo{
				{Name: "pnpm-lock.yaml", Type: "file"},
			},
		})

		require.Error(t, err)
		assert.Nil(t, plan)
		assert.ErrorContains(t, err, "package.json")
	})

	t.Run("fails when package json exists without a supported lockfile", func(t *testing.T) {
		plan, err := DetectBuilderFrontendExecutionPlan(&models.ListFilesResponse{
			Path: "/workspace",
			Files: []models.FileInfo{
				{Name: "package.json", Type: "file"},
			},
		})

		require.Error(t, err)
		assert.Nil(t, plan)
		assert.ErrorContains(t, err, "supported frontend lockfile is required")
	})

	t.Run("fails when go.mod exists without a root main package", func(t *testing.T) {
		plan, err := DetectBuilderFrontendExecutionPlan(&models.ListFilesResponse{
			Path: "/workspace",
			Files: []models.FileInfo{
				{Name: "go.mod", Type: "file"},
				{Name: "internal", Type: "dir"},
			},
		})

		require.Error(t, err)
		assert.Nil(t, plan)
		assert.ErrorContains(t, err, "root main.go is required")
	})

	t.Run("fails when python project markers exist without a root python entrypoint", func(t *testing.T) {
		plan, err := DetectBuilderFrontendExecutionPlan(&models.ListFilesResponse{
			Path: "/workspace",
			Files: []models.FileInfo{
				{Name: "requirements.txt", Type: "file"},
				{Name: "src", Type: "dir"},
			},
		})

		require.Error(t, err)
		assert.Nil(t, plan)
		assert.ErrorContains(t, err, "root app.py or main.py is required")
	})

	t.Run("fails when multiple supported lockfiles exist", func(t *testing.T) {
		plan, err := DetectBuilderFrontendExecutionPlan(&models.ListFilesResponse{
			Path: "/workspace",
			Files: []models.FileInfo{
				{Name: "package.json", Type: "file"},
				{Name: "pnpm-lock.yaml", Type: "file"},
				{Name: "package-lock.json", Type: "file"},
			},
		})

		require.Error(t, err)
		assert.Nil(t, plan)
		assert.ErrorContains(t, err, "multiple supported frontend lockfiles")
	})
}

func TestExecuteBuilderFrontendPlan(t *testing.T) {
	newPlan := func() *BuilderFrontendExecutionPlan {
		return &BuilderFrontendExecutionPlan{
			PackageManager: BuilderFrontendPackageManagerNPM,
			InstallCommand: []string{"npm", "ci"},
			BuildCommand:   []string{"npm", "run", "build"},
		}
	}

	t.Run("runs install then build and persists durable execution events", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		run := seedBuilderRunEventTestRun(t, "run-execute-plan-success")

		calls := make([]string, 0, 2)
		result, err := ExecuteBuilderFrontendPlan(context.Background(), run.ID, newPlan(), BuilderFrontendCommandRunnerFunc(func(ctx context.Context, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			calls = append(calls, string(step)+":"+strings.Join(command, " "))
			switch step {
			case BuilderFrontendExecutionStepInstall:
				require.NoError(t, appendLog("installing dependencies\n"))
			case BuilderFrontendExecutionStepBuild:
				require.NoError(t, appendLog("building frontend\n"))
			default:
				t.Fatalf("unexpected step %q", step)
			}
			return BuilderFrontendCommandResult{ExitCode: 0}, nil
		}))

		require.NoError(t, err)
		require.NotNil(t, result)
		assert.Equal(t, BuilderFrontendExecutionStepBuild, result.LastCompletedStep)
		assert.Equal(t, []string{
			"install:npm ci",
			"build:npm run build",
		}, calls)

		var persistedEvents []entities.BuilderRunEvent
		require.NoError(t, db.DB.Where("run_id = ?", run.ID).Order("sequence ASC").Find(&persistedEvents).Error)
		require.Len(t, persistedEvents, 5)
		assert.Equal(t, []entities.BuilderRunEventKind{
			entities.BuilderRunEventKindStatus,
			entities.BuilderRunEventKindLog,
			entities.BuilderRunEventKindStatus,
			entities.BuilderRunEventKindLog,
			entities.BuilderRunEventKindStatus,
		}, []entities.BuilderRunEventKind{
			persistedEvents[0].Kind,
			persistedEvents[1].Kind,
			persistedEvents[2].Kind,
			persistedEvents[3].Kind,
			persistedEvents[4].Kind,
		})
		assert.Equal(t, []string{
			"[system] installing frontend dependencies\n",
			"installing dependencies\n",
			"[system] running frontend build\n",
			"building frontend\n",
			"[system] frontend build completed\n",
		}, []string{
			persistedEvents[0].Message,
			persistedEvents[1].Message,
			persistedEvents[2].Message,
			persistedEvents[3].Message,
			persistedEvents[4].Message,
		})

		var persistedRun entities.BuilderRun
		require.NoError(t, db.DB.First(&persistedRun, "id = ?", run.ID).Error)
		assert.Equal(t, strings.Join([]string{
			"[system] installing frontend dependencies\n",
			"installing dependencies\n",
			"[system] running frontend build\n",
			"building frontend\n",
			"[system] frontend build completed\n",
		}, ""), persistedRun.ExecutionLog)
	})

	t.Run("classifies non-zero install exits as execution failure", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		run := seedBuilderRunEventTestRun(t, "run-execute-plan-install-failure")

		calls := make([]string, 0, 1)
		result, err := ExecuteBuilderFrontendPlan(context.Background(), run.ID, newPlan(), BuilderFrontendCommandRunnerFunc(func(ctx context.Context, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			calls = append(calls, string(step)+":"+strings.Join(command, " "))
			require.Equal(t, BuilderFrontendExecutionStepInstall, step)
			require.NoError(t, appendLog("npm ci failed\n"))
			return BuilderFrontendCommandResult{ExitCode: 17}, nil
		}))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, []string{"install:npm ci"}, calls)

		var execErr *BuilderFrontendExecutionError
		require.ErrorAs(t, err, &execErr)
		assert.Equal(t, BuilderFrontendExecutionStepInstall, execErr.Step)
		assert.Equal(t, 17, execErr.ExitCode)

		var persistedEvents []entities.BuilderRunEvent
		require.NoError(t, db.DB.Where("run_id = ?", run.ID).Order("sequence ASC").Find(&persistedEvents).Error)
		require.Len(t, persistedEvents, 3)
		assert.Equal(t, []entities.BuilderRunEventLevel{
			entities.BuilderRunEventLevelInfo,
			entities.BuilderRunEventLevelInfo,
			entities.BuilderRunEventLevelError,
		}, []entities.BuilderRunEventLevel{
			persistedEvents[0].Level,
			persistedEvents[1].Level,
			persistedEvents[2].Level,
		})
		assert.Equal(t, []string{
			"[system] installing frontend dependencies\n",
			"npm ci failed\n",
			"[system] install command exited with status 17\n",
		}, []string{
			persistedEvents[0].Message,
			persistedEvents[1].Message,
			persistedEvents[2].Message,
		})
	})

	t.Run("classifies non-zero build exits as execution failure", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		run := seedBuilderRunEventTestRun(t, "run-execute-plan-build-failure")

		calls := make([]string, 0, 2)
		result, err := ExecuteBuilderFrontendPlan(context.Background(), run.ID, newPlan(), BuilderFrontendCommandRunnerFunc(func(ctx context.Context, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			calls = append(calls, string(step)+":"+strings.Join(command, " "))
			switch step {
			case BuilderFrontendExecutionStepInstall:
				require.NoError(t, appendLog("dependencies installed\n"))
				return BuilderFrontendCommandResult{ExitCode: 0}, nil
			case BuilderFrontendExecutionStepBuild:
				require.NoError(t, appendLog("vite build failed\n"))
				return BuilderFrontendCommandResult{ExitCode: 2}, nil
			default:
				t.Fatalf("unexpected step %q", step)
				return BuilderFrontendCommandResult{}, nil
			}
		}))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, []string{
			"install:npm ci",
			"build:npm run build",
		}, calls)

		var execErr *BuilderFrontendExecutionError
		require.ErrorAs(t, err, &execErr)
		assert.Equal(t, BuilderFrontendExecutionStepBuild, execErr.Step)
		assert.Equal(t, 2, execErr.ExitCode)

		var persistedEvents []entities.BuilderRunEvent
		require.NoError(t, db.DB.Where("run_id = ?", run.ID).Order("sequence ASC").Find(&persistedEvents).Error)
		require.Len(t, persistedEvents, 5)
		assert.Equal(t, "[system] build command exited with status 2\n", persistedEvents[4].Message)
		assert.Equal(t, entities.BuilderRunEventLevelError, persistedEvents[4].Level)
	})

	t.Run("stops when the execution context is cancelled", func(t *testing.T) {
		setupBuilderSessionServiceTestDB(t)
		run := seedBuilderRunEventTestRun(t, "run-execute-plan-cancelled")

		ctx, cancel := context.WithCancel(context.Background())
		calls := make([]string, 0, 1)
		result, err := ExecuteBuilderFrontendPlan(ctx, run.ID, newPlan(), BuilderFrontendCommandRunnerFunc(func(ctx context.Context, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
			calls = append(calls, string(step)+":"+strings.Join(command, " "))
			require.Equal(t, BuilderFrontendExecutionStepInstall, step)
			require.NoError(t, appendLog("install interrupted\n"))
			cancel()
			return BuilderFrontendCommandResult{}, ctx.Err()
		}))

		require.Error(t, err)
		assert.Nil(t, result)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, []string{"install:npm ci"}, calls)

		var persistedEvents []entities.BuilderRunEvent
		require.NoError(t, db.DB.Where("run_id = ?", run.ID).Order("sequence ASC").Find(&persistedEvents).Error)
		require.Len(t, persistedEvents, 2)
		assert.Equal(t, []string{
			"[system] installing frontend dependencies\n",
			"install interrupted\n",
		}, []string{persistedEvents[0].Message, persistedEvents[1].Message})
	})
}
