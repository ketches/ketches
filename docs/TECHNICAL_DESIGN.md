# Ketches 技术设计方案

**文档版本**: v1.0  
**更新日期**: 2026-01-27  
**项目名称**: ketches (后端) / ketches-ui (前端)

---

## 1. 技术栈概览

### 1.1 前端技术栈 (ketches-ui)

| 技术 | 版本 | 用途 |
| ---- | ---- | ---- |
| React | 18.x | UI 框架 |
| TypeScript | 5.x | 类型安全 |
| Vite | 5.x | 构建工具 |
| shadcn/ui | latest | 组件库 |
| Tailwind CSS | 3.x | 样式框架 |
| React Router | 6.x | 路由管理 |
| Zustand | 4.x | 状态管理 |
| TanStack Query | 5.x | 服务端状态管理 |
| Axios | 1.x | HTTP 客户端 |
| xterm.js | 5.x | 终端模拟 |
| React Hook Form | 7.x | 表单处理 |
| Zod | 3.x | 数据验证 |

### 1.2 后端技术栈 (ketches)

| 技术 | 版本 | 用途 |
| ---- | ---- | ---- |
| Go | 1.24+ | 编程语言 |
| Gin | 1.9+ | Web 框架 |
| GORM | 1.25+ | ORM 框架 |
| client-go | 0.29+ | Kubernetes 客户端 |
| JWT | - | 身份认证 |
| Swagger | - | API 文档 |
| WebSocket | - | 实时通信 |

### 1.3 数据库支持

| 数据库 | 驱动 | 连接字符串示例 |
| ------ | ---- | ------------- |
| PostgreSQL | pgx | `host=localhost port=5432 user=postgres password=<db-password> dbname=ketches sslmode=disable` |
| MySQL | mysql | `user:password@tcp(localhost:3306)/ketches?charset=utf8mb4&parseTime=True&loc=Local` |

---

## 2. 系统架构

### 2.1 整体架构图

```txt
┌─────────────────────────────────────────────────────────────────────────────┐
│                              用户浏览器                                       │
└─────────────────────────────────────────────────────────────────────────────┘
                                    │
                                    │ HTTPS
                                    ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Nginx / Ingress                                     │
│                        (反向代理 & 静态资源)                                   │
└─────────────────────────────────────────────────────────────────────────────┘
                    │                               │
                    │ /                             │ /api/*
                    ▼                               ▼
┌──────────────────────────────┐    ┌──────────────────────────────────────────┐
│      ketches-ui (前端)        │    │           ketches (后端 API)              │
│  ┌────────────────────────┐  │    │  ┌────────────────────────────────────┐  │
│  │    React Application   │  │    │  │         Gin HTTP Server            │  │
│  │  ┌──────────────────┐  │  │    │  │  ┌──────────────────────────────┐  │  │
│  │  │   Pages/Views    │  │  │    │  │  │       Routes (v1)            │  │  │
│  │  ├──────────────────┤  │  │    │  │  ├──────────────────────────────┤  │  │
│  │  │   Components     │  │  │    │  │  │       Middlewares            │  │  │
│  │  │   (shadcn/ui)    │  │  │    │  │  │  (Auth, CORS, Permission)    │  │  │
│  │  ├──────────────────┤  │  │    │  │  ├──────────────────────────────┤  │  │
│  │  │   State (Zustand)│  │  │    │  │  │       Handlers               │  │  │
│  │  ├──────────────────┤  │  │    │  │  ├──────────────────────────────┤  │  │
│  │  │   API Client     │  │  │    │  │  │       Services               │  │  │
│  │  │   (Axios/Query)  │  │  │    │  │  ├──────────────────────────────┤  │  │
│  │  └──────────────────┘  │  │    │  │  │       Core (K8s Logic)       │  │  │
│  └────────────────────────┘  │    │  │  └──────────────────────────────┘  │  │
└──────────────────────────────┘    │  └────────────────────────────────────┘  │
                                    └──────────────────────────────────────────┘
                                                       │
                           ┌───────────────────────────┼───────────────────────────┐
                           │                           │                           │
                           ▼                           ▼                           ▼
                ┌─────────────────┐         ┌─────────────────┐         ┌─────────────────┐
                │    Database     │         │  Kubernetes     │         │  Kubernetes     │
                │  (PostgreSQL/   │         │  Cluster 1      │         │  Cluster N      │
                │   MySQL)        │         │  (client-go)    │         │  (client-go)    │
                └─────────────────┘         └─────────────────┘         └─────────────────┘
```

### 2.2 后端分层架构

