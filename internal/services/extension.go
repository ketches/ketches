package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/core"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/internal/secrets"
	"github.com/ketches/ketches/pkg/concurrency"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	memorycache "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

type builtinExtensionSeed struct {
	Name         string
	DisplayName  string
	Description  string
	Capabilities []string
	Metadata     map[string]any
	OCIUrl       string
	IconURL      string
}

var builtinExtensions = []builtinExtensionSeed{
	{
		Name:         "envoyGateway",
		DisplayName:  "Envoy Gateway",
		Description:  "Envoy Gateway provides Kubernetes Gateway API implementation based on Envoy Proxy.",
		Capabilities: []string{"gateway-api"},
		Metadata: map[string]any{
			"gateway_api": map[string]any{
				"controller_name": "gateway.envoyproxy.io/gatewayclass-controller",
			},
		},
		OCIUrl:  "oci://docker.io/envoyproxy/gateway-helm",
		IconURL: "",
	},
	{
		Name:         "kube-prometheus-stack",
		DisplayName:  "Kube Prometheus Stack",
		Description:  "Prometheus community monitoring stack for Kubernetes.",
		Capabilities: []string{"observability"},
		OCIUrl:       "oci://ghcr.io/prometheus-community/charts/kube-prometheus-stack",
		IconURL:      "",
	},
	{
		Name:        "metrics-server",
		DisplayName: "Metrics Server",
		Description: "Cluster-wide resource metrics aggregator for Kubernetes autoscaling.",
		OCIUrl:      "oci://registry-1.docker.io/bitnamicharts/metrics-server",
		IconURL:     "",
	},
}

var (
	launchClusterExtensionInstall          = defaultLaunchClusterExtensionInstall
	launchClusterExtensionUpgrade          = defaultLaunchClusterExtensionUpgrade
	launchClusterExtensionUninstall        = defaultLaunchClusterExtensionUninstall
	executeClusterExtensionInstall         = runClusterExtensionInstall
	ensureGatewayClassForExtensionInstall  = defaultEnsureGatewayClassForExtensionInstall
	ensureSharedGatewayForExtensionInstall = defaultEnsureSharedGatewayForExtensionInstall
)

// EnsureBuiltinExtensions seeds built-in OCI extensions during startup.
// Existing rows are updated to keep built-in metadata and OCI URL current.
// It uses a worker-pool upsert flow so adding more built-ins scales better.
func EnsureBuiltinExtensions() error {
	return runBuiltinExtensionUpserts(builtinExtensions)
}

func runBuiltinExtensionUpserts(items []builtinExtensionSeed) error {
	if err := concurrency.Run(items, 0, upsertBuiltinExtension); err != nil {
		return app.WrapErrorf(err, "builtin extension upsert failed: %w", err)
	}
	return nil
}

func upsertBuiltinExtension(ext builtinExtensionSeed) error {
	return db.DB.Transaction(func(tx *gorm.DB) error {
		var existing entities.Extension
		err := tx.Where("name = ?", ext.Name).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			item := &entities.Extension{
				ID:           uuid.New(),
				Name:         ext.Name,
				DisplayName:  ext.DisplayName,
				Description:  ext.Description,
				Capabilities: mustMarshalExtensionCapabilities(ext.Capabilities),
				Metadata:     mustMarshalExtensionMetadata(ext.Metadata),
				OCIUrl:       ext.OCIUrl,
				IconURL:      ext.IconURL,
				Builtin:      true,
				CreatedBy:    nil,
			}
			if err := tx.Create(item).Error; err != nil {
				return app.WrapErrorf(err, "failed to seed built-in extension %q: %w", ext.Name, err)
			}
			return nil
		}
		if err != nil {
			return app.WrapErrorf(err, "failed to query extension %q: %w", ext.Name, err)
		}

		updates := map[string]any{
			"display_name": ext.DisplayName,
			"description":  ext.Description,
			"capabilities": mustMarshalExtensionCapabilities(ext.Capabilities),
			"metadata":     mustMarshalExtensionMetadata(ext.Metadata),
			"oci_url":      ext.OCIUrl,
			"icon_url":     ext.IconURL,
			"builtin":      true,
		}
		if err := tx.Model(&entities.Extension{}).Where("id = ?", existing.ID).Updates(updates).Error; err != nil {
			return app.WrapErrorf(err, "failed to update built-in extension %q: %w", ext.Name, err)
		}
		return nil
	})
}

