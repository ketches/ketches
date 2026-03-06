package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	memorycache "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

// ListExtensions returns all extensions (builtin + admin-added) with install counts from DB.
func ListExtensions() ([]models.Extension, error) {
	var items []entities.Extension
	if err := db.DB.Order("builtin DESC, created_at ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("failed to list extensions: %w", err)
	}

	// Count installs per extension from the cluster_extensions table (fast, no Helm calls).
	type installCountRow struct {
		ExtensionID string
		Count       int
	}
	var counts []installCountRow
	db.DB.Model(&entities.ClusterExtension{}).
		Select("extension_id, COUNT(*) as count").
		Where("deleted_at IS NULL").
		Group("extension_id").
		Scan(&counts)
	installMap := make(map[string]int, len(counts))
	for _, row := range counts {
		installMap[row.ExtensionID] = row.Count
	}

	result := make([]models.Extension, 0, len(items))
	for _, item := range items {
		m := toExtensionModel(&item)
		m.InstallCount = installMap[m.ID]
		result = append(result, m)
	}
	return result, nil
}

// GetExtension returns a single extension by ID.
func GetExtension(extensionID string) (*models.Extension, error) {
	var item entities.Extension
	if err := db.DB.Where("id = ?", extensionID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("extension not found")
		}
		return nil, err
	}
	m := toExtensionModel(&item)
	return &m, nil
}

// GetExtensionEntity returns the raw extension entity by ID (used by other services).
func GetExtensionEntity(extensionID string) (*entities.Extension, error) {
	var item entities.Extension
	if err := db.DB.Where("id = ?", extensionID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("extension not found")
		}
		return nil, err
	}
	return &item, nil
}

// CreateExtension creates a new admin-added extension.
func CreateExtension(req *models.CreateExtensionRequest, createdBy string) (*models.Extension, error) {
	var existing entities.Extension
	if err := db.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("extension with name %q already exists", req.Name)
	}
	item := &entities.Extension{
		ID:          uuid.New(),
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Description: req.Description,
		OCIUrl:      req.OCIUrl,
		IconURL:     req.IconURL,
		Builtin:     false,
		CreatedBy:   createdBy,
	}
	if err := db.DB.Create(item).Error; err != nil {
		return nil, fmt.Errorf("failed to create extension: %w", err)
	}
	m := toExtensionModel(item)
	return &m, nil
}

// DeleteExtension removes a extension by ID (builtin extensions cannot be deleted).
func DeleteExtension(extensionID string) error {
	var item entities.Extension
	if err := db.DB.Where("id = ?", extensionID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("extension not found")
		}
		return err
	}
	if item.Builtin {
		return fmt.Errorf("built-in extensions cannot be deleted")
	}
	return db.DB.Delete(&item).Error
}

// UpdateExtension updates a non-builtin extension's metadata.
func UpdateExtension(extensionID string, req *models.UpdateExtensionRequest) (*models.Extension, error) {
	var item entities.Extension
	if err := db.DB.Where("id = ?", extensionID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("extension not found")
		}
		return nil, err
	}
	if item.Builtin {
		return nil, fmt.Errorf("built-in extensions cannot be modified")
	}
	if req.DisplayName != "" {
		item.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		item.Description = req.Description
	}
	if req.OCIUrl != "" {
		item.OCIUrl = req.OCIUrl
	}
	item.IconURL = req.IconURL
	if err := db.DB.Save(&item).Error; err != nil {
		return nil, fmt.Errorf("failed to update extension: %w", err)
	}
	m := toExtensionModel(&item)
	return &m, nil
}

