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

	t.Setenv("BUILDER_PROVIDER_REGISTRY_JSON", "")
	t.Setenv("BUILDER_MODEL_PROFILE_REGISTRY_JSON", "")
	t.Setenv("BUILDER_EXECUTOR_POLICY_REGISTRY_JSON", "")
	t.Setenv("BUILDER_DEFAULT_PROVIDER_KEY", "")
	t.Setenv("BUILDER_DEFAULT_MODEL_PROFILE_KEY", "")
	t.Setenv("BUILDER_DEFAULT_EXECUTOR_POLICY_KEY", "")
	t.Setenv("BUILDER_AGENT_BASE_URL", "")
	t.Setenv("BUILDER_AGENT_API_KEY", "")
	t.Setenv("BUILDER_AGENT_MODEL", "")
	t.Setenv("BUILDER_WORKSPACE_IMAGE", "")
	t.Setenv("BUILDER_WORKSPACE_ROOT", "")
	t.Setenv("BUILDER_SESSION_TTL_HOURS", "")

	InitConfig()

	if Config.BuilderProviderRegistryJSON != "" {
		t.Fatalf("expected default builder provider registry JSON %q, got %q", "", Config.BuilderProviderRegistryJSON)
	}
	if Config.BuilderModelProfileRegistryJSON != "" {
		t.Fatalf("expected default builder model profile registry JSON %q, got %q", "", Config.BuilderModelProfileRegistryJSON)
	}
	if Config.BuilderExecutorPolicyRegistryJSON != "" {
		t.Fatalf("expected default builder executor policy registry JSON %q, got %q", "", Config.BuilderExecutorPolicyRegistryJSON)
	}
	if Config.BuilderDefaultProviderKey != "default" {
		t.Fatalf("expected default builder provider key %q, got %q", "default", Config.BuilderDefaultProviderKey)
	}
	if Config.BuilderDefaultModelProfileKey != "builder-default" {
		t.Fatalf("expected default builder model profile key %q, got %q", "builder-default", Config.BuilderDefaultModelProfileKey)
	}
	if Config.BuilderDefaultExecutorPolicyKey != "workspace-only" {
		t.Fatalf("expected default builder executor policy key %q, got %q", "workspace-only", Config.BuilderDefaultExecutorPolicyKey)
	}
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

	t.Setenv("BUILDER_PROVIDER_REGISTRY_JSON", `[{"key":"default","base_url":"https://registry.example.com"}]`)
	t.Setenv("BUILDER_MODEL_PROFILE_REGISTRY_JSON", `[{"key":"builder-default","model":"gpt-5.4"}]`)
	t.Setenv("BUILDER_EXECUTOR_POLICY_REGISTRY_JSON", `[{"key":"workspace-only","executor_kind":"workspace_pod"}]`)
	t.Setenv("BUILDER_DEFAULT_PROVIDER_KEY", "openai-compatible-primary")
	t.Setenv("BUILDER_DEFAULT_MODEL_PROFILE_KEY", "builder-fast")
	t.Setenv("BUILDER_DEFAULT_EXECUTOR_POLICY_KEY", "workspace-plus-build")
	t.Setenv("BUILDER_AGENT_BASE_URL", "https://builder.example.com")
	t.Setenv("BUILDER_AGENT_API_KEY", "builder-secret")
	t.Setenv("BUILDER_AGENT_MODEL", "builder-model")
	t.Setenv("BUILDER_WORKSPACE_IMAGE", "ghcr.io/ketches/builder-workspace:latest")
	t.Setenv("BUILDER_WORKSPACE_ROOT", "/builder-workspace")
	t.Setenv("BUILDER_SESSION_TTL_HOURS", "72")

	InitConfig()

	if Config.BuilderProviderRegistryJSON != `[{"key":"default","base_url":"https://registry.example.com"}]` {
		t.Fatalf("expected builder provider registry JSON override %q, got %q", `[{"key":"default","base_url":"https://registry.example.com"}]`, Config.BuilderProviderRegistryJSON)
	}
	if Config.BuilderModelProfileRegistryJSON != `[{"key":"builder-default","model":"gpt-5.4"}]` {
		t.Fatalf("expected builder model profile registry JSON override %q, got %q", `[{"key":"builder-default","model":"gpt-5.4"}]`, Config.BuilderModelProfileRegistryJSON)
	}
	if Config.BuilderExecutorPolicyRegistryJSON != `[{"key":"workspace-only","executor_kind":"workspace_pod"}]` {
		t.Fatalf("expected builder executor policy registry JSON override %q, got %q", `[{"key":"workspace-only","executor_kind":"workspace_pod"}]`, Config.BuilderExecutorPolicyRegistryJSON)
	}
	if Config.BuilderDefaultProviderKey != "openai-compatible-primary" {
		t.Fatalf("expected builder default provider key override %q, got %q", "openai-compatible-primary", Config.BuilderDefaultProviderKey)
	}
	if Config.BuilderDefaultModelProfileKey != "builder-fast" {
		t.Fatalf("expected builder default model profile key override %q, got %q", "builder-fast", Config.BuilderDefaultModelProfileKey)
	}
	if Config.BuilderDefaultExecutorPolicyKey != "workspace-plus-build" {
		t.Fatalf("expected builder default executor policy key override %q, got %q", "workspace-plus-build", Config.BuilderDefaultExecutorPolicyKey)
	}
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
