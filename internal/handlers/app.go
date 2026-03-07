package handlers

import (
	"bufio"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
	"github.com/ketches/ketches/pkg/containerregistry"
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

	app, err := services.GetApp(c.Request.Context(), appID)
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

	app, err := services.GetApp(c.Request.Context(), appID)
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

	res := make([]models.AppResponse, len(apps))
	var wg sync.WaitGroup
	for i, a := range apps {
		wg.Add(1)
		go func(i int, a models.AppListRow) {
			defer wg.Done()
			res[i] = services.ToAppListResponse(c.Request.Context(), &a)
		}(i, a)
	}
	wg.Wait()

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

	res := make([]models.SimpleResponse, len(apps))
	var wg sync.WaitGroup
	for i, a := range apps {
		wg.Add(1)
		go func(i int, a entities.App) {
			defer wg.Done()
			res[i] = models.SimpleResponse{
				ID:          a.ID,
				Slug:        a.Slug,
				Name:        a.Name,
				Description: a.Description,
				Status:      a.DeployStatus,
				Metadata: map[string]string{
					"code_repository_id": derefString(a.CodeRepositoryID),
				},
			}
		}(i, a)
	}
	wg.Wait()

	api.Success(c, res)
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func CreateApp(c *gin.Context) {
	envID := c.Param("envID")
	var req models.CreateAppRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.CreateApp(c.Request.Context(), envID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, services.ToAppResponse(c.Request.Context(), app))
}

func GetApp(c *gin.Context) {
	appID := c.Param("appID")
	app, err := services.GetApp(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	api.Success(c, services.ToAppResponse(c.Request.Context(), app))
}

func UpdateAppBasic(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateBasicInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppBasic(c.Request.Context(), appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, services.ToAppResponse(c.Request.Context(), app))
}

func UpdateAppImage(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateAppImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppImage(c.Request.Context(), appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, services.ToAppResponse(c.Request.Context(), app))
}

func UpdateAppReplicas(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateAppReplicasRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppReplicas(c.Request.Context(), appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, services.ToAppResponse(c.Request.Context(), app))
}

func UpdateAppResources(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateAppResourcesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppResources(c.Request.Context(), appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, services.ToAppResponse(c.Request.Context(), app))
}

func UpdateAppAutoScaling(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateAppAutoScalingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppAutoScaling(c.Request.Context(), appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, services.ToAppResponse(c.Request.Context(), app))
}

func UpdateAppHealth(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateAppHealthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppHealth(c.Request.Context(), appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, services.ToAppResponse(c.Request.Context(), app))
}

func UpdateAppScheduling(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateAppSchedulingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppScheduling(c.Request.Context(), appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, services.ToAppResponse(c.Request.Context(), app))
}

func UpdateAppCommand(c *gin.Context) {
	appID := c.Param("appID")
	var req models.UpdateAppCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.UpdateAppCommand(c.Request.Context(), appID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, services.ToAppResponse(c.Request.Context(), app))
}

func DeleteApp(c *gin.Context) {
	appID := c.Param("appID")
	if err := services.DeleteApp(c.Request.Context(), appID); err != nil {
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

	if err := services.BatchDeleteApps(c.Request.Context(), req.IDs); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func ListAppInstances(c *gin.Context) {
	appID := c.Param("appID")
	instances, err := services.ListAppInstances(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, instances)
}

func ListAppInstanceEvents(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")

	app, err := services.GetApp(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	events, err := services.ListAppInstanceEvents(c.Request.Context(), app, instanceName)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, events)
}

func DeleteAppInstance(c *gin.Context) {
	appID := c.Param("appID")
	instanceName := c.Param("instanceName")

	app, err := services.GetApp(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	if err := services.DeleteAppInstance(c.Request.Context(), app, instanceName); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.NoContent(c)
}

func ApplyApp(c *gin.Context) {
	appID := c.Param("appID")
	app, err := services.GetApp(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	if err := services.ApplyApp(c.Request.Context(), app); err != nil {
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

	_, err := services.AppAction(c.Request.Context(), appID, req.Action)
	if err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	app, err := services.GetApp(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	status := services.GetAppStatus(c.Request.Context(), app)

	api.Success(c, models.AppActionResponse{
		Status: status,
	})
}

func GetAppAvailableActions(c *gin.Context) {
	appID := c.Param("appID")
	app, err := services.GetApp(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	status := services.GetAppStatus(c.Request.Context(), app)

	actions := core.GetAvailableActions(status)
	api.Success(c, models.AvailableActionsResponse{
		Actions: actions,
	})
}

func GetAppTopology(c *gin.Context) {
	appID := c.Param("appID")
	topology, err := services.GetAppTopology(c.Request.Context(), appID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, topology)
}

func GetAppTopologyResourceYaml(c *gin.Context) {
	appID := c.Param("appID")
	nodeID := c.Param("nodeID")
	yamlStr, err := services.GetAppTopologyResourceYaml(c.Request.Context(), appID, nodeID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}
	api.Success(c, gin.H{"yaml": yamlStr})
}

// GetImageMetadata fetches and returns container image metadata (ENV, VOLUME, EXPOSE, HEALTHCHECK).
// This is a read-only preview endpoint with no database side effects.
// Query params: image (required), registry_username (optional), registry_password (optional)
func GetImageMetadata(c *gin.Context) {
	image := c.Query("image")
	if image == "" {
		api.Error(c, http.StatusBadRequest, fmt.Errorf("image query parameter is required"))
		return
	}

	username := c.Query("registry_username")
	password := c.Query("registry_password")

	meta, err := containerregistry.FetchImageMetadata(c.Request.Context(), image, username, password)
	if err != nil {
		api.Error(c, http.StatusBadGateway, fmt.Errorf("failed to fetch image metadata: %w", err))
		return
	}

	resp := models.ImageMetadataResponse{}

	for _, ev := range meta.Env {
		resp.Env = append(resp.Env, models.ImageEnvVar{Key: ev.Key, Value: ev.Value})
	}
	resp.Volumes = meta.Volumes
	for _, pi := range meta.ExposedPorts {
		resp.ExposedPorts = append(resp.ExposedPorts, models.ImagePortInfo{Port: pi.Port, Protocol: pi.Protocol})
	}
	if hc := meta.HealthCheck; hc != nil {
		resp.HealthCheck = &models.ImageHealthCheck{
			Test:     hc.Test,
			Interval: int(hc.Interval.Seconds()),
			Timeout:  int(hc.Timeout.Seconds()),
			Retries:  hc.Retries,
		}
	}

	api.Success(c, resp)
}