// GetInstalledClustersForExtension returns all clusters that have a given extension installed,
// queried directly from the cluster_extensions table (no Helm calls).
func GetInstalledClustersForExtension(extensionID string) ([]models.InstalledCluster, error) {
	// Verify the extension exists.
	if _, err := GetExtensionEntity(extensionID); err != nil {
		return nil, err
	}

	// Join cluster_extensions with clusters.
	type row struct {
		ClusterID   string
		ClusterName string
		ReleaseName string
		Namespace   string
		Version     string
		Status      string
	}
	var rows []row
	err := db.DB.Table("cluster_extensions ce").
		Select("ce.cluster_id, c.name as cluster_name, ce.release_name, ce.namespace, ce.version, ce.status").
		Joins("JOIN clusters c ON c.id = ce.cluster_id").
		Where("ce.extension_id = ? AND ce.deleted_at IS NULL AND c.deleted_at IS NULL", extensionID).
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("failed to query installed clusters: %w", err)
	}

	result := make([]models.InstalledCluster, 0, len(rows))
	for _, r := range rows {
		result = append(result, models.InstalledCluster{
			ClusterID:   r.ClusterID,
			ClusterName: r.ClusterName,
			ReleaseName: r.ReleaseName,
			Namespace:   r.Namespace,
			Version:     r.Version,
			Status:      r.Status,
		})
	}
	return result, nil
}

// ListExtensionVersions lists OCI tags for an extension, sorted newest first.
func ListExtensionVersions(extensionID string) ([]models.ExtensionVersionInfo, error) {
	item, err := GetExtensionEntity(extensionID)
	if err != nil {
		return nil, err
	}

	// crane expects the repo without the "oci://" prefix.
	repo := strings.TrimPrefix(item.OCIUrl, "oci://")
	tags, err := crane.ListTags(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions for %q: %w", item.OCIUrl, err)
	}

	// Sort descending (newest first) using simple string sort; semver sorting
	// can be added later without API change.
	sort.Sort(sort.Reverse(sort.StringSlice(tags)))

	result := make([]models.ExtensionVersionInfo, 0, len(tags))
	for _, tag := range tags {
		result = append(result, models.ExtensionVersionInfo{Version: tag})
	}
	return result, nil
}

// GetExtensionValues pulls the OCI helm chart and returns its default values.yaml content.
func GetExtensionValues(extensionID, version string) (string, error) {
	item, err := GetExtensionEntity(extensionID)
	if err != nil {
		return "", err
	}

	// Pull the chart to a temp directory and load it.
	dir, err := os.MkdirTemp("", "ketches-ext-values-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	regClient, err := registry.NewClient()
	if err != nil {
		return "", fmt.Errorf("failed to create registry client: %w", err)
	}

	p := action.NewPullWithOpts(action.WithConfig(&action.Configuration{
		RegistryClient: regClient,
	}))
	p.Settings = cli.New()
	p.DestDir = dir
	p.Untar = true
	p.UntarDir = dir
	p.Version = version

	// Pull the OCI chart.
	if _, err := p.Run(item.OCIUrl); err != nil {
		return "", fmt.Errorf("failed to pull chart %q version %q: %w", item.OCIUrl, version, err)
	}

	// The chart is extracted to dir/<chart-name>/
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("failed to read extracted chart dir: %w", err)
	}
	var chartDir string
	for _, e := range entries {
		if e.IsDir() {
			chartDir = dir + "/" + e.Name()
			break
		}
	}
	if chartDir == "" {
		return "", fmt.Errorf("no chart directory found after pull")
	}

	chrt, err := loader.Load(chartDir)
	if err != nil {
		return "", fmt.Errorf("failed to load chart: %w", err)
	}

	// Return the raw values.yaml from the chart files.
	for _, f := range chrt.Raw {
		if f.Name == "values.yaml" {
			return string(f.Data), nil
		}
	}

	// Fallback: marshal the default values map.
	if len(chrt.Values) > 0 {
		out, err := yaml.Marshal(chrt.Values)
		if err != nil {
			return "", fmt.Errorf("failed to marshal values: %w", err)
		}
		return string(out), nil
	}
	return "", nil
}

// ListClusterExtensions returns all cluster extensions for a cluster from the DB.
func ListClusterExtensions(clusterID string) ([]models.ClusterExtension, error) {
	var records []entities.ClusterExtension
	if err := db.DB.Where("cluster_id = ?", clusterID).Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to list cluster extensions: %w", err)
	}
	result := make([]models.ClusterExtension, 0, len(records))
	for _, r := range records {
		result = append(result, toClusterExtensionModel(&r))
	}
	return result, nil
}

