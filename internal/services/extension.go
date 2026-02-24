package services

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

const (
	helmOperatorGroup     = "helm-operator.ketches.cn"
	helmOperatorVersion   = "v1alpha1"
	helmOperatorNamespace = "ketches"

	defaultSystemRepoName = "ketches-extensions"
	defaultSystemRepoURL  = "https://ketches.github.io/ketches-extension-charts"
)

var helmRepoGVR = schema.GroupVersionResource{
	Group:    helmOperatorGroup,
	Version:  helmOperatorVersion,
	Resource: "helmrepositories",
}

var helmReleaseGVR = schema.GroupVersionResource{
	Group:    helmOperatorGroup,
	Version:  helmOperatorVersion,
	Resource: "helmreleases",
}

// ========================
// HelmRepository Operations
// ========================

func ListHelmRepositories(clusterID string) ([]models.HelmRepositoryResponse, error) {
	dynClient, err := kube.GlobalClusterStore.GetDynamicClient(clusterID)
	if err != nil {
		return nil, err
	}

	list, err := dynClient.Resource(helmRepoGVR).Namespace(helmOperatorNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list helm repositories: %v", err)
	}

	repos := make([]models.HelmRepositoryResponse, 0, len(list.Items))
	for _, item := range list.Items {
		repos = append(repos, toHelmRepositoryResponse(&item))
	}
	return repos, nil
}

func GetHelmRepository(clusterID, name string) (*models.HelmRepositoryResponse, error) {
	dynClient, err := kube.GlobalClusterStore.GetDynamicClient(clusterID)
	if err != nil {
		return nil, err
	}

	obj, err := dynClient.Resource(helmRepoGVR).Namespace(helmOperatorNamespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get helm repository: %v", err)
	}

	resp := toHelmRepositoryResponse(obj)
	return &resp, nil
}

// GetChartValues returns the default values.yaml for a chart version from a Helm repository.
func GetChartValues(clusterID, repoName, chartName, version string) (string, error) {
	repoResp, err := GetHelmRepository(clusterID, repoName)
	if err != nil {
		return "", err
	}
	if repoResp.Type != "helm" {
		return "", fmt.Errorf("chart values are only supported for helm repositories, got type %q", repoResp.Type)
	}

	getters := getter.All(cli.New())
	// Resolve chart URL from repo index (no local repo file needed)
	chartURL, err := repo.FindChartInRepoURL(repoResp.URL, chartName, version, "", "", "", getters)
	if err != nil {
		return "", fmt.Errorf("failed to find chart in repository: %w", err)
	}

	dir, err := os.MkdirTemp("", "ketches-helm-values-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	// Download from resolved URL (version param ignored when ref is URL)
	dl := downloader.ChartDownloader{
		Out:     os.Stderr,
		Verify:  downloader.VerifyNever,
		Getters: getters,
	}
	savedPath, _, err := dl.DownloadTo(chartURL, "", dir)
	if err != nil {
		return "", fmt.Errorf("failed to download chart: %w", err)
	}

	chrt, err := loader.Load(savedPath)
	if err != nil {
		return "", fmt.Errorf("failed to load chart: %w", err)
	}

	values, err := chartutil.CoalesceValues(chrt, nil)
	if err != nil {
		return "", fmt.Errorf("failed to coalesce values: %w", err)
	}
	out, err := yaml.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("failed to marshal values: %w", err)
	}
	return string(out), nil
}

