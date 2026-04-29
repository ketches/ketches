package core

import (
	"testing"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildPluginContainerUsesAppPluginOverridesAndResources(t *testing.T) {
	metadata := AppMetadata{
		AppContext: &models.AppContext{
			EnvVars: []entities.AppEnvVar{
				{Key: "APP_SHARED", Value: "shared"},
			},
		},
	}

	plugin := &entities.Plugin{
		Slug:            "log-sidecar",
		Image:           "busybox:1.36",
		ImagePullPolicy: "IfNotPresent",
		EnvVars:         `[{"key":"PLUGIN_MODE","value":"template"}]`,
	}
	appPlugin := &entities.AppPlugin{
		EnvVars:       `[{"key":"PLUGIN_MODE","value":"runtime"},{"key":"PLUGIN_LEVEL","value":"debug"}]`,
		RequestCPU:    150,
		RequestMemory: 64,
		LimitCPU:      300,
		LimitMemory:   128,
	}

	container := metadata.buildPluginContainer(plugin, appPlugin)

	require.Equal(t, int64(150), container.Resources.Requests.Cpu().MilliValue())
	require.Equal(t, "64Mi", container.Resources.Requests.Memory().String())
	require.Equal(t, int64(300), container.Resources.Limits.Cpu().MilliValue())
	require.Equal(t, "128Mi", container.Resources.Limits.Memory().String())

	envMap := make(map[string]string, len(container.Env))
	for _, envVar := range container.Env {
		envMap[envVar.Name] = envVar.Value
	}

	assert.Equal(t, "shared", envMap["APP_SHARED"])
	assert.Equal(t, "runtime", envMap["PLUGIN_MODE"])
	assert.Equal(t, "debug", envMap["PLUGIN_LEVEL"])
}

func TestBuildPluginContainerFallsBackToPluginTemplateEnvVars(t *testing.T) {
	metadata := AppMetadata{
		AppContext: &models.AppContext{},
	}

	plugin := &entities.Plugin{
		Slug:    "init-sidecar",
		Image:   "busybox:1.36",
		EnvVars: `[{"key":"PLUGIN_MODE","value":"template"}]`,
	}
	appPlugin := &entities.AppPlugin{
		EnvVars: "null",
	}

	container := metadata.buildPluginContainer(plugin, appPlugin)

	require.Len(t, container.Env, 1)
	assert.Equal(t, "PLUGIN_MODE", container.Env[0].Name)
	assert.Equal(t, "template", container.Env[0].Value)
}
