package services

import (
	"errors"
	"strings"

	"github.com/ketches/ketches/internal/db/entities"
)

func DetectBuilderRuntimeValidationCommand(run *entities.BuilderRun, artifacts []entities.BuilderArtifact) ([]string, error) {
	if run == nil {
		return nil, errors.New("builder run is required")
	}

	switch strings.TrimSpace(stringPointerValue(run.PlannedProjectKind)) {
	case "go_api_service":
		if !hasBuilderArtifactPath(artifacts, "build/app") {
			return nil, errors.New("go runtime validation requires build/app artifact")
		}
		return []string{
			"sh",
			"-lc",
			`PORT=18080 ./build/app >/tmp/builder-runtime.log 2>&1 & pid=$!; sleep 2; if kill -0 "$pid" 2>/dev/null; then kill "$pid"; wait "$pid" || true; exit 0; fi; wait "$pid"; status=$?; if [ -f /tmp/builder-runtime.log ]; then cat /tmp/builder-runtime.log; fi; exit ${status:-1}`,
		}, nil
	case "python_api_service":
		if !hasBuilderArtifactPath(artifacts, "build/app.py") && !hasBuilderArtifactPath(artifacts, "build/main.py") {
			return nil, errors.New("python runtime validation requires build/app.py or build/main.py artifact")
		}
		return []string{
			"sh",
			"-lc",
			`if [ -f build/app.py ]; then python -c "import importlib.util; path='build/app.py'; spec=importlib.util.spec_from_file_location('builder_app', path); mod=importlib.util.module_from_spec(spec); spec.loader.exec_module(mod)"; else python -c "import importlib.util; path='build/main.py'; spec=importlib.util.spec_from_file_location('builder_app', path); mod=importlib.util.module_from_spec(spec); spec.loader.exec_module(mod)"; fi`,
		}, nil
	case "node_ssr_app":
		if !hasBuilderArtifactPath(artifacts, ".next/standalone/server.js") {
			return nil, nil
		}
		return []string{
			"sh",
			"-lc",
			`PORT=18081 node .next/standalone/server.js >/tmp/builder-runtime.log 2>&1 & pid=$!; sleep 3; if kill -0 "$pid" 2>/dev/null; then kill "$pid"; wait "$pid" || true; exit 0; fi; wait "$pid"; status=$?; if [ -f /tmp/builder-runtime.log ]; then cat /tmp/builder-runtime.log; fi; exit ${status:-1}`,
		}, nil
	default:
		return nil, nil
	}
}
