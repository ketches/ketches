package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type AppConfig struct {
	LogLevel              string
	Port                  string
	DBDriver              string
	DBSource              string
	DBHost                string
	DBPort                string
	DBName                string
	DBUsername            string
	DBPassword            string
	DBSSLMode             string
	DBAutoMigrate         bool
	JWTSecret             string
	CORSAllowedOrigins    string
	BuildLogBaseDir       string
	BuildLogRetentionDays int
}

var Config AppConfig

func InitConfig() {
	dbDriver := getEnv("DB_DRIVER", "sqlite")
	dbHost := getEnv("DB_HOST", "")
	dbPort := getEnv("DB_PORT", "")
	dbName := getEnv("DB_NAME", "")
	dbUsername := getEnv("DB_USERNAME", "")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbSSLMode := getEnv("DB_SSLMODE", "disable")
	dbAutoMigrate := getEnvBool("DB_AUTO_MIGRATE", false)

	Config = AppConfig{
		LogLevel:              getEnv("LOG_LEVEL", "info"),
		Port:                  getEnv("PORT", "8080"),
		DBDriver:              dbDriver,
		DBSource:              buildDBSource(dbDriver, dbHost, dbPort, dbName, dbUsername, dbPassword, dbSSLMode),
		DBHost:                dbHost,
		DBPort:                dbPort,
		DBName:                dbName,
		DBUsername:            dbUsername,
		DBPassword:            dbPassword,
		DBSSLMode:             dbSSLMode,
		DBAutoMigrate:         dbAutoMigrate,
		JWTSecret:             getEnv("JWT_SECRET", "ketches-secret-key"),
		CORSAllowedOrigins:    getEnv("CORS_ALLOWED_ORIGINS", ""),
		BuildLogBaseDir:       fallbackString(getEnv("BUILD_LOG_BASE_DIR", ""), "data/build-logs"),
		BuildLogRetentionDays: getEnvInt("BUILD_LOG_RETENTION_DAYS", 15),
	}
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
		return fallbackString(name, "ketches.db")
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
