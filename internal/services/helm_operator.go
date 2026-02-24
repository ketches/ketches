package services

import (
	"context"
	"fmt"
	"log"

	"github.com/ketches/helm-operator/pkg/installer"
	"github.com/ketches/ketches/internal/kube"
)

// GetHelmOperatorStatus checks the installation status of helm-operator in the specified cluster
func GetHelmOperatorStatus(clusterID string) (*installer.InstallationStatus, error) {
	cluster, err := GetCluster(clusterID)
	if err != nil {
		return nil, err
	}

	crClient, err := kube.CreateControllerRuntimeClientFromKubeConfig(cluster.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %v", err)
	}

	inst := installer.NewInstaller(crClient)
	return inst.GetStatus(context.Background())
}

// InstallHelmOperator installs helm-operator in the specified cluster and
// creates the default system helm repository.
func InstallHelmOperator(clusterID string) error {
	cluster, err := GetCluster(clusterID)
	if err != nil {
		return err
	}

	crClient, err := kube.CreateControllerRuntimeClientFromKubeConfig(cluster.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	inst := installer.NewInstaller(crClient)
	if err := inst.Install(context.Background()); err != nil {
		return err
	}

	// Auto-add default system helm repository after installation
	if err := EnsureDefaultHelmRepository(clusterID); err != nil {
		log.Printf("Warning: failed to create default helm repository for cluster %s: %v", clusterID, err)
		// Don't fail the whole install just because the default repo failed
	}

	return nil
}

// UninstallHelmOperator uninstalls helm-operator from the specified cluster
func UninstallHelmOperator(clusterID string) error {
	cluster, err := GetCluster(clusterID)
	if err != nil {
		return err
	}

	crClient, err := kube.CreateControllerRuntimeClientFromKubeConfig(cluster.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	inst := installer.NewInstaller(crClient)
	return inst.Uninstall(context.Background())
}
