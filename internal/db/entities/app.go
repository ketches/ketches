package entities

import "time"

type App struct {
	Base
	Slug        string `gorm:"type:varchar(64);not null;uniqueIndex:idx_env_app_slug"`
	Name        string `gorm:"type:varchar(128);not null"`
	Description string `gorm:"type:text"`
	EnvID       string `gorm:"type:varchar(36);not null;uniqueIndex:idx_env_app_slug"`
	AppType     string `gorm:"type:varchar(32);default:'Deployment'"`

	ContainerImage   string `gorm:"type:varchar(256);not null"`
	ContainerCommand string `gorm:"type:text"`
	RegistryUsername string `gorm:"type:varchar(128)"`
	RegistryPassword string `gorm:"type:varchar(256)"`

	Replicas      int `gorm:"type:int;default:1"`
	RequestCPU    int `gorm:"type:int;default:100"`
	RequestMemory int `gorm:"type:int;default:128"`
	LimitCPU      int `gorm:"type:int;default:1000"`
	LimitMemory   int `gorm:"type:int;default:512"`

	Edition       string `gorm:"type:varchar(36)"`
	ActualEdition string `gorm:"type:varchar(36)"`

	DeployStatus string `gorm:"type:varchar(32);default:'undeployed'"`

	// CodeRepositoryID: when set, this app was deployed from this code repository (build & release)
	CodeRepositoryID *string `gorm:"type:varchar(36);index"`

	Env            Env                `gorm:"foreignKey:EnvID"`
	EnvVars        []AppEnvVar        `gorm:"foreignKey:AppID"`
	Volumes        []AppVolume        `gorm:"foreignKey:AppID"`
	Gateways       []AppGateway       `gorm:"foreignKey:AppID"`
	Probes         []AppProbe         `gorm:"foreignKey:AppID"`
	ConfigFiles    []AppConfigFile    `gorm:"foreignKey:AppID"`
	SchedulingRule *AppSchedulingRule `gorm:"foreignKey:AppID"`
	AutoScaling    *AppAutoScaling    `gorm:"foreignKey:AppID"`
	AppPlugins     []AppPlugin        `gorm:"foreignKey:AppID"`
	BuildConfig    *AppBuildConfig    `gorm:"foreignKey:AppID"`
	Builds         []Build            `gorm:"foreignKey:AppID;constraint:false"`
	CodeRepository *CodeRepository    `gorm:"foreignKey:CodeRepositoryID"`
}

type AppEnvVar struct {
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	AppID     string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_app_env_key"`
	Key       string    `gorm:"type:varchar(256);not null;uniqueIndex:idx_app_env_key"`
	Value     string    `gorm:"type:text"`
}

type AppVolume struct {
	ID           string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
	AppID        string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_app_volume_slug;uniqueIndex:idx_app_volume_mount_path"`
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
	AppID       string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_app_gateway_port_proto"`
	Port        int       `gorm:"type:int;not null;uniqueIndex:idx_app_gateway_port_proto"`
	Protocol    string    `gorm:"type:varchar(16);not null;uniqueIndex:idx_app_gateway_port_proto"`
	Domain      string    `gorm:"type:varchar(256)"`
	Path        string    `gorm:"type:varchar(256);default:'/'"`
	GatewayPort int       `gorm:"type:int"`
	Exposed     bool      `gorm:"type:bool;default:false"`
	CertID      string    `gorm:"type:varchar(36)"`
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
	AppID     string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_app_config_file_slug;uniqueIndex:idx_app_config_file_mount_path"`
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
	ID        string    `gorm:"type:varchar(36);primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	AppID     string    `gorm:"type:varchar(36);index;not null"`
	PluginID  string    `gorm:"type:varchar(36);index;not null"`
	Enabled   bool      `gorm:"type:bool;default:true"`
	EnvVars   string    `gorm:"type:text"`
	Plugin    Plugin    `gorm:"foreignKey:PluginID"`
	App       App       `gorm:"foreignKey:AppID"`
}
