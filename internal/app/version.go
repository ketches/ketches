package app

import (
	"encoding/json"
	"fmt"
)

// Version variables - set at compile time via -ldflags
var (
	// Version is the semantic version string (e.g., "v1.2.3" or "v1.2.3-rc.1")
	// When not a tag, it includes branch + commit + dirty flag
	Version = "dev"

	// Commit is the git commit SHA
	Commit = "unknown"

	// BuildTime is the build timestamp (ISO 8601 format)
	BuildTime = "unknown"

	// Tag is the git tag if this build is from a tagged commit
	Tag = ""
)

// VersionInfo returns a human-readable version string
func VersionInfo() string {
	return Version
}

// FullVersionInfo returns detailed version information as a JSON string
func FullVersionInfo() string {
	info := map[string]string{
		"version":   Version,
		"commit":    Commit,
		"buildTime": BuildTime,
		"tag":       Tag,
	}
	bytes, _ := json.Marshal(info)
	return string(bytes)
}

// PrintVersionBanner prints structured version metadata at startup.
func PrintVersionBanner() {
	fmt.Println("========================================")
	fmt.Printf("Ketches Version : %s\n", Version)
	fmt.Printf("Commit          : %s\n", Commit)
	fmt.Printf("Build Time      : %s\n", BuildTime)
	if Tag != "" {
		fmt.Printf("Tag             : %s\n", Tag)
	}
	fmt.Println("========================================")
}
