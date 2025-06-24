package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// @Summary List Apps
// @Description List apps
// @Tags App
// @Accept json
// @Produce json
// @Param query query model.ListAppsRequest false "Query parameters for filtering and pagination"
// @Success 200 {object} api.Response{data=model.ListAppsResponse}
// @Router /api/v1/apps [get]
func ListApps(c *gin.Context) {
	var req models.ListAppsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	as := services.NewAppService()
	resp, err := as.ListApps(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, resp)
}

// @Summary All App Refs Under Environment
// @Description Get all apps for refs under a specific environment
// @Tags App
// @Accept json
// @Produce json
// @Param query query model.AllAppRefsRequest false "Query parameters for filtering refs"
// @Success 200 {object} api.Response{data=[]model.AppRef}
// @Router /api/v1/apps/refs [get]
func AllAppRefs(c *gin.Context) {
	var req models.AllAppRefsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	as := services.NewAppService()
	refs, err := as.AllAppRefs(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, refs)
}

// @Summary Get App
// @Description Get app by app ID
// @Tags App
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Success 200 {object} api.Response{data=model.AppModel}
// @Router /api/v1/apps/{appID} [get]
func GetApp(c *gin.Context) {
	var req models.GetAppRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	as := services.NewAppService()
	app, err := as.GetApp(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, app)
}

// @Summary Get App Ref
// @Description Get app ref by app ID
// @Tags App
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Success 200 {object} api.Response{data=model.AppRef}
// @Router /api/v1/apps/{appID}/ref [get]
func GetAppRef(c *gin.Context) {
	var req models.GetAppRefRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	as := services.NewAppService()
	ref, err := as.GetAppRef(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, ref)
}

// @Summary Create App
// @Description Create a new app
// @Tags App
// @Accept json
// @Produce json
// @Param app body model.CreateAppRequest true "App data"
// @Success 201 {object} api.Response{data=model.AppModel}
// @Router /api/v1/apps [post]
func CreateApp(c *gin.Context) {
	var req models.CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	as := services.NewAppService()
	app, err := as.CreateApp(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Created(c, app)
}

// @Summary Update App
// @Description Update an existing app
// @Tags App
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param app body model.UpdateAppRequest true "Updated app data"
// @Success 200 {object} api.Response{data=model.AppModel}
// @Router /api/v1/apps/{appID} [put]
func UpdateApp(c *gin.Context) {
	var req models.UpdateAppRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	as := services.NewAppService()
	app, err := as.UpdateApp(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, app)
}

// @Summary Delete App
// @Description Delete an app by app ID
// @Tags App
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Success 204 {object} api.Response{}
// @Router /api/v1/apps/{appID} [delete]
func DeleteApp(c *gin.Context) {
	var req models.DeleteAppRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	as := services.NewAppService()
	if err := as.DeleteApp(c, &req); err != nil {
		api.Error(c, err)
		return
	}

	api.NoContent(c)
}

// @Summary Update App Image
// @Description Update the image of an app
// @Tags App
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param image body model.UpdateAppImageRequest true "New app image"
// @Success 200 {object} api.Response{data=model.AppModel}
// @Router /api/v1/apps/{appID}/image [put]
func UpdateAppImage(c *gin.Context) {
	appID := c.Param("appID")
	if appID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "App ID is required"))
		return
	}
	var req models.UpdateAppImageRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = appID

	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	as := services.NewAppService()
	app, err := as.UpdateAppImage(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, app)
}

// @Summary App Action
// @Description Perform an action on an app (e.g., deploy, restart)
// @Tags App
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param action body model.AppActionRequest true "Action to perform on the app"
// @Success 200 {object} api.Response{data=model.AppModel}
// @Router /api/v1/apps/{appID}/action [post]
func AppAction(c *gin.Context) {
	appID := c.Param("appID")
	if appID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "App ID is required"))
		return
	}
	var req models.AppActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = appID

	as := services.NewAppService()
	app, err := as.AppAction(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, app)
}

// @Summary List App Instances
// @Description List instances of an app
// @Tags App
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Success 200 {object} api.Response{data=[]model.AppInstanceModel}
// @Router /api/v1/apps/{appID}/instances [get]
func ListAppInstances(c *gin.Context) {
	appID := c.Param("appID")
	if appID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "App ID is required"))
		return
	}

	as := services.NewAppService()
	instances, err := as.ListAppInstances(c, &models.ListAppInstancesRequest{
		AppID: appID,
	})
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, instances)
}

// @Summary Terminate App Instance
// @Description Terminate a specific instance of an app
// @Tags App
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param request body model.TerminateAppInstanceRequest true "Terminate app instance request"
// @Success 204 {object} api.Response{}
// @Router /api/v1/apps/{appID}/instances/terminate [post]
func TerminateAppInstance(c *gin.Context) {
	appID := c.Param("appID")
	if appID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "App ID is required"))
		return
	}

	var req models.TerminateAppInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.AppID = appID

	as := services.NewAppService()
	if err := as.TerminateAppInstance(c, &req); err != nil {
		api.Error(c, err)
		return
	}

	api.NoContent(c)
}

// @Summary View App Container Logs
// @Description View logs of a specific container in an app instance
// @Tags App
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param instanceName path string true "Instance name"
// @Param containerName path string true "Container name"
// @Param request query model.ViewAppContainerLogsRequest false "Query parameters for viewing logs"
// @Success 200 {object} api.Response{}
// @Router /api/v1/apps/{appID}/instances/{instanceName}/containers/{containerName}/logs [get]
func ViewAppContainerLogs(c *gin.Context) {
	appID := c.Param("appID")
	if appID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "App ID is required"))
		return
	}

	instanceName := c.Param("instanceName")
	if instanceName == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Instance name is required"))
		return
	}

	containerName := c.Param("containerName")
	if containerName == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Container name is required"))
		return
	}

	var req models.ViewAppContainerLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, "Invalid request body"))
		return
	}
	req.AppID = appID
	req.InstanceName = instanceName
	req.ContainerName = containerName

	req.Request = c.Request
	req.ResponseWriter = c.Writer

	as := services.NewAppService()
	if err := as.ViewAppContainerLogs(c, &req); err != nil {
		api.Error(c, err)
		return
	}
}

// @Summary Exec App Container Terminal
// @Description Exec into a terminal of a specific container in an app instance
// @Tags App
// @Accept json
// @Produce json
// @Param appID path string true "App ID"
// @Param instanceName path string true "Instance name"
// @Param containerName path string true "Container name"
// @Param request body model.ExecAppContainerTerminalRequest true "Request to execute command in container"
// @Success 200 {object} api.Response{}
// @Router /api/v1/apps/{appID}/instances/{instanceName}/containers/{containerName}/exec [get]
func ExecAppContainerTerminal(c *gin.Context) {
	appID := c.Param("appID")
	if appID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "App ID is required"))
		return
	}

	instanceName := c.Param("instanceName")
	if instanceName == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Instance name is required"))
		return
	}

	containerName := c.Param("containerName")
	if containerName == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Container name is required"))
		return
	}
	var req models.ExecAppContainerTerminalRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, "Invalid request body"))
		return
	}
	req.AppID = appID
	req.InstanceName = instanceName
	req.ContainerName = containerName

	req.Request = c.Request
	req.ResponseWriter = c.Writer

	as := services.NewAppService()
	if err := as.ExecAppContainerTerminal(c, &req); err != nil {
		api.Error(c, err)
		return
	}
}