func CreateHelmRepository(clusterID string, req *models.CreateHelmRepositoryRequest) (*models.HelmRepositoryResponse, error) {
	dynClient, err := kube.GlobalClusterStore.GetDynamicClient(clusterID)
	if err != nil {
		return nil, err
	}

	repoType := req.Type
	if repoType == "" {
		repoType = "helm"
	}

	spec := map[string]any{
		"url":  req.URL,
		"type": repoType,
	}

	// Add basic auth if provided
	if req.Username != "" || req.Password != "" {
		auth := map[string]any{
			"basic": map[string]any{},
		}
		basic := auth["basic"].(map[string]any)
		if req.Username != "" {
			basic["username"] = req.Username
		}
		if req.Password != "" {
			basic["password"] = req.Password
		}
		spec["auth"] = auth
	}

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": helmOperatorGroup + "/" + helmOperatorVersion,
			"kind":       "HelmRepository",
			"metadata": map[string]any{
				"name":      req.Name,
				"namespace": helmOperatorNamespace,
			},
			"spec": spec,
		},
	}

	created, err := dynClient.Resource(helmRepoGVR).Namespace(helmOperatorNamespace).Create(context.Background(), obj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create helm repository: %v", err)
	}

	resp := toHelmRepositoryResponse(created)
	return &resp, nil
}

func DeleteHelmRepository(clusterID, name string) error {
	dynClient, err := kube.GlobalClusterStore.GetDynamicClient(clusterID)
	if err != nil {
		return err
	}

	return dynClient.Resource(helmRepoGVR).Namespace(helmOperatorNamespace).Delete(context.Background(), name, metav1.DeleteOptions{})
}

// EnsureDefaultHelmRepository creates the default system helm repository if it doesn't exist.
func EnsureDefaultHelmRepository(clusterID string) error {
	dynClient, err := kube.GlobalClusterStore.GetDynamicClient(clusterID)
	if err != nil {
		return err
	}

	// Check if the default repo already exists
	_, err = dynClient.Resource(helmRepoGVR).Namespace(helmOperatorNamespace).Get(context.Background(), defaultSystemRepoName, metav1.GetOptions{})
	if err == nil {
		// Already exists
		return nil
	}

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": helmOperatorGroup + "/" + helmOperatorVersion,
			"kind":       "HelmRepository",
			"metadata": map[string]any{
				"name":      defaultSystemRepoName,
				"namespace": helmOperatorNamespace,
				"labels": map[string]any{
					"app.kubernetes.io/managed-by": "ketches",
					"ketches.cn/system":            "true",
				},
			},
			"spec": map[string]any{
				"url":  defaultSystemRepoURL,
				"type": "helm",
			},
		},
	}

	_, err = dynClient.Resource(helmRepoGVR).Namespace(helmOperatorNamespace).Create(context.Background(), obj, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create default helm repository: %v", err)
	}

	return nil
}

// ========================
// HelmRelease (Extension) Operations
// ========================

func ListExtensions(clusterID string) ([]models.ExtensionResponse, error) {
	dynClient, err := kube.GlobalClusterStore.GetDynamicClient(clusterID)
	if err != nil {
		return nil, err
	}

	list, err := dynClient.Resource(helmReleaseGVR).Namespace(helmOperatorNamespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list extensions: %v", err)
	}

	extensions := make([]models.ExtensionResponse, 0, len(list.Items))
	for _, item := range list.Items {
		extensions = append(extensions, toExtensionResponse(&item))
	}
	return extensions, nil
}

func GetExtension(clusterID, name string) (*models.ExtensionResponse, error) {
	dynClient, err := kube.GlobalClusterStore.GetDynamicClient(clusterID)
	if err != nil {
		return nil, err
	}

	obj, err := dynClient.Resource(helmReleaseGVR).Namespace(helmOperatorNamespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get extension: %v", err)
	}

	resp := toExtensionResponse(obj)
	return &resp, nil
}

