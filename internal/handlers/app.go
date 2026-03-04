package handlers

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
	wsPkg "github.com/ketches/ketches/pkg/websocket"
	"golang.org/x/net/websocket"
)

func StreamAppLogs(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")

	if containerName == "" {
		api.Error(c, http.StatusBadRequest, fmt.Errorf("container parameter is required"))
		return
	}

	tailLines := int64(1000)
	if lines := c.Query("tailLines"); lines != "" {
		if parsed, err := strconv.ParseInt(lines, 10, 64); err == nil && parsed > 0 {
			tailLines = parsed
		}
	}

	timestamps := false
	if ts := c.Query("timestamps"); ts == "true" {
		timestamps = true
	}

	app, err := services.GetApp(appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, fmt.Errorf("app not found: %v", err))
		return
	}

	w := c.Writer
	r := c.Request
	flusher, ok := w.(http.Flusher)
	if !ok {
		api.Error(c, http.StatusInternalServerError, fmt.Errorf("streaming unsupported"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	stream, err := services.StreamAppLogs(r.Context(), app, instanceName, containerName, tailLines, timestamps)
	if err != nil {
		fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
		flusher.Flush()
		return
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)

	for {
		select {
		case <-r.Context().Done():
			return
		default:
			if scanner.Scan() {
				txt := scanner.Text()
				fmt.Fprintf(w, "data: %s\n\n", txt)
				flusher.Flush()
			} else {
				return
			}
		}
	}
}

func ExecAppContainerTerminal(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")
	containerName := c.Query("container")

	app, err := services.GetApp(appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	w := c.Writer
	r := c.Request

	conn, err := wsPkg.NewConn(w, r)
	if err != nil {
		c.Error(fmt.Errorf("failed to upgrade to websocket: %v", err))
		return
	}
	defer conn.Close()

	sizeQueue := wsPkg.NewTerminalSizeQueue()
	defer sizeQueue.Close()

	stdinReader := wsPkg.NewDemuxReader(conn, sizeQueue)
	defer stdinReader.Close()

	stdout := wsPkg.NewWriter(conn)
	stderr := wsPkg.NewWriter(conn)

	err = services.ExecAppContainer(app, instanceName, containerName, stdinReader, stdout, stderr, true, sizeQueue)
	if err != nil {
		websocket.Message.Send(conn, []byte(fmt.Sprintf("Error: %v", err)))
	}
}

func ListApps(c *gin.Context) {
	envID := c.Param("envID")

	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, apps, err := services.ListApps(envID, req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	res := []models.AppResponse{}
	for _, a := range apps {
		res = append(res, toAppResponse(c, &a))
	}

	api.Success(c, models.ListAppResponse{
		Items:      res,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func ListAppsSimple(c *gin.Context) {
	envID := c.Param("envID")
	apps, err := services.ListAppsSimple(envID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	res := []models.SimpleResponse{}
	for _, a := range apps {
		codeRepoID := ""
		if a.CodeRepositoryID != nil {
			codeRepoID = *a.CodeRepositoryID
		}
		res = append(res, models.SimpleResponse{
			ID:          a.ID,
			Slug:        a.Slug,
			Name:        a.Name,
			Description: a.Description,
			Status:      getAppStatus(c, &a),
			Metadata: map[string]string{
				"code_repository_id": codeRepoID,
			},
		})
	}

	api.Success(c, res)
}

func toAppResponse(c *gin.Context, a *entities.App) models.AppResponse {
	status := getAppStatus(c, a)

	codeRepoID := ""
	if a.CodeRepositoryID != nil {
		codeRepoID = *a.CodeRepositoryID
	}
	res := models.AppResponse{
		ID:               a.ID,
		Slug:             a.Slug,
		Name:             a.Name,
		Description:      a.Description,
		EnvID:            a.EnvID,
		AppType:          a.AppType,
		CodeRepositoryID: codeRepoID,
		ContainerImage:   a.ContainerImage,
		ContainerCommand: a.ContainerCommand,
		RegistryUsername: a.RegistryUsername,
		RegistryPassword: a.RegistryPassword,
		Replicas:         a.Replicas,
		RequestCPU:       a.RequestCPU,
		RequestMemory:    a.RequestMemory,
		LimitCPU:         a.LimitCPU,
		LimitMemory:      a.LimitMemory,
		Status:           status,
		CreatedAt:        a.CreatedAt,
	}

	if a.AutoScaling != nil {
		res.AutoScaling = &models.AutoScalingSpec{
			MinReplicas:             a.AutoScaling.MinReplicas,
			MaxReplicas:             a.AutoScaling.MaxReplicas,
			TargetCPUUtilization:    a.AutoScaling.TargetCPUUtilization,
			TargetMemoryUtilization: a.AutoScaling.TargetMemoryUtilization,
		}
	}

	if a.SchedulingRule != nil {
		res.SchedulingRule = &models.SchedulingSpec{
			RuleType:     a.SchedulingRule.RuleType,
			NodeName:     a.SchedulingRule.NodeName,
			NodeSelector: a.SchedulingRule.NodeSelector,
			NodeAffinity: a.SchedulingRule.NodeAffinity,
			Tolerations:  a.SchedulingRule.Tolerations,
		}
	}

	return res
}

func CreateApp(c *gin.Context) {
	envID := c.Param("envID")
	var req models.CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.CreateApp(envID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, toAppResponse(c, app))
}

func GetApp(c *gin.Context) {
	appID := c.Param("appID")
	app, err := services.GetApp(appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	api.Success(c, toAppResponse(c, app))
}

func UpdateAppBasic(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateBasicInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppBasic(appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, toAppResponse(c, app))
}

func UpdateAppImage(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateAppImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppImage(appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, toAppResponse(c, app))
}

func UpdateAppReplicas(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateAppReplicasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppReplicas(appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, toAppResponse(c, app))
}

func UpdateAppResources(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateAppResourcesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppResources(appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, toAppResponse(c, app))
}

func UpdateAppAutoScaling(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateAppAutoScalingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppAutoScaling(appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, toAppResponse(c, app))
}

func UpdateAppHealth(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateAppHealthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppHealth(appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, toAppResponse(c, app))
}

func UpdateAppScheduling(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateAppSchedulingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppScheduling(appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, toAppResponse(c, app))
}

func UpdateAppCommand(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateAppCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppCommand(appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, toAppResponse(c, app))
}

func DeleteApp(c *gin.Context) {
	appID := c.Param("appID")
	if err := services.DeleteApp(appID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func BatchDeleteApps(c *gin.Context) {
	var req struct {
		IDs []string `json:"ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.BatchDeleteApps(req.IDs); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func ListAppInstances(c *gin.Context) {
	appID := c.Param("appID")
	instances, err := services.ListAppInstances(appID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, instances)
}

func ListAppInstanceEvents(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")

	app, err := services.GetApp(appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	events, err := services.ListAppInstanceEvents(app, instanceName)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, events)
}

func DeleteAppInstance(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")

	app, err := services.GetApp(appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	if err := services.DeleteAppInstance(app, instanceName); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func ApplyApp(c *gin.Context) {
	appID := c.Param("appID")
	app, err := services.GetApp(appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	if err := services.ApplyApp(app); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func AppAction(c *gin.Context) {
	appID := c.Param("appID")
	var req models.AppActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	_, err := services.AppAction(appID, req.Action)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.GetApp(appID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	status := getAppStatus(c, app)

	api.Success(c, models.AppActionResponse{
		Status: status,
	})
}

func GetAppAvailableActions(c *gin.Context) {
	appID := c.Param("appID")
	app, err := services.GetApp(appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	status := getAppStatus(c, app)

	actions := core.GetAvailableActions(status)
	api.Success(c, models.AvailableActionsResponse{
		Actions: actions,
	})
}

func GetAppTopology(c *gin.Context) {
	appID := c.Param("appID")
	topology, err := services.GetAppTopology(appID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, topology)
}

func GetAppTopologyResourceYaml(c *gin.Context) {
	appID := c.Param("appID")
	nodeID := c.Param("nodeID")
	yamlStr, err := services.GetAppTopologyResourceYaml(appID, nodeID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	api.Success(c, gin.H{"yaml": yamlStr})
}

func getAppStatus(c *gin.Context, app *entities.App) string {
	status := app.DeployStatus
	if status == "deployed" {
		calculatedStatus, err := core.CalculateAppStatus(c.Request.Context(), app)
		if err != nil {
			log.Printf("Failed to calculate app status for app %s: %v", app.ID, err)
		}
		status = string(calculatedStatus)
	}
	return status
}