```txt
┌─────────────────────────────────────────────────────────────────┐
│                        Routes Layer                             │
│  定义 API 路由，绑定 Handler                                      │
├─────────────────────────────────────────────────────────────────┤
│                      Middlewares Layer                          │
│  Auth (JWT验证) | CORS | Permission (权限校验) | RequestID        │
├─────────────────────────────────────────────────────────────────┤
│                       Handlers Layer                            │
│  处理 HTTP 请求/响应，参数校验，调用 Service                         │
├─────────────────────────────────────────────────────────────────┤
│                       Services Layer                            │
│  业务逻辑实现，编排 DB 和 Kubernetes 操作                           │
├─────────────────────────────────────────────────────────────────┤
│                         Core Layer                              │
│  Kubernetes 资源构建（Deployment, Service, Gateway 等）           │
├─────────────────────────────────────────────────────────────────┤
│                      Kube Client Layer                          │
│  client-go 封装，集群客户端管理，资源 CRUD                          │
├─────────────────────────────────────────────────────────────────┤
│                          DB Layer                               │
│  GORM ORM，Entity 定义，数据库迁移                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. 后端设计 (ketches)

### 3.1 项目结构

```txt
ketches/
├── cmd/
│   └── api/
│       └── main.go                 # API 服务入口
├── internal/
│   ├── api/
│   │   ├── context.go              # 请求上下文
│   │   ├── request.go              # 请求参数
│   │   ├── response.go             # 响应封装
│   │   ├── session.go              # 会话管理
│   │   └── validator.go            # 参数验证
│   ├── app/
│   │   ├── app.go                  # 应用初始化
│   │   ├── config.go               # 配置管理
│   │   ├── consts.go               # 常量定义
│   │   ├── error.go                # 错误定义
│   │   └── jwt.go                  # JWT 工具
│   ├── core/
│   │   ├── app_metadata.go         # 应用元数据
│   │   ├── app_metadata_builder.go # 元数据构建器
│   │   ├── app_status.go           # 应用状态计算
│   │   ├── apply_resource.go       # K8s 资源应用
│   │   ├── delete_resource.go      # K8s 资源删除
│   │   └── cluster_extension.go    # 集群扩展
│   ├── db/
│   │   ├── db.go                   # 数据库连接
│   │   ├── migration.go            # 数据库迁移
│   │   ├── postgres.go             # PostgreSQL 驱动
│   │   ├── mysql.go                # MySQL 驱动
│   │   ├── entities/               # 数据库实体
│   │   │   ├── base.go
│   │   │   ├── user.go
│   │   │   ├── cluster.go
│   │   │   ├── project.go
│   │   │   ├── env.go
│   │   │   ├── app.go
│   │   │   ├── app_config_file.go
│   │   │   ├── app_scheduling_rule.go
│   │   │   ├── audit.go
│   │   │   └── cert.go
│   │   └── orm/                    # ORM 查询封装
│   │       ├── app.go
│   │       ├── app_gateway.go
│   │       ├── cluster.go
│   │       ├── env.go
│   │       └── project.go
│   ├── handlers/                   # HTTP 处理器
│   │   ├── app.go
│   │   ├── app_config_file.go
│   │   ├── app_env_var.go
│   │   ├── app_gateway.go
│   │   ├── app_probe.go
│   │   ├── app_scheduling_rule.go
│   │   ├── app_volume.go
│   │   ├── cluster.go
│   │   ├── env.go
│   │   ├── healthz.go
│   │   ├── platform.go
│   │   ├── project.go
│   │   ├── user.go
│   │   └── version.go
│   ├── kube/                       # Kubernetes 客户端
│   │   ├── apply.go
│   │   ├── cluster.go
│   │   ├── deployment.go
│   │   ├── namespace.go
│   │   ├── persistent_volume_claim.go
│   │   ├── pod.go
│   │   ├── pod_event_handler.go
│   │   ├── service.go
│   │   ├── statefulset.go
│   │   └── store.go
│   ├── middlewares/                # 中间件
│   │   ├── auth.go
│   │   ├── cors.go
│   │   ├── permission.go
│   │   └── request_id.go
│   ├── models/                     # API 请求/响应模型
│   │   ├── app.go
│   │   ├── app_config_file.go
│   │   ├── app_env_var.go
│   │   ├── app_gateway.go
│   │   ├── app_probe.go
│   │   ├── app_scheduling_rule.go
│   │   ├── app_volume.go
│   │   ├── cluster.go
│   │   ├── env.go
│   │   ├── platform.go
│   │   ├── project.go
│   │   └── user.go
│   ├── routes/                     # 路由定义
│   │   ├── root.go
│   │   └── v1.go
│   └── services/                   # 业务服务
│       ├── app.go
│       ├── app_config_file.go
│       ├── app_env_var.go
│       ├── app_gateway.go
│       ├── app_probe.go
│       ├── app_scheduling_rule.go
│       ├── app_volume.go
│       ├── cluster.go
│       ├── env.go
│       ├── platform.go
│       ├── project.go
│       ├── service.go
│       └── user.go
├── pkg/
│   ├── kube/                       # Kubernetes 工具包
│   │   ├── dynamiclister/
│   │   ├── incluster/
│   │   └── util.go
│   ├── utils/
│   │   ├── humanize_time.go
│   │   └── ptr.go
│   ├── uuid/
│   │   └── uuid.go
│   └── websocket/
│       ├── websocket_conn.go
│       ├── websocket_reader.go
│       └── websocket_writer.go
├── openapi/                        # Swagger 文档
│   ├── docs.go
│   ├── swagger.json
│   └── swagger.yaml
├── ui/                             # 前端代码
│   └── ...
├── .env                            # 环境变量
├── go.mod
├── go.sum
├── Makefile
└── Dockerfile
```

### 3.2 数据库实体设计

#### 3.2.1 基础实体

```go
// entities/base.go
package entities

import (
    "time"
    "gorm.io/gorm"
)

type Base struct {
    ID        string         `gorm:"type:varchar(36);primaryKey"`
    CreatedAt time.Time      `gorm:"autoCreateTime"`
    UpdatedAt time.Time      `gorm:"autoUpdateTime"`
    DeletedAt gorm.DeletedAt `gorm:"index"`
}
```

#### 3.2.2 用户实体

```go
// entities/user.go
package entities

type User struct {
    Base
    Username     string `gorm:"type:varchar(64);uniqueIndex;not null"`
    Email        string `gorm:"type:varchar(128);uniqueIndex;not null"`
    Password     string `gorm:"type:varchar(128);not null"`
    Fullname     string `gorm:"type:varchar(64)"`
    Phone        string `gorm:"type:varchar(32)"`
    Gender       int    `gorm:"type:int;default:0"` // 0: unknown, 1: male, 2: female
    Role         string `gorm:"type:varchar(16);default:'user'"` // admin, user
    RefreshToken string `gorm:"type:text"`
}
```

#### 3.2.3 集群实体

```go
// entities/cluster.go
package entities

type Cluster struct {
    Base
    Slug        string `gorm:"type:varchar(64);uniqueIndex;not null"`
    Name string `gorm:"type:varchar(128);not null"`
    Description string `gorm:"type:text"`
    KubeConfig  string `gorm:"type:text;not null"` // 加密存储
    GatewayHost   string `gorm:"type:varchar(64)"`
    Enabled     bool   `gorm:"type:bool;default:true"`
}
```

#### 3.2.4 项目实体

```go
// entities/project.go
package entities

type Project struct {
    Base
    Slug        string          `gorm:"type:varchar(64);uniqueIndex;not null"`
    Name string          `gorm:"type:varchar(128);not null"`
    Description string          `gorm:"type:text"`
    Members     []ProjectMember `gorm:"foreignKey:ProjectID"`
    Envs        []Env           `gorm:"foreignKey:ProjectID"`
}

type ProjectMember struct {
    Base
    ProjectID   string `gorm:"type:varchar(36);index;not null"`
    UserID      string `gorm:"type:varchar(36);index;not null"`
    ProjectRole string `gorm:"type:varchar(16);not null"` // owner, developer, viewer
    User        User   `gorm:"foreignKey:UserID"`
}
```

#### 3.2.5 环境实体

```go
// entities/env.go
package entities

type Env struct {
    Base
    Slug             string `gorm:"type:varchar(64);not null"`
    Name      string `gorm:"type:varchar(128);not null"`
    Description      string `gorm:"type:text"`
    ProjectID        string `gorm:"type:varchar(36);index;not null"`
    ClusterID        string `gorm:"type:varchar(36);index;not null"`
    ClusterNamespace string `gorm:"type:varchar(64);not null"` // K8s Namespace
    Project          Project `gorm:"foreignKey:ProjectID"`
    Cluster          Cluster `gorm:"foreignKey:ClusterID"`
    Apps             []App   `gorm:"foreignKey:EnvID"`
}
```

#### 3.2.6 应用实体

```go
// entities/app.go
package entities