// ListExtensions returns all extensions (builtin + admin-added) with install counts from DB.
func ListExtensions() ([]models.Extension, error) {
	var items []entities.Extension
	if err := db.DB.Order("builtin DESC, created_at ASC").Find(&items).Error; err != nil {
		return nil, app.WrapErrorf(err, "failed to list extensions: %w", err)
	}

	// Count installs per extension from the cluster_extensions table (fast, no Helm calls).
	type installCountRow struct {
		ExtensionID string
		Count       int
	}
	var counts []installCountRow
	db.DB.Model(&entities.ClusterExtension{}).
		Select("extension_id, COUNT(*) as count").
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
			return nil, app.NewErrorf("extension not found")
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
			return nil, app.NewErrorf("extension not found")
		}
		return nil, err
	}
	return &item, nil
}

// CreateExtension creates a new admin-added extension.
func CreateExtension(req *models.CreateExtensionRequest, createdBy string) (*models.Extension, error) {
	var existing entities.Extension
	if err := db.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		return nil, app.NewErrorf("extension with name %q already exists", req.Name)
	}
	item := &entities.Extension{
		ID:           uuid.New(),
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		Capabilities: sanitizeAndMarshalExtensionCapabilities(req.Capabilities),
		Metadata:     sanitizeAndMarshalExtensionMetadata(req.Metadata),
		OCIUrl:       req.OCIUrl,
		IconURL:      req.IconURL,
		Builtin:      false,
		CreatedBy:    toNullableString(createdBy),
	}
	if err := db.DB.Create(item).Error; err != nil {
		return nil, app.WrapErrorf(err, "failed to create extension: %w", err)
	}
	m := toExtensionModel(item)
	return &m, nil
}

// DeleteExtension removes a extension by ID (builtin extensions cannot be deleted).
func DeleteExtension(extensionID string) error {
	var item entities.Extension
	if err := db.DB.Where("id = ?", extensionID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return app.NewErrorf("extension not found")
		}
		return err
	}
	if item.Builtin {
		return app.NewErrorf("built-in extensions cannot be deleted")
	}
	return db.DB.Delete(&item).Error
}

// UpdateExtension updates a non-builtin extension's metadata.
func UpdateExtension(extensionID string, req *models.UpdateExtensionRequest) (*models.Extension, error) {
	var item entities.Extension
	if err := db.DB.Where("id = ?", extensionID).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, app.NewErrorf("extension not found")
		}
		return nil, err
	}
	if item.Builtin {
		return nil, app.NewErrorf("built-in extensions cannot be modified")
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
	if req.Capabilities != nil {
		item.Capabilities = sanitizeAndMarshalExtensionCapabilities(*req.Capabilities)
	}
	item.Metadata = sanitizeAndMarshalExtensionMetadata(req.Metadata)
	item.IconURL = req.IconURL
	if err := db.DB.Save(&item).Error; err != nil {
		return nil, app.WrapErrorf(err, "failed to update extension: %w", err)
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
		Name        string
		ReleaseName string
		Namespace   string
		Version     string
		Status      string
	}
	var rows []row
	err := db.DB.Table("cluster_extensions ce").
		Select("ce.cluster_id, c.name as cluster_name, ce.name, ce.release_name, ce.namespace, ce.version, ce.status").
		Joins("JOIN clusters c ON c.id = ce.cluster_id").
		Where("ce.extension_id = ? AND c.deleted_at IS NULL", extensionID).
		Scan(&rows).Error
	if err != nil {
		return nil, app.WrapErrorf(err, "failed to query installed clusters: %w", err)
	}

	result := make([]models.InstalledCluster, 0, len(rows))
	for _, r := range rows {
		result = append(result, models.InstalledCluster{
			ClusterID:   r.ClusterID,
			ClusterName: r.ClusterName,
			Name:        firstNonEmpty(r.Name, r.ReleaseName),
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
		return nil, app.WrapErrorf(err, "failed to list versions for %q: %w", item.OCIUrl, err)
	}

	tags = sortExtensionVersions(tags)

	result := make([]models.ExtensionVersionInfo, 0, len(tags))
	for _, tag := range tags {
		result = append(result, models.ExtensionVersionInfo{Version: tag})
	}
	return result, nil
}

func sortExtensionVersions(tags []string) []string {
	type parsedVersion struct {
		original string
		version  *semver.Version
	}

	stableVersions := make([]parsedVersion, 0, len(tags))
	prereleaseVersions := make([]parsedVersion, 0, len(tags))
	otherTags := make([]string, 0, len(tags))

	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}

		version, err := semver.NewVersion(trimmed)
		if err != nil {
			otherTags = append(otherTags, trimmed)
			continue
		}

		item := parsedVersion{
			original: trimmed,
			version:  version,
		}
		if version.Prerelease() == "" {
			stableVersions = append(stableVersions, item)
			continue
		}
		prereleaseVersions = append(prereleaseVersions, item)
	}

	sort.Slice(stableVersions, func(i, j int) bool {
		return stableVersions[i].version.GreaterThan(stableVersions[j].version)
	})
	sort.Slice(prereleaseVersions, func(i, j int) bool {
		return prereleaseVersions[i].version.GreaterThan(prereleaseVersions[j].version)
	})
	sort.Sort(sort.Reverse(sort.StringSlice(otherTags)))

	result := make([]string, 0, len(stableVersions)+len(prereleaseVersions)+len(otherTags))
	for _, item := range stableVersions {
		result = append(result, item.original)
	}
	for _, item := range prereleaseVersions {
		result = append(result, item.original)
	}
	result = append(result, otherTags...)
	return result
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
		return "", app.WrapErrorf(err, "failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	regClient, err := registry.NewClient()
	if err != nil {
		return "", app.WrapErrorf(err, "failed to create registry client: %w", err)
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
		return "", app.WrapErrorf(err, "failed to pull chart %q version %q: %w", item.OCIUrl, version, err)
	}

	// The chart is extracted to dir/<chart-name>/
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", app.WrapErrorf(err, "failed to read extracted chart dir: %w", err)
	}
	var chartDir string
	for _, e := range entries {
		if e.IsDir() {
			chartDir = dir + "/" + e.Name()
			break
		}
	}
	if chartDir == "" {
		return "", app.NewErrorf("no chart directory found after pull")
	}

	chrt, err := loader.Load(chartDir)
	if err != nil {
		return "", app.WrapErrorf(err, "failed to load chart: %w", err)
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
			return "", app.WrapErrorf(err, "failed to marshal values: %w", err)
		}
		return string(out), nil
	}
	return "", nil
}

