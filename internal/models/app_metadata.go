package models

import "time"

// AppMetadata represents the application metadata for import/export.
type AppMetadata struct {
	AppName          string               `json:"app_name"`
	AppSlug          string               `json:"app_slug"`
	AppType          string               `json:"app_type"` // "Deployment" or "StatefulSet"
	Description      string               `json:"description"`
	ContainerImage   string               `json:"container_image"`
	ContainerCommand string               `json:"container_command,omitempty"`
	Replicas         int                  `json:"replicas"`
	RequestCPU       int                  `json:"request_cpu"`
	RequestMemory    int                  `json:"request_memory"`
	LimitCPU         int                  `json:"limit_cpu"`
	LimitMemory      int                  `json:"limit_memory"`
	RegistryUsername string               `json:"registry_username,omitempty"`
	RegistryPassword string               `json:"-"`
	EnvVars          []EnvVarMetadata     `json:"env_vars"`
	Volumes          []VolumeMetadata     `json:"volumes"`
	ConfigFiles      []ConfigFileMetadata `json:"config_files"`
	Gateways         []GatewayMetadata    `json:"gateways"`
	Probes           []ProbeMetadata      `json:"probes"`
	SchedulingRule   *SchedulingMetadata  `json:"scheduling_rule,omitempty"`
	AutoScaling      *AutoScalingMetadata `json:"auto_scaling,omitempty"`
}

// EnvVarMetadata represents environment variable metadata.
type EnvVarMetadata struct {
	Key      string `json:"key"`
	Value    string `json:"value,omitempty"`
	IsSecret bool   `json:"is_secret,omitempty"`
	HasValue bool   `json:"has_value,omitempty"`
}

// VolumeMetadata represents volume metadata.
type VolumeMetadata struct {
	Slug         string `json:"slug"`
	MountPath    string `json:"mount_path"`
	SubPath      string `json:"sub_path,omitempty"`
	HostPath     string `json:"host_path,omitempty"`
	VolumeType   string `json:"volume_type"`
	Capacity     int    `json:"capacity"`
	StorageClass string `json:"storage_class,omitempty"`
	VolumeMode   string `json:"volume_mode,omitempty"`
	AccessModes  string `json:"access_modes,omitempty"`
}

// ConfigFileMetadata represents configuration file metadata.
type ConfigFileMetadata struct {
	Slug      string `json:"slug"`
	MountPath string `json:"mount_path"`
	Content   string `json:"content,omitempty"`
	FileMode  string `json:"file_mode,omitempty"`
	IsSecret  bool   `json:"is_secret,omitempty"`
	HasValue  bool   `json:"has_value,omitempty"`
}

// GatewayMetadata represents gateway metadata.
type GatewayMetadata struct {
	Port        int                    `json:"port"`
	Protocol    string                 `json:"protocol"`
	GatewayPort int                    `json:"gateway_port,omitempty"`
	ServiceType string                 `json:"service_type,omitempty"`
	NodePort    int                    `json:"node_port,omitempty"`
	Routes      []GatewayRouteMetadata `json:"routes,omitempty"`
}

type GatewayRouteMetadata struct {
	Host               string                        `json:"host"`
	ListenerProtocol   string                        `json:"listener_protocol"`
	Path               string                        `json:"path,omitempty"`
	PathMatchType      string                        `json:"path_match_type,omitempty"`
	Enabled            bool                          `json:"enabled"`
	CertID             string                        `json:"cert_id,omitempty"`
	Matches            *GatewayRouteMatches          `json:"matches,omitempty"`
	Filters            *GatewayRouteFilters          `json:"filters,omitempty"`
	Timeouts           *GatewayRouteTimeouts         `json:"timeouts,omitempty"`
	Retry              *GatewayRouteRetry            `json:"retry,omitempty"`
	SessionPersistence *GatewaySessionPersistence    `json:"session_persistence,omitempty"`
	Extension          *GatewayRouteExtension        `json:"extension,omitempty"`
	Backends           []GatewayRouteBackendMetadata `json:"backends,omitempty"`
}

type GatewayRouteBackendMetadata struct {
	BackendAppSlug string `json:"backend_app_slug,omitempty"`
	BackendPort    int    `json:"backend_port"`
	Weight         int    `json:"weight"`
}

// ProbeMetadata represents probe metadata.
type ProbeMetadata struct {
	Type                string `json:"type"`
	ProbeMode           string `json:"probe_mode"`
	Enabled             bool   `json:"enabled"`
	HttpGetPath         string `json:"http_get_path,omitempty"`
	HttpGetPort         int    `json:"http_get_port,omitempty"`
	TcpSocketPort       int    `json:"tcp_socket_port,omitempty"`
	ExecCommand         string `json:"exec_command,omitempty"`
	InitialDelaySeconds int    `json:"initial_delay_seconds"`
	PeriodSeconds       int    `json:"period_seconds"`
	TimeoutSeconds      int    `json:"timeout_seconds"`
	SuccessThreshold    int    `json:"success_threshold"`
	FailureThreshold    int    `json:"failure_threshold"`
}