type App struct {
    Base
    Slug             string `gorm:"type:varchar(64);not null"`
    Name      string `gorm:"type:varchar(128);not null"`
    Description      string `gorm:"type:text"`
    EnvID            string `gorm:"type:varchar(36);index;not null"`
    AppType          string `gorm:"type:varchar(32);default:'Deployment'"` // Deployment, StatefulSet
    
    // Container
    ContainerImage   string `gorm:"type:varchar(256);not null"`
    ContainerCommand string `gorm:"type:text"`
    RegistryUsername string `gorm:"type:varchar(128)"`
    RegistryPassword string `gorm:"type:varchar(256)"` // 加密存储
    
    // Resources
    Replicas      int `gorm:"type:int;default:1"`
    RequestCPU    int `gorm:"type:int;default:100"`    // milliCPU
    RequestMemory int `gorm:"type:int;default:128"`    // MiB
    LimitCPU      int `gorm:"type:int;default:1000"`   // milliCPU
    LimitMemory   int `gorm:"type:int;default:512"`    // MiB
    
    // Version
    Edition       string `gorm:"type:varchar(36)"`
    ActualEdition string `gorm:"type:varchar(36)"`
    
    // Relations
    Env            Env                 `gorm:"foreignKey:EnvID"`
    EnvVars        []AppEnvVar         `gorm:"foreignKey:AppID"`
    Volumes        []AppVolume         `gorm:"foreignKey:AppID"`
    Gateways       []AppGateway        `gorm:"foreignKey:AppID"`
    Probes         []AppProbe          `gorm:"foreignKey:AppID"`
    ConfigFiles    []AppConfigFile     `gorm:"foreignKey:AppID"`
    SchedulingRule *AppSchedulingRule  `gorm:"foreignKey:AppID"`
}

type AppEnvVar struct {
    Base
    AppID string `gorm:"type:varchar(36);index;not null"`
    Key   string `gorm:"type:varchar(256);not null"`
    Value string `gorm:"type:text"`
}

type AppVolume struct {
    Base
    AppID        string `gorm:"type:varchar(36);index;not null"`
    Slug         string `gorm:"type:varchar(64);not null"`
    MountPath    string `gorm:"type:varchar(256);not null"`
    SubPath      string `gorm:"type:varchar(256)"`
    VolumeType   string `gorm:"type:varchar(32);not null"` // pvc, emptyDir, hostPath
    Capacity     int    `gorm:"type:int;default:1"`        // GiB
    StorageClass string `gorm:"type:varchar(64)"`
    VolumeMode   string `gorm:"type:varchar(16);default:'Filesystem'"` // Filesystem, Block
    AccessModes  string `gorm:"type:varchar(128);default:'ReadWriteOnce'"` // JSON array
}

type AppGateway struct {
    Base
    AppID       string `gorm:"type:varchar(36);index;not null"`
    Port        int    `gorm:"type:int;not null"`
    Protocol    string `gorm:"type:varchar(16);not null"` // http, https, tcp, udp
    Domain      string `gorm:"type:varchar(256)"`
    Path        string `gorm:"type:varchar(256);default:'/'"`
    GatewayPort int    `gorm:"type:int"`
    Exposed     bool   `gorm:"type:bool;default:false"`
    CertID      string `gorm:"type:varchar(36)"`
}

type AppProbe struct {
    Base
    AppID               string `gorm:"type:varchar(36);index;not null"`
    Type                string `gorm:"type:varchar(16);not null"` // liveness, readiness, startup
    ProbeMode           string `gorm:"type:varchar(16);not null"` // httpGet, tcpSocket, exec
    Enabled             bool   `gorm:"type:bool;default:false"`
    HttpGetPath         string `gorm:"type:varchar(256)"`
    HttpGetPort         int    `gorm:"type:int"`
    TcpSocketPort       int    `gorm:"type:int"`
    ExecCommand         string `gorm:"type:text"`
    InitialDelaySeconds int    `gorm:"type:int;default:0"`
    PeriodSeconds       int    `gorm:"type:int;default:10"`
    TimeoutSeconds      int    `gorm:"type:int;default:1"`
    SuccessThreshold    int    `gorm:"type:int;default:1"`
    FailureThreshold    int    `gorm:"type:int;default:3"`
}

type AppConfigFile struct {
    Base
    AppID     string `gorm:"type:varchar(36);index;not null"`
    Slug      string `gorm:"type:varchar(64);not null"`
    MountPath string `gorm:"type:varchar(256);not null"`
    Content   string `gorm:"type:text;not null"`
    FileMode  string `gorm:"type:varchar(8);default:'0644'"`
}

