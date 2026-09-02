package app

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	LogLevel                        string
	Environment                     string
	Port                            string
	DBDriver                        string
	DBSource                        string
	DBHost                          string
	DBPort                          string
	DBName                          string
	DBUsername                      string
	DBPassword                      string
	DBSSLMode                       string
	DBTLSMode                       string
	DBAutoMigrate                   bool
	DBMaxIdleConns                  int
	DBMaxOpenConns                  int
	DBConnMaxLifetimeMinutes        int
	DBConnMaxIdleTimeMinutes        int
	JWTSecret                       string
	JWTIssuer                       string
	JWTAudience                     string
	AccessTokenTTLMinutes           int
	RefreshTokenTTLHours            int
	SecretEncryptionKey             string
	PreviousSecretEncryptionKeys    string
	BootstrapAdminUsername          string
	BootstrapAdminPassword          string
	SignUpEmailVerificationRequired bool
	CORSAllowedOrigins              string
	TrustedProxies                  []string
	SMTPHost                        string
	SMTPPort                        int
	SMTPUsername                    string
	SMTPPassword                    string
	SMTPFrom                        string
	BuildLogBaseDir                 string
	BuildLogRetentionDays           int
	EgressAllowedHosts              string
	AllowHostPathVolumes            bool
}

var Config AppConfig

var (
	ErrJWTSecretNotConfigured              = errors.New("JWT_SECRET must be configured")
	ErrSecretEncryptionKeyNotConfigured    = errors.New("SECRET_ENCRYPTION_KEY must be configured")
	ErrSecretEncryptionKeyTooShort         = errors.New("SECRET_ENCRYPTION_KEY must be at least 32 bytes")
	ErrBootstrapAdminPasswordNotConfigured = errors.New("BOOTSTRAP_ADMIN_PASSWORD must be configured")
	ErrBootstrapAdminPasswordTooShort      = errors.New("BOOTSTRAP_ADMIN_PASSWORD must be at least 12 characters")
	ErrDatabaseTLSRequired                 = errors.New("database TLS is required outside local development")
)

func InitConfig() {
	dbDriver := getEnv("DB_DRIVER", "postgres")
	dbHost := getEnv("DB_HOST", "")
	dbPort := getEnv("DB_PORT", "")
	dbName := getEnv("DB_NAME", "")
	dbUsername := getEnv("DB_USERNAME", "")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")
	dbTLSMode := strings.TrimSpace(getEnv("DB_TLS", ""))
	dbAutoMigrate := getEnvBool("DB_AUTO_MIGRATE", true)
	Config = AppConfig{
		LogLevel:                        getEnv("LOG_LEVEL", "info"),
		Environment:                     strings.ToLower(strings.TrimSpace(getEnv("APP_ENV", "development"))),
		Port:                            getEnv("PORT", "8080"),
		DBDriver:                        dbDriver,
		DBSource:                        buildDBSource(dbDriver, dbHost, dbPort, dbName, dbUsername, dbPassword, dbSSLMode, dbTLSMode),
		DBHost:                          dbHost,
		DBPort:                          dbPort,
		DBName:                          dbName,
		DBUsername:                      dbUsername,
		DBPassword:                      dbPassword,
		DBSSLMode:                       dbSSLMode,
		DBTLSMode:                       dbTLSMode,
		DBAutoMigrate:                   dbAutoMigrate,
		DBMaxIdleConns:                  getEnvInt("DB_MAX_IDLE_CONNS", 10),
		DBMaxOpenConns:                  getEnvInt("DB_MAX_OPEN_CONNS", 50),
		DBConnMaxLifetimeMinutes:        getEnvInt("DB_CONN_MAX_LIFETIME_MINUTES", 60),
		DBConnMaxIdleTimeMinutes:        getEnvInt("DB_CONN_MAX_IDLE_TIME_MINUTES", 30),
		JWTSecret:                       strings.TrimSpace(getEnv("JWT_SECRET", "")),
		JWTIssuer:                       fallbackString(getEnv("JWT_ISSUER", ""), "ketches"),
		JWTAudience:                     fallbackString(getEnv("JWT_AUDIENCE", ""), "ketches-ui"),
		AccessTokenTTLMinutes:           getEnvInt("ACCESS_TOKEN_TTL_MINUTES", 60),
		RefreshTokenTTLHours:            getEnvInt("REFRESH_TOKEN_TTL_HOURS", 24*7),
		SecretEncryptionKey:             strings.TrimSpace(getEnv("SECRET_ENCRYPTION_KEY", "")),
		PreviousSecretEncryptionKeys:    strings.TrimSpace(getEnv("PREVIOUS_SECRET_ENCRYPTION_KEYS", "")),
		BootstrapAdminUsername:          strings.TrimSpace(getEnv("BOOTSTRAP_ADMIN_USERNAME", "")),
		BootstrapAdminPassword:          getEnv("BOOTSTRAP_ADMIN_PASSWORD", ""),
		SignUpEmailVerificationRequired: getEnvBool("SIGN_UP_EMAIL_VERIFICATION_REQUIRED", true),
		CORSAllowedOrigins:              getEnv("CORS_ALLOWED_ORIGINS", ""),
		TrustedProxies:                  splitCommaSeparated(getEnv("TRUSTED_PROXIES", "")),
		SMTPHost:                        strings.TrimSpace(getEnv("SMTP_HOST", "")),
		SMTPPort:                        getEnvInt("SMTP_PORT", 587),
		SMTPUsername:                    strings.TrimSpace(getEnv("SMTP_USERNAME", "")),
		SMTPPassword:                    getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:                        strings.TrimSpace(getEnv("SMTP_FROM", "")),
		BuildLogBaseDir:                 fallbackString(getEnv("BUILD_LOG_BASE_DIR", ""), "data/build-logs"),
		BuildLogRetentionDays:           getEnvInt("BUILD_LOG_RETENTION_DAYS", 15),
		EgressAllowedHosts:              getEnv("EGRESS_ALLOWED_HOSTS", ""),
		AllowHostPathVolumes:            getEnvBool("ALLOW_HOSTPATH_VOLUMES", false),
	}
}

