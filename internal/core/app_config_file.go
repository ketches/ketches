package core

import (
	"context"
	"fmt"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SyncConfigMapToK8s synchronizes config files to a Kubernetes ConfigMap
func SyncConfigMapToK8s(ctx context.Context, app *entities.App) error {
	if app.Env.ClusterID == "" {
		return fmt.Errorf("app environment has no cluster configured")
	}

	client, err := kube.GlobalClusterStore.GetClient(app.Env.ClusterID)
	if err != nil {
		return err
	}

	metadata := &AppMetadata{App: app}
	
	// If no config files, delete the ConfigMap if it exists
	if len(app.ConfigFiles) == 0 {
		err := client.CoreV1().ConfigMaps(app.Env.ClusterNamespace).Delete(
			ctx,
			app.Slug+"-config",
			metav1.DeleteOptions{},
		)
		if err != nil && !errors.IsNotFound(err) {
			return err
		}
		return nil
	}

	// Build ConfigMap with all config files
	cm := metadata.BuildConfigMap()
	
	// Try to get existing ConfigMap
	_, err = client.CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			// Create new ConfigMap
			_, err = client.CoreV1().ConfigMaps(cm.Namespace).Create(ctx, cm, metav1.CreateOptions{})
			return err
		}
		return err
	}

	// Update existing ConfigMap
	_, err = client.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, cm, metav1.UpdateOptions{})
	return err
}