// ListClusterExtensions returns all cluster extensions for a cluster from the DB.
func ListClusterExtensions(clusterID string) ([]models.ClusterExtension, error) {
	var records []entities.ClusterExtension
	if err := db.DB.Where("cluster_id = ?", clusterID).Find(&records).Error; err != nil {
		return nil, app.WrapErrorf(err, "failed to list cluster extensions: %w", err)
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
			return nil, app.NewErrorf("cluster extension not found")
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
		return nil, app.WrapErrorf(err, "extension not found: %w", err)
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}
	normalizedReleaseName := sanitizeClusterExtensionReleaseName(req.ReleaseName)
	if err := validateClusterExtensionReleaseName(normalizedReleaseName); err != nil {
		return nil, err
	}

	// Check uniqueness: same cluster, namespace, extensionID.
	var existing entities.ClusterExtension
	err = db.DB.Where("cluster_id = ? AND namespace = ? AND extension_id = ?",
		clusterID, namespace, req.ExtensionID).First(&existing).Error
	if err == nil {
		return nil, app.NewErrorf("extension already installed in namespace %q", namespace)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, app.WrapErrorf(err, "failed to query existing cluster extension: %w", err)
	}

	record := &entities.ClusterExtension{
		ID:              uuid.New(),
		ClusterID:       clusterID,
		ExtensionID:     req.ExtensionID,
		Namespace:       namespace,
		ReleaseName:     normalizedReleaseName,
		Name:            clusterExtensionDisplayName(ext),
		Version:         req.Version,
		CreateNamespace: req.CreateNamespace,
		Values:          req.Values,
		Status:          entities.ClusterExtensionStatusPending,
		Phase:           "installing",
		InstalledBy:     toNullableString(installedBy),
	}
	if err := db.DB.Create(record).Error; err != nil {
		return nil, app.WrapErrorf(err, "failed to create cluster extension record: %w", err)
	}

	launchClusterExtensionInstall(clusterID, ext, record)

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

	updates := map[string]any{
		"status":        string(entities.ClusterExtensionStatusUpgrading),
		"phase":         "upgrading",
		"error_message": "",
	}
	if req.Version != "" {
		updates["version"] = req.Version
		record.Version = req.Version
	}
	if req.Values != "" {
		updates["values"] = req.Values
		record.Values = req.Values
	}
	if err := db.DB.Model(record).Updates(updates).Error; err != nil {
		return nil, app.WrapErrorf(err, "failed to update cluster extension status: %w", err)
	}
	record.Status = entities.ClusterExtensionStatusUpgrading
	record.Phase = "upgrading"
	record.ErrorMessage = ""

	launchClusterExtensionUpgrade(clusterID, ext, record)

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

	if err := db.DB.Model(record).Updates(map[string]any{
		"status":        string(entities.ClusterExtensionStatusUninstalling),
		"phase":         "uninstalling",
		"error_message": "",
	}).Error; err != nil {
		return app.WrapErrorf(err, "failed to update cluster extension status: %w", err)
	}
	record.Status = entities.ClusterExtensionStatusUninstalling
	record.Phase = "uninstalling"
	record.ErrorMessage = ""

	launchClusterExtensionUninstall(clusterID, record)

	return nil
}

