package core

import (
	"strings"
	"testing"
)

func TestBuildctlArgs_DefaultSinglePlatform(t *testing.T) {
	args, err := buildctlArgs(BuildkitOptions{
		DockerfilePath:   "apps/api/Dockerfile",
		BuildContext:     "apps/api",
		ImageDestination: "registry.example.com/demo/api:v1.2.3",
		Platforms:        "linux/amd64",
	})
	if err != nil {
		t.Fatalf("buildctlArgs returned error: %v", err)
	}

	expected := []string{
		"--addr=tcp://ketches-buildkitd.ketches-build.svc.cluster.local:1234",
		"build",
		"--progress=plain",
		"--frontend=dockerfile.v0",
		"--local=context=/workspace/apps/api",
		"--local=dockerfile=/workspace/apps/api",
		"--opt=filename=Dockerfile",
		"--opt=platform=linux/amd64",
		"--output=type=image,name=registry.example.com/demo/api:v1.2.3,push=true",
	}
	for _, item := range expected {
		if !containsArg(args, item) {
			t.Fatalf("expected %q in args, got %v", item, args)
		}
	}
}

func TestBuildctlArgs_MultiPlatformWithRegistryCache(t *testing.T) {
	args, err := buildctlArgs(BuildkitOptions{
		DockerfilePath:       "apps/api/Dockerfile.release",
		BuildContext:         "apps/api",
		ImageDestination:     "registry.example.com/demo/api:v1.2.3",
		Platforms:            "linux/amd64,linux/arm64",
		RegistryCacheEnabled: true,
		RegistryCacheRef:     "registry.example.com/demo/api:buildcache-setting-1",
		BuildArgs: []string{
			"APP_ENV=production",
			"COMMIT_SHA=abc123",
		},
	})
	if err != nil {
		t.Fatalf("buildctlArgs returned error: %v", err)
	}

	expected := []string{
		"--opt=platform=linux/amd64,linux/arm64",
		"--opt=filename=Dockerfile.release",
		"--opt=build-arg:APP_ENV=production",
		"--opt=build-arg:COMMIT_SHA=abc123",
		"--import-cache=type=registry,ref=registry.example.com/demo/api:buildcache-setting-1",
		"--export-cache=type=registry,ref=registry.example.com/demo/api:buildcache-setting-1,mode=max",
	}
	for _, item := range expected {
		if !containsArg(args, item) {
			t.Fatalf("expected %q in args, got %v", item, args)
		}
	}
}

func TestBuildctlArgs_DisabledRegistryCacheSkipsImportExport(t *testing.T) {
	args, err := buildctlArgs(BuildkitOptions{
		DockerfilePath:       "Dockerfile",
		BuildContext:         ".",
		ImageDestination:     "registry.example.com/demo/api:v1.2.3",
		Platforms:            "linux/amd64",
		RegistryCacheEnabled: false,
		RegistryCacheRef:     "registry.example.com/demo/api:buildcache-setting-1",
	})
	if err != nil {
		t.Fatalf("buildctlArgs returned error: %v", err)
	}

	for _, arg := range args {
		if strings.HasPrefix(arg, "--import-cache=") || strings.HasPrefix(arg, "--export-cache=") {
			t.Fatalf("expected no cache import/export args, got %v", args)
		}
	}
}

func TestBuildctlArgs_DoesNotInjectRuntimePlatformBuildArgs(t *testing.T) {
	args, err := buildctlArgs(BuildkitOptions{
		DockerfilePath:   "Dockerfile",
		BuildContext:     ".",
		ImageDestination: "registry.example.com/demo/api:v1.2.3",
		Platforms:        "linux/amd64",
		BuildArgs:        []string{"APP_ENV=production"},
	})
	if err != nil {
		t.Fatalf("buildctlArgs returned error: %v", err)
	}

	disallowed := []string{
		"BUILDPLATFORM=",
		"TARGETPLATFORM=",
		"TARGETOS=",
		"TARGETARCH=",
	}
	for _, arg := range args {
		for _, prefix := range disallowed {
			if strings.Contains(arg, prefix) {
				t.Fatalf("expected no implicit runtime platform build arg %q, got %v", prefix, args)
			}
		}
	}
}

func TestDefaultRegistryCacheRef_UsesBuildSettingID(t *testing.T) {
	ref := defaultRegistryCacheRef("registry.example.com/demo/api:v1.2.3", "setting-1")
	if ref != "registry.example.com/demo/api:buildcache-setting-1" {
		t.Fatalf("unexpected cache ref %q", ref)
	}
}
