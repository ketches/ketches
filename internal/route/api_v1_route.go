package route

import (
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/handler"
	"github.com/ketches/ketches/internal/middleware"
)

type APIV1Route struct {
	handler.Handler
	*gin.RouterGroup
}

func NewAPIV1Route(e *gin.Engine) *APIV1Route {
	r := e.Group("/api/v1")
	r.Use(middleware.Auth())
	return &APIV1Route{
		Handler:     handler.NewHandler(),
		RouterGroup: r,
	}
}

func (r *APIV1Route) Register() {
	registerClusterRoute(r)
	registerUserRoute(r)
	registerSpaceRoute(r)
	registerApplicationRoute(r)
	// Note: add more routes here
}

func registerClusterRoute(r *APIV1Route) {
	// cluster := r.Group("/clusters")

	// cluster.GET("", handler.ListClusters)
	// cluster.GET("/:id", handler.GetCluster)
}

func registerUserRoute(r *APIV1Route) {
	user := r.Group("/users")

	user.GET("", handler.ListUsers)
	user.GET("/:account_id", handler.GetUser)
	user.POST("/sign-up", handler.UserSignUp)
	user.POST("/sign-in", handler.UserSignIn)
	user.POST("/sign-out", handler.UserSignOut)
	user.PUT("/:account_id", handler.UserUpdate)
	user.POST("/refresh-token", handler.UserRefreshToken)
	user.PUT("/reset-password", handler.UserResetPassword)
	user.PUT("/rename", handler.UserRename)
	user.DELETE("/:account_id", handler.DeleteUser)
}

func registerSpaceRoute(r *APIV1Route) {
	space := r.Group("/spaces")

	space.GET("", handler.ListSpaces)
	space.GET("/:space_id", handler.GetSpace)
	space.POST("", handler.CreateSpace)
	space.DELETE("/:space_id", handler.DeleteSpace)
	r.GET("/:space_id/users", handler.ListUsers)
}

func registerApplicationRoute(r *APIV1Route) {
	application := r.Group("/apps")

	application.GET("", handler.ListApplications)
	application.GET("/:application_id", handler.GetApplication)
	application.POST("", handler.CreateApplication)
	application.PUT("/:application_id/start", handler.StartApplication)
	application.PUT("/:application_id/stop", handler.StopApplication)
	application.PUT("/:application_id/restart", handler.RestartApplication)
	application.GET("/:application_id/pod-containers", handler.GetApplicationPodsAndContainers)
	application.GET("/:application_id/logs", handler.GetApplicationContainerLogs)
	application.DELETE("/:application_id", handler.DeleteApplication)
	application.POST("/:application_id/backup", handler.BackupApplication)
	application.POST("/:application_id/restore", handler.RestoreApplication)
}