type AppSchedulingRule struct {
    Base
    AppID        string `gorm:"type:varchar(36);uniqueIndex;not null"`
    RuleType     string `gorm:"type:varchar(32)"` // nodeName, nodeSelector, nodeAffinity
    NodeName     string `gorm:"type:varchar(256)"`
    NodeSelector string `gorm:"type:text"` // JSON array of "key=value"
    NodeAffinity string `gorm:"type:text"` // JSON array of expressions
    Tolerations  string `gorm:"type:text"` // JSON array of tolerations
}
```

### 3.3 API 设计

#### 3.3.1 路由结构

```go
// routes/v1.go
func SetupV1Routes(r *gin.RouterGroup) {
    v1 := r.Group("/api/v1")
    
    // 公开接口（无需认证）
    users := v1.Group("/users")
    {
        users.POST("/sign-up", handlers.SignUp)
        users.POST("/sign-in", handlers.SignIn)
        users.POST("/refresh-token", handlers.RefreshToken)
    }
    
    // 需要认证的接口
    authorized := v1.Group("")
    authorized.Use(middlewares.Auth())
    {
        // 用户
        users := authorized.Group("/users")
        {
            users.GET("", handlers.ListUsers)
            users.GET("/:userID", handlers.GetUser)
            users.PUT("/:userID", handlers.UpdateUser)
            users.DELETE("/:userID", handlers.DeleteUser)
            users.PUT("/:userID/reset-password", handlers.ResetPassword)
            users.PUT("/:userID/change-role", middlewares.AdminOnly(), handlers.ChangeRole)
            users.POST("/:userID/sign-out", handlers.SignOut)
            users.GET("/resources", handlers.GetUserResources)
        }
        
        // 集群（仅管理员）
        clusters := authorized.Group("/clusters")
        clusters.Use(middlewares.AdminOnly())
        {
            clusters.GET("", handlers.ListClusters)
            clusters.POST("", handlers.CreateCluster)
            clusters.GET("/:clusterID", handlers.GetCluster)
            clusters.PUT("/:clusterID", handlers.UpdateCluster)
            clusters.DELETE("/:clusterID", handlers.DeleteCluster)
            clusters.PUT("/:clusterID/enable", handlers.EnableCluster)
            clusters.PUT("/:clusterID/disable", handlers.DisableCluster)
            clusters.POST("/ping", handlers.PingCluster)
            clusters.GET("/:clusterID/nodes", handlers.ListClusterNodes)
            clusters.GET("/:clusterID/extensions", handlers.ListClusterExtensions)
        }
        
        // 项目
        projects := authorized.Group("/projects")
        {
            projects.GET("", handlers.ListProjects)
            projects.POST("", handlers.CreateProject)
            projects.GET("/refs", handlers.ListProjectRefs)
            
            project := projects.Group("/:projectID")
            project.Use(middlewares.ProjectMember())
            {
                project.GET("", handlers.GetProject)
                project.PUT("", middlewares.ProjectOwner(), handlers.UpdateProject)
                project.DELETE("", middlewares.ProjectOwner(), handlers.DeleteProject)
                project.GET("/statistics", handlers.GetProjectStatistics)
                project.GET("/members", handlers.ListProjectMembers)
                project.POST("/members", middlewares.ProjectOwner(), handlers.InviteProjectMembers)
                project.PUT("/members/:userID", middlewares.ProjectOwner(), handlers.UpdateProjectMember)
                project.DELETE("/members", middlewares.ProjectOwner(), handlers.RemoveProjectMember)
                
                // 环境
                project.GET("/envs", handlers.ListEnvs)
                project.POST("/envs", middlewares.ProjectDeveloper(), handlers.CreateEnv)
            }
        }
        
        // 环境
        envs := authorized.Group("/envs")
        {
            env := envs.Group("/:envID")
            env.Use(middlewares.EnvAccess())
            {
                env.GET("", handlers.GetEnv)
                env.PUT("", middlewares.EnvOwner(), handlers.UpdateEnv)
                env.DELETE("", middlewares.EnvOwner(), handlers.DeleteEnv)
                
                // 应用
                env.GET("/apps", handlers.ListApps)
                env.POST("/apps", middlewares.EnvDeveloper(), handlers.CreateApp)
            }
        }
        
        // 应用
        apps := authorized.Group("/apps")
        {
            app := apps.Group("/:appID")
            app.Use(middlewares.AppAccess())
            {
                app.GET("", handlers.GetApp)
                app.PUT("", middlewares.AppDeveloper(), handlers.UpdateApp)
                app.DELETE("", middlewares.AppDeveloper(), handlers.DeleteApp)
                app.POST("/action", middlewares.AppDeveloper(), handlers.AppAction)
                app.PUT("/image", middlewares.AppDeveloper(), handlers.UpdateAppImage)
                app.PUT("/resource", middlewares.AppDeveloper(), handlers.SetAppResource)
                app.PUT("/command", middlewares.AppDeveloper(), handlers.SetAppCommand)
                
                // 应用实例
                app.GET("/instances", handlers.ListAppInstances)
                app.POST("/instances/terminate", middlewares.AppDeveloper(), handlers.TerminateAppInstance)
                app.GET("/instances/:instanceName/containers/:containerName/logs", handlers.ViewAppContainerLogs)
                app.GET("/instances/:instanceName/containers/:containerName/exec", handlers.ExecAppContainerTerminal)
                
                // 环境变量
                app.GET("/env-vars", handlers.ListAppEnvVars)
                app.POST("/env-vars", middlewares.AppDeveloper(), handlers.CreateAppEnvVar)
                
                // 存储卷
                app.GET("/volumes", handlers.ListAppVolumes)
                app.POST("/volumes", middlewares.AppDeveloper(), handlers.CreateAppVolume)
                
                // 配置文件
                app.GET("/config-files", handlers.ListAppConfigFiles)
                app.POST("/config-files", middlewares.AppDeveloper(), handlers.CreateAppConfigFile)
                
                // 网关
                app.GET("/gateways", handlers.ListAppGateways)
                app.POST("/gateways", middlewares.AppDeveloper(), handlers.CreateAppGateway)
                
                // 健康检查
                app.GET("/probes", handlers.ListAppProbes)
                app.POST("/probes", middlewares.AppDeveloper(), handlers.CreateAppProbe)
                
                // 调度规则
                app.GET("/scheduling-rule", handlers.GetAppSchedulingRule)
                app.PUT("/scheduling-rule", middlewares.AppDeveloper(), handlers.SetAppSchedulingRule)
            }
        }
        
        // 平台统计
        authorized.GET("/statistics", middlewares.AdminOnly(), handlers.GetPlatformStatistics)
        authorized.GET("/admin/resources", middlewares.AdminOnly(), handlers.GetAdminResources)
    }
}
```

#### 3.3.2 响应格式

```go
// api/response.go
package api

type Response struct {
    Data  any `json:"data,omitempty"`
    Error string      `json:"error,omitempty"`
}

func Success(c *gin.Context, data any) {
    c.JSON(http.StatusOK, Response{Data: data})
}

func Created(c *gin.Context, data any) {
    c.JSON(http.StatusCreated, Response{Data: data})
}

func NoContent(c *gin.Context) {
    c.Status(http.StatusNoContent)
}

func Error(c *gin.Context, status int, err error) {
    c.JSON(status, Response{Error: err.Error()})
}
```

### 3.4 Kubernetes 集成

#### 3.4.1 集群客户端管理

```go
// kube/store.go
package kube

import (
    "sync"
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/tools/clientcmd"
)

type ClusterStore struct {
    clients sync.Map // map[clusterID]*kubernetes.Clientset
}

func (s *ClusterStore) GetClient(clusterID string) (*kubernetes.Clientset, error) {
    if client, ok := s.clients.Load(clusterID); ok {
        return client.(*kubernetes.Clientset), nil
    }
    return nil, ErrClusterNotFound
}

func (s *ClusterStore) AddClient(clusterID, kubeConfig string) error {
    config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
    if err != nil {
        return err
    }
    
    client, err := kubernetes.NewForConfig(config)
    if err != nil {
        return err
    }
    
    s.clients.Store(clusterID, client)
    return nil
}

func (s *ClusterStore) RemoveClient(clusterID string) {
    s.clients.Delete(clusterID)
}
```

#### 3.4.2 应用部署核心逻辑

```go
// core/apply_resource.go
package core

