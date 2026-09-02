package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/middlewares"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
	wsPkg "github.com/ketches/ketches/pkg/websocket"
)

func ListClusters(c *gin.Context) {
	var req models.PaginationRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	req.Validate()

	total, clusters, err := services.ListClusters(req.Page, req.PageSize, req.Search)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	res := []models.ClusterResponse{}
	for _, cl := range clusters {
		res = append(res, models.ClusterResponse{
			ID:                     cl.ID,
			Slug:                   cl.Slug,
			Name:                   cl.Name,
			Description:            cl.Description,
			Enabled:                cl.Enabled,
			ApiServer:              cl.ApiServer,
			GatewayHost:            cl.GatewayHost,
			HasKubeConfig:          cl.KubeConfig != "",
			ConnectionStatus:       cl.ConnectionStatus,
			ConnectionStatusReason: cl.ConnectionStatusReason,
			LastCheckedAt:          cl.LastCheckedAt,
			CreatedAt:              cl.CreatedAt,
		})
	}
	slog.Debug(fmt.Sprintf("Listing clusters: found %d records out of %d total", len(res), total))
	api.Success(c, models.ListClusterResponse{
		Items:      res,
		Pagination: models.BuildPaginationResponse(total, req.Page, req.PageSize),
	})
}

func ListClustersSimple(c *gin.Context) {
	clusters, err := services.ListClustersSimple()
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, clusters)
}

func ListPublicClusters(c *gin.Context) {
	claims := api.GetClaims(c)
	if claims == nil {
		api.Error(c, http.StatusUnauthorized, app.NewErrorf("unauthorized"))
		return
	}

	projectID := queryProjectID(c)
	if claims.Role != app.UserRoleAdmin {
		if requireProjectAccess(c, projectID) == nil {
			return
		}
	}

	clusters, err := services.ListClustersSimple()
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	result := make([]models.SimpleCluster, 0, len(clusters))
	for _, cl := range clusters {
		if cl.Enabled {
			result = append(result, cl)
		}
	}

	api.Success(c, result)
}

