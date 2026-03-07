package core

import (
	"context"
	"fmt"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

func ApplyApp(ctx context.Context, appCtx *models.AppContext) error {
	metadata := &AppMetadata{AppContext: appCtx}

	client, err := kube.GlobalClusterStore.GetClient(appCtx.EnvContext.Env.ClusterID)
	if err != nil {
		return err
	}

	ns := metadata.BuildNamespace()
	if _, err := client.CoreV1().Namespaces().Get(ctx, ns.Name, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			if _, err := client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	if len(appCtx.ConfigFiles) > 0 {
		cm := metadata.BuildConfigMap()
		if _, err := client.CoreV1().ConfigMaps(cm.Namespace).Get(ctx, cm.Name, metav1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				if _, err := client.CoreV1().ConfigMaps(cm.Namespace).Create(ctx, cm, metav1.CreateOptions{}); err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			if _, err := client.CoreV1().ConfigMaps(cm.Namespace).Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
				return err
			}
		}
	}

	if appCtx.App.RegistryUsername != "" {
		secret := metadata.BuildRegistrySecret()
		if _, err := client.CoreV1().Secrets(secret.Namespace).Get(ctx, secret.Name, metav1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				if _, err := client.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			if _, err := client.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{}); err != nil {
				return err
			}
		}
	}

	for _, v := range appCtx.Volumes {
		if v.VolumeType == app.VolumeTypePVC {
			switch appCtx.App.AppType {
			case app.AppTypeDeployment:
				pvc := metadata.BuildPVC(v)
				if _, err := client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(ctx, pvc.Name, metav1.GetOptions{}); err != nil {
					if errors.IsNotFound(err) {
						if _, err := client.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(ctx, pvc, metav1.CreateOptions{}); err != nil {
							return err
						}
					} else {
						return err
					}
				}
			case app.AppTypeStatefulSet:
				// For StatefulSet, PVCs are created by the StatefulSet controller based on the volumeClaimTemplates
				// So we don't need to create them here. Just ensure they are defined in the volumeClaimTemplates.
			default:
				return fmt.Errorf("unsupported app type '%s' for volume '%s'", appCtx.App.AppType, v.Slug)
			}
		}
	}

	if appCtx.App.AppType == app.AppTypeStatefulSet {
		sts := metadata.BuildStatefulSet()
		if _, err := client.AppsV1().StatefulSets(sts.Namespace).Get(ctx, sts.Name, metav1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				if _, err := client.AppsV1().StatefulSets(sts.Namespace).Create(ctx, sts, metav1.CreateOptions{}); err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			if _, err := client.AppsV1().StatefulSets(sts.Namespace).Update(ctx, sts, metav1.UpdateOptions{}); err != nil {
				return err
			}
		}
	} else {
		deploy := metadata.BuildDeployment()
		if _, err := client.AppsV1().Deployments(deploy.Namespace).Get(ctx, deploy.Name, metav1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				if _, err := client.AppsV1().Deployments(deploy.Namespace).Create(ctx, deploy, metav1.CreateOptions{}); err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			if _, err := client.AppsV1().Deployments(deploy.Namespace).Update(ctx, deploy, metav1.UpdateOptions{}); err != nil {
				return err
			}
		}
	}

	if appCtx.AutoScaling != nil {
		hpa := metadata.BuildHorizontalPodAutoscaler()
		if _, err := client.AutoscalingV2().HorizontalPodAutoscalers(hpa.Namespace).Get(ctx, hpa.Name, metav1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				if _, err := client.AutoscalingV2().HorizontalPodAutoscalers(hpa.Namespace).Create(ctx, hpa, metav1.CreateOptions{}); err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			if _, err := client.AutoscalingV2().HorizontalPodAutoscalers(hpa.Namespace).Update(ctx, hpa, metav1.UpdateOptions{}); err != nil {
				return err
			}
		}
	}

	SyncGatewaysToK8s(ctx, appCtx)

	return nil
}

func ApplyResource(ctx context.Context, client *kubernetes.Clientset, res runtime.Object) error {
	return nil
}
