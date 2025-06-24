package kube

import (
	"context"
	"log"
	"net/http"

	"github.com/ketches/ketches/internal/app"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func CreateDeployment(ctx context.Context, clusterID string, deployment *appsv1.Deployment) (*appsv1.Deployment, app.Error) {
	clientset, e := ClusterClientset(ctx, clusterID, false)
	if e != nil {
		return nil, e
	}

	newDeployment, err := clientset.AppsV1().Deployments(deployment.Namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil, app.NewError(http.StatusConflict, "Deployment already exists")
		}
		log.Printf("failed to create deployment %s/%s: %v", deployment.Namespace, deployment.Name, err)
		return nil, app.ErrClusterOperationFailed
	}
	return newDeployment, nil
}

func UpdateDeployment(ctx context.Context, clusterID string, deployment *appsv1.Deployment) (*appsv1.Deployment, app.Error) {
	clientset, err := ClusterClientset(ctx, clusterID, false)
	if err != nil {
		return nil, err
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := clientset.AppsV1().Deployments(deployment.Namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		deployment.ResourceVersion = current.ResourceVersion

		_, err = clientset.AppsV1().Deployments(deployment.Namespace).Update(ctx, deployment, metav1.UpdateOptions{})
		return err
	}); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Deployment not found")
		}
		return nil, app.ErrClusterOperationFailed
	}
	return deployment, nil
}

func GetDeployment(ctx context.Context, clusterID, namespace, name string) (*appsv1.Deployment, app.Error) {
	store, e := ClusterStore(ctx, clusterID)
	if e != nil {
		return nil, e
	}

	deployment, err := store.DeploymentLister().Deployments(namespace).Get(name)
	if err != nil {
		return nil, app.NewError(http.StatusNotFound, "Deployment not found")
	}

	return deployment, nil
}

func DeleteDeployment(ctx context.Context, clusterID, namespace, name string) app.Error {
	clientset, err := ClusterClientset(ctx, clusterID, false)
	if err != nil {
		return err
	}

	if err := clientset.AppsV1().Deployments(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return app.ErrClusterOperationFailed
		}
	}
	return nil
}
