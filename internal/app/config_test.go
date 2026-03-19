package app

import (
	"os"
	"testing"
)

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()

	value, exists := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unset %s: %v", key, err)
	}

	t.Cleanup(func() {
		var restoreErr error
		if exists {
			restoreErr = os.Setenv(key, value)
		} else {
			restoreErr = os.Unsetenv(key)
		}
		if restoreErr != nil {
			t.Fatalf("restore %s: %v", key, restoreErr)
		}
	})
}

func TestInitConfig_DefaultsToPostgresDriver(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	unsetEnvForTest(t, "DB_DRIVER")
	unsetEnvForTest(t, "DB_SOURCE")
	unsetEnvForTest(t, "DB_HOST")
	unsetEnvForTest(t, "DB_PORT")
	unsetEnvForTest(t, "DB_NAME")
	unsetEnvForTest(t, "DB_USERNAME")
	unsetEnvForTest(t, "DB_PASSWORD")
	unsetEnvForTest(t, "DB_SSLMODE")

	InitConfig()

	if Config.DBDriver != "postgres" {
		t.Fatalf("expected default db driver %q, got %q", "postgres", Config.DBDriver)
	}
	expectedSource := "host=localhost port=5432 user=postgres password= dbname=ketches sslmode=disable"
	if Config.DBSource != expectedSource {
		t.Fatalf("expected default db source %q, got %q", expectedSource, Config.DBSource)
	}
}

func TestInitConfig_DefaultsBuildLogSettings(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	t.Setenv("BUILD_LOG_BASE_DIR", "")
	t.Setenv("BUILD_LOG_RETENTION_DAYS", "")

	InitConfig()

	if Config.BuildLogBaseDir != "data/build-logs" {
		t.Fatalf("expected default build log base dir %q, got %q", "data/build-logs", Config.BuildLogBaseDir)
	}
	if Config.BuildLogRetentionDays != 15 {
		t.Fatalf("expected default build log retention days %d, got %d", 15, Config.BuildLogRetentionDays)
	}
}

func TestInitConfig_UsesBuildLogEnvOverrides(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	t.Setenv("BUILD_LOG_BASE_DIR", "/tmp/build-logs")
	t.Setenv("BUILD_LOG_RETENTION_DAYS", "30")

	InitConfig()

	if Config.BuildLogBaseDir != "/tmp/build-logs" {
		t.Fatalf("expected build log base dir override %q, got %q", "/tmp/build-logs", Config.BuildLogBaseDir)
	}
	if Config.BuildLogRetentionDays != 30 {
		t.Fatalf("expected build log retention days override %d, got %d", 30, Config.BuildLogRetentionDays)
	}
}

func TestInitConfig_DefaultsBuilderSettings(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	t.Setenv("BUILDER_AGENT_BASE_URL", "")
	t.Setenv("BUILDER_AGENT_API_KEY", "")
	t.Setenv("BUILDER_AGENT_MODEL", "")
	t.Setenv("BUILDER_WORKSPACE_IMAGE", "")
	t.Setenv("BUILDER_WORKSPACE_ROOT", "")
	t.Setenv("BUILDER_SESSION_TTL_HOURS", "")

	InitConfig()

	if Config.BuilderAgentBaseURL != "" {
		t.Fatalf("expected default builder agent base URL %q, got %q", "", Config.BuilderAgentBaseURL)
	}
	if Config.BuilderAgentAPIKey != "" {
		t.Fatalf("expected default builder agent API key %q, got %q", "", Config.BuilderAgentAPIKey)
	}
	if Config.BuilderAgentModel != "" {
		t.Fatalf("expected default builder agent model %q, got %q", "", Config.BuilderAgentModel)
	}
	if Config.BuilderWorkspaceImage != "" {
		t.Fatalf("expected default builder workspace image %q, got %q", "", Config.BuilderWorkspaceImage)
	}
	if Config.BuilderWorkspaceRoot != "/workspace" {
		t.Fatalf("expected default builder workspace root %q, got %q", "/workspace", Config.BuilderWorkspaceRoot)
	}
	if Config.BuilderSessionTTLHours != 24 {
		t.Fatalf("expected default builder session TTL hours %d, got %d", 24, Config.BuilderSessionTTLHours)
	}
}

func TestInitConfig_UsesBuilderEnvOverrides(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	t.Setenv("BUILDER_AGENT_BASE_URL", "https://builder.example.com")
	t.Setenv("BUILDER_AGENT_API_KEY", "builder-secret")
	t.Setenv("BUILDER_AGENT_MODEL", "builder-model")
	t.Setenv("BUILDER_WORKSPACE_IMAGE", "ghcr.io/ketches/builder-workspace:latest")
	t.Setenv("BUILDER_WORKSPACE_ROOT", "/builder-workspace")
	t.Setenv("BUILDER_SESSION_TTL_HOURS", "72")

	InitConfig()

	if Config.BuilderAgentBaseURL != "https://builder.example.com" {
		t.Fatalf("expected builder agent base URL override %q, got %q", "https://builder.example.com", Config.BuilderAgentBaseURL)
	}
	if Config.BuilderAgentAPIKey != "builder-secret" {
		t.Fatalf("expected builder agent API key override %q, got %q", "builder-secret", Config.BuilderAgentAPIKey)
	}
	if Config.BuilderAgentModel != "builder-model" {
		t.Fatalf("expected builder agent model override %q, got %q", "builder-model", Config.BuilderAgentModel)
	}
	if Config.BuilderWorkspaceImage != "ghcr.io/ketches/builder-workspace:latest" {
		t.Fatalf("expected builder workspace image override %q, got %q", "ghcr.io/ketches/builder-workspace:latest", Config.BuilderWorkspaceImage)
	}
	if Config.BuilderWorkspaceRoot != "/builder-workspace" {
		t.Fatalf("expected builder workspace root override %q, got %q", "/builder-workspace", Config.BuilderWorkspaceRoot)
	}
	if Config.BuilderSessionTTLHours != 72 {
		t.Fatalf("expected builder session TTL hours override %d, got %d", 72, Config.BuilderSessionTTLHours)
	}
}

func TestBuildDBSourceUsesExplicitSource(t *testing.T) {
	t.Setenv("DB_SOURCE", "custom-source")

	result := buildDBSource("postgres", "db", "5432", "ketches", "user", "pass", "disable")

	if result != "custom-source" {
		t.Fatalf("expected explicit DB_SOURCE to win, got %q", result)
	}
}

func TestBuildDBSourceBuildsPostgresDSN(t *testing.T) {
	result := buildDBSource("postgres", "db", "5432", "ketches", "user", "pass", "require")
	expected := "host=db port=5432 user=user password=pass dbname=ketches sslmode=require"

	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestBuildDBSourceBuildsMySQLDSN(t *testing.T) {
	result := buildDBSource("mysql", "db", "3306", "ketches", "user", "pass", "")
	expected := "user:pass@tcp(db:3306)/ketches?charset=utf8mb4&parseTime=True&loc=Local"

	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestBuildDBSourceReturnsEmptyForUnsupportedDriver(t *testing.T) {
	result := buildDBSource("sqlite", "", "", "custom.db", "", "", "")

	if result != "" {
		t.Fatalf("expected unsupported driver to produce empty source, got %q", result)
	}
}
