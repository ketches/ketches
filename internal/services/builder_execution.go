package services

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
)

type BuilderFrontendPackageManager string

const (
	BuilderFrontendPackageManagerPNPM BuilderFrontendPackageManager = "pnpm"
	BuilderFrontendPackageManagerNPM  BuilderFrontendPackageManager = "npm"
	BuilderFrontendPackageManagerYarn BuilderFrontendPackageManager = "yarn"
)

type BuilderFrontendExecutionPlan struct {
	PackageManager BuilderFrontendPackageManager
	InstallCommand []string
	BuildCommand   []string
}

type BuilderFrontendExecutionStep string

const (
	BuilderFrontendExecutionStepInstall BuilderFrontendExecutionStep = "install"
	BuilderFrontendExecutionStepBuild   BuilderFrontendExecutionStep = "build"
)

type BuilderFrontendCommandResult struct {
	ExitCode int
}

type BuilderFrontendCommandRunner interface {
	RunBuilderFrontendCommand(ctx context.Context, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error)
}

type BuilderFrontendCommandRunnerFunc func(ctx context.Context, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error)

func (fn BuilderFrontendCommandRunnerFunc) RunBuilderFrontendCommand(ctx context.Context, step BuilderFrontendExecutionStep, command []string, appendLog func(string) error) (BuilderFrontendCommandResult, error) {
	if fn == nil {
		return BuilderFrontendCommandResult{}, errors.New("builder frontend command runner is required")
	}
	return fn(ctx, step, command, appendLog)
}

type BuilderFrontendExecutionResult struct {
	LastCompletedStep BuilderFrontendExecutionStep
}

type BuilderFrontendExecutionError struct {
	Step     BuilderFrontendExecutionStep
	ExitCode int
}

type builderFrontendExecutionEventWriter struct {
	appendLog    func(ctx context.Context, message string) error
	appendStatus func(ctx context.Context, level entities.BuilderRunEventLevel, message string) error
}

func (err *BuilderFrontendExecutionError) Error() string {
	if err == nil {
		return "builder frontend execution failed"
	}
	return fmt.Sprintf("%s command exited with status %d", err.Step, err.ExitCode)
}

func (writer builderFrontendExecutionEventWriter) AppendLog(ctx context.Context, message string) error {
	if writer.appendLog == nil {
		return errors.New("builder frontend execution log writer is required")
	}
	return writer.appendLog(ctx, message)
}

func (writer builderFrontendExecutionEventWriter) AppendStatus(ctx context.Context, level entities.BuilderRunEventLevel, message string) error {
	if writer.appendStatus == nil {
		return errors.New("builder frontend execution status writer is required")
	}
	return writer.appendStatus(ctx, level, message)
}

type builderFrontendLockfilePlan struct {
	packageManager BuilderFrontendPackageManager
	lockfileName   string
	installCommand []string
	buildCommand   []string
}

var builderFrontendLockfilePlans = []builderFrontendLockfilePlan{
	{
		packageManager: BuilderFrontendPackageManagerPNPM,
		lockfileName:   "pnpm-lock.yaml",
		installCommand: []string{"pnpm", "install", "--frozen-lockfile"},
		buildCommand:   []string{"pnpm", "build"},
	},
	{
		packageManager: BuilderFrontendPackageManagerNPM,
		lockfileName:   "package-lock.json",
		installCommand: []string{"npm", "ci"},
		buildCommand:   []string{"npm", "run", "build"},
	},
	{
		packageManager: BuilderFrontendPackageManagerYarn,
		lockfileName:   "yarn.lock",
		installCommand: []string{"yarn", "install", "--frozen-lockfile"},
		buildCommand:   []string{"yarn", "build"},
	},
}

func ExecuteBuilderFrontendPlan(ctx context.Context, runID string, plan *BuilderFrontendExecutionPlan, runner BuilderFrontendCommandRunner) (*BuilderFrontendExecutionResult, error) {
	return executeBuilderFrontendPlanWithEventWriter(ctx, runID, plan, runner, builderFrontendExecutionEventWriter{
		appendLog: func(ctx context.Context, message string) error {
			_, err := AppendBuilderRunExecutionLogEvent(ctx, runID, nil, message)
			return err
		},
		appendStatus: func(ctx context.Context, level entities.BuilderRunEventLevel, message string) error {
			_, err := AppendBuilderRunExecutionStatusEvent(ctx, runID, nil, level, message)
			return err
		},
	})
}

