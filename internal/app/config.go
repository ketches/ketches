package app

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	LogLevel                          string
	Port                              string
	DBDriver                          string
	DBSource                          string
	DBHost                            string
	DBPort                            string
	DBName                            string
	DBUsername                        string
	DBPassword                        string
	DBSSLMode                         string
	DBAutoMigrate                     bool
	DBMaxIdleConns                    int
	DBMaxOpenConns                    int
	DBConnMaxLifetimeMinutes          int
	DBConnMaxIdleTimeMinutes          int
	JWTSecret                         string
	JWTIssuer                         string
	JWTAudience                       string
	AccessTokenTTLMinutes             int
	RefreshTokenTTLHours              int
	SecretEncryptionKey               string
	BootstrapAdminUsername            string
	BootstrapAdminPassword            string
	CORSAllowedOrigins                string
	SMTPHost                          string
	SMTPPort                          int
	SMTPUsername                      string
	SMTPPassword                      string
	SMTPFrom                          string
	BuildLogBaseDir                   string
	BuildLogRetentionDays             int
	BuilderSnapshotBaseDir            string
	BuilderProviderRegistryJSON       string
	BuilderModelProfileRegistryJSON   string
	BuilderExecutorPolicyRegistryJSON string
	BuilderExecutionCatalogJSON       string
	BuilderDefaultProviderKey         string
	BuilderDefaultModelProfileKey     string
	BuilderDefaultExecutorPolicyKey   string
	BuilderAgentBaseURL               string
	BuilderAgentAPIKey                string
	BuilderAgentModel                 string
	BuilderWorkspaceImage             string
	BuilderWorkspaceRoot              string
	BuilderSessionTTLHours            int
}

var Config AppConfig

var (
	ErrJWTSecretNotConfigured           = errors.New("JWT_SECRET must be configured")
	ErrSecretEncryptionKeyNotConfigured = errors.New("SECRET_ENCRYPTION_KEY must be configured")
	ErrBootstrapAdminConfigIncomplete   = errors.New("BOOTSTRAP_ADMIN_USERNAME and BOOTSTRAP_ADMIN_PASSWORD must be configured together")
	ErrBootstrapAdminPasswordTooShort   = errors.New("BOOTSTRAP_ADMIN_PASSWORD must be at least 12 characters")
)

func InitConfig() {
	dbDriver := getEnv("DB_DRIVER", "postgres")
	dbHost := getEnv("DB_HOST", "")
	dbPort := getEnv("DB_PORT", "")
	dbName := getEnv("DB_NAME", "")
	dbUsername := getEnv("DB_USERNAME", "")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")
	dbAutoMigrate := getEnvBool("DB_AUTO_MIGRATE", false)

	Config = AppConfig{
		LogLevel:                          getEnv("LOG_LEVEL", "info"),
		Port:                              getEnv("PORT", "8080"),
		DBDriver:                          dbDriver,
		DBSource:                          buildDBSource(dbDriver, dbHost, dbPort, dbName, dbUsername, dbPassword, dbSSLMode),
		DBHost:                            dbHost,
		DBPort:                            dbPort,
		DBName:                            dbName,
		DBUsername:                        dbUsername,
		DBPassword:                        dbPassword,
		DBSSLMode:                         dbSSLMode,
		DBAutoMigrate:                     dbAutoMigrate,
		DBMaxIdleConns:                    getEnvInt("DB_MAX_IDLE_CONNS", 10),
		DBMaxOpenConns:                    getEnvInt("DB_MAX_OPEN_CONNS", 50),
		DBConnMaxLifetimeMinutes:          getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 60),
		DBConnMaxIdleTimeMinutes:          getEnvInt("DB_CONN_MAX_IDLE_TIME_MINUTES", 30),
		JWTSecret:                         strings.TrimSpace(getEnv("JWT_SECRET", "")),
		JWTIssuer:                         fallbackString(getEnv("JWT_ISSUER", ""), "ketches"),
		JWTAudience:                       fallbackString(getEnv("JWT_AUDIENCE", ""), "ketches-ui"),
		AccessTokenTTLMinutes:             getEnvInt("ACCESS_TOKEN_TTL_MINUTES", 15),
		RefreshTokenTTLHours:              getEnvInt("REFRESH_TOKEN_TTL_HOURS", 24*7),
		SecretEncryptionKey:               strings.TrimSpace(getEnv("SECRET_ENCRYPTION_KEY", "")),
		BootstrapAdminUsername:            strings.TrimSpace(getEnv("BOOTSTRAP_ADMIN_USERNAME", "")),
		BootstrapAdminPassword:            getEnv("BOOTSTRAP_ADMIN_PASSWORD", ""),
		CORSAllowedOrigins:                getEnv("CORS_ALLOWED_ORIGINS", ""),
		SMTPHost:                          strings.TrimSpace(getEnv("SMTP_HOST", "")),
		SMTPPort:                          getEnvInt("SMTP_PORT", 587),
		SMTPUsername:                      strings.TrimSpace(getEnv("SMTP_USERNAME", "")),
		SMTPPassword:                      getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:                          strings.TrimSpace(getEnv("SMTP_FROM", "")),
		BuildLogBaseDir:                   fallbackString(getEnv("BUILD_LOG_BASE_DIR", ""), "data/build-logs"),
		BuildLogRetentionDays:             getEnvInt("BUILD_LOG_RETENTION_DAYS", 15),
		BuilderSnapshotBaseDir:            fallbackString(getEnv("BUILDER_SNAPSHOT_BASE_DIR", ""), "data/builder-previews"),
		BuilderProviderRegistryJSON:       getEnv("BUILDER_PROVIDER_REGISTRY_JSON", ""),
		BuilderModelProfileRegistryJSON:   getEnv("BUILDER_MODEL_PROFILE_REGISTRY_JSON", ""),
		BuilderExecutorPolicyRegistryJSON: getEnv("BUILDER_EXECUTOR_POLICY_REGISTRY_JSON", ""),
		BuilderExecutionCatalogJSON:       getEnv("BUILDER_EXECUTION_CATALOG_JSON", ""),
		BuilderDefaultProviderKey:         fallbackString(getEnv("BUILDER_DEFAULT_PROVIDER_KEY", ""), "default"),
		BuilderDefaultModelProfileKey:     fallbackString(getEnv("BUILDER_DEFAULT_MODEL_PROFILE_KEY", ""), "builder-default"),
		BuilderDefaultExecutorPolicyKey:   fallbackString(getEnv("BUILDER_DEFAULT_EXECUTOR_POLICY_KEY", ""), "workspace-only"),
		BuilderAgentBaseURL:               getEnv("BUILDER_AGENT_BASE_URL", ""),
		BuilderAgentAPIKey:                getEnv("BUILDER_AGENT_API_KEY", ""),
		BuilderAgentModel:                 getEnv("BUILDER_AGENT_MODEL", ""),
		BuilderWorkspaceImage:             fallbackString(getEnv("BUILDER_WORKSPACE_IMAGE", ""), "node:22-bookworm"),
		BuilderWorkspaceRoot:              fallbackString(getEnv("BUILDER_WORKSPACE_ROOT", ""), "/workspace"),
		BuilderSessionTTLHours:            getEnvInt("BUILDER_SESSION_TTL_HOURS", 24),
	}
}

