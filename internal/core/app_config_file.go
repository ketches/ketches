package core

import (
	"context"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// SyncConfigMapToK8s synchronizes public configuration through a ConfigMap and
// sensitive configuration through a Secret.
func SyncConfigMapToK8s(ctx context.Context, appCtx *models.AppContext) error {
	return withAppReconcileLockContext(ctx, appCtx, func(latest *models.AppContext) error {
		if latest.EnvContext.Env.ClusterID == "" {
			return app.NewErrorf("app environment has no cluster configured")
		}
		client, err := kube.GlobalClusterStore.GetClient(latest.EnvContext.Env.ClusterID)
		if err != nil {
			return err
		}

		return syncConfigMapToK8s(ctx, client, latest)
	})
}

func syncConfigMapToK8s(ctx context.Context, client kubernetes.Interface, appCtx *models.AppContext) error {
	metadata := &AppMetadata{AppContext: appCtx}
	namespace := appCtx.EnvContext.Env.ClusterNamespace
	envSecretName := appCtx.App.Slug + "-env-secret"

	// Config-file CRUD must never publish pending environment variables. It
	// also must not trigger a rollout when the deployed workload still requires
	// an environment Secret that has disappeared outside this operation.
	envSecretData, envSecretExists, err := getSecretDataIfExists(ctx, client, namespace, envSecretName)
	if err != nil {
		return err
	}
	workloadReferencesMissingEnvSecret, err := workloadReferencesSecret(
		ctx,
		client,
		appCtx.App.AppType,
		namespace,
		appCtx.App.Slug,
		envSecretName,
		!envSecretExists,
	)
	if err != nil {
		return err
	}
	if workloadReferencesMissingEnvSecret {
		return app.NewErrorf("deployed workload references missing environment Secret %q", envSecretName)
	}

	revision, err := metadata.buildConfigRevisionWithEnvSecretData(envSecretData)
	if err != nil {
		return err
	}
	metadata.configRevision = revision

	// Create/update resources before changing the workload. In particular, a
	// public-to-secret (or secret-to-public) transition must not expose a
	// template that references a resource which has not been created yet.
	hasConfigMap := metadata.hasNonSecretConfigFiles()
	hasConfigSecret := metadata.hasSecretConfigFiles()
	if hasConfigMap {
		if err := applyConfigMap(ctx, client, metadata.BuildConfigMap()); err != nil {
			return err
		}
	}
	if hasConfigSecret {
		secret, err := metadata.BuildConfigSecret()
		if err != nil {
			return err
		}
		if err := applySecret(ctx, client, secret); err != nil {
			return err
		}
	}

	var desiredTemplate *corev1.PodTemplateSpec
	if appCtx.App.AppType == app.AppTypeStatefulSet {
		desiredTemplate = &metadata.BuildStatefulSet().Spec.Template
	} else {
		desiredTemplate = &metadata.BuildDeployment().Spec.Template
	}
	if err := updateAppConfigWorkload(ctx, client, appCtx.App.AppType, namespace, appCtx.App.Slug, desiredTemplate, revision); err != nil {
		return err
	}

	// Delete obsolete resources only after the workload update has committed.
	// If the workload is not deployed yet, updateAppConfigWorkload is a no-op,
	// and there are no live Pod references to protect.
	if !hasConfigMap {
		if err := deleteConfigMapIfExists(ctx, client, namespace, appCtx.App.Slug+"-config"); err != nil {
			return err
		}
	}
	if !hasConfigSecret {
		if err := deleteSecretIfExists(ctx, client, namespace, appCtx.App.Slug+"-config-secret"); err != nil {
			return err
		}
	}
	return nil
}

func getSecretDataIfExists(ctx context.Context, client kubernetes.Interface, namespace, name string) (map[string][]byte, bool, error) {
	secret, err := client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return secret.Data, true, nil
}

func workloadReferencesSecret(
	ctx context.Context,
	client kubernetes.Interface,
	appType, namespace, name, secretName string,
	onlyIfMissing bool,
) (bool, error) {
	if !onlyIfMissing {
		return false, nil
	}

	var template *corev1.PodTemplateSpec
	if appType == app.AppTypeStatefulSet {
		statefulSet, err := client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		template = &statefulSet.Spec.Template
	} else {
		deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		template = &deployment.Spec.Template
	}
	return podTemplateReferencesSecret(template, secretName), nil
}

func podTemplateReferencesSecret(template *corev1.PodTemplateSpec, secretName string) bool {
	containers := make([]corev1.Container, 0, len(template.Spec.InitContainers)+len(template.Spec.Containers))
	containers = append(containers, template.Spec.InitContainers...)
	containers = append(containers, template.Spec.Containers...)
	for _, container := range containers {
		for _, env := range container.Env {
			if env.ValueFrom != nil && env.ValueFrom.SecretKeyRef != nil && env.ValueFrom.SecretKeyRef.Name == secretName {
				return true
			}
		}
		for _, envFrom := range container.EnvFrom {
			if envFrom.SecretRef != nil && envFrom.SecretRef.Name == secretName {
				return true
			}
		}
	}
	return false
}
