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
// @Param query query model.ListClustersRequest false "Query parameters for filtering and pagination"
// @Success 200 {object} api.Response{data=model.ListClustersResponse}
// @Router /api/v1/clusters [get]
func ListClusters(c *gin.Context) {
	var req models.ListClustersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	cs := services.NewClusterService()
	resp, err := cs.ListClusters(c, &req)
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
// @Success 200 {object} api.Response{data=[]model.ClusterRef}
// @Router /api/v1/clusters/refs [get]
func AllClusterRefs(c *gin.Context) {
	cs := services.NewClusterService()
	refs, err := cs.AllClusterRefs(c)
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
// @Success 200 {object} api.Response{data=model.ClusterModel}
// @Router /api/v1/clusters/{clusterID} [get]
func GetCluster(c *gin.Context) {
	clusterID := c.Param("clusterID")
	if clusterID == "" {
		api.Error(c, app.NewError(http.StatusBadRequest, "Cluster ID is required"))
		return
	}

	cs := services.NewClusterService()
	cluster, err := cs.GetCluster(c, &models.GetClusterRequest{
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
// @Success 200 {object} api.Response{data=model.ClusterRef}
// @Router /api/v1/clusters/{clusterID}/ref [get]
func GetClusterRef(c *gin.Context) {
	var req models.GetClusterRefRequest
	if err := c.ShouldBindUri(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	as := services.NewClusterService()
	ref, err := as.GetClusterRef(c, &req)
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
// @Param request body model.CreateClusterRequest true "Cluster information"
// @Success 201 {object} api.Response{data=model.ClusterModel}
// @Router /api/v1/clusters [post]
func CreateCluster(c *gin.Context) {
	var req models.CreateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.Error(c, app.NewError(http.StatusBadRequest, err.Error()))
		return
	}

	cs := services.NewClusterService()
	cluster, err := cs.CreateCluster(c, &req)
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
// @Param request body model.UpdateClusterRequest true "Updated cluster information"
// @Success 200 {object} api.Response{data=model.ClusterModel}
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

	cs := services.NewClusterService()
	cluster, err := cs.UpdateCluster(c, &req)
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

	cs := services.NewClusterService()
	if err := cs.DeleteCluster(c, &models.DeleteClusterRequest{
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

	cs := services.NewClusterService()
	err := cs.EnableCluster(c, &req)
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

	cs := services.NewClusterService()
	err := cs.DisableCluster(c, &req)
	if err != nil {
		api.Error(c, err)
		return
	}

	api.Success(c, nil)
}
