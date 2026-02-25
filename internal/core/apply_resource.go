package core

import (
	"context"

	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/kube"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
)

func ApplyApp(ctx context.Context, app *entities.App) error {
	metadata := &AppMetadata{App: app}

	client, err := kube.GlobalClusterStore.GetClient(app.Env.ClusterID)
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

	if len(app.ConfigFiles) > 0 {
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

	if app.RegistryUsername != "" {
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

	for _, v := range app.Volumes {
		if v.VolumeType == "pvc" {
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
		}
	}

	if app.AppType == "StatefulSet" {
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

	if app.AutoScaling != nil {
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

	SyncGatewaysToK8s(ctx, app)

	return nil
}

func ApplyResource(ctx context.Context, client *kubernetes.Clientset, res runtime.Object) error {
	return nil
}
