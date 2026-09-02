package app

import (
	"errors"
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

func TestInitConfig_DefaultsTokenTTLs(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	t.Setenv("ACCESS_TOKEN_TTL_MINUTES", "")
	t.Setenv("REFRESH_TOKEN_TTL_HOURS", "")

	InitConfig()

	if Config.AccessTokenTTLMinutes != 60 {
		t.Fatalf("expected default access token ttl minutes %d, got %d", 60, Config.AccessTokenTTLMinutes)
	}
	if Config.RefreshTokenTTLHours != 24*7 {
		t.Fatalf("expected default refresh token ttl hours %d, got %d", 24*7, Config.RefreshTokenTTLHours)
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

func TestInitConfig_UsesSecretEncryptionKeyOverride(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	t.Setenv("SECRET_ENCRYPTION_KEY", "test-master-key")

	InitConfig()

	if Config.SecretEncryptionKey != "test-master-key" {
		t.Fatalf("expected secret encryption key override %q, got %q", "test-master-key", Config.SecretEncryptionKey)
	}
}

func TestBuildDBSourceUsesExplicitSource(t *testing.T) {
	t.Setenv("DB_SOURCE", "custom-source")

	result := buildDBSource("postgres", "db", "5432", "ketches", "user", "pass", "disable", "")

	if result != "custom-source" {
		t.Fatalf("expected explicit DB_SOURCE to win, got %q", result)
	}
}

func TestBuildDBSourceBuildsPostgresDSN(t *testing.T) {
	result := buildDBSource("postgres", "db", "5432", "ketches", "user", "pass", "require", "")
	expected := "host=db port=5432 user=user password=pass dbname=ketches sslmode=require"

	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestBuildDBSourceBuildsMySQLDSN(t *testing.T) {
	result := buildDBSource("mysql", "db", "3306", "ketches", "user", "pass", "", "")
	expected := "user:pass@tcp(db:3306)/ketches?charset=utf8mb4&parseTime=True&loc=Local"

	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestBuildDBSourceReturnsEmptyForUnsupportedDriver(t *testing.T) {
	result := buildDBSource("sqlite", "", "", "custom.db", "", "", "", "")

	if result != "" {
		t.Fatalf("expected unsupported driver to produce empty source, got %q", result)
	}
}

func TestValidateDatabaseTLSRejectsInsecurePostgresProductionDSN(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() { Config = originalConfig })

	Config = AppConfig{
		Environment: "production",
		DBDriver:    "postgres",
		DBSource:    "postgres://user:pass@db/ketches?sslmode=prefer",
		DBSSLMode:   "verify-full",
	}
	if err := ValidateDatabaseTLS(); !errors.Is(err, ErrDatabaseTLSRequired) {
		t.Fatalf("expected production PostgreSQL TLS error, got %v", err)
	}

	Config.DBSource = "host=db dbname=ketches sslmode=verify-full"
	if err := ValidateDatabaseTLS(); err != nil {
		t.Fatalf("expected verify-full PostgreSQL DSN to pass, got %v", err)
	}

	Config.DBSource = "host=db dbname=ketches"
	Config.DBSSLMode = "verify-full"
	if err := ValidateDatabaseTLS(); !errors.Is(err, ErrDatabaseTLSRequired) {
		t.Fatalf("expected explicit PostgreSQL DSN without sslmode to fail, got %v", err)
	}
}

func TestValidateDatabaseTLSRejectsInsecureMySQLProductionDSN(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() { Config = originalConfig })

	Config = AppConfig{
		Environment: "production",
		DBDriver:    "mysql",
		DBSource:    "user:pass@tcp(db:3306)/ketches?parseTime=true&tls=false",
	}
	if err := ValidateDatabaseTLS(); !errors.Is(err, ErrDatabaseTLSRequired) {
		t.Fatalf("expected production MySQL TLS error, got %v", err)
	}

	Config.DBSource = "user:pass@tcp(db:3306)/ketches?parseTime=true&tls=true"
	if err := ValidateDatabaseTLS(); err != nil {
		t.Fatalf("expected MySQL tls=true DSN to pass, got %v", err)
	}

	Config.DBSource = "user:pass@tcp(db:3306)/ketches?parseTime=true"
	Config.DBTLSMode = "true"
	if err := ValidateDatabaseTLS(); !errors.Is(err, ErrDatabaseTLSRequired) {
		t.Fatalf("expected explicit MySQL DSN without tls to fail, got %v", err)
	}
}

func TestValidateDatabaseTLSAllowsLocalDevelopmentWithoutTLS(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() { Config = originalConfig })

	Config = AppConfig{Environment: "development", DBDriver: "postgres", DBSSLMode: "disable"}
	if err := ValidateDatabaseTLS(); err != nil {
		t.Fatalf("expected local development without TLS to pass, got %v", err)
	}
}

func TestInitConfigReadsTrustedProxies(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() { Config = originalConfig })
	t.Setenv("TRUSTED_PROXIES", "10.0.0.0/8, 192.168.1.10")

	InitConfig()
	if len(Config.TrustedProxies) != 2 || Config.TrustedProxies[0] != "10.0.0.0/8" || Config.TrustedProxies[1] != "192.168.1.10" {
		t.Fatalf("unexpected trusted proxies: %#v", Config.TrustedProxies)
	}
}

func TestInitConfig_ReadsBootstrapAdminConfig(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	t.Setenv("BOOTSTRAP_ADMIN_USERNAME", "bootstrap-admin")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "bootstrap-password-123")

	InitConfig()

	if Config.BootstrapAdminUsername != "bootstrap-admin" {
		t.Fatalf("expected bootstrap admin username %q, got %q", "bootstrap-admin", Config.BootstrapAdminUsername)
	}
	if Config.BootstrapAdminPassword != "bootstrap-password-123" {
		t.Fatalf("expected bootstrap admin password override %q, got %q", "bootstrap-password-123", Config.BootstrapAdminPassword)
	}
}