func RetryClusterExtension(clusterID, id string, req *models.RetryClusterExtensionRequest) (*models.ClusterExtension, error) {
	record, err := getClusterExtensionEntity(clusterID, id)
	if err != nil {
		return nil, err
	}
	originalReleaseName := record.ReleaseName
	normalizedReleaseName := sanitizeClusterExtensionReleaseName(originalReleaseName)
	if normalizedReleaseName != originalReleaseName {
		if record.Phase != "" && record.Phase != "installing" {
			return nil, validateClusterExtensionReleaseName(normalizedReleaseName)
		}
		if err := validateClusterExtensionReleaseName(normalizedReleaseName); err != nil {
			return nil, err
		}
		if err := db.DB.Model(record).Update("release_name", normalizedReleaseName).Error; err != nil {
			return nil, app.WrapErrorf(err, "failed to update cluster extension release name: %w", err)
		}
		slog.Warn("[extension] normalized invalid release name for retry",
			"cluster_id", clusterID,
			"extension_id", record.ExtensionID,
			"old_release_name", originalReleaseName,
			"new_release_name", normalizedReleaseName,
		)
		record.ReleaseName = normalizedReleaseName
	}
	if err := validateClusterExtensionReleaseName(record.ReleaseName); err != nil {
		return nil, err
	}
	if record.Status != entities.ClusterExtensionStatusFailed {
		return nil, app.NewErrorf("only failed extensions can be retried")
	}

	switch record.Phase {
	case "", "installing":
		ext, err := GetExtensionEntity(record.ExtensionID)
		if err != nil {
			return nil, err
		}
		retryInstallUpdates, err := applyRetryInstallEdits(record, req)
		if err != nil {
			return nil, err
		}
		updates := map[string]any{
			"status":        string(entities.ClusterExtensionStatusPending),
			"phase":         "installing",
			"error_message": "",
		}
		for key, value := range retryInstallUpdates {
			updates[key] = value
		}
		if _, ok := updates["name"]; !ok {
			record.Name = firstNonEmpty(record.Name, clusterExtensionDisplayName(ext), record.ReleaseName)
			updates["name"] = record.Name
		}
		if err := db.DB.Model(record).Updates(updates).Error; err != nil {
			return nil, app.WrapErrorf(err, "failed to update cluster extension status: %w", err)
		}
		record.Status = entities.ClusterExtensionStatusPending
		record.Phase = "installing"
		record.ErrorMessage = ""
		launchClusterExtensionInstall(clusterID, ext, record)
	case "upgrading":
		ext, err := GetExtensionEntity(record.ExtensionID)
		if err != nil {
			return nil, err
		}
		if err := db.DB.Model(record).Updates(map[string]any{
			"status":        string(entities.ClusterExtensionStatusUpgrading),
			"phase":         "upgrading",
			"error_message": "",
		}).Error; err != nil {
			return nil, app.WrapErrorf(err, "failed to update cluster extension status: %w", err)
		}
		record.Status = entities.ClusterExtensionStatusUpgrading
		record.Phase = "upgrading"
		record.ErrorMessage = ""
		if record.Name == "" {
			record.Name = clusterExtensionDisplayName(ext)
			_ = db.DB.Model(record).Update("name", record.Name).Error
		}
		launchClusterExtensionUpgrade(clusterID, ext, record)
	case "uninstalling":
		if err := db.DB.Model(record).Updates(map[string]any{
			"status":        string(entities.ClusterExtensionStatusUninstalling),
			"phase":         "uninstalling",
			"error_message": "",
		}).Error; err != nil {
			return nil, app.WrapErrorf(err, "failed to update cluster extension status: %w", err)
		}
		record.Status = entities.ClusterExtensionStatusUninstalling
		record.Phase = "uninstalling"
		record.ErrorMessage = ""
		launchClusterExtensionUninstall(clusterID, record)
	default:
		return nil, app.NewErrorf("unsupported retry phase %q", record.Phase)
	}

	m := toClusterExtensionModel(record)
	return &m, nil
}

