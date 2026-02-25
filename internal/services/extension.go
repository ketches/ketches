package services

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/ketches/ketches/internal/models"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
	"helm.sh/helm/v3/pkg/release"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	 memorycache "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/yaml"
)

// ListExtensionVersions lists OCI tags for a catalog item, sorted newest first.
func ListExtensionVersions(itemID string) ([]models.ExtensionVersionInfo, error) {
	item, err := GetExtensionCatalogItemEntity(itemID)
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
func GetExtensionValues(itemID, version string) (string, error) {
	item, err := GetExtensionCatalogItemEntity(itemID)
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

// ListExtensions lists all helm releases installed in a cluster.
func ListExtensions(clusterID string) ([]models.InstalledExtension, error) {
	actionConfig, cleanup, err := newHelmActionConfig(clusterID, "")
	if err != nil {
		return nil, err
	}
	defer cleanup()

	l := action.NewList(actionConfig)
	l.AllNamespaces = true
	l.SetStateMask()

	releases, err := l.Run()
	if err != nil {
		return nil, fmt.Errorf("failed to list extensions: %w", err)
	}

	result := make([]models.InstalledExtension, 0, len(releases))
	for _, r := range releases {
		result = append(result, toInstalledExtension(r))
	}
	return result, nil
}

// GetExtension returns a single installed helm release by name.
func GetExtension(clusterID, name string) (*models.InstalledExtension, error) {
	// Get the release status.
	releases, err := ListExtensions(clusterID)
	if err != nil {
		return nil, err
	}
	for _, ext := range releases {
		if ext.Name == name {
			return &ext, nil
		}
	}
	return nil, fmt.Errorf("extension %q not found", name)
}

// InstallExtension installs an OCI helm chart into the specified cluster.
func InstallExtension(clusterID string, req *models.InstallExtensionRequest) (*models.InstalledExtension, error) {
	// Resolve catalog item to get OCI URL.
	item, err := GetExtensionCatalogItemEntity(req.CatalogItemID)
	if err != nil {
		return nil, fmt.Errorf("catalog item not found: %w", err)
	}

	releaseNamespace := req.ReleaseNamespace
	if releaseNamespace == "" {
		releaseNamespace = "default"
	}

	actionConfig, cleanup, err := newHelmActionConfig(clusterID, releaseNamespace)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	// Pull the chart.
	chrt, err := pullChart(item.OCIUrl, req.ChartVersion, actionConfig.RegistryClient)
	if err != nil {
		return nil, err
	}

	installer := action.NewInstall(actionConfig)
	installer.ReleaseName = req.Name
	installer.Namespace = releaseNamespace
	installer.CreateNamespace = req.CreateNamespace

	vals, err := parseValues(req.Values)
	if err != nil {
		return nil, fmt.Errorf("failed to parse values: %w", err)
	}

	rel, err := installer.RunWithContext(context.Background(), chrt, vals)
	if err != nil {
		return nil, fmt.Errorf("failed to install extension %q: %w", req.Name, err)
	}

	ext := toInstalledExtension(rel)
	ext.CatalogItemID = req.CatalogItemID
	return &ext, nil
}

// UpdateExtension upgrades an installed helm release with optional new version / values.
func UpdateExtension(clusterID, name string, req *models.UpdateExtensionRequest) (*models.InstalledExtension, error) {
	// Look up the current release to obtain the OCI URL and namespace.
	releases, err := ListExtensions(clusterID)
	if err != nil {
		return nil, err
	}
	var current *models.InstalledExtension
	for i := range releases {
		if releases[i].Name == name {
			current = &releases[i]
			break
		}
	}
	if current == nil {
		return nil, fmt.Errorf("extension %q not found", name)
	}

	actionConfig, cleanup, err := newHelmActionConfig(clusterID, current.ReleaseNamespace)
	if err != nil {
		return nil, err
	}
	defer cleanup()

	targetVersion := req.ChartVersion
	if targetVersion == "" {
		targetVersion = current.ChartVersion
	}

	chrt, err := pullChart(current.OCIUrl, targetVersion, actionConfig.RegistryClient)
	if err != nil {
		return nil, err
	}

	upgrader := action.NewUpgrade(actionConfig)
	upgrader.Namespace = current.ReleaseNamespace

	vals, err := parseValues(req.Values)
	if err != nil {
		return nil, fmt.Errorf("failed to parse values: %w", err)
	}

	rel, err := upgrader.RunWithContext(context.Background(), name, chrt, vals)
	if err != nil {
		return nil, fmt.Errorf("failed to update extension %q: %w", name, err)
	}

	ext := toInstalledExtension(rel)
	ext.CatalogItemID = current.CatalogItemID
	return &ext, nil
}

// UninstallExtension removes an installed helm release from a cluster.
func UninstallExtension(clusterID, name string) error {
	// Find the release to get its namespace.
	releases, err := ListExtensions(clusterID)
	if err != nil {
		return err
	}
	var releaseNamespace string
	for _, ext := range releases {
		if ext.Name == name {
			releaseNamespace = ext.ReleaseNamespace
			break
		}
	}
	if releaseNamespace == "" {
		releaseNamespace = "default"
	}

	actionConfig, cleanup, err := newHelmActionConfig(clusterID, releaseNamespace)
	if err != nil {
		return err
	}
	defer cleanup()

	uninstaller := action.NewUninstall(actionConfig)
	if _, err := uninstaller.Run(name); err != nil {
		return fmt.Errorf("failed to uninstall extension %q: %w", name, err)
	}
	return nil
}

// ========================
// Internal helpers
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

// toInstalledExtension converts a helm release to the API model.
func toInstalledExtension(r *release.Release) models.InstalledExtension {
	status := ""
	if r.Info != nil {
		status = r.Info.Status.String()
	}
	createdAt := time.Time{}
	if r.Info != nil {
		createdAt = r.Info.FirstDeployed.Time
	}
	chartVersion := ""
	appVersion := ""
	ociUrl := ""
	if r.Chart != nil && r.Chart.Metadata != nil {
		chartVersion = r.Chart.Metadata.Version
		appVersion = r.Chart.Metadata.AppVersion
		// For OCI charts, reconstruct the OCI URL from chart metadata sources.
		if len(r.Chart.Metadata.Sources) > 0 {
			ociUrl = r.Chart.Metadata.Sources[0]
		}
	}

	vals := ""
	if len(r.Config) > 0 {
		if out, err := yaml.Marshal(r.Config); err == nil {
			vals = string(out)
		}
	}

	return models.InstalledExtension{
		Name:             r.Name,
		OCIUrl:           ociUrl,
		ChartVersion:     chartVersion,
		ReleaseNamespace: r.Namespace,
		Status:           status,
		AppVersion:       appVersion,
		Values:           vals,
		Revision:         r.Version,
		CreatedAt:        createdAt,
	}
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