func ApplyApp(ctx context.Context, app *entities.App, cluster *entities.Cluster) error {
    // 1. 构建应用元数据
    metadata := NewAppMetadataBuilder(app).
        WithEnvVars().
        WithVolumes().
        WithConfigFiles().
        WithProbes().
        WithSchedulingRules().
        Build()
    
    // 2. 生成 Kubernetes 资源
    resources := []runtime.Object{}
    
    // Namespace
    resources = append(resources, metadata.BuildNamespace())
    
    // ConfigMap (if config files exist)
    if len(app.ConfigFiles) > 0 {
        resources = append(resources, metadata.BuildConfigMap())
    }
    
    // PVC (if volumes exist)
    for _, vol := range app.Volumes {
        if vol.VolumeType == "pvc" {
            resources = append(resources, metadata.BuildPVC(vol))
        }
    }
    
    // Deployment or StatefulSet
    if app.AppType == "Deployment" {
        resources = append(resources, metadata.BuildDeployment())
    } else {
        resources = append(resources, metadata.BuildStatefulSet())
    }
    
    // Service
    resources = append(resources, metadata.BuildService())
    
    // Gateway (if gateways exist)
    for _, gw := range app.Gateways {
        if gw.Exposed {
            resources = append(resources, metadata.BuildHTTPRoute(gw))
        }
    }
    
    // 3. 应用资源到集群
    client, err := kube.GetClient(cluster.ID)
    if err != nil {
        return err
    }
    
    for _, res := range resources {
        if err := kube.ApplyResource(ctx, client, res); err != nil {
            return err
        }
    }
    
    return nil
}
```

### 3.5 认证与授权

#### 3.5.1 JWT 认证

```go
// app/jwt.go
package app

import (
    "time"
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID   string `json:"user_id"`
    Username string `json:"username"`
    Role     string `json:"role"`
    jwt.RegisteredClaims
}

func GenerateAccessToken(user *entities.User) (string, error) {
    claims := Claims{
        UserID:   user.ID,
        Username: user.Username,
        Role:     user.Role,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(Config.JWTSecret))
}

func GenerateRefreshToken(user *entities.User) (string, error) {
    claims := Claims{
        UserID: user.ID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(Config.JWTSecret))
}
```

#### 3.5.2 权限中间件

```go
// middlewares/permission.go
package middlewares

func AdminOnly() gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := api.GetClaims(c)
        if claims.Role != "admin" {
            api.Error(c, http.StatusForbidden, ErrAdminRequired)
            c.Abort()
            return
        }
        c.Next()
    }
}

func ProjectMember() gin.HandlerFunc {
    return func(c *gin.Context) {
        claims := api.GetClaims(c)
        projectID := c.Param("projectID")
        
        // Admin 跳过检查
        if claims.Role == "admin" {
            c.Next()
            return
        }
        
        // 检查是否为项目成员
        member, err := services.GetProjectMember(projectID, claims.UserID)
        if err != nil {
            api.Error(c, http.StatusForbidden, ErrNotProjectMember)
            c.Abort()
            return
        }
        
        c.Set("projectMember", member)
        c.Next()
    }
}

func ProjectOwner() gin.HandlerFunc {
    return func(c *gin.Context) {
        member := c.MustGet("projectMember").(*entities.ProjectMember)
        if member.ProjectRole != "owner" {
            api.Error(c, http.StatusForbidden, ErrOwnerRequired)
            c.Abort()
            return
        }
        c.Next()
    }
}

func ProjectDeveloper() gin.HandlerFunc {
    return func(c *gin.Context) {
        member := c.MustGet("projectMember").(*entities.ProjectMember)
        if member.ProjectRole == "viewer" {
            api.Error(c, http.StatusForbidden, ErrDeveloperRequired)
            c.Abort()
            return
        }
        c.Next()
    }
}
```

---

## 4. 前端设计 (ketches-ui)

The frontend code is in the `ui` directory.

### 4.1 项目结构

```txt
ui/
├── public/
│   └── favicon.ico
├── src/
│   ├── api/                        # API 客户端
│   │   ├── client.ts               # Axios 实例配置
│   │   ├── types.ts                # API 类型定义
│   │   ├── auth.ts                 # 认证相关 API
│   │   ├── users.ts                # 用户管理 API
│   │   ├── clusters.ts             # 集群管理 API
│   │   ├── projects.ts             # 项目管理 API
│   │   ├── envs.ts                 # 环境管理 API
│   │   └── apps.ts                 # 应用管理 API
│   ├── components/                 # 通用组件
│   │   ├── ui/                     # shadcn/ui 组件
│   │   │   ├── button.tsx
│   │   │   ├── card.tsx
│   │   │   ├── dialog.tsx
│   │   │   ├── dropdown-menu.tsx
│   │   │   ├── form.tsx
│   │   │   ├── input.tsx
│   │   │   ├── select.tsx
│   │   │   ├── table.tsx
│   │   │   ├── tabs.tsx
│   │   │   ├── toast.tsx
│   │   │   └── ...
│   │   ├── layout/                 # 布局组件
│   │   │   ├── app-layout.tsx
│   │   │   ├── sidebar.tsx
│   │   │   ├── header.tsx
│   │   │   └── breadcrumb.tsx
│   │   ├── shared/                 # 共享业务组件
│   │   │   ├── color-badge.tsx
│   │   │   ├── empty-state.tsx
│   │   │   └── ...
│   │   ├── cluster/                # 集群相关组件
│   │   │   ├── cluster-card.tsx
│   │   │   ├── cluster-form.tsx
│   │   │   └── node-list.tsx
│   │   ├── project/                # 项目相关组件
│   │   │   ├── project-card.tsx
│   │   │   ├── project-form.tsx
│   │   │   └── member-list.tsx
│   │   ├── env/                    # 环境相关组件
│   │   │   ├── env-card.tsx
│   │   │   └── env-form.tsx
│   │   ├── app/                    # 应用相关组件
│   │   │   ├── app-card.tsx
│   │   │   ├── app-form.tsx
│   │   │   ├── app-status.tsx
│   │   │   ├── instance-list.tsx
│   │   │   ├── log-viewer.tsx
│   │   │   ├── terminal.tsx
│   │   │   ├── env-var-editor.tsx
│   │   │   ├── volume-editor.tsx
│   │   │   ├── gateway-editor.tsx
│   │   │   ├── probe-editor.tsx
│   │   │   └── scheduling-editor.tsx
│   │   └── spot/                   # 全局搜索
│   │       └── spotlight.tsx
│   ├── hooks/                      # 自定义 Hooks
│   │   ├── use-auth.ts
│   │   ├── use-toast.ts
│   │   ├── use-debounce.ts
│   │   └── use-websocket.ts
│   ├── lib/                        # 工具库
│   │   ├── utils.ts                # 通用工具函数
│   │   ├── cn.ts                   # className 合并
│   │   └── constants.ts            # 常量定义
│   ├── pages/                      # 页面组件
│   │   ├── auth/
│   │   │   ├── sign-in.tsx
│   │   │   └── sign-up.tsx
│   │   ├── dashboard/
│   │   │   └── index.tsx
│   │   ├── clusters/
│   │   │   ├── index.tsx
│   │   │   ├── [clusterId]/
│   │   │   │   └── index.tsx
│   │   │   └── create.tsx
│   │   ├── users/
│   │   │   └── index.tsx
│   │   ├── projects/
│   │   │   ├── index.tsx
│   │   │   ├── [projectId]/
│   │   │   │   ├── index.tsx
│   │   │   │   ├── members.tsx
│   │   │   │   └── settings.tsx
│   │   │   └── create.tsx
│   │   ├── envs/
│   │   │   ├── [envId]/
│   │   │   │   ├── index.tsx
│   │   │   │   └── settings.tsx
│   │   │   └── create.tsx
│   │   ├── apps/
│   │   │   ├── [appId]/
│   │   │   │   ├── index.tsx
│   │   │   │   ├── config.tsx
│   │   │   │   ├── instances.tsx
│   │   │   │   └── logs.tsx
│   │   │   └── create.tsx
│   │   └── settings/
│   │       └── profile.tsx
│   ├── routes/                     # 路由配置
│   │   └── index.tsx
│   ├── stores/                     # 状态管理
│   │   ├── auth.ts                 # 认证状态
│   │   ├── user.ts                 # 用户状态
│   │   └── theme.ts                # 主题状态
│   ├── types/                      # 类型定义
│   │   ├── user.ts
│   │   ├── cluster.ts
│   │   ├── project.ts
│   │   ├── env.ts
│   │   └── app.ts
│   ├── App.tsx                     # 应用入口
│   ├── main.tsx                    # React 入口
│   └── index.css                   # 全局样式
├── components.json                 # shadcn/ui 配置
├── tailwind.config.js              # Tailwind 配置
├── tsconfig.json                   # TypeScript 配置
├── vite.config.ts                  # Vite 配置
├── package.json
└── .env
```

### 4.2 状态管理

#### 4.2.1 认证状态

```typescript
// stores/auth.ts
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface User {
  userID: string
  username: string
  email: string
  role: string
  fullname?: string
}