func reconcileClusterExtensionInstallSuccess(clusterID string, ext *entities.Extension, record *entities.ClusterExtension) error {
	if ext == nil || !extensionHasCapability(ext, "gateway-api") {
		return nil
	}

	controllerName := extensionGatewayControllerName(ext)
	if controllerName == "" {
		return app.NewErrorf("extension %q is marked as gateway-api but has no gateway controller name configured", ext.Name)
	}

	gatewayClassName := buildManagedGatewayClassName(record.ReleaseName)
	if err := ensureGatewayClassForExtensionInstall(clusterID, gatewayClassName, controllerName); err != nil {
		return err
	}

	provider := &entities.ClusterGatewayProvider{
		ID:                 uuid.New(),
		ClusterID:          clusterID,
		SourceType:         "managed",
		DisplayName:        firstNonEmpty(record.Name, clusterExtensionDisplayName(ext), gatewayClassName),
		GatewayClassName:   gatewayClassName,
		ControllerName:     controllerName,
		ExtensionID:        &record.ExtensionID,
		ClusterExtensionID: &record.ID,
	}
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&entities.ClusterGatewayProvider{}).Where("cluster_id = ? AND is_default = ?", clusterID, true).Count(&count).Error; err != nil {
			return err
		}
		provider.IsDefault = count == 0
		if err := upsertClusterGatewayProvider(tx, provider); err != nil {
			return err
		}
		if provider.IsDefault {
			if err := setDefaultClusterGatewayProvider(tx, clusterID, gatewayClassName); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if provider.IsDefault {
		if err := ensureSharedGatewayForExtensionInstall(clusterID); err != nil {
			return err
		}
	}

	return nil
}

func buildManagedGatewayClassName(releaseName string) string {
	return fmt.Sprintf("ketches-%s", sanitizeClusterExtensionReleaseName(releaseName))
}

func applyRetryInstallEdits(record *entities.ClusterExtension, req *models.RetryClusterExtensionRequest) (map[string]any, error) {
	updates := map[string]any{}
	if req == nil {
		return updates, nil
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, app.NewErrorf("name is required")
		}
		updates["name"] = name
		record.Name = name
	}
	if req.Version != nil {
		version := strings.TrimSpace(*req.Version)
		updates["version"] = version
		record.Version = version
	}
	if req.Values != nil {
		updates["values"] = *req.Values
		record.Values = *req.Values
	}
	return updates, nil
}

// getClusterExtensionEntity fetches the raw entity (internal helper).
func getClusterExtensionEntity(clusterID, id string) (*entities.ClusterExtension, error) {
	var record entities.ClusterExtension
	if err := db.DB.Where("cluster_id = ? AND id = ?", clusterID, id).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, app.NewErrorf("cluster extension not found")
		}
		return nil, err
	}
	return &record, nil
}

// toClusterExtensionModel converts a ClusterExtension entity to the API model.
func toClusterExtensionModel(e *entities.ClusterExtension) models.ClusterExtension {
	return models.ClusterExtension{
		ID:              e.ID,
		ClusterID:       e.ClusterID,
		ExtensionID:     e.ExtensionID,
		Name:            firstNonEmpty(e.Name, e.ReleaseName),
		Namespace:       e.Namespace,
		ReleaseName:     e.ReleaseName,
		Version:         e.Version,
		CreateNamespace: e.CreateNamespace,
		Values:          e.Values,
		Status:          string(e.Status),
		Phase:           e.Phase,
		ErrorMessage:    e.ErrorMessage,
		CreatedAt:       e.CreatedAt,
	}
}

func defaultEnsureGatewayClassForExtensionInstall(clusterID, gatewayClassName, controllerName string) error {
	return core.EnsureGatewayClass(context.Background(), clusterID, gatewayClassName, controllerName)
}

func defaultEnsureSharedGatewayForExtensionInstall(clusterID string) error {
	return core.EnsureSharedGateway(context.Background(), clusterID)
}

