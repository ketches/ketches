package services

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/utils"
	"k8s.io/apimachinery/pkg/labels"
)

type ClusterService interface {
	ListClusters(ctx context.Context, req *models.ListClustersRequest) (*models.ListClustersResponse, app.Error)
	AllClusterRefs(ctx context.Context) ([]*models.ClusterRef, app.Error)
	GetCluster(ctx context.Context, req *models.GetClusterRequest) (*models.ClusterModel, app.Error)
	GetClusterRef(ctx context.Context, req *models.GetClusterRefRequest) (*models.ClusterRef, app.Error)
	CreateCluster(ctx context.Context, req *models.CreateClusterRequest) (*models.ClusterModel, app.Error)
	UpdateCluster(ctx context.Context, req *models.UpdateClusterRequest) (*models.ClusterModel, app.Error)
	DeleteCluster(ctx context.Context, req *models.DeleteClusterRequest) app.Error
	EnableCluster(ctx context.Context, req *models.EnabledClusterRequest) app.Error
	DisableCluster(ctx context.Context, req *models.DisableClusterRequest) app.Error
	PingClusterKubeConfig(ctx context.Context, req *models.PingClusterKubeConfigRequest) bool
	ListClusterExtensions(ctx context.Context, req *models.ListClusterExtensionsRequest) (*models.ListClusterExtensionsResponse, app.Error)
}

type clusterService struct {
	Service
}

func NewClusterService() ClusterService {
	return &clusterService{
		Service: LoadService(),
	}
}

func (s *clusterService) ListClusters(ctx context.Context, req *models.ListClustersRequest) (*models.ListClustersResponse, app.Error) {
	query := db.Instance().Model(&entities.Cluster{})
	if len(req.Query) > 0 {
		query = db.CaseInsensitiveLike(query, req.Query, "slug", "display_name")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Printf("failed to count clusters for user %s: %v", api.UserID(ctx), err)
		return nil, app.ErrDatabaseOperationFailed
	}

	var clusters []*entities.Cluster
	if err := req.PagedSQL(query).Find(&clusters).Error; err != nil {
		log.Printf("failed to list clusters for user %s: %v", api.UserID(ctx), err)
		return nil, app.ErrDatabaseOperationFailed
	}

	result := &models.ListClustersResponse{
		Total:   total,
		Records: make([]*models.ClusterModel, 0, len(clusters)),
	}

	var wg sync.WaitGroup
	for _, cluster := range clusters {
		item := &models.ClusterModel{
			ClusterID:   cluster.ID,
			Slug:        cluster.Slug,
			DisplayName: cluster.DisplayName,
			Description: cluster.Description,
			Enabled:     cluster.Enabled,
		}
		if api.IsAdmin(ctx) {
			item.KubeConfig = cluster.KubeConfig
		}

		wg.Add(1)
		go func(item *models.ClusterModel) {
			defer wg.Done()
			kstore, err := kube.ClusterStore(ctx, item.ClusterID)
			if err != nil {
				log.Printf("failed to get cluster store for %s: %v", item.ClusterID, err)
				return
			}
			nodes, e := kstore.NodeLister().List(labels.Everything())
			if e != nil {
				log.Printf("failed to list nodes for cluster %s: %v", item.ClusterID, e)
				return
			}
			if len(nodes) == 0 {
				return
			}

			item.NodeCount = len(nodes)
			item.Connectable = true
			item.ServerVersion = nodes[0].Status.NodeInfo.KubeletVersion
			for _, node := range nodes {
				for _, condition := range node.Status.Conditions {
					if condition.Type == "Ready" && condition.Status == "True" {
						item.ReadyNodeCount++
					}
				}
			}
		}(item)

		result.Records = append(result.Records, item)
	}
	wg.Wait()

	return result, nil
}

