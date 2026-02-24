package app

import (
	"os"
)

type AppConfig struct {
	LogLevel           string
	Port               string
	DBDriver           string
	DBSource           string
	JWTSecret          string
	CORSAllowedOrigins string
}

var Config AppConfig

func InitConfig() {
	Config = AppConfig{
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		Port:               getEnv("PORT", "8080"),
		DBDriver:           getEnv("DB_DRIVER", "sqlite"),
		DBSource:           getEnv("DB_SOURCE", "ketches.db"),
		JWTSecret:          getEnv("JWT_SECRET", "ketches-secret-key"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", ""),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