// SchedulingMetadata represents scheduling rule metadata.
type SchedulingMetadata struct {
	RuleType     string `json:"rule_type,omitempty"`
	NodeName     string `json:"node_name,omitempty"`
	NodeSelector string `json:"node_selector,omitempty"`
	NodeAffinity string `json:"node_affinity,omitempty"`
	Tolerations  string `json:"tolerations,omitempty"`
}

// AutoScalingMetadata represents auto-scaling metadata.
type AutoScalingMetadata struct {
	MinReplicas             int `json:"min_replicas"`
	MaxReplicas             int `json:"max_replicas"`
	TargetCPUUtilization    int `json:"target_cpu_utilization"`
	TargetMemoryUtilization int `json:"target_memory_utilization"`
}

// KetchesMetadataFile represents the root structure of the export file.
type KetchesMetadataFile struct {
	Version    string        `json:"version"`
	Type       string        `json:"type"`
	Apps       []AppMetadata `json:"apps"`
	ExportedAt time.Time     `json:"exported_at"`
	ExportedBy string        `json:"exported_by,omitempty"`
}

// ToCreateAppRequest converts AppMetadata to CreateAppRequest.
func (m *AppMetadata) ToCreateAppRequest() *CreateAppRequest {
	req := &CreateAppRequest{
		Slug:             m.AppSlug,
		Name:             m.AppName,
		Description:      m.Description,
		AppType:          m.AppType,
		ContainerImage:   m.ContainerImage,
		RegistryUsername: m.RegistryUsername,
		Replicas:         m.Replicas,
		ContainerCommand: m.ContainerCommand,
		RequestCPU:       m.RequestCPU,
		RequestMemory:    m.RequestMemory,
		LimitCPU:         m.LimitCPU,
		LimitMemory:      m.LimitMemory,
	}

	if m.AutoScaling != nil {
		req.AutoScaling = &AutoScalingSpec{
			MinReplicas:             m.AutoScaling.MinReplicas,
			MaxReplicas:             m.AutoScaling.MaxReplicas,
			TargetCPUUtilization:    m.AutoScaling.TargetCPUUtilization,
			TargetMemoryUtilization: m.AutoScaling.TargetMemoryUtilization,
		}
	}

	if m.SchedulingRule != nil {
		req.SchedulingRule = &SchedulingSpec{
			RuleType:     m.SchedulingRule.RuleType,
			NodeName:     m.SchedulingRule.NodeName,
			NodeSelector: m.SchedulingRule.NodeSelector,
			NodeAffinity: m.SchedulingRule.NodeAffinity,
			Tolerations:  m.SchedulingRule.Tolerations,
		}
	}

	if len(m.Probes) > 0 {
		req.Probes = make([]ProbeSpec, len(m.Probes))
		for i, p := range m.Probes {
			req.Probes[i] = ProbeSpec(p)
		}
	}

	if len(m.Gateways) > 0 {
		req.Gateways = make([]GatewaySpec, len(m.Gateways))
		for i, g := range m.Gateways {
			req.Gateways[i] = gatewayMetadataToSpec(g)
		}
	}

	return req
}

func gatewayMetadataToSpec(g GatewayMetadata) GatewaySpec {
	spec := GatewaySpec{
		Port:        g.Port,
		Protocol:    g.Protocol,
		GatewayPort: g.GatewayPort,
		ServiceType: g.ServiceType,
		NodePort:    g.NodePort,
	}
	if len(g.Routes) == 0 {
		return spec
	}

	spec.Routes = make([]GatewayRouteSpec, len(g.Routes))
	for i, route := range g.Routes {
		spec.Routes[i] = GatewayRouteSpec{
			Host:               route.Host,
			ListenerProtocol:   route.ListenerProtocol,
			Path:               route.Path,
			PathMatchType:      route.PathMatchType,
			Enabled:            route.Enabled,
			CertID:             route.CertID,
			Matches:            route.Matches,
			Filters:            route.Filters,
			Timeouts:           route.Timeouts,
			Retry:              route.Retry,
			SessionPersistence: route.SessionPersistence,
			Extension:          route.Extension,
		}
		if len(route.Backends) > 0 {
			spec.Routes[i].Backends = make([]GatewayRouteBackendSpec, len(route.Backends))
			for j, backend := range route.Backends {
				spec.Routes[i].Backends[j] = GatewayRouteBackendSpec{
					BackendAppSlug: backend.BackendAppSlug,
					BackendPort:    backend.BackendPort,
					Weight:         backend.Weight,
				}
			}
		}
	}
	return spec
}