func defaultLaunchClusterExtensionInstall(clusterID string, ext *entities.Extension, record *entities.ClusterExtension) {
	go func() {
		if err := db.DB.Model(record).Updates(map[string]any{
			"status":        string(entities.ClusterExtensionStatusInstalling),
			"phase":         "installing",
			"error_message": "",
		}).Error; err != nil {
			slog.Error(fmt.Sprintf("[extension] failed to mark %q as installing: %v", record.ReleaseName, err))
			return
		}

		if err := executeClusterExtensionInstall(clusterID, ext, record); err != nil {
			slog.Error("[extension] install failed",
				"cluster_id", clusterID,
				"extension_id", record.ExtensionID,
				"release_name", record.ReleaseName,
				"namespace", record.Namespace,
				"version", record.Version,
				"error", err,
			)
			db.DB.Model(record).Updates(map[string]any{
				"status":        string(entities.ClusterExtensionStatusFailed),
				"phase":         "installing",
				"error_message": err.Error(),
			})
			return
		}

		if err := reconcileClusterExtensionInstallSuccess(clusterID, ext, record); err != nil {
			slog.Error("[extension] install post-reconcile failed",
				"cluster_id", clusterID,
				"extension_id", record.ExtensionID,
				"release_name", record.ReleaseName,
				"namespace", record.Namespace,
				"version", record.Version,
				"error", err,
			)
			db.DB.Model(record).Updates(map[string]any{
				"status":        string(entities.ClusterExtensionStatusFailed),
				"phase":         "installing",
				"error_message": err.Error(),
			})
			return
		}

		db.DB.Model(record).Updates(map[string]any{
			"status":        string(entities.ClusterExtensionStatusDeployed),
			"error_message": "",
			"phase":         "",
		})
	}()
}

func defaultLaunchClusterExtensionUpgrade(clusterID string, ext *entities.Extension, record *entities.ClusterExtension) {
	go func() {
		if err := runClusterExtensionUpgrade(clusterID, ext, record); err != nil {
			slog.Error("[extension] upgrade failed",
				"cluster_id", clusterID,
				"extension_id", record.ExtensionID,
				"release_name", record.ReleaseName,
				"namespace", record.Namespace,
				"version", record.Version,
				"error", err,
			)
			db.DB.Model(record).Updates(map[string]any{
				"status":        string(entities.ClusterExtensionStatusFailed),
				"phase":         "upgrading",
				"error_message": err.Error(),
			})
			return
		}

		db.DB.Model(record).Updates(map[string]any{
			"status":        string(entities.ClusterExtensionStatusDeployed),
			"error_message": "",
			"phase":         "",
		})
	}()
}

func defaultLaunchClusterExtensionUninstall(clusterID string, record *entities.ClusterExtension) {
	go func() {
		if err := runClusterExtensionUninstall(clusterID, record); err != nil {
			slog.Error(fmt.Sprintf("[extension] helm uninstall %q failed: %v", record.ReleaseName, err))
			db.DB.Model(record).Updates(map[string]any{
				"status":        string(entities.ClusterExtensionStatusFailed),
				"phase":         "uninstalling",
				"error_message": err.Error(),
			})
			return
		}

		db.DB.Unscoped().Delete(record)
	}()
}

func runClusterExtensionInstall(clusterID string, ext *entities.Extension, record *entities.ClusterExtension) error {
	actionConfig, cleanup, err := newHelmActionConfig(clusterID, record.Namespace)
	if err != nil {
		return err
	}
	defer cleanup()

	chrt, vals, err := prepareClusterExtensionChart(actionConfig, ext, record.Version, record.Values)
	if err != nil {
		return err
	}

	historyClient := action.NewHistory(actionConfig)
	historyClient.Max = 1
	versions, err := historyClient.Run(record.ReleaseName)
	if err != nil {
		if errors.Is(err, driver.ErrReleaseNotFound) {
			return runClusterExtensionHelmInstall(actionConfig, record, chrt, vals, false)
		}
		return err
	}

	if isClusterExtensionReleaseUninstalled(versions) {
		return runClusterExtensionHelmInstall(actionConfig, record, chrt, vals, true)
	}

	return runClusterExtensionHelmUpgrade(actionConfig, record, chrt, vals)
}

func runClusterExtensionUpgrade(clusterID string, ext *entities.Extension, record *entities.ClusterExtension) error {
	actionConfig, cleanup, err := newHelmActionConfig(clusterID, record.Namespace)
	if err != nil {
		return err
	}
	defer cleanup()

	chrt, vals, err := prepareClusterExtensionChart(actionConfig, ext, record.Version, record.Values)
	if err != nil {
		return err
	}

	return runClusterExtensionHelmUpgrade(actionConfig, record, chrt, vals)
}

