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
	r.Use(middlewares.Auth())
	return &APIV1Route{
		RouterGroup: r,
	}
}

func (r *APIV1Route) Register() {
	registerClusterRoute(r)
	registerUserRoute(r)
	registerProjectRoute(r)
	registerEnvRoute(r)
	registerAppRoute(r)
}

func registerClusterRoute(r *APIV1Route) {
	clusters := r.Group("/clusters")

	clusters.GET("", handlers.ListClusters)
	clusters.GET("/refs", handlers.AllClusterRefs)
	clusters.GET("/:clusterID", handlers.GetCluster)
	clusters.GET("/:clusterID/ref", handlers.GetClusterRef)
	clusters.POST("", handlers.CreateCluster)
	clusters.PUT("/:clusterID", handlers.UpdateCluster)
	clusters.DELETE("/:clusterID", handlers.DeleteCluster)
	clusters.PUT("/:clusterID/enable", handlers.EnableCluster)
	clusters.PUT("/:clusterID/disable", handlers.DisableCluster)
}

func registerUserRoute(r *APIV1Route) {
	users := r.Group("/users")

	users.GET("", handlers.ListUsers)
	users.GET("/:userID", handlers.GetUser)
	users.POST("/sign-up", handlers.UserSignUp)
	users.POST("/sign-in", handlers.UserSignIn)
	users.POST("/sign-out", handlers.UserSignOut)
	users.POST("/refresh-token", handlers.UserRefreshToken)
	users.PUT("/:userID", handlers.UserUpdate)
	users.PUT("/:userID/reset-password", handlers.UserResetPassword)
	users.PUT("/:userID/rename", handlers.UserRename)
	users.DELETE("/:userID", handlers.DeleteUser)
}

func registerProjectRoute(r *APIV1Route) {
	projects := r.Group("/projects")

	projects.GET("", handlers.ListProjects)
	projects.GET("/refs", handlers.AllProjectRefs)
	projects.GET("/:projectID", handlers.GetProject)
	projects.GET("/:projectID/ref", handlers.GetProjectRef)
	projects.POST("", handlers.CreateProject)
	projects.PUT("/:projectID", handlers.UpdateProject)
	projects.DELETE("/:projectID", handlers.DeleteProject)

	projects.GET("/:projectID/members", handlers.ListProjectMembers)
	projects.GET("/:projectID/members/addable", handlers.ListAddableProjectMembers)
	projects.POST("/:projectID/members", handlers.AddProjectMembers)
	projects.PUT("/:projectID/members/:userID", handlers.UpdateProjectMember)
	projects.DELETE("/:projectID/members", handlers.RemoveProjectMember)
}

func registerEnvRoute(r *APIV1Route) {
	envs := r.Group("/envs")

	envs.GET("", handlers.ListEnvs)
	envs.GET("/refs", handlers.AllEnvRefs)
	envs.GET("/:envID", handlers.GetEnv)
	envs.GET("/:envID/ref", handlers.GetEnvRef)
	envs.POST("", handlers.CreateEnv)
	envs.PUT("/:envID", handlers.UpdateEnv)
	envs.DELETE("/:envID", handlers.DeleteEnv)
}

func registerAppRoute(r *APIV1Route) {
	apps := r.Group("/apps")

	apps.GET("", handlers.ListApps)
	apps.GET("/refs", handlers.AllAppRefs)
	apps.GET("/:appID", handlers.GetApp)
	apps.GET("/:appID/ref", handlers.GetAppRef)
	apps.POST("", handlers.CreateApp)
	apps.PUT("/:appID", handlers.UpdateApp)
	apps.DELETE("/:appID", handlers.DeleteApp)
	apps.PUT("/:appID/image", handlers.UpdateAppImage)
	apps.POST("/:appID/action", handlers.AppAction)
	apps.GET("/:appID/instances", handlers.ListAppInstances)
	apps.DELETE("/:appID/instances", handlers.TerminateAppInstance)
	apps.GET("/:appID/instances/:instanceName/containers/:containerName/logs", handlers.ViewAppContainerLogs)
	apps.GET("/:appID/instances/:instanceName/containers/:containerName/exec", handlers.ExecAppContainerTerminal)
}