interface AuthState {
  user: User | null
  accessToken: string | null
  refreshToken: string | null
  isAuthenticated: boolean
  
  setAuth: (user: User, accessToken: string, refreshToken: string) => void
  updateTokens: (accessToken: string, refreshToken: string) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      accessToken: null,
      refreshToken: null,
      isAuthenticated: false,
      
      setAuth: (user, accessToken, refreshToken) =>
        set({
          user,
          accessToken,
          refreshToken,
          isAuthenticated: true,
        }),
      
      updateTokens: (accessToken, refreshToken) =>
        set({ accessToken, refreshToken }),
      
      logout: () =>
        set({
          user: null,
          accessToken: null,
          refreshToken: null,
          isAuthenticated: false,
        }),
    }),
    {
      name: 'auth-storage',
    }
  )
)
```

### 4.3 API 客户端

```typescript
// api/client.ts
import axios, { AxiosInstance, AxiosError } from 'axios'
import { useAuthStore } from '@/stores/auth'

const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || '/api'

const client: AxiosInstance = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器
client.interceptors.request.use((config) => {
  const { accessToken } = useAuthStore.getState()
  if (accessToken) {
    config.headers.Authorization = `Bearer ${accessToken}`
  }
  return config
})

// 响应拦截器
client.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config
    
    // Token 过期，尝试刷新
    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true
      
      try {
        const { refreshToken, updateTokens, logout } = useAuthStore.getState()
        
        if (!refreshToken) {
          logout()
          return Promise.reject(error)
        }
        
        const response = await axios.post(`${API_BASE_URL}/v1/users/refresh-token`, null, {
          headers: { Authorization: `Bearer ${refreshToken}` },
        })
        
        const { accessToken: newAccessToken, refreshToken: newRefreshToken } = response.data.data
        updateTokens(newAccessToken, newRefreshToken)
        
        originalRequest.headers.Authorization = `Bearer ${newAccessToken}`
        return client(originalRequest)
      } catch (refreshError) {
        useAuthStore.getState().logout()
        return Promise.reject(refreshError)
      }
    }
    
    return Promise.reject(error)
  }
)

export default client
```

### 4.4 路由配置

```typescript
// routes/index.tsx
import { createBrowserRouter, Navigate } from 'react-router-dom'
import { useAuthStore } from '@/stores/auth'

// Layouts
import AppLayout from '@/components/layout/app-layout'
import AuthLayout from '@/components/layout/auth-layout'

// Pages
import SignIn from '@/pages/auth/sign-in'
import SignUp from '@/pages/auth/sign-up'
import Dashboard from '@/pages/dashboard'
import ClusterList from '@/pages/clusters'
import ClusterDetail from '@/pages/clusters/[clusterId]'
import UserList from '@/pages/users'
import ProjectList from '@/pages/projects'
import ProjectDetail from '@/pages/projects/[projectId]'
import EnvDetail from '@/pages/envs/[envId]'
import AppDetail from '@/pages/apps/[appId]'

// Protected Route
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  
  if (!isAuthenticated) {
    return <Navigate to="/sign-in" replace />
  }
  
  return <>{children}</>
}

// Admin Route
function AdminRoute({ children }: { children: React.ReactNode }) {
  const user = useAuthStore((state) => state.user)
  
  if (user?.role !== 'admin') {
    return <Navigate to="/" replace />
  }
  
  return <>{children}</>
}

export const router = createBrowserRouter([
  // Auth routes
  {
    element: <AuthLayout />,
    children: [
      { path: '/sign-in', element: <SignIn /> },
      { path: '/sign-up', element: <SignUp /> },
    ],
  },
  
  // App routes
  {
    element: (
      <ProtectedRoute>
        <AppLayout />
      </ProtectedRoute>
    ),
    children: [
      { index: true, element: <Dashboard /> },
      
      // Admin routes
      {
        path: 'clusters',
        element: <AdminRoute><ClusterList /></AdminRoute>,
      },
      {
        path: 'clusters/:clusterId',
        element: <AdminRoute><ClusterDetail /></AdminRoute>,
      },
      {
        path: 'users',
        element: <AdminRoute><UserList /></AdminRoute>,
      },
      
      // User routes
      { path: 'projects', element: <ProjectList /> },
      { path: 'projects/:projectId', element: <ProjectDetail /> },
      { path: 'envs/:envId', element: <EnvDetail /> },
      { path: 'apps/:appId', element: <AppDetail /> },
    ],
  },
])
```

### 4.5 核心组件示例

#### 4.5.1 应用状态徽章

```typescript
// components/app/app-status.tsx
import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

type AppStatus = 
  | 'undeployed' 
  | 'starting' 
  | 'running' 
  | 'stopped' 
  | 'stopping'
  | 'updating'
  | 'abnormal'
  | 'completed'
  | 'debugging'
  | 'unknown'