func InstallExtension(clusterID string, req *models.InstallExtensionRequest) (*models.ExtensionResponse, error) {
	dynClient, err := kube.GlobalClusterStore.GetDynamicClient(clusterID)
	if err != nil {
		return nil, err
	}

	releaseNamespace := req.ReleaseNamespace
	if releaseNamespace == "" {
		releaseNamespace = "default"
	}

	// Build chart spec
	chart := map[string]any{
		"name": req.ChartName,
	}
	if req.ChartVersion != "" {
		chart["version"] = req.ChartVersion
	}
	if req.Repository != "" {
		chart["repository"] = map[string]any{
			"name":      req.Repository,
			"namespace": helmOperatorNamespace,
		}
	}
	if req.RepositoryURL != "" {
		chart["repositoryURL"] = req.RepositoryURL
	}
	if req.OCIRepository != "" {
		chart["ociRepository"] = req.OCIRepository
	}

	spec := map[string]any{
		"chart": chart,
		"release": map[string]any{
			"name":            req.Name,
			"namespace":       releaseNamespace,
			"createNamespace": req.CreateNamespace,
		},
	}

	if req.Values != "" {
		spec["values"] = req.Values
	}

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": helmOperatorGroup + "/" + helmOperatorVersion,
			"kind":       "HelmRelease",
			"metadata": map[string]any{
				"name":      req.Name,
				"namespace": helmOperatorNamespace,
				"labels": map[string]any{
					"app.kubernetes.io/managed-by": "ketches",
				},
			},
			"spec": spec,
		},
	}

	created, err := dynClient.Resource(helmReleaseGVR).Namespace(helmOperatorNamespace).Create(context.Background(), obj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to install extension: %v", err)
	}

	resp := toExtensionResponse(created)
	return &resp, nil
}

func UpdateExtension(clusterID, name string, req *models.UpdateExtensionRequest) (*models.ExtensionResponse, error) {
	dynClient, err := kube.GlobalClusterStore.GetDynamicClient(clusterID)
	if err != nil {
		return nil, err
	}

	obj, err := dynClient.Resource(helmReleaseGVR).Namespace(helmOperatorNamespace).Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get extension: %v", err)
	}

	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	if spec == nil {
		return nil, fmt.Errorf("extension spec not found")
	}

	if req.ChartVersion != "" {
		if err := unstructured.SetNestedField(obj.Object, req.ChartVersion, "spec", "chart", "version"); err != nil {
			return nil, err
		}
	}

	if req.Values != "" {
		if err := unstructured.SetNestedField(obj.Object, req.Values, "spec", "values"); err != nil {
			return nil, err
		}
	}

	if req.Suspended != nil {
		if err := unstructured.SetNestedField(obj.Object, *req.Suspended, "spec", "suspend"); err != nil {
			return nil, err
		}
	}

	updated, err := dynClient.Resource(helmReleaseGVR).Namespace(helmOperatorNamespace).Update(context.Background(), obj, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update extension: %v", err)
	}

	resp := toExtensionResponse(updated)
	return &resp, nil
}

func UninstallExtension(clusterID, name string) error {
	dynClient, err := kube.GlobalClusterStore.GetDynamicClient(clusterID)
	if err != nil {
		return err
	}

	return dynClient.Resource(helmReleaseGVR).Namespace(helmOperatorNamespace).Delete(context.Background(), name, metav1.DeleteOptions{})
}

// ========================
// Response Converters
// ========================