func ValidateRuntimeConfig() error {
	switch {
	case Config.JWTSecret == "":
		return ErrJWTSecretNotConfigured
	case Config.SecretEncryptionKey == "":
		return ErrSecretEncryptionKeyNotConfigured
	}

	hasBootstrapUsername := strings.TrimSpace(Config.BootstrapAdminUsername) != ""
	hasBootstrapPassword := strings.TrimSpace(Config.BootstrapAdminPassword) != ""

	if hasBootstrapUsername != hasBootstrapPassword {
		return ErrBootstrapAdminConfigIncomplete
	}
	if hasBootstrapPassword && len(strings.TrimSpace(Config.BootstrapAdminPassword)) < 12 {
		return ErrBootstrapAdminPasswordTooShort
	}
	if Config.AccessTokenTTLMinutes < 1 {
		Config.AccessTokenTTLMinutes = 15
	}
	if Config.RefreshTokenTTLHours < 1 {
		Config.RefreshTokenTTLHours = 24 * 7
	}
	if Config.DBMaxIdleConns < 1 {
		Config.DBMaxIdleConns = 10
	}
	if Config.DBMaxOpenConns < 1 {
		Config.DBMaxOpenConns = 50
	}
	if Config.DBConnMaxLifetimeMinutes < 1 {
		Config.DBConnMaxLifetimeMinutes = 60
	}
	if Config.DBConnMaxIdleTimeMinutes < 1 {
		Config.DBConnMaxIdleTimeMinutes = 30
	}

	return nil
}

func buildDBSource(driver, host, port, name, username, password, sslMode string) string {
	if source, ok := os.LookupEnv("DB_SOURCE"); ok && strings.TrimSpace(source) != "" {
		return source
	}

	switch driver {
	case "postgres":
		resolvedHost := fallbackString(host, "localhost")
		resolvedPort := fallbackString(port, "5432")
		resolvedName := fallbackString(name, "ketches")
		resolvedUsername := fallbackString(username, "postgres")
		resolvedSSLMode := fallbackString(sslMode, "disable")
		return fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			resolvedHost,
			resolvedPort,
			resolvedUsername,
			password,
			resolvedName,
			resolvedSSLMode,
		)
	case "mysql":
		resolvedHost := fallbackString(host, "localhost")
		resolvedPort := fallbackString(port, "3306")
		resolvedName := fallbackString(name, "ketches")
		resolvedUsername := fallbackString(username, "root")
		return fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			resolvedUsername,
			password,
			resolvedHost,
			resolvedPort,
			resolvedName,
		)
	default:
		return ""
	}
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := strconv.ParseBool(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}

	return parsed
}

func getEnvInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}

	return parsed
}