func executeBuilderFrontendPlanWithEventWriter(ctx context.Context, runID string, plan *BuilderFrontendExecutionPlan, runner BuilderFrontendCommandRunner, eventWriter builderFrontendExecutionEventWriter) (*BuilderFrontendExecutionResult, error) {
	if runID == "" {
		return nil, errors.New("builder run id is required")
	}
	if plan == nil {
		return nil, errors.New("builder frontend execution plan is required")
	}
	if runner == nil {
		return nil, errors.New("builder frontend command runner is required")
	}

	result := &BuilderFrontendExecutionResult{}
	if err := executeBuilderFrontendStep(ctx, plan.InstallCommand, BuilderFrontendExecutionStepInstall, runner, eventWriter); err != nil {
		return nil, err
	}
	result.LastCompletedStep = BuilderFrontendExecutionStepInstall

	if err := executeBuilderFrontendStep(ctx, plan.BuildCommand, BuilderFrontendExecutionStepBuild, runner, eventWriter); err != nil {
		return nil, err
	}
	result.LastCompletedStep = BuilderFrontendExecutionStepBuild

	if err := eventWriter.AppendStatus(ctx, entities.BuilderRunEventLevelInfo, "[system] frontend build completed\n"); err != nil {
		return nil, err
	}

	return result, nil
}

func DetectBuilderFrontendExecutionPlan(listing *models.ListFilesResponse) (*BuilderFrontendExecutionPlan, error) {
	if listing == nil {
		return nil, errors.New("builder frontend workspace listing is required")
	}

	hasPackageJSON := false
	detectedPlans := make([]builderFrontendLockfilePlan, 0, 1)

	for _, file := range listing.Files {
		if file.Type != "" && file.Type != "file" {
			continue
		}

		switch file.Name {
		case "package.json":
			hasPackageJSON = true
		case "pnpm-lock.yaml", "package-lock.json", "yarn.lock":
			for _, candidate := range builderFrontendLockfilePlans {
				if candidate.lockfileName == file.Name {
					detectedPlans = append(detectedPlans, candidate)
					break
				}
			}
		}
	}

	if len(detectedPlans) > 1 {
		lockfiles := make([]string, 0, len(detectedPlans))
		for _, detectedPlan := range detectedPlans {
			lockfiles = append(lockfiles, detectedPlan.lockfileName)
		}
		sort.Strings(lockfiles)
		return nil, fmt.Errorf("multiple supported frontend lockfiles detected: %s", strings.Join(lockfiles, ", "))
	}

	if len(detectedPlans) == 0 {
		if hasPackageJSON {
			return nil, errors.New("supported frontend lockfile is required for builder frontend execution")
		}
		return nil, errors.New("supported frontend lockfile is required for builder frontend execution")
	}
	if !hasPackageJSON {
		return nil, errors.New("package.json is required for builder frontend execution")
	}

	detectedPlan := detectedPlans[0]
	return &BuilderFrontendExecutionPlan{
		PackageManager: detectedPlan.packageManager,
		InstallCommand: append([]string(nil), detectedPlan.installCommand...),
		BuildCommand:   append([]string(nil), detectedPlan.buildCommand...),
	}, nil
}

func executeBuilderFrontendStep(ctx context.Context, command []string, step BuilderFrontendExecutionStep, runner BuilderFrontendCommandRunner, eventWriter builderFrontendExecutionEventWriter) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if len(command) == 0 {
		return fmt.Errorf("%s command is required", step)
	}

	if err := eventWriter.AppendStatus(ctx, entities.BuilderRunEventLevelInfo, builderFrontendStepStartMessage(step)); err != nil {
		return err
	}

	commandCopy := append([]string(nil), command...)
	commandResult, err := runner.RunBuilderFrontendCommand(ctx, step, commandCopy, func(message string) error {
		if message == "" {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return eventWriter.AppendLog(ctx, message)
	})
	if err != nil {
		return err
	}
	if commandResult.ExitCode != 0 {
		if err := eventWriter.AppendStatus(ctx, entities.BuilderRunEventLevelError, builderFrontendStepFailureMessage(step, commandResult.ExitCode)); err != nil {
			return err
		}
		return &BuilderFrontendExecutionError{
			Step:     step,
			ExitCode: commandResult.ExitCode,
		}
	}

	return nil
}

func builderFrontendStepStartMessage(step BuilderFrontendExecutionStep) string {
	switch step {
	case BuilderFrontendExecutionStepInstall:
		return "[system] installing frontend dependencies\n"
	case BuilderFrontendExecutionStepBuild:
		return "[system] running frontend build\n"
	default:
		return fmt.Sprintf("[system] running %s command\n", step)
	}
}

func builderFrontendStepFailureMessage(step BuilderFrontendExecutionStep, exitCode int) string {
	return fmt.Sprintf("[system] %s command exited with status %d\n", step, exitCode)
}