func runClusterExtensionUninstall(clusterID string, record *entities.ClusterExtension) error {
	actionConfig, cleanup, err := newHelmActionConfig(clusterID, record.Namespace)
	if err != nil {
		return err
	}
	defer cleanup()

	uninstaller := action.NewUninstall(actionConfig)
	_, err = uninstaller.Run(record.ReleaseName)
	return err
}

func prepareClusterExtensionChart(actionConfig *action.Configuration, ext *entities.Extension, version string, values string) (*chart.Chart, map[string]any, error) {
	chrt, err := pullChart(ext.OCIUrl, version, actionConfig.RegistryClient)
	if err != nil {
		return nil, nil, err
	}

	vals, err := parseValues(values)
	if err != nil {
		return nil, nil, err
	}

	return chrt, vals, nil
}

func runClusterExtensionHelmInstall(actionConfig *action.Configuration, record *entities.ClusterExtension, chrt *chart.Chart, vals map[string]any, replace bool) error {
	installer := action.NewInstall(actionConfig)
	installer.ReleaseName = record.ReleaseName
	installer.Namespace = record.Namespace
	installer.CreateNamespace = record.CreateNamespace
	installer.Replace = replace
	_, err := installer.RunWithContext(context.Background(), chrt, vals)
	return err
}

func runClusterExtensionHelmUpgrade(actionConfig *action.Configuration, record *entities.ClusterExtension, chrt *chart.Chart, vals map[string]any) error {
	upgrader := action.NewUpgrade(actionConfig)
	upgrader.Namespace = record.Namespace
	_, err := upgrader.RunWithContext(context.Background(), record.ReleaseName, chrt, vals)
	return err
}

func isClusterExtensionReleaseUninstalled(versions []*release.Release) bool {
	if len(versions) == 0 {
		return false
	}
	latest := versions[len(versions)-1]
	if latest == nil || latest.Info == nil {
		return false
	}
	return latest.Info.Status == release.StatusUninstalled
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

	plaintextKubeConfig, err := secrets.DecryptString(cluster.KubeConfig)
	if err != nil {
		return nil, func() {}, app.WrapErrorf(err, "failed to decrypt kubeconfig: %w", err)
	}

	// Write kubeconfig to a temp file since ConfigFlags expects a path.
	f, err := os.CreateTemp("", "ketches-kubeconfig-*")
	if err != nil {
		return nil, func() {}, app.WrapErrorf(err, "failed to create temp kubeconfig file: %w", err)
	}
	cleanup := func() {
		_ = os.Remove(f.Name())
	}
	if _, err := f.WriteString(plaintextKubeConfig); err != nil {
		cleanup()
		return nil, func() {}, app.WrapErrorf(err, "failed to write kubeconfig: %w", err)
	}
	_ = f.Close()

	// Validate the kubeconfig is parseable.
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(plaintextKubeConfig))
	if err != nil {
		cleanup()
		return nil, func() {}, app.WrapErrorf(err, "invalid kubeconfig for cluster %q: %w", clusterID, err)
	}
	_ = restConfig

	if namespace == "" {
		namespace = "default"
	}

	kubeConfigPath := f.Name()
	cfgFlags := newConfigFlagsFromPath(kubeConfigPath, namespace)

	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(cfgFlags, namespace, "secrets", func(format string, v ...any) {
		slog.Info(fmt.Sprintf("[helm] "+format, v...))
	}); err != nil {
		cleanup()
		return nil, func() {}, app.WrapErrorf(err, "failed to init helm action config: %w", err)
	}

	// Attach a registry client for OCI pulls.
	regClient, err := registry.NewClient()
	if err != nil {
		cleanup()
		return nil, func() {}, app.WrapErrorf(err, "failed to create registry client: %w", err)
	}
	actionConfig.RegistryClient = regClient

	return actionConfig, cleanup, nil
}