const statusConfig: Record<AppStatus, { label: string; className: string }> = {
  undeployed: { label: '未部署', className: 'bg-gray-100 text-gray-800' },
  starting: { label: '启动中', className: 'bg-blue-100 text-blue-800 animate-pulse' },
  running: { label: '运行中', className: 'bg-green-100 text-green-800' },
  stopped: { label: '已停止', className: 'bg-gray-100 text-gray-800' },
  stopping: { label: '停止中', className: 'bg-yellow-100 text-yellow-800 animate-pulse' },
  updating: { label: '更新中', className: 'bg-blue-100 text-blue-800 animate-pulse' },
  abnormal: { label: '异常', className: 'bg-red-100 text-red-800' },
  completed: { label: '已完成', className: 'bg-green-100 text-green-800' },
  debugging: { label: '调试中', className: 'bg-purple-100 text-purple-800' },
  unknown: { label: '未知', className: 'bg-gray-100 text-gray-800' },
}

interface AppStatusBadgeProps {
  status: AppStatus
  className?: string
}

export function AppStatusBadge({ status, className }: AppStatusBadgeProps) {
  const config = statusConfig[status] || statusConfig.unknown
  
  return (
    <Badge className={cn(config.className, className)} variant="outline">
      {config.label}
    </Badge>
  )
}
```

#### 4.5.2 终端组件

```typescript
// components/app/terminal.tsx
import { useEffect, useRef } from 'react'
import { Terminal as XTerm } from 'xterm'
import { FitAddon } from 'xterm-addon-fit'
import { WebLinksAddon } from 'xterm-addon-web-links'
import 'xterm/css/xterm.css'

interface TerminalProps {
  appId: string
  instanceName: string
  containerName: string
}