func CreateCluster(c *gin.Context) {
	var req models.CreateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	cluster, err := services.CreateCluster(&req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Created(c, models.ClusterResponse{
		ID:                     cluster.ID,
		Slug:                   cluster.Slug,
		Name:                   cluster.Name,
		Description:            cluster.Description,
		Enabled:                cluster.Enabled,
		ApiServer:              cluster.ApiServer,
		GatewayHost:            cluster.GatewayHost,
		HasKubeConfig:          cluster.KubeConfig != "",
		ConnectionStatus:       cluster.ConnectionStatus,
		ConnectionStatusReason: cluster.ConnectionStatusReason,
		LastCheckedAt:          cluster.LastCheckedAt,
		CreatedAt:              cluster.CreatedAt,
	})
}

func GetCluster(c *gin.Context) {
	clusterID := c.Param("clusterID")
	cluster, err := services.GetCluster(clusterID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	hasPrometheusIntegration, err := services.HasPrometheusIntegration(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ClusterResponse{
		ID:                       cluster.ID,
		Slug:                     cluster.Slug,
		Name:                     cluster.Name,
		Description:              cluster.Description,
		Enabled:                  cluster.Enabled,
		ApiServer:                cluster.ApiServer,
		GatewayHost:              cluster.GatewayHost,
		HasKubeConfig:            cluster.KubeConfig != "",
		HasPrometheusIntegration: hasPrometheusIntegration,
		ConnectionStatus:         cluster.ConnectionStatus,
		ConnectionStatusReason:   cluster.ConnectionStatusReason,
		LastCheckedAt:            cluster.LastCheckedAt,
		CreatedAt:                cluster.CreatedAt,
	})
}

func GetPublicCluster(c *gin.Context) {
	clusterID := c.Param("clusterID")
	projectID := queryProjectID(c)

	if requireClusterProjectAccess(c, projectID, clusterID) == nil {
		return
	}

	cluster, err := services.GetSimpleCluster(clusterID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	if !cluster.Enabled {
		api.Error(c, http.StatusNotFound, app.NewErrorf("cluster not found"))
		return
	}

	hasPrometheusIntegration, err := services.HasPrometheusIntegration(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, gin.H{
		"id":                         cluster.ID,
		"slug":                       cluster.Slug,
		"name":                       cluster.Name,
		"description":                cluster.Description,
		"enabled":                    cluster.Enabled,
		"connection_status":          cluster.ConnectionStatus,
		"has_prometheus_integration": hasPrometheusIntegration,
	})
}

func UpdateCluster(c *gin.Context) {
	clusterID := c.Param("clusterID")
	var req models.UpdateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	cluster, err := services.UpdateCluster(clusterID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ClusterResponse{
		ID:                     cluster.ID,
		Slug:                   cluster.Slug,
		Name:                   cluster.Name,
		Description:            cluster.Description,
		Enabled:                cluster.Enabled,
		ApiServer:              cluster.ApiServer,
		GatewayHost:            cluster.GatewayHost,
		HasKubeConfig:          cluster.KubeConfig != "",
		ConnectionStatus:       cluster.ConnectionStatus,
		ConnectionStatusReason: cluster.ConnectionStatusReason,
		LastCheckedAt:          cluster.LastCheckedAt,
		CreatedAt:              cluster.CreatedAt,
	})
}

func UpdateClusterBasic(c *gin.Context) {
	clusterID := c.Param("clusterID")
	var req models.UpdateBasicInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	cluster, err := services.UpdateClusterBasic(clusterID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ClusterResponse{
		ID:                     cluster.ID,
		Slug:                   cluster.Slug,
		Name:                   cluster.Name,
		Description:            cluster.Description,
		Enabled:                cluster.Enabled,
		ApiServer:              cluster.ApiServer,
		GatewayHost:            cluster.GatewayHost,
		HasKubeConfig:          cluster.KubeConfig != "",
		ConnectionStatus:       cluster.ConnectionStatus,
		ConnectionStatusReason: cluster.ConnectionStatusReason,
		LastCheckedAt:          cluster.LastCheckedAt,
		CreatedAt:              cluster.CreatedAt,
	})
}

func CreateClusterGatewayProvider(c *gin.Context) {
	clusterID := c.Param("clusterID")
	var req models.CreateClusterGatewayProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}
	provider, err := services.CreateClusterGatewayProvider(clusterID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Created(c, models.ClusterGatewayProvider{
		ID:               provider.ID,
		ClusterID:        provider.ClusterID,
		SourceType:       provider.SourceType,
		DisplayName:      provider.DisplayName,
		GatewayClassName: provider.GatewayClassName,
		ControllerName:   provider.ControllerName,
		IsDefault:        provider.IsDefault,
	})
}

func DeleteClusterGatewayProvider(c *gin.Context) {
	clusterID := c.Param("clusterID")
	providerID := c.Param("providerID")
	if err := services.DeleteClusterGatewayProvider(clusterID, providerID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func ListClusterGatewayProviders(c *gin.Context) {
	clusterID := c.Param("clusterID")
	items, err := services.ListClusterGatewayProviders(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, items)
}

func ListClusterGatewayClasses(c *gin.Context) {
	clusterID := c.Param("clusterID")
	items, err := services.ListClusterGatewayClasses(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, items)
}

func UpdateClusterDefaultGatewayClass(c *gin.Context) {
	clusterID := c.Param("clusterID")
	var req models.UpdateClusterGatewayClassRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	cluster, err := services.UpdateClusterDefaultGatewayClass(clusterID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ClusterResponse{
		ID:                     cluster.ID,
		Slug:                   cluster.Slug,
		Name:                   cluster.Name,
		Description:            cluster.Description,
		Enabled:                cluster.Enabled,
		ApiServer:              cluster.ApiServer,
		GatewayHost:            cluster.GatewayHost,
		HasKubeConfig:          cluster.KubeConfig != "",
		ConnectionStatus:       cluster.ConnectionStatus,
		ConnectionStatusReason: cluster.ConnectionStatusReason,
		LastCheckedAt:          cluster.LastCheckedAt,
		CreatedAt:              cluster.CreatedAt,
	})
}

func DeleteCluster(c *gin.Context) {
	clusterID := c.Param("clusterID")
	if err := services.DeleteCluster(clusterID); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.NoContent(c)
}

func ListClusterNodes(c *gin.Context) {
	clusterID := c.Param("clusterID")
	nodes, err := services.ListClusterNodes(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, nodes)
}

func GetClusterNode(c *gin.Context) {
	clusterID := c.Param("clusterID")
	nodeName := c.Param("nodeName")
	node, err := services.GetClusterNode(clusterID, nodeName)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, node)
}

func UpdateClusterNodeLabels(c *gin.Context) {
	clusterID := c.Param("clusterID")
	nodeName := c.Param("nodeName")
	var req models.UpdateNodeLabelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.UpdateClusterNodeLabels(clusterID, nodeName, req.Labels); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, nil)
}

func UpdateClusterNodeAnnotations(c *gin.Context) {
	clusterID := c.Param("clusterID")
	nodeName := c.Param("nodeName")
	var req models.UpdateNodeAnnotationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.UpdateClusterNodeAnnotations(clusterID, nodeName, req.Annotations); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, nil)
}

func UpdateClusterNodeTaints(c *gin.Context) {
	clusterID := c.Param("clusterID")
	nodeName := c.Param("nodeName")
	var req models.UpdateNodeTaintsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.UpdateClusterNodeTaints(clusterID, nodeName, req.Taints); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, nil)
}

func CordonClusterNode(c *gin.Context) {
	clusterID := c.Param("clusterID")
	nodeName := c.Param("nodeName")
	var req models.CordonNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.CordonClusterNode(clusterID, nodeName, req.Cordon); err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, nil)
}

