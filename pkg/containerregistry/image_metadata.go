// Package containerregistry provides utilities for interacting with container image registries.
package containerregistry

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

// EnvVar represents a container environment variable with a key-value pair.
type EnvVar struct {
	Key   string
	Value string
}

// PortInfo represents an exposed container port with protocol.
type PortInfo struct {
	Port     int
	Protocol string // "TCP" or "UDP"
}

// HealthCheck represents the container's built-in health check configuration.
type HealthCheck struct {
	Test     []string
	Interval time.Duration
	Timeout  time.Duration
	Retries  int
}

// ImageMetadata holds parsed metadata extracted from a container image's config.
type ImageMetadata struct {
	Env          []EnvVar
	Volumes      []string // absolute mount paths
	ExposedPorts []PortInfo
	HealthCheck  *HealthCheck // nil if not defined
}

// FetchImageMetadata fetches and parses the metadata from a container image.
// If username/password are empty, anonymous access is attempted.
// Returns an error if the image reference is invalid or the registry is unreachable.
func FetchImageMetadata(ctx context.Context, image, username, password string) (*ImageMetadata, error) {
	ref, err := name.ParseReference(image)
	if err != nil {
		return nil, fmt.Errorf("invalid image reference %q: %w", image, err)
	}

	var authOpt remote.Option
	if username != "" {
		authOpt = remote.WithAuth(&authn.Basic{
			Username: username,
			Password: password,
		})
	} else {
		authOpt = remote.WithAuthFromKeychain(authn.DefaultKeychain)
	}

	img, err := remote.Image(ref, remote.WithContext(ctx), authOpt)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image %q: %w", image, err)
	}

	cfg, err := img.ConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to read image config for %q: %w", image, err)
	}

	meta := &ImageMetadata{}

	// Parse ENV vars: each entry is "KEY=VALUE" or "KEY"
	for _, e := range cfg.Config.Env {
		parts := strings.SplitN(e, "=", 2)
		ev := EnvVar{Key: parts[0]}
		if len(parts) == 2 {
			ev.Value = parts[1]
		}
		meta.Env = append(meta.Env, ev)
	}

	// Parse VOLUME declarations: map keys are absolute paths
	for path := range cfg.Config.Volumes {
		meta.Volumes = append(meta.Volumes, path)
	}

	// Parse EXPOSE declarations: format is "80/tcp", "53/udp", "8080" (default tcp)
	for portProto := range cfg.Config.ExposedPorts {
		parts := strings.SplitN(string(portProto), "/", 2)
		port, err := strconv.Atoi(parts[0])
		if err != nil {
			continue // skip malformed entries
		}
		proto := "TCP"
		if len(parts) == 2 && strings.ToUpper(parts[1]) == "UDP" {
			proto = "UDP"
		}
		meta.ExposedPorts = append(meta.ExposedPorts, PortInfo{Port: port, Protocol: proto})
	}

	// Parse HEALTHCHECK
	if hc := cfg.Config.Healthcheck; hc != nil && len(hc.Test) > 0 {
		meta.HealthCheck = &HealthCheck{
			Test:     hc.Test,
			Interval: hc.Interval,
			Timeout:  hc.Timeout,
			Retries:  hc.Retries,
		}
	}

	return meta, nil
}
