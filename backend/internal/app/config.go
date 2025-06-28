package app

import (
	"os"
)

// GetEnv returns the value of the environment variable named by the key.
// It returns defaultVal if the variable is not present.
func GetEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
