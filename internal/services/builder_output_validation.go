package services

import (
	"strings"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db/entities"
)

func ValidateBuilderExecutionOutputs(run *entities.BuilderRun, artifacts []entities.BuilderArtifact) error {
	if run == nil {
		return app.NewErrorf("builder run is required")
	}
	if len(artifacts) == 0 {
		return app.NewErrorf("builder output artifacts are required")
	}

	projectKind := strings.TrimSpace(stringPointerValue(run.PlannedProjectKind))
	switch projectKind {
	case "go_api_service":
		if hasBuilderArtifactPath(artifacts, "build/app") {
			return nil
		}
		return app.NewErrorf("go api validation failed: build/app artifact is required")
	case "python_api_service":
		if hasBuilderArtifactPath(artifacts, "build/app.py") || hasBuilderArtifactPath(artifacts, "build/main.py") {
			return nil
		}
		return app.NewErrorf("python api validation failed: build/app.py or build/main.py artifact is required")
	case "node_ssr_app":
		if hasBuilderArtifactPathPrefix(artifacts, ".next/") {
			return nil
		}
		return app.NewErrorf("node ssr validation failed: .next output is required")
	case "full_stack_app", "static_frontend_app", "":
		if hasBuilderArtifactPath(artifacts, "dist/index.html") || hasBuilderArtifactPath(artifacts, "build/index.html") {
			return nil
		}
		return app.NewErrorf("static frontend validation failed: index.html build output is required")
	default:
		return nil
	}
}

func hasBuilderArtifactPath(artifacts []entities.BuilderArtifact, expectedPath string) bool {
	for _, artifact := range artifacts {
		if artifact.Path == expectedPath {
			return true
		}
	}
	return false
}

func hasBuilderArtifactPathPrefix(artifacts []entities.BuilderArtifact, prefix string) bool {
	for _, artifact := range artifacts {
		if strings.HasPrefix(artifact.Path, prefix) {
			return true
		}
	}
	return false
}