// pullChart downloads an OCI helm chart and returns the loaded chart object.
func pullChart(ociUrl, version string, regClient *registry.Client) (*chart.Chart, error) {
	dir, err := os.MkdirTemp("", "ketches-helm-pull-*")
	if err != nil {
		return nil, app.WrapErrorf(err, "failed to create temp dir: %w", err)
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
		return nil, app.WrapErrorf(err, "failed to pull chart %q version %q: %w", ociUrl, version, err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, app.WrapErrorf(err, "failed to read extracted chart dir: %w", err)
	}
	var chartDir string
	for _, e := range entries {
		if e.IsDir() {
			chartDir = dir + "/" + e.Name()
			break
		}
	}
	if chartDir == "" {
		return nil, app.NewErrorf("no chart directory found after pull for %q", ociUrl)
	}

	return loader.Load(chartDir)
}

// parseValues parses a YAML values string into a map. Empty string returns empty map.
var nonHelmReleaseNameChars = regexp.MustCompile(`[^a-z0-9-]+`)
var repeatedHelmReleaseNameHyphens = regexp.MustCompile(`-+`)

func extensionMetadata(ext *entities.Extension) map[string]any {
	if ext == nil || len(ext.Metadata) == 0 {
		return nil
	}

	var metadata map[string]any
	if err := json.Unmarshal(ext.Metadata, &metadata); err != nil {
		return nil
	}
	return metadata
}

func extensionGatewayControllerName(ext *entities.Extension) string {
	metadata := extensionMetadata(ext)
	if metadata == nil {
		return ""
	}
	gatewayMetadata, ok := metadata["gateway_api"].(map[string]any)
	if !ok {
		return ""
	}
	controllerName, _ := gatewayMetadata["controller_name"].(string)
	return strings.TrimSpace(controllerName)
}

func sanitizeAndMarshalExtensionMetadata(metadata map[string]any) entities.JSONBlob {
	if len(metadata) == 0 {
		return nil
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return nil
	}
	return entities.JSONBlob(encoded)
}

func mustMarshalExtensionMetadata(metadata map[string]any) entities.JSONBlob {
	return sanitizeAndMarshalExtensionMetadata(metadata)
}

func extensionCapabilities(ext *entities.Extension) []string {
	if ext == nil || strings.TrimSpace(ext.Capabilities) == "" {
		return nil
	}

	var capabilities []string
	if err := json.Unmarshal([]byte(ext.Capabilities), &capabilities); err != nil {
		return nil
	}

	return sanitizeExtensionCapabilities(capabilities)
}

func extensionHasCapability(ext *entities.Extension, capability string) bool {
	for _, item := range extensionCapabilities(ext) {
		if item == capability {
			return true
		}
	}
	return false
}

func sanitizeExtensionCapabilities(capabilities []string) []string {
	seen := make(map[string]struct{}, len(capabilities))
	result := make([]string, 0, len(capabilities))
	for _, capability := range capabilities {
		normalized := strings.ToLower(strings.TrimSpace(capability))
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	sort.Strings(result)
	return result
}

func sanitizeAndMarshalExtensionCapabilities(capabilities []string) string {
	sanitized := sanitizeExtensionCapabilities(capabilities)
	if len(sanitized) == 0 {
		return ""
	}
	encoded, err := json.Marshal(sanitized)
	if err != nil {
		return ""
	}
	return string(encoded)
}

func mustMarshalExtensionCapabilities(capabilities []string) string {
	return sanitizeAndMarshalExtensionCapabilities(capabilities)
}

func clusterExtensionDisplayName(ext *entities.Extension) string {
	if ext == nil {
		return ""
	}
	return firstNonEmpty(ext.DisplayName, ext.Name)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func sanitizeClusterExtensionReleaseName(name string) string {
	if name == "" {
		return ""
	}

	var b strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			prev := rune(name[i-1])
			if (prev >= 'a' && prev <= 'z') || (prev >= '0' && prev <= '9') {
				b.WriteByte('-')
			}
		}
		b.WriteRune(r)
	}

	normalized := strings.ToLower(b.String())
	normalized = nonHelmReleaseNameChars.ReplaceAllString(normalized, "-")
	normalized = repeatedHelmReleaseNameHyphens.ReplaceAllString(normalized, "-")
	normalized = strings.Trim(normalized, "-")
	if len(normalized) > 53 {
		normalized = strings.Trim(normalized[:53], "-")
	}
	return normalized
}

func validateClusterExtensionReleaseName(name string) error {
	if err := chartutil.ValidateReleaseName(name); err != nil {
		return app.NewErrorf("invalid release name %q: use lowercase letters, numbers, hyphens, or dots, and keep it within 53 characters", name)
	}
	return nil
}

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
		ID:           e.ID,
		Name:         e.Name,
		DisplayName:  e.DisplayName,
		Description:  e.Description,
		Capabilities: extensionCapabilities(e),
		Metadata:     extensionMetadata(e),
		OCIUrl:       e.OCIUrl,
		IconURL:      e.IconURL,
		Builtin:      e.Builtin,
		CreatedAt:    e.CreatedAt,
	}
}

func toNullableString(v string) *string {
	if v == "" {
		return nil
	}
	vv := v
	return &vv
}
