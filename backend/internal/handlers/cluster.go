package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/services"
)

// @Summary List Clusters
// @Description List clusters
// @Tags Cluster
// @Accept json
// @Produce json
// @Param query query models.ListClustersRequest false "Query parameters for filtering and pagination"
// @Success 200 {object} api.Response{data=models.ListClustersResponse}
// @Router /api/v1/clusters [get]
func ListClusters(c *gin.Context) {
	var req models.ListClustersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	s := services.NewClusterService()
	resp, err := s.ListClusters(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, resp)
}

// @Summary All Cluster Refs
// @Description Get all clusters for refs
// @Tags Cluster
// @Accept json
// @Produce json
// @Success 200 {object} api.Response{data=[]models.ClusterRef}
// @Router /api/v1/clusters/refs [get]
func AllClusterRefs(c *gin.Context) {
	s := services.NewClusterService()
	refs, err := s.AllClusterRefs(c)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, refs)
}

// @Summary Get Cluster
// @Description Get cluster by cluster ID
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Success 200 {object} api.Response{data=models.ClusterModel}
// @Router /api/v1/clusters/{clusterID} [get]
func GetCluster(c *gin.Context) {
	clusterID := c.Param("clusterID")
	if clusterID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Cluster ID is required"))
		return
	}

	s := services.NewClusterService()
	cluster, err := s.GetCluster(c, &models.GetClusterRequest{
		ClusterID: clusterID,
	})
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, cluster)
}

// @Summary Get Cluster Ref
// @Description Get cluster ref by cluster ID
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Success 200 {object} api.Response{data=models.ClusterRef}
// @Router /api/v1/clusters/{clusterID}/ref [get]
func GetClusterRef(c *gin.Context) {
	var req models.GetClusterRefRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	s := services.NewClusterService()
	ref, err := s.GetClusterRef(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, ref)
}

// @Summary Create Cluster
// @Description Create a new cluster
// @Tags Cluster
// @Accept json
// @Produce json
// @Param request body models.CreateClusterRequest true "Cluster information"
// @Success 201 {object} api.Response{data=models.ClusterModel}
// @Router /api/v1/clusters [post]
func CreateCluster(c *gin.Context) {
	var req models.CreateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	s := services.NewClusterService()
	cluster, err := s.CreateCluster(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Created(c, cluster)
}

// @Summary Update Cluster
// @Description Update an existing cluster
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Param request body models.UpdateClusterRequest true "Updated cluster information"
// @Success 200 {object} api.Response{data=models.ClusterModel}
// @Router /api/v1/clusters/{clusterID} [put]
func UpdateCluster(c *gin.Context) {
	clusterID := c.Param("clusterID")
	if clusterID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Cluster ID is required"))
		return
	}

	var req models.UpdateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.ClusterID = clusterID

	s := services.NewClusterService()
	cluster, err := s.UpdateCluster(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, cluster)
}

// @Summary Delete Cluster
// @Description Delete a cluster by cluster ID
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Success 204
// @Router /api/v1/clusters/{clusterID} [delete]
func DeleteCluster(c *gin.Context) {
	clusterID := c.Param("clusterID")
	if clusterID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Cluster ID is required"))
		return
	}

	s := services.NewClusterService()
	if err := s.DeleteCluster(c, &models.DeleteClusterRequest{
		ClusterID: clusterID,
	}); err != nil {
		api.Error(c, err)
		return
	}

	api.NoContent(c)
}

// @Summary Enable Cluster
// @Description Enable a cluster by cluster ID
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Success 200 {object} api.Response{}
// @Router /api/v1/clusters/{clusterID}/enable [put]
func EnableCluster(c *gin.Context) {
	clusterID := c.Param("clusterID")
	if clusterID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Cluster ID is required"))
		return
	}

	var req models.EnabledClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.ClusterID = clusterID

	s := services.NewClusterService()
	err := s.EnableCluster(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, nil)
}

// @Summary Disable Cluster
// @Description Disable a cluster by cluster ID
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Success 200 {object} api.Response{}
// @Router /api/v1/clusters/{clusterID}/disable [put]
func DisableCluster(c *gin.Context) {
	clusterID := c.Param("clusterID")
	if clusterID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Cluster ID is required"))
		return
	}

	var req models.DisableClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.ClusterID = clusterID

	s := services.NewClusterService()
	err := s.DisableCluster(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, nil)
}

// @Summary Ping Cluster KubeConfig
// @Description Ping a cluster's KubeConfig to check if it is connectable
// @Tags Cluster
// @Accept json
// @Produce json
// @Param request body models.PingClusterKubeConfigRequest true "KubeConfig to ping"
// @Success 200 {object} api.Response{data=bool}
// @Router /api/v1/clusters/ping [post]
func PingClusterKubeConfig(c *gin.Context) {
	var req models.PingClusterKubeConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	s := services.NewClusterService()
	result := s.PingClusterKubeConfig(c, &req)
	api.Success(c, result)
}