export function Terminal({ appId, instanceName, containerName }: TerminalProps) {
  const terminalRef = useRef<HTMLDivElement>(null)
  const xtermRef = useRef<XTerm | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  
  useEffect(() => {
    if (!terminalRef.current) return
    
    // 初始化终端
    const xterm = new XTerm({
      cursorBlink: true,
      fontSize: 14,
      fontFamily: 'Menlo, Monaco, "Courier New", monospace',
      theme: {
        background: '#1e1e1e',
        foreground: '#d4d4d4',
      },
    })
    
    const fitAddon = new FitAddon()
    const webLinksAddon = new WebLinksAddon()
    
    xterm.loadAddon(fitAddon)
    xterm.loadAddon(webLinksAddon)
    xterm.open(terminalRef.current)
    fitAddon.fit()
    
    xtermRef.current = xterm
    
    // 建立 WebSocket 连接
    const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/v1/apps/${appId}/instances/${instanceName}/containers/${containerName}/exec`
    const ws = new WebSocket(wsUrl)
    wsRef.current = ws
    
    ws.onopen = () => {
      xterm.writeln('Connected to container terminal...')
    }
    
    ws.onmessage = (event) => {
      xterm.write(event.data)
    }
    
    ws.onerror = () => {
      xterm.writeln('\r\n\x1b[31mConnection error\x1b[0m')
    }
    
    ws.onclose = () => {
      xterm.writeln('\r\n\x1b[33mConnection closed\x1b[0m')
    }
    
    // 发送用户输入
    xterm.onData((data) => {
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(data)
      }
    })
    
    // 处理窗口大小变化
    const handleResize = () => {
      fitAddon.fit()
      if (ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
          type: 'resize',
          cols: xterm.cols,
          rows: xterm.rows,
        }))
      }
    }
    
    window.addEventListener('resize', handleResize)
    
    return () => {
      window.removeEventListener('resize', handleResize)
      ws.close()
      xterm.dispose()
    }
  }, [appId, instanceName, containerName])
  
  return (
    <div 
      ref={terminalRef} 
      className="h-full w-full bg-[#1e1e1e] rounded-lg overflow-hidden"
    />
  )
}
```

### 4.6 Tailwind 配置

```javascript
// tailwind.config.js
/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: ["class"],
  content: [
    './pages/**/*.{ts,tsx}',
    './components/**/*.{ts,tsx}',
    './app/**/*.{ts,tsx}',
    './src/**/*.{ts,tsx}',
  ],
  theme: {
    container: {
      center: true,
      padding: "2rem",
      screens: {
        "2xl": "1400px",
      },
    },
    extend: {
      colors: {
        border: "hsl(var(--border))",
        input: "hsl(var(--input))",
        ring: "hsl(var(--ring))",
        background: "hsl(var(--background))",
        foreground: "hsl(var(--foreground))",
        primary: {
          DEFAULT: "hsl(var(--primary))",
          foreground: "hsl(var(--primary-foreground))",
        },
        secondary: {
          DEFAULT: "hsl(var(--secondary))",
          foreground: "hsl(var(--secondary-foreground))",
        },
        destructive: {
          DEFAULT: "hsl(var(--destructive))",
          foreground: "hsl(var(--destructive-foreground))",
        },
        muted: {
          DEFAULT: "hsl(var(--muted))",
          foreground: "hsl(var(--muted-foreground))",
        },
        accent: {
          DEFAULT: "hsl(var(--accent))",
          foreground: "hsl(var(--accent-foreground))",
        },
        popover: {
          DEFAULT: "hsl(var(--popover))",
          foreground: "hsl(var(--popover-foreground))",
        },
        card: {
          DEFAULT: "hsl(var(--card))",
          foreground: "hsl(var(--card-foreground))",
        },
      },
      borderRadius: {
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      keyframes: {
        "accordion-down": {
          from: { height: 0 },
          to: { height: "var(--radix-accordion-content-height)" },
        },
        "accordion-up": {
          from: { height: "var(--radix-accordion-content-height)" },
          to: { height: 0 },
        },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
}
```

---

## 5. 部署架构

### 5.1 Kubernetes 部署

```yaml
# deploy/kubernetes/manifests.yaml
---
apiVersion: v1
kind: Namespace
metadata:
  name: ketches-system

---
apiVersion: v1
kind: ConfigMap
metadata:
  name: ketches-config
  namespace: ketches-system
data:
  PORT: "8080"
  LOG_LEVEL: "info"
  DB_DRIVER: "postgres"
  DB_HOST: "postgres"
  DB_PORT: "5432"
  DB_NAME: "ketches"
  DB_USERNAME: "ketches"
  DB_SSLMODE: "require"
  DB_AUTO_MIGRATE: "true"
  CORS_ALLOWED_ORIGINS: "https://app.example.com"

---
apiVersion: v1
kind: Secret
metadata:
  name: ketches-secret
  namespace: ketches-system
type: Opaque
stringData:
  JWT_SECRET: ""
  SECRET_ENCRYPTION_KEY: ""
  DB_PASSWORD: ""

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ketches-api
  namespace: ketches-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: ketches-api
  template:
    metadata:
      labels:
        app: ketches-api
    spec:
      containers:
      - name: ketches-api
        image: ketches/ketches-api:latest
        ports:
        - containerPort: 8080
        envFrom:
        - configMapRef:
            name: ketches-config
        - secretRef:
            name: ketches-secret
        resources:
          requests:
            cpu: 100m
            memory: 128Mi
          limits:
            cpu: 500m
            memory: 512Mi
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5

---
apiVersion: v1
kind: Service
metadata:
  name: ketches-api
  namespace: ketches-system
spec:
  selector:
    app: ketches-api
  ports:
  - port: 8080
    targetPort: 8080

---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ketches-ui
  namespace: ketches-system
spec:
  replicas: 2
  selector:
    matchLabels:
      app: ketches-ui
  template:
    metadata:
      labels:
        app: ketches-ui
    spec:
      containers:
      - name: ketches-ui
        image: ketches/ketches-ui:latest
        ports:
        - containerPort: 80
        resources:
          requests:
            cpu: 50m
            memory: 64Mi
          limits:
            cpu: 200m
            memory: 256Mi

---
apiVersion: v1
kind: Service
metadata:
  name: ketches-ui
  namespace: ketches-system
spec:
  selector:
    app: ketches-ui
  ports:
  - port: 80
    targetPort: 80

---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: ketches-ingress
  namespace: ketches-system
  annotations:
    nginx.ingress.kubernetes.io/proxy-body-size: "50m"
spec:
  rules:
  - host: ketches.example.com
    http:
      paths:
      - path: /api
        pathType: Prefix
        backend:
          service:
            name: ketches-api
            port:
              number: 8080
      - path: /
        pathType: Prefix
        backend:
          service:
            name: ketches-ui
            port:
              number: 80
```

### 5.2 Docker Compose 部署

```yaml
# docker-compose.yaml
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_USER: ketches
      POSTGRES_PASSWORD: "${POSTGRES_PASSWORD:?set POSTGRES_PASSWORD}"
      POSTGRES_DB: ketches
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ketches"]
      interval: 10s
      timeout: 5s
      retries: 5

  ketches-api:
    image: ketches/ketches-api:latest
    build:
      context: .
      dockerfile: Dockerfile.backend
    environment:
      PORT: "8080"
      LOG_LEVEL: "info"
      DB_DRIVER: "postgres"
      DB_HOST: "postgres"
      DB_PORT: "5432"
      DB_NAME: "ketches"
      DB_USERNAME: "ketches"
      DB_PASSWORD: "${POSTGRES_PASSWORD:?set POSTGRES_PASSWORD}"
      DB_SSLMODE: "require"
      JWT_SECRET: "${JWT_SECRET:?set JWT_SECRET}"
      SECRET_ENCRYPTION_KEY: "${SECRET_ENCRYPTION_KEY:?set SECRET_ENCRYPTION_KEY}"
    depends_on:
      postgres:
        condition: service_healthy
    ports:
      - "8080:8080"

  ketches-ui:
    image: ketches/ketches-ui:latest
    build:
      context: .
      dockerfile: Dockerfile.frontend
    ports:
      - "80:80"
    depends_on:
      - ketches-api

volumes:
  postgres_data:
```

---

## 6. 开发规范

### 6.1 代码规范

#### 后端 (Go)

- 使用 `gofmt` 和 `goimports` 格式化代码
- 遵循 Effective Go 和 Go Code Review Comments
- 错误处理：返回错误而非 panic
- 命名规范：使用驼峰命名，导出函数首字母大写

#### 前端 (TypeScript/React)

- 使用 ESLint + Prettier 格式化代码
- 组件使用函数式组件 + Hooks
- 类型优先：避免使用 any
- 文件命名：使用 kebab-case

### 6.2 Git 提交规范

```txt
<type>(<scope>): <subject>

<body>

<footer>
```

Type:

- `feat`: 新功能
- `fix`: Bug 修复
- `docs`: 文档
- `style`: 格式
- `refactor`: 重构
- `test`: 测试
- `chore`: 构建/工具

### 6.3 API 版本管理

- URL 版本化：`/api/v1/...`
- 向后兼容：新版本添加字段，不删除现有字段
- 废弃通知：通过响应头 `X-Deprecated` 标记

---

## 7. 测试策略

### 7.1 后端测试

```txt
tests/
├── unit/                   # 单元测试
│   ├── services/
│   └── utils/
├── integration/            # 集成测试
│   ├── api/
│   └── db/
└── e2e/                    # 端到端测试
    └── scenarios/
```

### 7.2 前端测试

```txt
tests/
├── unit/                   # 组件单元测试 (Vitest)
├── integration/            # 组件集成测试
└── e2e/                    # E2E 测试 (Playwright)
```

---

## 8. 监控与日志

### 8.1 日志规范

```go
// 使用结构化日志
logger.Info("app deployed",
    "appID", app.ID,
    "appName", app.Slug,
    "cluster", cluster.Slug,
    "duration", duration,
)
```

### 8.2 指标采集

- HTTP 请求指标（延迟、状态码）
- Kubernetes 操作指标
- 数据库查询指标

### 8.3 健康检查

```txt
GET /healthz              # 基础健康检查
GET /readyz               # 就绪检查（包含数据库连接）
```

---

## 附录

### A. 环境变量清单

| 变量 | 描述 | 默认值 |
| ---- | ---- | ------ |
| PORT | API 监听端口 | 8080 |
| LOG_LEVEL | 日志级别 | info |
| DB_DRIVER | 数据库驱动 | postgres |
| DB_SOURCE | 完整数据库连接串（优先于拆分变量） | 空 |
| DB_AUTO_MIGRATE | 启动时是否执行 GORM AutoMigrate | true |
| JWT_SECRET | JWT 密钥 | 无（必填） |
| SECRET_ENCRYPTION_KEY | 敏感数据静态加密密钥 | 无（必填） |

### B. API 错误码

| 错误码 | HTTP 状态 | 描述 |
| ------ | -------- | ---- |
| ERR_UNAUTHORIZED | 401 | 未认证 |
| ERR_FORBIDDEN | 403 | 权限不足 |
| ERR_NOT_FOUND | 404 | 资源不存在 |
| ERR_VALIDATION | 400 | 参数验证失败 |
| ERR_INTERNAL | 500 | 内部错误 |

### C. 参考文档

- [Gin Web Framework](https://gin-gonic.com/)
- [GORM](https://gorm.io/)
- [client-go](https://github.com/kubernetes/client-go)
- [React](https://react.dev/)
- [shadcn/ui](https://ui.shadcn.com/)
- [Tailwind CSS](https://tailwindcss.com/)
- [Vite](https://vitejs.dev/)
