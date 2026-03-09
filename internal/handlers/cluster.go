package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
	wsPkg "github.com/ketches/ketches/pkg/websocket"
	"golang.org/x/net/websocket"
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
			KubeConfig:             cl.KubeConfig,
			GatewayIP:              cl.GatewayIP,
			ConnectionStatus:       cl.ConnectionStatus,
			ConnectionStatusReason: cl.ConnectionStatusReason,
			LastCheckedAt:          cl.LastCheckedAt,
			CreatedAt:              cl.CreatedAt,
		})
	}

	log.Printf("Listing clusters: found %d records out of %d total", len(res), total)
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
	clusters, err := services.ListClustersSimple()
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}

	var result []models.SimpleCluster
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
		GatewayIP:              cluster.GatewayIP,
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

	api.Success(c, models.ClusterResponse{
		ID:                     cluster.ID,
		Slug:                   cluster.Slug,
		Name:                   cluster.Name,
		Description:            cluster.Description,
		Enabled:                cluster.Enabled,
		KubeConfig:             cluster.KubeConfig,
		GatewayIP:              cluster.GatewayIP,
		ConnectionStatus:       cluster.ConnectionStatus,
		ConnectionStatusReason: cluster.ConnectionStatusReason,
		LastCheckedAt:          cluster.LastCheckedAt,
		CreatedAt:              cluster.CreatedAt,
	})
}

func GetPublicCluster(c *gin.Context) {
	clusterID := c.Param("clusterID")
	cluster, err := services.GetSimpleCluster(clusterID)
	if err != nil {
		api.Error(c, http.StatusNotFound, err)
		return
	}

	if !cluster.Enabled {
		api.Error(c, http.StatusNotFound, fmt.Errorf("cluster not found"))
		return
	}

	api.Success(c, cluster)
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
		GatewayIP:              cluster.GatewayIP,
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
		GatewayIP:              cluster.GatewayIP,
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
		api.Error(c, http.StatusBadRequest, fmt.Errorf("namespace is required"))
		return
	}
	services, err := services.ListClusterServices(clusterID, namespace)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, services)
}

func ListStorageClasses(c *gin.Context) {
	clusterID := c.Param("clusterID")
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

	err = services.ExecClusterNodeTerminal(clusterID, nodeName, stdinReader, stdout, stderr)
	if err != nil {
		websocket.Message.Send(conn, []byte(fmt.Sprintf("Error: %v", err)))
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
		GatewayIP:              cluster.GatewayIP,
		ConnectionStatus:       cluster.ConnectionStatus,
		ConnectionStatusReason: cluster.ConnectionStatusReason,
		LastCheckedAt:          cluster.LastCheckedAt,
		CreatedAt:              cluster.CreatedAt,
	})
}

// GetClusterGatewayAPIStatus checks whether Gateway API CRDs are installed on the cluster.
func GetClusterGatewayAPIStatus(c *gin.Context) {
	clusterID := c.Param("clusterID")
	installed, err := core.ClusterHasGatewayAPICRDs(clusterID)
	if err != nil {
		api.Error(c, http.StatusInternalServerError, err)
		return
	}
	api.Success(c, gin.H{"installed": installed})
}
