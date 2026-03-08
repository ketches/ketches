package models

import (
	"github.com/ketches/ketches/internal/db/entities"
)

// AppContext encapsulates an app and all its operational data for the core layer
type AppContext struct {
	App            entities.App
	EnvContext     EnvContext
	EnvVars        []entities.AppEnvVar
	Volumes        []entities.AppVolume
	Gateways       []entities.AppGateway
	Probes         []entities.AppProbe
	ConfigFiles    []entities.AppConfigFile
	SchedulingRule *entities.AppSchedulingRule
	AutoScaling    *entities.AppAutoScaling
	AppPlugins     []entities.AppPlugin
	Plugins        map[string]entities.Plugin
	BuildSetting   *entities.BuildSetting
}
