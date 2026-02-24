package core

import (
	"context"
	"fmt"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SyncVolumeToK8s synchronizes a volume to Kubernetes (creates/updates PVC for persistent volumes)
func SyncVolumeToK8s(ctx context.Context, app *entities.App, volume *entities.AppVolume) error {
	if app.Env.ClusterID == "" {
		return fmt.Errorf("app environment has no cluster configured")
	}

	// Only create PVC for pvc type volumes
	if volume.VolumeType != "pvc" {
		return nil
	}

	client, err := kube.GlobalClusterStore.GetClient(app.Env.ClusterID)
	if err != nil {
		return err
	}

	metadata := &AppMetadata{App: app}
	
	// Build PVC
	pvc := metadata.BuildPVC(*volume)
	
	// Try to get existing PVC
	_, err = client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(ctx, pvc.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new PVC
			_, err = client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// PVC already exists - note: PVC specs cannot be updated after creation
	// Only certain fields like resources.requests can be updated
	// For major changes, user should delete and recreate
	return nil
}

// DeleteVolumeFromK8s deletes a PVC from Kubernetes
func DeleteVolumeFromK8s(ctx context.Context, app *entities.App, volume *entities.AppVolume) error {
	if app.Env.ClusterID == "" {
		return fmt.Errorf("app environment has no cluster configured")
	}

	// Only delete PVC for pvc type volumes
	if volume.VolumeType != "pvc" {
		return nil
	}

	client, err := kube.GlobalClusterStore.GetClient(app.Env.ClusterID)
	if err != nil {
		return err
	}

	// Delete PVC
	err = client.CoreV1().PersistentVolumeClaims(app.Env.ClusterNamespace).Delete(
		ctx,
		volume.Slug,
		metav1.DeleteOptions{},
	)
	
	if err != nil && !errors.IsNotFound(err) {
		return err
	}

	return nil
}
