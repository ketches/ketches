package entities

type App struct {
	UUIDBase
	Slug             string `json:"slug" gorm:"not null;uniqueIndex:idx_envID_slug;size:36"`        // Unique identifier for the app, typically a URL-friendly name
	DisplayName      string `json:"displayName" gorm:"not null;size:255"`                           // Human-readable name for the app
	Description      string `json:"description" gorm:"size:255"`                                    // Optional description of the app
	AppType          string `json:"appType" gorm:"not null;size:16"`                                // Type of app (e.g., 'Deployment', 'StatefulSet')
	Replicas         int32  `json:"replicas" gorm:"not null;default:1"`                             // Number of replicas for the app
	ContainerImage   string `json:"containerImage" gorm:"size:255"`                                 // Business image URL or path
	RegistryUsername string `json:"registryUsername" gorm:"size:64"`                                // Docker username for the app
	RegistryPassword string `json:"registryPassword" gorm:"size:255"`                               // Docker password for the app
	ContainerCommand string `json:"containerCommand" gorm:"type:text"`                              // Optional command to run in the container
	RequestCPU       int32  `json:"requestCPU" gorm:"not null;default:200"`                         // CPU request in milliCPU (e.g., 500 for 0.5 CPU, 1000 for 1 CPU)
	RequestMemory    int32  `json:"requestMemory" gorm:"not null;default:256"`                      // Memory request in MiB
	LimitCPU         int32  `json:"limitCPU" gorm:"not null;default:200"`                           // CPU limit in milliCPU (e.g., 1000 for 1 CPU, 2000 for 2 CPUs)
	LimitMemory      int32  `json:"limitMemory" gorm:"not null;default:256"`                        // Memory limit in MiB
	Edition          string `json:"edition" gorm:"size:64"`                                         // Edition of the app
	EnvID            string `json:"envID" gorm:"not null;uniqueIndex:idx_envID_slug;index;size:64"` // Env UUID this app belongs to
	EnvSlug          string `json:"envSlug" gorm:"not null;size:36"`                                // Env slug this app belongs to, typically a URL-friendly name
	ProjectID        string `json:"projectID" gorm:"not null;index;size:64"`                        // Project UUID this app belongs to
	ProjectSlug      string `json:"projectSlug" gorm:"not null;size:36"`                            // Project slug this app belongs to, typically a URL-friendly name
	ClusterID        string `json:"clusterID" gorm:"not null;index;size:64"`                        // Cluster UUID where this app is deployed
	ClusterSlug      string `json:"clusterSlug" gorm:"not null;size:36"`                            // Cluster slug where this app is deployed, typically a URL-friendly name
	ClusterNamespace string `json:"clusterNamespace" gorm:"not null;size:64"`                       // Cluster namespace where this app is deployed
	AuditBase
}

type AppEnvVar struct {
	UUIDBase
	AppID string `json:"appID" gorm:"not null;uniqueIndex:idx_appID_key;index;size:36"` // App UUID this environment variable belongs to
	Key   string `json:"key" gorm:"not null;uniqueIndex:idx_appID_key;size:64"`         // Key of the environment variable
	Value string `json:"value" gorm:"not null;size:255"`                                // Value of the environment variable
	AuditBase
}

type AppVolume struct {
	UUIDBase
	AppID        string `json:"appID" gorm:"not null;uniqueIndex:idx_appID_slug;uniqueIndex:idx_appID_mountPath_subPath;size:36"` // App UUID this volume belongs to
	Slug         string `json:"slug" gorm:"not null;uniqueIndex:idx_appID_slug;size:64"`                                          // Volume slug
	MountPath    string `json:"mountPath" gorm:"not null;uniqueIndex:idx_appID_mountPath_subPath;size:255"`                       // Mount path in container
	SubPath      string `json:"subPath" gorm:"uniqueIndex:idx_appID_mountPath_subPath;size:255"`                                  // Optional subPath for the volume
	VolumeMode   string `json:"volumeMode" gorm:"not null;size:16;default:Filesystem"`                                            // Volume mode (e.g. Filesystem, Block)
	Capacity     int    `json:"capacity" gorm:"not null"`                                                                         // Capacity, in MiB
	VolumeType   string `json:"volumeType" gorm:"not null;size:32"`                                                               // Type (e.g. emptyDir, pvc, hostPath)
	AccessModes  string `json:"accessModes" gorm:"not null;size:255"`                                                             // Access modes, semicolon separated (e.g. "ReadWriteOnce;ReadOnlyMany")
	StorageClass string `json:"storageClass" gorm:"size:64"`                                                                      // StorageClass for PVC
	AuditBase
}

type AppGateway struct {
	UUIDBase
	AppID       string `json:"appID" gorm:"not null;uniqueIndex:idx_appID_domain_path;uniqueIndex:idx_appID_gatewayPort;index;size:36"`
	Port        int32  `json:"port" gorm:"not null"`
	Protocol    string `json:"protocol" gorm:"not null;size:16"`
	Domain      string `json:"domain" gorm:"not null;uniqueIndex:idx_appID_domain_path;size:255"`
	Path        string `json:"path" gorm:"not null;uniqueIndex:idx_appID_domain_path;size:255"`
	CertID      string `json:"certID" gorm:"size:36"`
	GatewayPort int32  `json:"gatewayPort" gorm:"not null;uniqueIndex:idx_appID_gatewayPort;default:80"` // Port on the gateway to expose this app
	Exposed     bool   `json:"exposed" gorm:"not null;default:false"`
	EnvID       string `json:"envID" gorm:"not null;index;size:36"`     // Env UUID this gateway belongs to
	ProjectID   string `json:"projectID" gorm:"not null;index;size:36"` // Project UUID this gateway belongs to
	AuditBase
}

type AppProbe struct {
	UUIDBase
	AppID               string `json:"appID" gorm:"not null;uniqueIndex:idx_appID_type;index;size:36"`
	Type                string `json:"type" gorm:"not null;uniqueIndex:idx_appID_type;size:20"` // liveness, readiness, startup
	Enabled             bool   `json:"enabled" gorm:"not null;default:true"`
	InitialDelaySeconds int32  `json:"initialDelaySeconds" gorm:"not null;default:30"`
	PeriodSeconds       int32  `json:"periodSeconds" gorm:"not null;default:10"`
	TimeoutSeconds      int32  `json:"timeoutSeconds" gorm:"not null;default:5"`
	SuccessThreshold    int32  `json:"successThreshold" gorm:"not null;default:1"`
	FailureThreshold    int32  `json:"failureThreshold" gorm:"not null;default:3"`
	ProbeMode           string `json:"probeMode" gorm:"not null;size:16"` // httpGet, tcpSocket, exec
	HTTPGetPath         string `json:"httpGetPath" gorm:"size:255"`
	HTTPGetPort         int    `json:"httpGetPort" gorm:"default:0"`
	TCPSocketPort       int    `json:"tcpSocketPort" gorm:"default:0"`
	ExecCommand         string `json:"execCommand" gorm:"type:text"`
	AuditBase
}
