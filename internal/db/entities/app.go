package entities

import "time"

const DefaultAppPluginRequestCPU = 100
const DefaultAppPluginRequestMemory = 128
const DefaultAppPluginLimitCPU = 500
const DefaultAppPluginLimitMemory = 256

type App struct {
	Base
	Slug        string `gorm:"type:varchar(64);not null;uniqueIndex:idx_env_app_slug"`
	Name        string `gorm:"type:varchar(128);not null"`
	Description string `gorm:"type:text"`
	EnvID       string `gorm:"type:varchar(36);not null;uniqueIndex:idx_env_app_slug;index"`
	AppType     string `gorm:"type:varchar(32);default:'Deployment'"`

	ContainerImage   string `gorm:"type:varchar(256);not null"`
	ImagePullPolicy  string `gorm:"type:varchar(32);default:'IfNotPresent'"`
	ContainerCommand string `gorm:"type:text"`
	RegistryUsername string `gorm:"type:varchar(128)"`
	RegistryPassword string `gorm:"type:varchar(256)"`

	Replicas         int     `gorm:"type:int;default:1"`
	RequestCPU       int     `gorm:"type:int;default:100"`
	RequestMemory    int     `gorm:"type:int;default:128"`
	LimitCPU         int     `gorm:"type:int;default:1000"`
	LimitMemory      int     `gorm:"type:int;default:512"`
	DeployStatus     string  `gorm:"type:varchar(32);default:'undeployed'"`
	CodeRepositoryID *string `gorm:"type:varchar(36);index"`
}

type AppEnvVar struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	AppID     string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_app_env_key;index"`
	Key       string    `gorm:"type:varchar(256);not null;uniqueIndex:idx_app_env_key"`
	Value     string    `gorm:"type:text"`
}

type AppVolume struct {
	ID           string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
	AppID        string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_app_volume_slug;uniqueIndex:idx_app_volume_mount_path;index"`
	Slug         string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_app_volume_slug"`
	MountPath    string    `gorm:"type:varchar(256);not null;uniqueIndex:idx_app_volume_mount_path"`
	SubPath      string    `gorm:"type:varchar(256)"`
	VolumeType   string    `gorm:"type:varchar(32);not null"`
	Capacity     int       `gorm:"type:int;default:1"`
	StorageClass string    `gorm:"type:varchar(64)"`
	VolumeMode   string    `gorm:"type:varchar(16);default:'Filesystem'"`
	AccessModes  string    `gorm:"type:varchar(128);default:'ReadWriteOnce'"`
}

type AppGateway struct {
	ID          string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
	AppID       string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_app_gateway_port_proto;index"`
	Port        int       `gorm:"type:int;not null;uniqueIndex:idx_app_gateway_port_proto"`
	Protocol    string    `gorm:"type:varchar(16);not null;uniqueIndex:idx_app_gateway_port_proto"`
	GatewayPort int       `gorm:"type:int"`
	ServiceType string    `gorm:"type:varchar(16);default:'ClusterIP'"`
	NodePort    int       `gorm:"type:int"` // 0 = auto-assigned by K8s
}

type AppGatewayHTTPRoute struct {
	ID                     string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt              time.Time `gorm:"autoCreateTime"`
	UpdatedAt              time.Time `gorm:"autoUpdateTime"`
	AppGatewayID           string    `gorm:"type:varchar(36);not null;index;uniqueIndex:idx_gateway_route_unique,priority:1"`
	Host                   string    `gorm:"type:varchar(256);not null;uniqueIndex:idx_gateway_route_unique,priority:2"`
	ListenerProtocol       string    `gorm:"type:varchar(16);not null;uniqueIndex:idx_gateway_route_unique,priority:3"`
	PathMatchType          string    `gorm:"type:varchar(32);default:'PathPrefix';uniqueIndex:idx_gateway_route_unique,priority:4"`
	Path                   string    `gorm:"type:varchar(256);default:'/';uniqueIndex:idx_gateway_route_unique,priority:5"`
	Enabled                bool      `gorm:"type:bool;default:true"`
	CertID                 *string   `gorm:"type:varchar(36);index"`
	MatchesJSON            JSONBlob  `gorm:"type:json"`
	FiltersJSON            JSONBlob  `gorm:"type:json"`
	TimeoutsJSON           JSONBlob  `gorm:"type:json"`
	RetryJSON              JSONBlob  `gorm:"type:json"`
	SessionPersistenceJSON JSONBlob  `gorm:"type:json"`
	ExtensionJSON          JSONBlob  `gorm:"type:json"`
	SortOrder              int       `gorm:"type:int;default:0"`
}