// @Summary List nodes of a cluster
// @Description Get all nodes of the specified cluster
// @Tags Cluster
// @Param clusterID path string true "Cluster ID"
// @Success 200 {object} api.Response{data=[]models.ClusterNodeModel}
// @Router /api/v1/clusters/{clusterID}/nodes [get]
func ListClusterNodes(c *gin.Context) {
	svc := services.NewClusterService()
	nodes, err := svc.ListClusterNodes(c.Request.Context(), &models.ListClusterNodesRequest{
		ClusterID: c.Param("clusterID"),
	})
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, nodes)
}

// @Summary List references of cluster nodes
// @Description Get all node references of the specified cluster
// @Tags Cluster
// @Param clusterID path string true "Cluster ID"
// @Success 200 {object} api.Response{data=[]models.ClusterNodeRef}
// @Router /api/v1/clusters/{clusterID}/nodes/refs [get]
func ListClusterNodeRefs(c *gin.Context) {
	svc := services.NewClusterService()
	refs, err := svc.ListClusterNodeRefs(c.Request.Context(), &models.ListClusterNodeRefsRequest{
		ClusterID: c.Param("clusterID"),
	})
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, refs)
}

// @Summary Get Cluster Node
// @Description Get details of a specific node in the cluster
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Param nodeName path string true "Node Name"
// @Success 200 {object} api.Response{data=models.ClusterNodeModel}
// @Router /api/v1/clusters/{clusterID}/nodes/{nodeName} [get]
func GetClusterNode(c *gin.Context) {
	svc := services.NewClusterService()
	node, err := svc.GetClusterNode(c.Request.Context(), &models.GetClusterNodeRequest{
		ClusterID: c.Param("clusterID"),
		NodeName:  c.Param("nodeName"),
	})
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, node)
}

// @Summary List Cluster Extensions
// @Description List extensions available in a cluster
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Success 200 {object} api.Response{data=models.ListClusterExtensionsResponse}
// @Router /api/v1/clusters/{clusterID}/extensions [get]
func ListClusterExtensions(c *gin.Context) {
	clusterID := c.Param("clusterID")
	if clusterID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Cluster ID is required"))
		return
	}

	var req models.ListClusterExtensionsRequest
	req.ClusterID = clusterID

	s := services.NewClusterService()
	resp, err := s.ListClusterExtensions(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, resp)
}

// @Summary List Cluster Node Labels
// @Description Get all node labels from the specified cluster
// @Tags Cluster
// @Param clusterID path string true "Cluster ID"
// @Success 200 {object} api.Response{data=[]string}
// @Router /api/v1/clusters/{clusterID}/nodes/labels [get]
func ListClusterNodeLabels(c *gin.Context) {
	svc := services.NewClusterService()
	labels, err := svc.ListClusterNodeLabels(c.Request.Context(), &models.ListClusterNodeLabelsRequest{
		ClusterID: c.Param("clusterID"),
	})
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, labels)
}

// @Summary List Cluster Node Taints
// @Description Get all node taints from the specified cluster
// @Tags Cluster
// @Param clusterID path string true "Cluster ID"
// @Success 200 {object} api.Response{data=[]models.ClusterNodeTaintModel}
// @Router /api/v1/clusters/{clusterID}/nodes/taints [get]
func ListClusterNodeTaints(c *gin.Context) {
	svc := services.NewClusterService()
	taints, err := svc.ListClusterNodeTaints(c.Request.Context(), &models.ListClusterNodeTaintsRequest{
		ClusterID: c.Param("clusterID"),
	})
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, taints)
}

// @Summary Check Cluster Extension Feature Enabled
// @Description Check if the cluster extension feature is enabled for the specified cluster
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Success 200 {object} api.Response{data=bool}
// @Router /api/v1/clusters/{clusterID}/extensions/feature-enabled [get]
func CheckClusterExtensionFeatureEnabled(c *gin.Context) {
	svc := services.NewClusterService()
	enabled, err := svc.CheckClusterExtensionFeatureEnabled(c.Request.Context(), &models.CheckClusterExtensionFeatureEnabledRequest{
		ClusterID: c.Param("clusterID"),
	})
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, enabled)
}

