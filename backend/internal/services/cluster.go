package services

import (
	"context"
	"log"
	"net/http"
	"slices"
	"strings"
	"sync"

	helmoperatorv1alpha1 "github.com/ketches/helm-operator/api/v1alpha1"
	helmoperatorinstaller "github.com/ketches/helm-operator/pkg/installer"
	"github.com/ketches/ketches/internal/api"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/db/orm"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	ListClusterNodes(ctx context.Context, req *models.ListClusterNodesRequest) ([]*models.ClusterNodeModel, app.Error)
	ListClusterNodeRefs(ctx context.Context, req *models.ListClusterNodeRefsRequest) ([]*models.ClusterNodeRef, app.Error)
	GetClusterNode(ctx context.Context, req *models.GetClusterNodeRequest) (*models.ClusterNodeModel, app.Error)
	ListClusterNodeLabels(ctx context.Context, req *models.ListClusterNodeLabelsRequest) ([]string, app.Error)
	ListClusterNodeTaints(ctx context.Context, req *models.ListClusterNodeTaintsRequest) ([]*models.ClusterNodeTaintModel, app.Error)
	ListClusterExtensions(ctx context.Context, req *models.ListClusterExtensionsRequest) (*models.ListClusterExtensionsResponse, app.Error)
	EnableClusterExtension(ctx context.Context, req *models.EnableClusterExtensionRequest) app.Error
	CheckClusterExtensionFeatureEnabled(ctx context.Context, req *models.CheckClusterExtensionFeatureEnabledRequest) (bool, app.Error)
	InstallClusterExtension(ctx context.Context, req *models.InstallClusterExtensionRequest) app.Error
	UninstallClusterExtension(ctx context.Context, req *models.UninstallClusterExtensionRequest) app.Error
	GetClusterExtensionValues(ctx context.Context, req *models.GetClusterExtensionValuesRequest) (string, app.Error)
	GetInstalledExtensionValues(ctx context.Context, req *models.GetInstalledExtensionValuesRequest) (string, app.Error)
	UpdateClusterExtension(ctx context.Context, req *models.UpdateClusterExtensionRequest) app.Error
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

	if err := db.Instance().Select("DisplayName", "KubeConfig", "Description", "UpdatedBy").Updates(&entities.Cluster{
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

func (s *clusterService) ListClusterNodes(ctx context.Context, req *models.ListClusterNodesRequest) ([]*models.ClusterNodeModel, app.Error) {
	kstore, err := kube.ClusterStore(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	nodes, e := kstore.NodeLister().List(labels.Everything())
	if e != nil {
		log.Printf("failed to list nodes for cluster %s: %v", req.ClusterID, e)
		return nil, app.ErrClusterOperationFailed
	}

	var result []*models.ClusterNodeModel
	for _, n := range nodes {
		result = append(result, clusterNodeFrom(req.ClusterID, n))
	}

	return result, nil
}

func (s *clusterService) ListClusterNodeRefs(ctx context.Context, req *models.ListClusterNodeRefsRequest) ([]*models.ClusterNodeRef, app.Error) {
	cluster, err := orm.GetClusterByID(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	kstore, err := kube.ClusterStore(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	nodes, e := kstore.NodeLister().List(labels.Everything())
	if e != nil {
		log.Printf("failed to list nodes for cluster %s: %v", req.ClusterID, e)
		return nil, app.ErrClusterOperationFailed
	}

	var result []*models.ClusterNodeRef
	for _, node := range nodes {
		var (
			internalIP string
		)
		for _, addr := range node.Status.Addresses {
			if addr.Type == corev1.NodeInternalIP {
				internalIP = addr.Address
			}
		}

		result = append(result, &models.ClusterNodeRef{
			NodeName:           node.Name,
			NodeIP:             internalIP,
			ClusterID:          cluster.ID,
			ClusterSlug:        cluster.Slug,
			ClusterDisplayName: cluster.DisplayName,
		})
	}

	return result, nil
}

func (s *clusterService) GetClusterNode(ctx context.Context, req *models.GetClusterNodeRequest) (*models.ClusterNodeModel, app.Error) {
	kstore, err := kube.ClusterStore(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	node, e := kstore.NodeLister().Get(req.NodeName)
	if e != nil {
		log.Printf("failed to get node %s for cluster %s: %v", req.NodeName, req.ClusterID, e)
		if k8serrors.IsNotFound(e) {
			return nil, app.NewError(http.StatusNotFound, "node not found")
		}
		return nil, app.ErrClusterOperationFailed
	}

	return clusterNodeFrom(req.ClusterID, node), nil
}

func (s *clusterService) ListClusterNodeLabels(ctx context.Context, req *models.ListClusterNodeLabelsRequest) ([]string, app.Error) {
	store, err := kube.ClusterStore(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	nodes, kubeErr := store.NodeLister().List(labels.Everything())
	if kubeErr != nil {
		log.Printf("failed to list nodes: %v", kubeErr)
		return nil, app.ErrClusterOperationFailed
	}

	kvM := make(map[string]struct{}, len(nodes[0].Labels))
	for _, node := range nodes {
		for key, value := range node.Labels {
			if value == "" {
				kvM[key] = struct{}{}
			} else {
				kvM[key+"="+value] = struct{}{}
			}
		}
	}

	var result []string
	for kv := range kvM {
		result = append(result, kv)
	}

	slices.Sort(result)
	return result, nil
}

func (s *clusterService) ListClusterNodeTaints(ctx context.Context, req *models.ListClusterNodeTaintsRequest) ([]*models.ClusterNodeTaintModel, app.Error) {
	store, err := kube.ClusterStore(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	result := make([]*models.ClusterNodeTaintModel, 0)
	nodes, kubeErr := store.NodeLister().List(labels.Everything())
	if kubeErr != nil {
		log.Printf("failed to list nodes: %v", kubeErr)
		return nil, app.ErrClusterOperationFailed
	}

	for _, node := range nodes {
		for _, taint := range node.Spec.Taints {
			if taint.Key == "node.kubernetes.io/unschedulable" {
				continue
			}
			found := false
			for _, item := range result {
				if item.Key == taint.Key {
					found = true
					if !slices.Contains(item.Values, taint.Value) {
						item.Values = append(item.Values, taint.Value)
					} else {
						continue
					}
				}
			}
			if !found {
				result = append(result, &models.ClusterNodeTaintModel{
					Key:    taint.Key,
					Values: []string{taint.Value},
				})
				continue
			}
		}
	}
	return result, err
}

func (s *clusterService) CheckClusterExtensionFeatureEnabled(ctx context.Context, req *models.CheckClusterExtensionFeatureEnabledRequest) (bool, app.Error) {
	kclient, err := kube.ClusterRuntimeClient(ctx, req.ClusterID)
	if err != nil {
		return false, err
	}

	var crd apiextensionsv1.CustomResourceDefinition
	if err := kclient.Get(ctx, client.ObjectKey{Name: "helmrepositories.helm-operator.ketches.cn"}, &crd); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Printf("crd HelmRepository not found for cluster %s: %v", req.ClusterID, err)
			return false, nil
		}
		log.Printf("failed to get crd HelmRepository for cluster %s: %v", req.ClusterID, err)
		return false, app.ErrClusterOperationFailed
	}

	return true, nil
}

func (s *clusterService) EnableClusterExtension(ctx context.Context, req *models.EnableClusterExtensionRequest) app.Error {
	kclient, err := kube.ClusterRuntimeClient(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := helmoperatorinstaller.NewInstaller(kclient).Install(ctx); err != nil {
		log.Printf("failed to install helm operator for cluster %s: %v", req.ClusterID, err)
		return app.NewError(http.StatusInternalServerError, "failed to enable cluster extension feature")
	}

	core.ApplyResource(ctx, kclient, &helmoperatorv1alpha1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ketches-extension-charts",
			Namespace: "ketches",
			Labels: map[string]string{
				"ketches.cn/owned": "true",
			},
		},
		Spec: helmoperatorv1alpha1.HelmRepositorySpec{
			URL:      "https://ketches.github.io/ketches-extension-charts",
			Type:     "helm",
			Interval: "30m",
			Timeout:  "10m",
		},
	})

	return nil
}

func (s *clusterService) ListClusterExtensions(ctx context.Context, req *models.ListClusterExtensionsRequest) (*models.ListClusterExtensionsResponse, app.Error) {
	kclient, err := kube.ClusterRuntimeClient(ctx, req.ClusterID)
	if err != nil {
		return nil, err
	}

	var helmrepository helmoperatorv1alpha1.HelmRepository
	if err := kclient.Get(ctx, client.ObjectKey{Namespace: "ketches", Name: "ketches-extension-charts"}, &helmrepository); err != nil {
		if k8serrors.IsNotFound(err) {
			log.Printf("ketches helm repository not found for cluster %s: %v", req.ClusterID, err)
			result := make(models.ListClusterExtensionsResponse)
			return &result, nil
		}
		log.Printf("failed to get ketches helm repository for cluster %s: %v", req.ClusterID, err)
		return nil, app.ErrClusterOperationFailed
	}

	// Get all installed HelmReleases
	var helmreleases helmoperatorv1alpha1.HelmReleaseList
	if err := kclient.List(ctx, &helmreleases, client.MatchingLabels{"ketches.cn/owned": "ketches"}); err != nil {
		log.Printf("failed to list helm releases for cluster %s: %v", req.ClusterID, err)
		return nil, app.ErrClusterOperationFailed
	}

	// Create a map of installed releases
	installedReleases := make(map[string]*helmoperatorv1alpha1.HelmRelease)
	for i := range helmreleases.Items {
		release := &helmreleases.Items[i]
		if extensionName, ok := release.Labels["extension.ketches.cn/name"]; ok {
			installedReleases[extensionName] = release
		}
	}

	result := make(models.ListClusterExtensionsResponse, len(helmrepository.Status.Charts))

	for _, entry := range helmrepository.Status.Charts {
		// Convert ChartVersion slice to string slice
		versions := make([]string, len(entry.Versions))
		for i, v := range entry.Versions {
			versions[i] = v.Version
		}

		extension := &models.ClusterExtensionModel{
			ExtensionID:   entry.Name,
			Slug:          entry.Name,
			DisplayName:   entry.Name,
			Description:   entry.Description,
			InstallMethod: "helm",
			Installed:     false,
			Status:        "available",
			Versions:      versions,
		}

		// Check if this extension is installed
		if release, installed := installedReleases[entry.Name]; installed {
			extension.Installed = true
			extension.Version = release.Spec.Chart.Version
			extension.CreatedAt = release.CreationTimestamp.Format("2006-01-02 15:04:05")
			extension.UpdatedAt = release.CreationTimestamp.Format("2006-01-02 15:04:05")

			// Determine status from HelmRelease conditions
			if len(release.Status.Conditions) > 0 {
				for _, condition := range release.Status.Conditions {
					if condition.Type == "Ready" {
						if condition.Status == metav1.ConditionTrue {
							extension.Status = "installed"
						} else {
							extension.Status = "failed"
						}
						break
					}
				}
			} else {
				extension.Status = "installing"
			}

			// Update timestamps if available
			if release.Status.HelmRelease != nil {
				if release.Status.HelmRelease.LastDeployed != nil {
					extension.UpdatedAt = release.Status.HelmRelease.LastDeployed.Format("2006-01-02 15:04:05")
				}
			}
		}

		result[entry.Name] = extension
	}

	return &result, nil
}

func (s *clusterService) InstallClusterExtension(ctx context.Context, req *models.InstallClusterExtensionRequest) app.Error {
	kclient, err := kube.ClusterRuntimeClient(ctx, req.ClusterID)
	if err != nil {
		return err
	}

	release := &helmoperatorv1alpha1.HelmRelease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.ExtensionName,
			Namespace: req.Namespace,
			Labels: map[string]string{
				"ketches.cn/owned":             "ketches",
				"extension.ketches.cn/name":    req.ExtensionName,
				"extension.ketches.cn/version": req.Type,
			},
		},
		Spec: helmoperatorv1alpha1.HelmReleaseSpec{
			Chart: helmoperatorv1alpha1.ChartSpec{
				Name:    req.ExtensionName,
				Version: req.Version,
				Repository: &helmoperatorv1alpha1.RepositoryReference{
					Name:      "ketches-extension-charts",
					Namespace: "ketches",
				},
			},
			Release: &helmoperatorv1alpha1.ReleaseSpec{
				Name:      req.ExtensionName,
				Namespace: "ketches",
			},
			Values: req.Values,
			Install: &helmoperatorv1alpha1.InstallSpec{
				Timeout:     "10m",
				Wait:        true,
				WaitForJobs: true,
			},
			Upgrade: &helmoperatorv1alpha1.UpgradeSpec{
				Timeout:       "10m",
				Wait:          true,
				CleanupOnFail: true,
			},
			Interval: "1h",
		},
	}

	if err := kclient.Create(ctx, release); err != nil {
		log.Printf("failed to create helm release %s for cluster %s: %v", req.ExtensionName, req.ClusterID, err)
		if k8serrors.IsAlreadyExists(err) {
			return app.NewError(http.StatusConflict, "extension already installed")
		}
		return app.ErrClusterOperationFailed
	}
	return nil
}

func (s *clusterService) UninstallClusterExtension(ctx context.Context, req *models.UninstallClusterExtensionRequest) app.Error {
	kclient, err := kube.ClusterRuntimeClient(ctx, req.ClusterID)
	if err != nil {
		return err
	}

	// Find the HelmRelease for this extension
	var helmreleases helmoperatorv1alpha1.HelmReleaseList
	if err := kclient.List(ctx, &helmreleases, client.MatchingLabels{
		"ketches.cn/owned":          "ketches",
		"extension.ketches.cn/name": req.ExtensionName,
	}); err != nil {
		log.Printf("failed to list helm releases for extension %s in cluster %s: %v", req.ExtensionName, req.ClusterID, err)
		return app.ErrClusterOperationFailed
	}

	if len(helmreleases.Items) == 0 {
		return app.NewError(http.StatusNotFound, "extension not found or not installed")
	}

	// Delete the HelmRelease
	release := &helmreleases.Items[0]
	if err := kclient.Delete(ctx, release); err != nil {
		log.Printf("failed to delete helm release %s for cluster %s: %v", req.ExtensionName, req.ClusterID, err)
		if k8serrors.IsNotFound(err) {
			return app.NewError(http.StatusNotFound, "extension not found")
		}
		return app.ErrClusterOperationFailed
	}

	return nil
}

func (s *clusterService) GetClusterExtensionValues(ctx context.Context, req *models.GetClusterExtensionValuesRequest) (string, app.Error) {
	// Generate ConfigMap name using the same logic as helm-operator
	configMapName := s.generateConfigMapName("ketches-extension-charts", req.ExtensionName, req.Version)

	// Get the ConfigMap
	kstore, err := kube.ClusterStore(ctx, req.ClusterID)
	if err != nil {
		return "", err
	}

	cm, e := kstore.ConfigMapLister().ConfigMaps("ketches").Get(configMapName)
	if e != nil {
		log.Printf("failed to get ConfigMap %s for extension %s in cluster %s: %v", configMapName, req.ExtensionName, req.ClusterID, e)
		if k8serrors.IsNotFound(e) {
			return "", nil // Return empty string if ConfigMap doesn't exist
		}
		return "", app.ErrClusterOperationFailed
	}

	// Return the values.yaml content
	if values, ok := cm.Data["values.yaml"]; ok {
		return values, nil
	}

	return "", nil
}

func (s *clusterService) GetInstalledExtensionValues(ctx context.Context, req *models.GetInstalledExtensionValuesRequest) (string, app.Error) {
	kclient, err := kube.ClusterRuntimeClient(ctx, req.ClusterID)
	if err != nil {
		return "", err
	}

	// Find the HelmRelease for this extension
	var helmreleases helmoperatorv1alpha1.HelmReleaseList
	if err := kclient.List(ctx, &helmreleases, client.MatchingLabels{
		"ketches.cn/owned":          "ketches",
		"extension.ketches.cn/name": req.ExtensionName,
	}); err != nil {
		log.Printf("failed to list helm releases for extension %s in cluster %s: %v", req.ExtensionName, req.ClusterID, err)
		return "", app.ErrClusterOperationFailed
	}

	if len(helmreleases.Items) == 0 {
		return "", app.NewError(http.StatusNotFound, "extension not found or not installed")
	}

	// Return the current values from the HelmRelease
	release := &helmreleases.Items[0]
	return release.Spec.Values, nil
}

func (s *clusterService) UpdateClusterExtension(ctx context.Context, req *models.UpdateClusterExtensionRequest) app.Error {
	kclient, err := kube.ClusterRuntimeClient(ctx, req.ClusterID)
	if err != nil {
		return err
	}

	// Find the existing HelmRelease
	var helmreleases helmoperatorv1alpha1.HelmReleaseList
	if err := kclient.List(ctx, &helmreleases, client.MatchingLabels{
		"ketches.cn/owned":          "ketches",
		"extension.ketches.cn/name": req.ExtensionName,
	}); err != nil {
		log.Printf("failed to list helm releases for extension %s in cluster %s: %v", req.ExtensionName, req.ClusterID, err)
		return app.ErrClusterOperationFailed
	}

	if len(helmreleases.Items) == 0 {
		return app.NewError(http.StatusNotFound, "extension not found or not installed")
	}

	// Update the HelmRelease
	release := &helmreleases.Items[0]
	if req.Version != "" {
		release.Spec.Chart.Version = req.Version
	}
	if req.Values != "" {
		release.Spec.Values = req.Values
	}

	if err := kclient.Update(ctx, release); err != nil {
		log.Printf("failed to update helm release %s for cluster %s: %v", req.ExtensionName, req.ClusterID, err)
		return app.ErrClusterOperationFailed
	}

	return nil
}

// generateConfigMapName generates the ConfigMap name using the same logic as helm-operator
func (s *clusterService) generateConfigMapName(repoName, chartName, version string) string {
	return "helm-values-" + repoName + "-" + chartName + "-" + strings.ReplaceAll(version, ".", "-")
}

func clusterNodeFrom(clusterID string, node *corev1.Node) *models.ClusterNodeModel {
	var (
		internalIP, externalIP string
		roles                  []string
		ready                  bool
	)
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			internalIP = addr.Address
		}
		if addr.Type == corev1.NodeExternalIP {
			externalIP = addr.Address
		}
	}
	for k := range node.Labels {
		if k == "node-role.kubernetes.io/master" || k == "node-role.kubernetes.io/control-plane" {
			roles = append(roles, "master")
		} else if strings.HasPrefix(k, "node-role.kubernetes.io/") {
			roles = append(roles, strings.TrimPrefix(k, "node-role.kubernetes.io/"))
		}
	}
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			ready = cond.Status == corev1.ConditionTrue
		}
	}

	return &models.ClusterNodeModel{
		NodeName:                node.Name,
		Roles:                   roles,
		CreatedAt:               utils.HumanizeTime(node.CreationTimestamp.Time),
		Version:                 node.Status.NodeInfo.KubeletVersion,
		InternalIP:              internalIP,
		ExternalIP:              externalIP,
		OSImage:                 node.Status.NodeInfo.OSImage,
		KernelVersion:           node.Status.NodeInfo.KernelVersion,
		OperatingSystem:         node.Status.NodeInfo.OperatingSystem,
		Architecture:            node.Status.NodeInfo.Architecture,
		ContainerRuntimeVersion: node.Status.NodeInfo.ContainerRuntimeVersion,
		KubeletVersion:          node.Status.NodeInfo.KubeletVersion,
		PodCIDR:                 node.Spec.PodCIDR,
		Ready:                   ready,
		ClusterID:               clusterID,
	}
}