type AppGatewayHTTPRouteBackend struct {
	ID           string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
	RouteID      string    `gorm:"type:varchar(36);not null;index"`
	BackendAppID string    `gorm:"type:varchar(36);not null;index"`
	BackendPort  int       `gorm:"type:int;not null"`
	Weight       int       `gorm:"type:int;default:1"`
}

type AppProbe struct {
	ID                  string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt           time.Time `gorm:"autoCreateTime"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime"`
	AppID               string    `gorm:"type:varchar(36);index;not null"`
	Type                string    `gorm:"type:varchar(16);not null"`
	ProbeMode           string    `gorm:"type:varchar(16);not null"`
	Enabled             bool      `gorm:"type:bool;default:false"`
	HttpGetPath         string    `gorm:"type:varchar(256)"`
	HttpGetPort         int       `gorm:"type:int"`
	TcpSocketPort       int       `gorm:"type:int"`
	ExecCommand         string    `gorm:"type:text"`
	InitialDelaySeconds int       `gorm:"type:int;default:0"`
	PeriodSeconds       int       `gorm:"type:int;default:10"`
	TimeoutSeconds      int       `gorm:"type:int;default:1"`
	SuccessThreshold    int       `gorm:"type:int;default:1"`
	FailureThreshold    int       `gorm:"type:int;default:3"`
}

type AppConfigFile struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	AppID     string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_app_config_file_slug;uniqueIndex:idx_app_config_file_mount_path;index"`
	Slug      string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_app_config_file_slug"`
	MountPath string    `gorm:"type:varchar(256);not null;uniqueIndex:idx_app_config_file_mount_path"`
	Content   string    `gorm:"type:text;not null"`
	FileMode  string    `gorm:"type:varchar(8);default:'0644'"`
}

type AppSchedulingRule struct {
	ID           string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
	AppID        string    `gorm:"type:varchar(36);uniqueIndex;not null"`
	RuleType     string    `gorm:"type:varchar(32)"`
	NodeName     string    `gorm:"type:varchar(256)"`
	NodeSelector string    `gorm:"type:text"`
	NodeAffinity string    `gorm:"type:text"`
	Tolerations  string    `gorm:"type:text"`
}

type AppAutoScaling struct {
	ID                      string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt               time.Time `gorm:"autoCreateTime"`
	UpdatedAt               time.Time `gorm:"autoUpdateTime"`
	AppID                   string    `gorm:"type:varchar(36);uniqueIndex;not null"`
	MinReplicas             int       `gorm:"type:int;default:1"`
	MaxReplicas             int       `gorm:"type:int;default:10"`
	TargetCPUUtilization    int       `gorm:"type:int"`
	TargetMemoryUtilization int       `gorm:"type:int"`
}

type AppPlugin struct {
	ID            string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime"`
	AppID         string    `gorm:"type:varchar(36);index;not null"`
	PluginID      string    `gorm:"type:varchar(36);index;not null"`
	Enabled       bool      `gorm:"type:bool;default:true"`
	EnvVars       string    `gorm:"type:text"`
	RequestCPU    int       `gorm:"type:int;default:100"`
	RequestMemory int       `gorm:"type:int;default:128"`
	LimitCPU      int       `gorm:"type:int;default:500"`
	LimitMemory   int       `gorm:"type:int;default:256"`
}

func NormalizeAppPluginResources(appPlugin AppPlugin) AppPlugin {
	if appPlugin.RequestCPU != 0 || appPlugin.RequestMemory != 0 || appPlugin.LimitCPU != 0 || appPlugin.LimitMemory != 0 {
		return appPlugin
	}

	appPlugin.RequestCPU = DefaultAppPluginRequestCPU
	appPlugin.RequestMemory = DefaultAppPluginRequestMemory
	appPlugin.LimitCPU = DefaultAppPluginLimitCPU
	appPlugin.LimitMemory = DefaultAppPluginLimitMemory

	return appPlugin
}