// GetClusterExtension returns a single cluster extension by ID from the DB.
func GetClusterExtension(clusterID, id string) (*models.ClusterExtension, error) {
	var record entities.ClusterExtension
	if err := db.DB.Where("cluster_id = ? AND id = ?", clusterID, id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("cluster extension not found")
		}
		return nil, err
	}
	m := toClusterExtensionModel(&record)
	return &m, nil
}

// InstallClusterExtension creates a DB record (status=pending) and runs the Helm install asynchronously.
// Returns 202-style result immediately.
func InstallClusterExtension(clusterID string, req *models.InstallExtensionRequest, installedBy string) (*models.ClusterExtension, error) {
	ext, err := GetExtensionEntity(req.ExtensionID)
	if err != nil {
		return nil, fmt.Errorf("extension not found: %w", err)
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Check uniqueness: same cluster, namespace, extensionID.
	var existing entities.ClusterExtension
	if err := db.DB.Where("cluster_id = ? AND namespace = ? AND extension_id = ? AND deleted_at IS NULL",
		clusterID, namespace, req.ExtensionID).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("extension already installed in namespace %q", namespace)
	}

	record := &entities.ClusterExtension{
		ID:          uuid.New(),
		ClusterID:   clusterID,
		ExtensionID: req.ExtensionID,
		Namespace:   namespace,
		ReleaseName: req.ReleaseName,
		Version:     req.Version,
		Values:      req.Values,
		Status:      entities.ClusterExtensionStatusPending,
		Phase:       "installing",
		InstalledBy: installedBy,
	}
	if err := db.DB.Create(record).Error; err != nil {
		return nil, fmt.Errorf("failed to create cluster extension record: %w", err)
	}

	go func() {
		// Update status to installing.
		db.DB.Model(record).Update("status", entities.ClusterExtensionStatusInstalling)

		actionConfig, cleanup, err := newHelmActionConfig(clusterID, namespace)
		if err != nil {
			db.DB.Model(record).Updates(map[string]any{"status": string(entities.ClusterExtensionStatusFailed), "error_message": err.Error()})
			return
		}
		defer cleanup()

		chrt, err := pullChart(ext.OCIUrl, req.Version, actionConfig.RegistryClient)
		if err != nil {
			db.DB.Model(record).Updates(map[string]any{"status": string(entities.ClusterExtensionStatusFailed), "error_message": err.Error()})
			return
		}

		installer := action.NewInstall(actionConfig)
		installer.ReleaseName = req.ReleaseName
		installer.Namespace = namespace
		installer.CreateNamespace = req.CreateNamespace

		vals, err := parseValues(req.Values)
		if err != nil {
			db.DB.Model(record).Updates(map[string]any{"status": string(entities.ClusterExtensionStatusFailed), "error_message": err.Error()})
			return
		}

		if _, err := installer.RunWithContext(context.Background(), chrt, vals); err != nil {
			db.DB.Model(record).Updates(map[string]any{"status": string(entities.ClusterExtensionStatusFailed), "error_message": err.Error()})
			return
		}

		db.DB.Model(record).Updates(map[string]any{"status": string(entities.ClusterExtensionStatusDeployed), "error_message": "", "phase": ""})
	}()

	m := toClusterExtensionModel(record)
	return &m, nil
}

