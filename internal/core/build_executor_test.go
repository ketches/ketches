package core

import (
	"strings"
	"testing"

	"github.com/ketches/ketches/internal/db/entities"
)

func TestBuildKanikoArgsIncludesDefaultIgnorePaths(t *testing.T) {
	t.Setenv("KANIKO_EXTRA_IGNORE_PATHS", "")

	args := buildKanikoArgs("Dockerfile", ".", "registry.example.com/demo/app:latest", "", &entities.ContainerRegistry{})

	if !containsArg(args, "--ignore-path=/product_uuid") {
		t.Fatalf("expected default ignore path for /product_uuid, got %v", args)
	}
}

func TestBuildKanikoArgsIncludesConfiguredIgnorePathsWithoutDuplicates(t *testing.T) {
	t.Setenv("KANIKO_EXTRA_IGNORE_PATHS", "/product_uuid,/var/run/secrets/kubernetes.io\n/mnt/cache")

	args := buildKanikoArgs("Dockerfile", ".", "registry.example.com/demo/app:latest", "", &entities.ContainerRegistry{})

	if countArg(args, "--ignore-path=/product_uuid") != 1 {
		t.Fatalf("expected /product_uuid to be present once, got %v", args)
	}
	if !containsArg(args, "--ignore-path=/var/run/secrets/kubernetes.io") {
		t.Fatalf("expected configured ignore path for kubernetes secrets, got %v", args)
	}
	if !containsArg(args, "--ignore-path=/mnt/cache") {
		t.Fatalf("expected configured ignore path for /mnt/cache, got %v", args)
	}
}

func containsArg(args []string, target string) bool {
	return countArg(args, target) > 0
}

func countArg(args []string, target string) int {
	count := 0
	for _, arg := range args {
		if strings.TrimSpace(arg) == target {
			count++
		}
	}
	return count
}