func (s *clusterService) AllClusterRefs(ctx context.Context) ([]*models.ClusterRef, app.Error) {
	result := []*models.ClusterRef{}
	if err := db.Instance().Model(&entities.Cluster{}).Where("enabled = ?", true).Find(&result).Error; err != nil {
		log.Printf("failed to list cluster refs: %v", err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

func (s *clusterService) GetCluster(ctx context.Context, req *models.GetClusterRequest) (*models.ClusterModel, app.Error) {
	cluster := &entities.Cluster{}
	if err := db.Instance().Where("id = ?", req.ClusterID).First(cluster).Error; err != nil {
		log.Printf("failed to get cluster %s for user %s: %v", req.ClusterID, api.UserID(ctx), err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Cluster not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	result := &models.ClusterModel{
		ClusterID:   cluster.ID,
		Slug:        cluster.Slug,
		DisplayName: cluster.DisplayName,
		Description: cluster.Description,
		Enabled:     cluster.Enabled,
	}
	if api.IsAdmin(ctx) {
		result.KubeConfig = cluster.KubeConfig
	}

	return result, nil
}

func (s *clusterService) GetClusterRef(ctx context.Context, req *models.GetClusterRefRequest) (*models.ClusterRef, app.Error) {
	result := &models.ClusterRef{}
	if err := db.Instance().Model(&entities.Cluster{}).First(result, "id = ?", req.ClusterID).Error; err != nil {
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Cluster not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	return result, nil
}

func (s *clusterService) CreateCluster(ctx context.Context, req *models.CreateClusterRequest) (*models.ClusterModel, app.Error) {
	cluster := &entities.Cluster{
		Slug:        req.Slug,
		DisplayName: req.DisplayName,
		KubeConfig:  req.KubeConfig,
		GatewayIP:   req.GatewayIP,
		Description: req.Description,
		Enabled:     true,
		AuditBase: entities.AuditBase{
			CreatedBy: api.UserID(ctx),
			UpdatedBy: api.UserID(ctx),
		},
	}

	if err := db.Instance().Create(cluster).Error; err != nil {
		log.Printf("failed to create cluster for user %s: %v", api.UserID(ctx), err)
		if db.IsErrDuplicatedKey(err) {
			return nil, app.NewError(http.StatusConflict, "cluster with this slug already exists")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	return &models.ClusterModel{
		ClusterID:   cluster.ID,
		Slug:        cluster.Slug,
		DisplayName: cluster.DisplayName,
		Description: cluster.Description,
		KubeConfig:  cluster.KubeConfig,
		Enabled:     cluster.Enabled,
	}, nil
}

func (s *clusterService) UpdateCluster(ctx context.Context, req *models.UpdateClusterRequest) (*models.ClusterModel, app.Error) {
	cluster := &entities.Cluster{}
	if err := db.Instance().Where("id = ?", req.ClusterID).First(cluster).Error; err != nil {
		log.Printf("failed to get cluster %s for user %s: %v", req.ClusterID, api.UserID(ctx), err)
		if db.IsErrRecordNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "cluster not found")
		}
		return nil, app.ErrDatabaseOperationFailed
	}

	cluster.DisplayName = req.DisplayName
	cluster.KubeConfig = req.KubeConfig
	cluster.Description = req.Description

	if err := db.Instance().Updates(&entities.Cluster{
		UUIDBase:    cluster.UUIDBase,
		DisplayName: cluster.DisplayName,
		KubeConfig:  cluster.KubeConfig,
		Description: cluster.Description,
		AuditBase: entities.AuditBase{
			UpdatedBy: api.UserID(ctx),
		},
	}).Error; err != nil {
		log.Printf("failed to update cluster %s for user %s: %v", req.ClusterID, api.UserID(ctx), err)
		return nil, app.ErrDatabaseOperationFailed
	}

	return &models.ClusterModel{
		ClusterID:   cluster.ID,
		Slug:        cluster.Slug,
		DisplayName: cluster.DisplayName,
		Description: cluster.Description,
		KubeConfig:  cluster.KubeConfig,
		Enabled:     cluster.Enabled,
	}, nil
}

func (s *clusterService) DeleteCluster(ctx context.Context, req *models.DeleteClusterRequest) app.Error {
	var envCount int64
	if err := db.Instance().Model(&entities.Env{}).Where("cluster_id = ?", req.ClusterID).Count(&envCount).Error; err != nil {
		log.Printf("failed to count environments for cluster %s for user %s: %v", req.ClusterID, api.UserID(ctx), err)
		return app.ErrDatabaseOperationFailed
	}
	if envCount > 0 {
		log.Printf("cannot delete cluster %s for user %s: cluster has associated environments", req.ClusterID, api.UserID(ctx))
		return app.NewError(http.StatusConflict, "cluster has associated environments")
	}

	if err := db.Instance().Delete(&entities.Cluster{}, "id = ?", req.ClusterID).Error; err != nil {
		log.Printf("failed to delete cluster %s for user %s: %v", req.ClusterID, api.UserID(ctx), err)
		return app.ErrDatabaseOperationFailed
	}

	return nil
}

func (s *clusterService) EnableCluster(ctx context.Context, req *models.EnabledClusterRequest) app.Error {
	if err := db.Instance().Updates(&entities.Cluster{
		UUIDBase: entities.UUIDBase{
			ID: req.ClusterID},
		Enabled: true,
		AuditBase: entities.AuditBase{
			UpdatedBy: api.UserID(ctx),
		},
	}).Error; err != nil {
		log.Printf("failed to enable cluster %s for user %s: %v", req.ClusterID, api.UserID(ctx), err)
		return app.ErrDatabaseOperationFailed
	}

	return nil
}

func (s *clusterService) DisableCluster(ctx context.Context, req *models.DisableClusterRequest) app.Error {
	if err := db.Instance().Updates(&entities.Cluster{
		UUIDBase: entities.UUIDBase{
			ID: req.ClusterID,
		},
		Enabled: false,
		AuditBase: entities.AuditBase{
			UpdatedBy: api.UserID(ctx),
		},
	}).Error; err != nil {
		log.Printf("failed to disable cluster %s for user %s: %v", req.ClusterID, api.UserID(ctx), err)
		return app.ErrDatabaseOperationFailed
	}

	return nil
}

func (s *clusterService) PingClusterKubeConfig(ctx context.Context, req *models.PingClusterKubeConfigRequest) bool {
	return kube.CheckKubeConfigBytes([]byte(req.KubeConfig))
}

func (s *clusterService) ListClusterExtensions(ctx context.Context, req *models.ListClusterExtensionsRequest) (*models.ListClusterExtensionsResponse, app.Error) {
	cli, err := kube.ClusterRuntimeClient(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	nativeExtensions := core.CheckNativeExtensions(ctx, cli)
	result := make(models.ListClusterExtensionsResponse, len(nativeExtensions))
	for _, ext := range nativeExtensions {
		result[ext.Slug] = &models.ClusterExtensionModel{
			Slug:        ext.Slug,
			DisplayName: ext.DisplayName,
			Description: ext.Description,
			Installed:   ext.Installed,
			Version:     ext.Version,
			CreatedAt:   utils.HumanizeTime(ext.CreatedAt),
		}
	}

	return &result, nil
}
