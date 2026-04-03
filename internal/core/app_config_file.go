package core

import (
	"context"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SyncConfigMapToK8s synchronizes config files to a Kubernetes ConfigMap
func SyncConfigMapToK8s(ctx context.Context, appCtx *models.AppContext) error {
	if appCtx.EnvContext.Env.ClusterID == "" {
		return app.NewErrorf("app environment has no cluster configured")
	}

	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}

	metadata := &AppMetadata{AppContext: appCtx}

	// If no config files, delete the ConfigMap if it exists
	if len(appCtx.ConfigFiles) == 0 {
		err := client.CoreV1().ConfigMaps(appCtx.EnvContext.Env.ClusterNamespace).Delete(
			ctx,
			appCtx.App.Slug+"-config",
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