func ValidateRuntimeConfig() error {
	switch {
	case Config.JWTSecret == "":
		return ErrJWTSecretNotConfigured
	case Config.SecretEncryptionKey == "":
		return ErrSecretEncryptionKeyNotConfigured
	case len([]byte(strings.TrimSpace(Config.SecretEncryptionKey))) < 32:
		return ErrSecretEncryptionKeyTooShort
	case strings.TrimSpace(Config.BootstrapAdminPassword) == "":
		return ErrBootstrapAdminPasswordNotConfigured
	}

	if len(strings.TrimSpace(Config.BootstrapAdminPassword)) < 12 {
		return ErrBootstrapAdminPasswordTooShort
	}
	if err := ValidateDatabaseTLS(); err != nil {
		return err
	}
	if Config.AccessTokenTTLMinutes < 1 {
		Config.AccessTokenTTLMinutes = 60
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

func buildDBSource(driver, host, port, name, username, password, sslMode, tlsMode string) string {
	if source, ok := os.LookupEnv("DB_SOURCE"); ok && strings.TrimSpace(source) != "" {
		return source
	}

	switch driver {
	case "postgres", "postgresql":
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
		source := fmt.Sprintf(
			"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			resolvedUsername,
			password,
			resolvedHost,
			resolvedPort,
			resolvedName,
		)
		if strings.TrimSpace(tlsMode) != "" {
			source += "&tls=" + url.QueryEscape(strings.TrimSpace(tlsMode))
		}
		return source
	default:
		return ""
	}
}

func ValidateDatabaseTLS() error {
	if isLocalEnvironment(Config.Environment) {
		return nil
	}

	switch strings.ToLower(strings.TrimSpace(Config.DBDriver)) {
	case "postgres", "postgresql":
		mode := postgresSSLMode(Config.DBSource, Config.DBSSLMode)
		switch mode {
		case "require", "verify-ca", "verify-full":
			return nil
		default:
			return WrapErrorf(ErrDatabaseTLSRequired, "PostgreSQL sslmode=%q is not allowed in %s", mode, Config.Environment)
		}
	case "mysql":
		mode := mysqlTLSMode(Config.DBSource, Config.DBTLSMode)
		if mode == "" || mode == "false" || mode == "skip-verify" || mode == "preferred" {
			return WrapErrorf(ErrDatabaseTLSRequired, "MySQL tls=%q is not allowed in %s", mode, Config.Environment)
		}
	}
	return nil
}

func isLocalEnvironment(environment string) bool {
	switch strings.ToLower(strings.TrimSpace(environment)) {
	case "", "local", "development", "dev", "test":
		return true
	default:
		return false
	}
}

func postgresSSLMode(source, fallback string) string {
	trimmed := strings.TrimSpace(source)
	if parsed, err := url.Parse(trimmed); err == nil && (parsed.Scheme == "postgres" || parsed.Scheme == "postgresql") {
		if mode := strings.TrimSpace(parsed.Query().Get("sslmode")); mode != "" {
			return strings.ToLower(mode)
		}
	}
	for _, field := range strings.Fields(trimmed) {
		key, value, ok := strings.Cut(field, "=")
		if ok && strings.EqualFold(strings.TrimSpace(key), "sslmode") {
			return strings.ToLower(strings.Trim(strings.TrimSpace(value), "'\""))
		}
	}
	if trimmed != "" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(fallback))
}

func mysqlTLSMode(source, fallback string) string {
	if index := strings.Index(source, "?"); index >= 0 {
		if values, err := url.ParseQuery(source[index+1:]); err == nil {
			if mode := strings.TrimSpace(values.Get("tls")); mode != "" {
				return strings.ToLower(mode)
			}
		}
	}
	if strings.TrimSpace(source) != "" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(fallback))
}

func splitCommaSeparated(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
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