// UpgradeClusterExtension upgrades an installed cluster extension asynchronously.
func UpgradeClusterExtension(clusterID, id string, req *models.UpgradeExtensionRequest) (*models.ClusterExtension, error) {
	record, err := getClusterExtensionEntity(clusterID, id)
	if err != nil {
		return nil, err
	}

	ext, err := GetExtensionEntity(record.ExtensionID)
	if err != nil {
		return nil, err
	}

	// Update record with new version/values and set status to upgrading.
	updates := map[string]any{"status": string(entities.ClusterExtensionStatusUpgrading), "phase": "upgrading"}
	if req.Version != "" {
		updates["version"] = req.Version
		record.Version = req.Version
	}
	if req.Values != "" {
		updates["values"] = req.Values
		record.Values = req.Values
	}
	db.DB.Model(record).Updates(updates)

	targetVersion := record.Version
	namespace := record.Namespace
	releaseName := record.ReleaseName

	go func() {
		actionConfig, cleanup, err := newHelmActionConfig(clusterID, namespace)
		if err != nil {
			db.DB.Model(record).Updates(map[string]any{"status": string(entities.ClusterExtensionStatusFailed), "error_message": err.Error()})
			return
		}
		defer cleanup()

		chrt, err := pullChart(ext.OCIUrl, targetVersion, actionConfig.RegistryClient)
		if err != nil {
			db.DB.Model(record).Updates(map[string]any{"status": string(entities.ClusterExtensionStatusFailed), "error_message": err.Error()})
			return
		}

		upgrader := action.NewUpgrade(actionConfig)
		upgrader.Namespace = namespace

		vals, err := parseValues(record.Values)
		if err != nil {
			db.DB.Model(record).Updates(map[string]any{"status": string(entities.ClusterExtensionStatusFailed), "error_message": err.Error()})
			return
		}

		if _, err := upgrader.RunWithContext(context.Background(), releaseName, chrt, vals); err != nil {
			db.DB.Model(record).Updates(map[string]any{"status": string(entities.ClusterExtensionStatusFailed), "error_message": err.Error()})
			return
		}

		db.DB.Model(record).Updates(map[string]any{"status": string(entities.ClusterExtensionStatusDeployed), "error_message": "", "phase": ""})
	}()

	m := toClusterExtensionModel(record)
	return &m, nil
}

// UninstallClusterExtension sets status to "uninstalling" and runs helm uninstall asynchronously.
// On success the DB record is hard-deleted; on failure status is set to "failed" with phase="uninstalling".
func UninstallClusterExtension(clusterID, id string) error {
	record, err := getClusterExtensionEntity(clusterID, id)
	if err != nil {
		return err
	}

	releaseName := record.ReleaseName
	namespace := record.Namespace

	// Mark as uninstalling (do NOT delete yet).
	if err := db.DB.Model(record).Updates(map[string]any{
		"status":        string(entities.ClusterExtensionStatusUninstalling),
		"phase":         "uninstalling",
		"error_message": "",
	}).Error; err != nil {
		return fmt.Errorf("failed to update cluster extension status: %w", err)
	}

	go func() {
		actionConfig, cleanup, err := newHelmActionConfig(clusterID, namespace)
		if err != nil {
			log.Printf("[extension] failed to init helm config for uninstall: %v", err)
			db.DB.Model(record).Updates(map[string]any{
				"status":        string(entities.ClusterExtensionStatusFailed),
				"error_message": err.Error(),
			})
			return
		}
		defer cleanup()

		uninstaller := action.NewUninstall(actionConfig)
		if _, err := uninstaller.Run(releaseName); err != nil {
			log.Printf("[extension] helm uninstall %q failed: %v", releaseName, err)
			db.DB.Model(record).Updates(map[string]any{
				"status":        string(entities.ClusterExtensionStatusFailed),
				"error_message": err.Error(),
			})
			return
		}

		// Success: hard-delete the record.
		db.DB.Unscoped().Delete(record)
	}()

	return nil
}

// getClusterExtensionEntity fetches the raw entity (internal helper).
func getClusterExtensionEntity(clusterID, id string) (*entities.ClusterExtension, error) {
	var record entities.ClusterExtension
	if err := db.DB.Where("cluster_id = ? AND id = ?", clusterID, id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("cluster extension not found")
		}
		return nil, err
	}
	return &record, nil
}

// toClusterExtensionModel converts a ClusterExtension entity to the API model.
func toClusterExtensionModel(e *entities.ClusterExtension) models.ClusterExtension {
	return models.ClusterExtension{
		ID:           e.ID,
		ClusterID:    e.ClusterID,
		ExtensionID:  e.ExtensionID,
		Namespace:    e.Namespace,
		ReleaseName:  e.ReleaseName,
		Version:      e.Version,
		Values:       e.Values,
		Status:       string(e.Status),
		Phase:        e.Phase,
		ErrorMessage: e.ErrorMessage,
		CreatedAt:    e.CreatedAt,
	}
}

// ========================
// Helm helpers
// ========================