func toHelmRepositoryResponse(obj *unstructured.Unstructured) models.HelmRepositoryResponse {
	url, _, _ := unstructured.NestedString(obj.Object, "spec", "url")
	repoType, _, _ := unstructured.NestedString(obj.Object, "spec", "type")
	if repoType == "" {
		repoType = "helm"
	}

	// Check system label
	labels, _, _ := unstructured.NestedStringMap(obj.Object, "metadata", "labels")
	system := labels["ketches.cn/system"] == "true"

	// Parse status conditions
	ready := false
	message := ""
	conditions, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	for _, c := range conditions {
		cond, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if condType, _ := cond["type"].(string); condType == "Ready" {
			ready = cond["status"] == "True"
			message, _ = cond["message"].(string)
		}
	}

	// Parse charts from status
	charts := make([]models.HelmChartInfo, 0)
	chartsRaw, _, _ := unstructured.NestedSlice(obj.Object, "status", "charts")
	for _, ch := range chartsRaw {
		chartMap, ok := ch.(map[string]any)
		if !ok {
			continue
		}
		chartInfo := models.HelmChartInfo{
			Name:        getStringField(chartMap, "name"),
			Description: getStringField(chartMap, "description"),
		}

		versionsRaw, ok := chartMap["versions"].([]any)
		if ok {
			for _, v := range versionsRaw {
				vMap, ok := v.(map[string]any)
				if !ok {
					continue
				}
				vInfo := models.HelmChartVersionInfo{
					Version:    getStringField(vMap, "version"),
					AppVersion: getStringField(vMap, "appVersion"),
				}
				if createdStr := getStringField(vMap, "created"); createdStr != "" {
					if t, err := time.Parse(time.RFC3339, createdStr); err == nil {
						vInfo.Created = &t
					}
				}
				chartInfo.Versions = append(chartInfo.Versions, vInfo)
			}
		}
		charts = append(charts, chartInfo)
	}

	// Parse stats
	totalCharts, _, _ := unstructured.NestedFieldNoCopy(obj.Object, "status", "stats", "totalCharts")
	totalChartsInt := 0
	if tc, ok := totalCharts.(int64); ok {
		totalChartsInt = int(tc)
	}

	// Parse last sync time
	var lastSyncTime *time.Time
	if syncTimeStr, ok, _ := unstructured.NestedString(obj.Object, "status", "lastSyncTime"); ok && syncTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, syncTimeStr); err == nil {
			lastSyncTime = &t
		}
	}

	// Parse creation time
	createdAt := obj.GetCreationTimestamp().Time

	return models.HelmRepositoryResponse{
		Name:         obj.GetName(),
		URL:          url,
		Type:         repoType,
		Ready:        ready,
		Message:      message,
		Charts:       charts,
		TotalCharts:  totalChartsInt,
		LastSyncTime: lastSyncTime,
		CreatedAt:    createdAt,
		System:       system,
	}
}

func toExtensionResponse(obj *unstructured.Unstructured) models.ExtensionResponse {
	chartName, _, _ := unstructured.NestedString(obj.Object, "spec", "chart", "name")
	chartVersion, _, _ := unstructured.NestedString(obj.Object, "spec", "chart", "version")
	repoName, _, _ := unstructured.NestedString(obj.Object, "spec", "chart", "repository", "name")
	ociRepo, _, _ := unstructured.NestedString(obj.Object, "spec", "chart", "ociRepository")
	releaseName, _, _ := unstructured.NestedString(obj.Object, "spec", "release", "name")
	releaseNamespace, _, _ := unstructured.NestedString(obj.Object, "spec", "release", "namespace")
	values, _, _ := unstructured.NestedString(obj.Object, "spec", "values")
	suspended, _, _ := unstructured.NestedBool(obj.Object, "spec", "suspend")

	if releaseName == "" {
		releaseName = obj.GetName()
	}
	if releaseNamespace == "" {
		releaseNamespace = "default"
	}

	// Parse status
	ready := false
	message := ""
	conditions, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	for _, c := range conditions {
		cond, ok := c.(map[string]any)
		if !ok {
			continue
		}
		if condType, _ := cond["type"].(string); condType == "Ready" {
			ready = cond["status"] == "True"
			message, _ = cond["message"].(string)
		}
	}

	status, _, _ := unstructured.NestedString(obj.Object, "status", "helmRelease", "status")
	revision, _, _ := unstructured.NestedFieldNoCopy(obj.Object, "status", "helmRelease", "revision")
	revisionInt := 0
	if r, ok := revision.(int64); ok {
		revisionInt = int(r)
	}
	appVersion, _, _ := unstructured.NestedString(obj.Object, "status", "helmRelease", "appVersion")
	originalValues, _, _ := unstructured.NestedString(obj.Object, "status", "originalValues")

	return models.ExtensionResponse{
		Name:             obj.GetName(),
		ChartName:        chartName,
		ChartVersion:     chartVersion,
		Repository:       repoName,
		OCIRepository:    ociRepo,
		ReleaseNamespace: releaseNamespace,
		ReleaseName:      releaseName,
		Status:           status,
		Ready:            ready,
		Message:          message,
		Revision:         revisionInt,
		AppVersion:       appVersion,
		Suspended:        suspended,
		Values:           values,
		OriginalValues:   originalValues,
		CreatedAt:        obj.GetCreationTimestamp().Time,
	}
}

func getStringField(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