// @Summary Enable Cluster Extension
// @Description Enable the cluster extension feature for the specified cluster
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Success 200 {object} api.Response{}
// @Router /api/v1/clusters/{clusterID}/extensions/enable [post]
func EnableClusterExtension(c *gin.Context) {
	svc := services.NewClusterService()
	err := svc.EnableClusterExtension(c.Request.Context(), &models.EnableClusterExtensionRequest{
		ClusterID: c.Param("clusterID"),
	})
	if err != nil {
		api.Error(c, err)
		return
	}
	api.Success(c, nil)
}

// @Summary Install Cluster Extension
// @Description Install an extension in the specified cluster
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Param request body models.InstallClusterExtensionRequest true "Extension installation information"
// @Success 200 {object} api.Response{}
// @Router /api/v1/clusters/{clusterID}/extensions/install [post]
func InstallClusterExtension(c *gin.Context) {
	clusterID := c.Param("clusterID")
	if clusterID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Cluster ID is required"))
		return
	}

	var req models.InstallClusterExtensionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.ClusterID = clusterID

	s := services.NewClusterService()
	err := s.InstallClusterExtension(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, nil)
}

// @Summary Uninstall Cluster Extension
// @Description Uninstall an extension from the specified cluster
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Param extensionName path string true "Extension Name"
// @Success 200 {object} api.Response{}
// @Router /api/v1/clusters/{clusterID}/extensions/{extensionName} [delete]
func UninstallClusterExtension(c *gin.Context) {
	clusterID := c.Param("clusterID")
	extensionName := c.Param("extensionName")
	if clusterID == "" || extensionName == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Cluster ID and Extension Name are required"))
		return
	}

	s := services.NewClusterService()
	err := s.UninstallClusterExtension(c, &models.UninstallClusterExtensionRequest{
		ClusterID:     clusterID,
		ExtensionName: extensionName,
	})
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, nil)
}


// GetClusterExtensionValues godoc
// @Summary Get cluster extension default values
// @Description Get the default values.yaml for a specific cluster extension version
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Param extensionName path string true "Extension Name"
// @Param version path string true "Extension Version"
// @Success 200 {object} api.Response{data=string}
// @Failure 400 {object} api.Response
// @Failure 404 {object} api.Response
// @Failure 500 {object} api.Response
// @Router /api/v1/clusters/{clusterID}/extensions/{extensionName}/values/{version} [get]
func GetClusterExtensionValues(c *gin.Context) {
	clusterID := c.Param("clusterID")
	extensionName := c.Param("extensionName")
	version := c.Param("version")
	
	if clusterID == "" || extensionName == "" || version == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Cluster ID, Extension Name and Version are required"))
		return
	}

	s := services.NewClusterService()
	values, err := s.GetClusterExtensionValues(c, &models.GetClusterExtensionValuesRequest{
		ClusterID:     clusterID,
		ExtensionName: extensionName,
		Version:       version,
	})
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, values)
}

// GetInstalledExtensionValues godoc
// @Summary Get installed extension current values
// @Description Get the current values.yaml for an installed cluster extension
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Param extensionName path string true "Extension Name"
// @Success 200 {object} api.Response{data=string}
// @Failure 400 {object} api.Response
// @Failure 404 {object} api.Response
// @Failure 500 {object} api.Response
// @Router /api/v1/clusters/{clusterID}/extensions/{extensionName}/installed-values [get]
func GetInstalledExtensionValues(c *gin.Context) {
	clusterID := c.Param("clusterID")
	extensionName := c.Param("extensionName")
	
	if clusterID == "" || extensionName == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Cluster ID and Extension Name are required"))
		return
	}

	s := services.NewClusterService()
	values, err := s.GetInstalledExtensionValues(c, &models.GetInstalledExtensionValuesRequest{
		ClusterID:     clusterID,
		ExtensionName: extensionName,
	})
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, values)
}

// UpdateClusterExtension godoc
// @Summary Update cluster extension
// @Description Update an installed cluster extension
// @Tags Cluster
// @Accept json
// @Produce json
// @Param clusterID path string true "Cluster ID"
// @Param extensionName path string true "Extension Name"
// @Param request body models.UpdateClusterExtensionRequest true "Extension update information"
// @Success 200 {object} api.Response{}
// @Failure 400 {object} api.Response
// @Failure 404 {object} api.Response
// @Failure 500 {object} api.Response
// @Router /api/v1/clusters/{clusterID}/extensions/{extensionName}/update [put]
func UpdateClusterExtension(c *gin.Context) {
	clusterID := c.Param("clusterID")
	extensionName := c.Param("extensionName")
	
	if clusterID == "" || extensionName == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Cluster ID and Extension Name are required"))
		return
	}

	var req models.UpdateClusterExtensionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}
	req.ClusterID = clusterID
	req.ExtensionName = extensionName

	s := services.NewClusterService()
	err := s.UpdateClusterExtension(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, nil)
}