// newHelmActionConfig creates a helm action.Configuration backed by the cluster kubeconfig.
// The caller must call the returned cleanup function to remove the temp kubeconfig file.
func newHelmActionConfig(clusterID, namespace string) (*action.Configuration, func(), error) {
	cluster, err := GetCluster(clusterID)
	if err != nil {
		return nil, func() {}, err
	}

	// Write kubeconfig to a temp file since ConfigFlags expects a path.
	f, err := os.CreateTemp("", "ketches-kubeconfig-*")
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to create temp kubeconfig file: %w", err)
	}
	cleanup := func() {
		_ = os.Remove(f.Name())
	}
	if _, err := f.WriteString(cluster.KubeConfig); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("failed to write kubeconfig: %w", err)
	}
	_ = f.Close()

	// Validate the kubeconfig is parseable.
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.KubeConfig))
	if err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("invalid kubeconfig for cluster %q: %w", clusterID, err)
	}
	_ = restConfig

	if namespace == "" {
		namespace = "default"
	}

	kubeConfigPath := f.Name()
	cfgFlags := newConfigFlagsFromPath(kubeConfigPath, namespace)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(cfgFlags, namespace, "secrets", func(format string, v ...any) {
		log.Printf("[helm] "+format, v...)
	}); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("failed to init helm action config: %w", err)
	}

	// Attach a registry client for OCI pulls.
	regClient, err := registry.NewClient()
	if err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("failed to create registry client: %w", err)
	}
	actionConfig.RegistryClient = regClient

	return actionConfig, cleanup, nil
}

// pullChart downloads an OCI helm chart and returns the loaded chart object.
func pullChart(ociUrl, version string, regClient *registry.Client) (*chart.Chart, error) {
	dir, err := os.MkdirTemp("", "ketches-helm-pull-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	p := action.NewPullWithOpts(action.WithConfig(&action.Configuration{
		RegistryClient: regClient,
	}))
	p.Settings = cli.New()
	p.DestDir = dir
	p.Untar = true
	p.UntarDir = dir
	if version != "" {
		p.Version = version
	}

	if _, err := p.Run(ociUrl); err != nil {
		return nil, fmt.Errorf("failed to pull chart %q version %q: %w", ociUrl, version, err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read extracted chart dir: %w", err)
	}
	var chartDir string
	for _, e := range entries {
		if e.IsDir() {
			chartDir = dir + "/" + e.Name()
			break
		}
	}
	if chartDir == "" {
		return nil, fmt.Errorf("no chart directory found after pull for %q", ociUrl)
	}

	return loader.Load(chartDir)
}

// parseValues parses a YAML values string into a map. Empty string returns empty map.
func parseValues(raw string) (map[string]any, error) {
	if raw == "" {
		return map[string]any{}, nil
	}
	vals := map[string]any{}
	if err := yaml.Unmarshal([]byte(raw), &vals); err != nil {
		return nil, err
	}
	return vals, nil
}

// configFlagsAdapter implements genericclioptions.RESTClientGetter using a kubeconfig path.
// We embed ConfigFlags from k8s.io/cli-runtime to avoid re-implementing the interface.
type configFlagsAdapter struct {
	kubeConfigPath string
	namespace      string
}

// newConfigFlagsFromPath returns a CLI-runtime ConfigFlags pointing at the given kubeconfig path.
func newConfigFlagsFromPath(path, namespace string) *configFlagsAdapter {
	return &configFlagsAdapter{kubeConfigPath: path, namespace: namespace}
}

func (c *configFlagsAdapter) ToRESTConfig() (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags("", c.kubeConfigPath)
}

func (c *configFlagsAdapter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	restConfig, err := c.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	return memorycache.NewMemCacheClient(discovery.NewDiscoveryClientForConfigOrDie(restConfig)), nil
}

func (c *configFlagsAdapter) ToRESTMapper() (meta.RESTMapper, error) {
	dc, err := c.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(dc)
	return mapper, nil
}

func (c *configFlagsAdapter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: c.kubeConfigPath},
		&clientcmd.ConfigOverrides{},
	)
}

// toExtensionModel converts the DB entity to the API model.
func toExtensionModel(e *entities.Extension) models.Extension {
	return models.Extension{
		ID:          e.ID,
		Name:        e.Name,
		DisplayName: e.DisplayName,
		Description: e.Description,
		OCIUrl:      e.OCIUrl,
		IconURL:     e.IconURL,
		Builtin:     e.Builtin,
		CreatedAt:   e.CreatedAt,
	}
}