func TestInitConfig_DefaultsSignUpEmailVerificationToTrue(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	unsetEnvForTest(t, "SIGN_UP_EMAIL_VERIFICATION_REQUIRED")

	InitConfig()

	if !Config.SignUpEmailVerificationRequired {
		t.Fatalf("expected sign-up email verification to default to true")
	}
}

func TestInitConfig_ReadsSignUpEmailVerificationOverride(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	t.Setenv("SIGN_UP_EMAIL_VERIFICATION_REQUIRED", "false")

	InitConfig()

	if Config.SignUpEmailVerificationRequired {
		t.Fatalf("expected sign-up email verification override to be false")
	}
}

func TestValidateRuntimeConfig_RejectsMissingSecrets(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	Config = AppConfig{}

	err := ValidateRuntimeConfig()
	if !errors.Is(err, ErrJWTSecretNotConfigured) {
		t.Fatalf("expected missing JWT secret error, got %v", err)
	}

	Config.JWTSecret = "0123456789abcdef0123456789abcdef"
	err = ValidateRuntimeConfig()
	if !errors.Is(err, ErrSecretEncryptionKeyNotConfigured) {
		t.Fatalf("expected missing secret encryption key error, got %v", err)
	}

	Config.SecretEncryptionKey = "fedcba9876543210fedcba9876543210"
	err = ValidateRuntimeConfig()
	if !errors.Is(err, ErrBootstrapAdminPasswordNotConfigured) {
		t.Fatalf("expected missing bootstrap admin password error, got %v", err)
	}
}

func TestValidateRuntimeConfigRejectsWeakSecretEncryptionKey(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	Config = AppConfig{
		JWTSecret:              "0123456789abcdef0123456789abcdef",
		SecretEncryptionKey:    "too-short",
		BootstrapAdminPassword: "bootstrap-password-123",
	}

	if err := ValidateRuntimeConfig(); !errors.Is(err, ErrSecretEncryptionKeyTooShort) {
		t.Fatalf("expected weak secret encryption key error, got %v", err)
	}
}

func TestValidateRuntimeConfigRejectsMissingBootstrapAdminPassword(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	Config = AppConfig{
		JWTSecret:           "0123456789abcdef0123456789abcdef",
		SecretEncryptionKey: "fedcba9876543210fedcba9876543210",
	}

	if err := ValidateRuntimeConfig(); !errors.Is(err, ErrBootstrapAdminPasswordNotConfigured) {
		t.Fatalf("expected missing bootstrap admin password error, got %v", err)
	}
}

func TestValidateRuntimeConfig_RejectsShortExplicitBootstrapPassword(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	Config = AppConfig{
		JWTSecret:              "0123456789abcdef0123456789abcdef",
		SecretEncryptionKey:    "fedcba9876543210fedcba9876543210",
		BootstrapAdminUsername: "bootstrap-admin",
		BootstrapAdminPassword: "short",
	}

	Config.BootstrapAdminPassword = "short"
	err := ValidateRuntimeConfig()
	if !errors.Is(err, ErrBootstrapAdminPasswordTooShort) {
		t.Fatalf("expected short bootstrap password error, got %v", err)
	}
}

func TestValidateRuntimeConfig_AllowsExplicitSecurityConfig(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() {
		Config = originalConfig
	})

	Config = AppConfig{
		JWTSecret:              "0123456789abcdef0123456789abcdef",
		SecretEncryptionKey:    "fedcba9876543210fedcba9876543210",
		BootstrapAdminUsername: "bootstrap-admin",
		BootstrapAdminPassword: "bootstrap-password-123",
	}

	if err := ValidateRuntimeConfig(); err != nil {
		t.Fatalf("expected valid runtime config, got %v", err)
	}
}

func TestInitConfigDefaultsDatabaseAutoMigrateToEnabled(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() { Config = originalConfig })
	unsetEnvForTest(t, "DB_AUTO_MIGRATE")

	InitConfig()

	if !Config.DBAutoMigrate {
		t.Fatal("expected database auto migration to be enabled by default")
	}
}

func TestInitConfigHonorsDisabledDatabaseAutoMigrate(t *testing.T) {
	originalConfig := Config
	t.Cleanup(func() { Config = originalConfig })
	t.Setenv("DB_AUTO_MIGRATE", "false")

	InitConfig()

	if Config.DBAutoMigrate {
		t.Fatal("expected DB_AUTO_MIGRATE=false to disable database auto migration")
	}
}
