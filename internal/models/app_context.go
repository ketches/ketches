package models

import (
	"github.com/ketches/ketches/internal/db/entities"
)

// AppContext encapsulates an app and all its operational data for the core layer
type AppContext struct {
	App             entities.App
	EnvContext      EnvContext
	EnvVars         []entities.AppEnvVar
	Volumes         []entities.AppVolume
	Gateways        []entities.AppGateway
	GatewayRoutes   []entities.AppGatewayHTTPRoute
	GatewayBackends []entities.AppGatewayHTTPRouteBackend
	Probes          []entities.AppProbe
	ConfigFiles     []entities.AppConfigFile
	SchedulingRule  *entities.AppSchedulingRule
	AutoScaling     *entities.AppAutoScaling
	AppPlugins      []entities.AppPlugin
	Plugins         map[string]entities.Plugin
	// PodAccessPolicy is used by internal workloads that are not application
	// Pods (for example, a workspace). HTTP application operations use
	// the application identity labels instead.
	PodAccessPolicy *PodAccessPolicy
}

// PodAccessPolicy identifies an internal Pod that may be used by a scoped
// service operation. The managed label is always required in addition to the
// labels declared here.
type PodAccessPolicy struct {
	RequiredLabels map[string]string
}
