package core

import (
	"fmt"
	"path"
	"strings"

	"github.com/ketches/ketches/internal/app"
)

const buildkitServiceAddress = "tcp://ketches-buildkitd.ketches-build.svc.cluster.local:1234"

type BuildkitOptions struct {
	DockerfilePath       string
	BuildContext         string
	ImageDestination     string
	Platforms            string
	RegistryCacheEnabled bool
	RegistryCacheRef     string
	BuildArgs            []string
}

func defaultRegistryCacheRef(imageDestination, buildSettingID string) string {
	imageDestination = sanitizeImageReference(strings.TrimSpace(imageDestination))
	if imageDestination == "" || strings.TrimSpace(buildSettingID) == "" {
		return ""
	}

	if idx := strings.Index(imageDestination, "@"); idx >= 0 {
		imageDestination = imageDestination[:idx]
	}

	lastSlash := strings.LastIndex(imageDestination, "/")
	lastColon := strings.LastIndex(imageDestination, ":")
	if lastColon > lastSlash {
		imageDestination = imageDestination[:lastColon]
	}

	return fmt.Sprintf("%s:buildcache-%s", imageDestination, strings.TrimSpace(buildSettingID))
}

func resolveDockerfileLocalPaths(buildContext, dockerfilePath string) (contextDir, dockerfileDir, filename string, err error) {
	contextDir = path.Clean(strings.TrimSpace(buildContext))
	if contextDir == "" {
		contextDir = "."
	}

	dockerfilePath = path.Clean(strings.TrimSpace(dockerfilePath))
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}

	if path.IsAbs(contextDir) || path.IsAbs(dockerfilePath) {
		return "", "", "", app.NewErrorf("absolute build paths are not supported")
	}

	filename = path.Base(dockerfilePath)
	dockerfileDir = path.Dir(dockerfilePath)
	if dockerfileDir == "." {
		dockerfileDir = contextDir
	}

	return contextDir, dockerfileDir, filename, nil
}

func buildctlArgs(opts BuildkitOptions) ([]string, error) {
	contextDir, dockerfileDir, filename, err := resolveDockerfileLocalPaths(opts.BuildContext, opts.DockerfilePath)
	if err != nil {
		return nil, err
	}

	platforms := strings.TrimSpace(opts.Platforms)
	if platforms == "" {
		platforms = "linux/amd64"
	}

	args := []string{
		fmt.Sprintf("--addr=%s", buildkitServiceAddress),
		"build",
		"--progress=plain",
		"--frontend=dockerfile.v0",
		fmt.Sprintf("--local=context=/workspace/%s", contextDir),
		fmt.Sprintf("--local=dockerfile=/workspace/%s", dockerfileDir),
		fmt.Sprintf("--opt=filename=%s", filename),
		fmt.Sprintf("--opt=platform=%s", platforms),
		fmt.Sprintf("--output=type=image,name=%s,push=true", strings.TrimSpace(opts.ImageDestination)),
	}

	for _, buildArg := range opts.BuildArgs {
		buildArg = strings.TrimSpace(buildArg)
		if buildArg == "" {
			continue
		}
		args = append(args, fmt.Sprintf("--opt=build-arg:%s", buildArg))
	}

	cacheRef := strings.TrimSpace(opts.RegistryCacheRef)
	if opts.RegistryCacheEnabled && cacheRef != "" {
		args = append(args,
			fmt.Sprintf("--import-cache=type=registry,ref=%s", cacheRef),
			fmt.Sprintf("--export-cache=type=registry,ref=%s,mode=max", cacheRef),
		)
	}

	return args, nil
}
