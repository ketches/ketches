package app

import (
	"os"
)

// AppConfig holds the application configuration
// type AppConfig struct {
// 	Host      string
// 	Port      int32
// 	RunMode   string
// 	JWTSecret string
// }

// // DBConfig holds the database configuration
// type DBConfig struct {
// 	Type string
// 	DNS  string
// }

// // ConfigFromEnv returns config from environment variables
// func ConfigFromEnv() (AppConfig, DBConfig) {
// 	port, _ := strconv.Atoi(GetEnv("APP_PORT", "8080"))
// 	return AppConfig{
// 			Host:      GetEnv("APP_HOST", "0.0.0.0"),
// 			Port:      int32(port),
// 			RunMode:   GetEnv("APP_RUNMODE", "dev"),
// 			JWTSecret: GetEnv("APP_JWT_SECRET", "ketches"),
// 		}, DBConfig{
// 			Type: GetEnv("DB_TYPE", "sqlite"),
// 			DNS:  GetEnv("DB_DNS", "file:ketches.db?cache=shared&mode=rwc"),
// 		}
// }

// GetEnv returns the value of the environment variable named by the key.
// It returns defaultVal if the variable is not present.
func GetEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
