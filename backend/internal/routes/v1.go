package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/handlers"
	"github.com/ketches/ketches/internal/middlewares"
)

type APIV1Route struct {
	*gin.RouterGroup
}

func NewAPIV1Route(e *gin.Engine) *APIV1Route {
	r := e.Group("/api/v1")

	// No authentication required for these routes
	r.POST("/users/sign-in", handlers.UserSignIn)
	r.POST("/users/sign-up", handlers.UserSignUp)
	r.POST("/users/refresh-token", handlers.UserRefreshToken)
	r.POST("/users/reset-password", handlers.UserResetPassword)

	auth := r.Group("", middlewares.Auth())
	return &APIV1Route{
		RouterGroup: auth,
	}
}

func (r *APIV1Route) Register() {
	registerPlatformRoute(r)
	registerClusterRoute(r)
	registerUserRoute(r)
	registerProjectRoute(r)
	registerEnvRoute(r)
	registerAppRoute(r)
	registerAppEnvVarRoute(r)
}

func registerPlatformRoute(r *APIV1Route) {
	// Routes that require admin permissions
	adminOnly := r.Group("", middlewares.AdminOnly())
	adminOnly.GET("/statistics", handlers.GetPlatformStatistics)
	adminOnly.GET("/admin/resources", handlers.GetAdminResources)
}

func registerClusterRoute(r *APIV1Route) {
	clusters := r.Group("/clusters")
	clusters.GET("/refs", handlers.AllClusterRefs)
	clusters.GET("/:clusterID/ref", handlers.GetClusterRef)
	clusters.GET("/:clusterID/nodes/refs", handlers.ListClusterNodeRefs)

	// Routes that require admin permissions
	adminOnly := clusters.Group("", middlewares.AdminOnly())
	adminOnly.GET("", handlers.ListClusters)
	adminOnly.GET("/:clusterID", handlers.GetCluster)
	adminOnly.POST("", handlers.CreateCluster)
	adminOnly.PUT("/:clusterID", handlers.UpdateCluster)
	adminOnly.DELETE("/:clusterID", handlers.DeleteCluster)
	adminOnly.PUT("/:clusterID/enable", handlers.EnableCluster)
	adminOnly.PUT("/:clusterID/disable", handlers.DisableCluster)
	adminOnly.POST("/ping", handlers.PingClusterKubeConfig)
	adminOnly.GET("/:clusterID/nodes", handlers.ListClusterNodes)
	adminOnly.GET("/:clusterID/nodes/:nodeName", handlers.GetClusterNode)
	adminOnly.GET("/:clusterID/extensions", handlers.ListClusterExtensions)
}

func registerUserRoute(r *APIV1Route) {
	users := r.Group("/users")

	users.GET("", handlers.ListUsers)
	users.GET("/:userID", handlers.GetUser)
	users.POST("/sign-out", handlers.UserSignOut)
	users.PUT("/:userID", handlers.UserUpdate)
	users.PUT("/:userID/reset-password", handlers.UserResetPassword)
	users.PUT("/:userID/rename", handlers.UserRename)
	users.DELETE("/:userID", handlers.DeleteUser)
	users.GET("/resources", handlers.GetUserResources)
}

func registerProjectRoute(r *APIV1Route) {
	projects := r.Group("/projects")

	projects.GET("", handlers.ListProjects)
	projects.GET("/refs", handlers.AllProjectRefs)
	projects.POST("", handlers.CreateProject)

	// Routes that require project membership
	projectMember := projects.Group("/:projectID", middlewares.ProjectMember())
	projectMember.GET("/statistics", handlers.GetProjectStatistics)
	projectMember.GET("", handlers.GetProject)
	projectMember.GET("/ref", handlers.GetProjectRef)
	projectMember.GET("/members", handlers.ListProjectMembers)
	projectMember.GET("/members/addable", handlers.ListAddableProjectMembers)
	projectMember.GET("/envs", handlers.ListEnvs)
	projectMember.GET("/envs/refs", handlers.AllEnvRefs)

	// Routes that require project owner role
	projectOwner := projects.Group("/:projectID", middlewares.ProjectOwnerOnly())
	projectOwner.PUT("", handlers.UpdateProject)
	projectOwner.DELETE("", handlers.DeleteProject)
	projectOwner.POST("/members", handlers.AddProjectMembers)
	projectOwner.PUT("/members/:userID", handlers.UpdateProjectMember)
	projectOwner.DELETE("/members", handlers.RemoveProjectMember)

	// Routes that require project developer or above role
	projectDeveloper := projects.Group("/:projectID", middlewares.ProjectDeveloperOrAbove())
	projectDeveloper.POST("/envs", handlers.CreateEnv)
}