func ListClusterNamespaces(c *gin.Context) {
	clusterID := c.Param("clusterID")
	namespaces, err := services.ListClusterNamespaces(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, namespaces)
}

func ListClusterServices(c *gin.Context) {
	clusterID := c.Param("clusterID")
	namespace := c.Query("namespace")
	if namespace == "" {
		api.Error(c, http.StatusBadRequest, app.NewErrorf("namespace is required"))
		return
	}

	withPorts := c.Query("with_ports")
	if withPorts == "1" || withPorts == "true" {
		serviceDetails, err := services.ListClusterServicesWithPorts(clusterID, namespace)
		if err != nil {
			api.Error(c, http.StatusInternalServerError, err)
			return
		}
		api.Success(c, serviceDetails)
		return
	}

	serviceNames, err := services.ListClusterServices(clusterID, namespace)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, serviceNames)
}

func ListStorageClasses(c *gin.Context) {
	clusterID := c.Param("clusterID")
	projectID := queryProjectID(c)

	if requireClusterProjectAccess(c, projectID, clusterID) == nil {
		return
	}

	storageClasses, err := services.ListStorageClasses(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, storageClasses)
}

func ExecClusterNodeTerminal(c *gin.Context) {
	clusterID := c.Param("clusterID")
	nodeName := c.Param("nodeName")

	w := c.Writer
	r := c.Request

	conn, err := wsPkg.NewConn(w, r, wsPkg.OriginValidator(middlewares.IsAllowedOrigin))
	if err != nil {
		_ = c.Error(app.NewErrorf("failed to upgrade to websocket: %v", err))
		return
	}
	defer func() {
		_ = conn.Close()
	}()

	sizeQueue := wsPkg.NewTerminalSizeQueue()
	defer sizeQueue.Close()

	stdinReader := wsPkg.NewDemuxReader(conn, sizeQueue)
	defer func() {
		_ = stdinReader.Close()
	}()

	stdout, stderr := wsPkg.NewWriterPair(conn)
	defer stdout.Close()
	defer stderr.Close()

	execCtx, cancel := context.WithTimeout(r.Context(), wsPkg.SessionTimeout)
	defer cancel()
	go func() {
		select {
		case <-stdinReader.Done():
			cancel()
		case <-stdout.Done():
			cancel()
		case <-execCtx.Done():
			_ = conn.Close()
		}
	}()

	err = services.ExecClusterNodeTerminal(execCtx, clusterID, nodeName, stdinReader, stdout, stderr)
	if err != nil && execCtx.Err() == nil {
		if _, sendErr := stderr.Write([]byte(fmt.Sprintf("Error: %v", err))); sendErr != nil {
			_ = c.Error(sendErr)
		}
	}
}

func PingCluster(c *gin.Context) {
	var req models.PingClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := services.PingCluster(&req); err != nil {
		api.Error(c, http.StatusBadGateway, err)
		return
	}

	api.Success(c, gin.H{"message": "Connection successful"})
}

func CheckClusterConnectivity(c *gin.Context) {
	clusterID := c.Param("clusterID")

	services.CheckClusterConnectivity(clusterID)

	api.Success(c, gin.H{"message": "Connectivity check started"})
}

func CheckAllClustersConnectivity(c *gin.Context) {
	services.CheckAllClustersConnectivity()

	api.Success(c, gin.H{"message": "Connectivity check started for all clusters"})
}

func UpdateClusterCredentials(c *gin.Context) {
	clusterID := c.Param("clusterID")
	var req models.UpdateClusterCredentialsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, http.StatusBadRequest, err)
		return
	}

	cluster, err := services.UpdateClusterCredentials(clusterID, &req)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	api.Success(c, models.ClusterResponse{
		ID:                     cluster.ID,
		Slug:                   cluster.Slug,
		Name:                   cluster.Name,
		Description:            cluster.Description,
		Enabled:                cluster.Enabled,
		ApiServer:              cluster.ApiServer,
		GatewayHost:            cluster.GatewayHost,
		HasKubeConfig:          cluster.KubeConfig != "",
		ConnectionStatus:       cluster.ConnectionStatus,
		ConnectionStatusReason: cluster.ConnectionStatusReason,
		LastCheckedAt:          cluster.LastCheckedAt,
		CreatedAt:              cluster.CreatedAt,
	})
}

// GetClusterGatewayAPIStatus checks whether Gateway API CRDs are installed on the cluster.
func GetClusterGatewayAPIStatus(c *gin.Context) {
	clusterID := c.Param("clusterID")
	projectID := queryProjectID(c)

	if requireClusterProjectAccess(c, projectID, clusterID) == nil {
		return
	}

	installed, err := core.ClusterHasGatewayAPICRDs(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, gin.H{"installed": installed})
}
