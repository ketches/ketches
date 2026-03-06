package models

import (
	"github.com/ketches/ketches/internal/db/entities"
)

// AppContext encapsulates an app and all its operational data for the core layer
type AppContext struct {
	App            entities.App
	Env            entities.Env
	Cluster        entities.Cluster
	EnvVars        []entities.AppEnvVar
	Volumes        []entities.AppVolume
	Gateways       []entities.AppGateway
	Probes         []entities.AppProbe
	ConfigFiles    []entities.AppConfigFile
	SchedulingRule *entities.AppSchedulingRule
	AutoScaling    *entities.AppAutoScaling
	AppPlugins     []entities.AppPlugin
	BuildConfig    *entities.AppBuildConfig
}