func registerEnvRoute(r *APIV1Route) {
	envs := r.Group("/envs/:envID")

	// Routes that require project membership
	projectMember := envs.Group("", middlewares.ProjectMember())
	projectMember.GET("", handlers.GetEnv)
	projectMember.GET("/ref", handlers.GetEnvRef)
	projectMember.GET("/apps", handlers.ListApps)
	projectMember.GET("/apps/refs", handlers.AllAppRefs)

	// Routes that require project owner role
	projectOwner := envs.Group("", middlewares.ProjectOwnerOnly())
	projectOwner.PUT("", handlers.UpdateEnv)
	projectOwner.DELETE("", handlers.DeleteEnv)

	// Routes that require project developer or above role
	projectDeveloper := envs.Group("", middlewares.ProjectDeveloperOrAbove())
	projectDeveloper.POST("/apps", handlers.CreateApp)
}

func registerAppRoute(r *APIV1Route) {
	apps := r.Group("/apps/:appID")

	// Routes that require project membership (read-only)
	projectMember := apps.Group("", middlewares.ProjectMember())
	projectMember.GET("", handlers.GetApp)
	projectMember.GET("/ref", handlers.GetAppRef)
	projectMember.GET("/instances", handlers.ListAppInstances)
	projectMember.GET("/running/info", handlers.GetAppRunningInfo, middlewares.RequestIDMiddleware())
	projectMember.GET("/envVars", handlers.ListAppEnvVars)
	projectMember.GET("/volumes", handlers.ListAppVolumes)
	projectMember.GET("/gateways", handlers.NewAppGatewayHandler().ListAppGateways)
	projectMember.GET("/probes", handlers.NewAppProbeHandler().ListAppProbes)

	// Routes that require developer or owner role (read-write)
	projectDeveloper := apps.Group("", middlewares.ProjectDeveloperOrAbove())
	projectDeveloper.PUT("", handlers.UpdateApp)
	projectDeveloper.DELETE("", handlers.DeleteApp)
	projectDeveloper.PUT("/image", handlers.UpdateAppImage)
	projectDeveloper.PUT("/command", handlers.SetAppCommand)
	projectDeveloper.PUT("/resource", handlers.SetAppResource)
	projectDeveloper.POST("/action", handlers.AppAction)
	projectDeveloper.DELETE("/instances", handlers.TerminateAppInstance)
	projectDeveloper.GET("/instances/:instanceName/containers/:containerName/logs", handlers.ViewAppContainerLogs)
	projectDeveloper.GET("/instances/:instanceName/containers/:containerName/exec", handlers.ExecAppContainerTerminal)

	projectDeveloper.POST("/envVars", handlers.CreateAppEnvVar)
	projectDeveloper.PUT("/envVars/:envVarID", handlers.UpdateAppEnvVar)
	projectDeveloper.DELETE("/envVars", handlers.DeleteAppEnvVars)

	projectDeveloper.POST("/volumes", handlers.CreateAppVolume)
	projectDeveloper.PUT("/volumes/:volumeID", handlers.UpdateAppVolume)
	projectDeveloper.DELETE("/volumes", handlers.DeleteAppVolumes)

	appGatewayHandler := handlers.NewAppGatewayHandler()
	projectDeveloper.POST("/gateways", appGatewayHandler.CreateAppGateway)
	projectDeveloper.PUT("/gateways/:gatewayID", appGatewayHandler.UpdateAppGateway)
	projectDeveloper.PUT("/gateways/:gatewayID/toggle", appGatewayHandler.ToggleAppGatewayExposed)
	projectDeveloper.DELETE("/gateways", appGatewayHandler.DeleteAppGateways)

	appProbeHandler := handlers.NewAppProbeHandler()
	projectDeveloper.POST("/probes", appProbeHandler.CreateAppProbe)
	projectDeveloper.PUT("/probes/:probeID", appProbeHandler.UpdateAppProbe)
	projectDeveloper.PUT("/probes/:probeID/toggle", appProbeHandler.ToggleAppProbe)
	projectDeveloper.DELETE("/probes/:probeID", appProbeHandler.DeleteAppProbe)
}

func registerAppEnvVarRoute(r *APIV1Route) {
	// Routes that require developer or owner role (write)
}
