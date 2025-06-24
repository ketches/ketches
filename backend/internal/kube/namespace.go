package kube

import (
	"context"
	"net/http"

	"github.com/ketches/ketches/internal/app"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func CreateNamespace(ctx context.Context, clusterID string, namespace *corev1.Namespace) (*corev1.Namespace, app.Error) {
	clientset, e := ClusterClientset(ctx, clusterID, false)
	if e != nil {
		return nil, e
	}

	newNamespace, err := clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil, app.NewError(http.StatusConflict, "Namespace already exists")
		}
		return nil, app.ErrClusterOperationFailed
	}
	return newNamespace, nil
}

func DeleteNamespace(ctx context.Context, clusterID, namespace string) app.Error {
	clientset, e := ClusterClientset(ctx, clusterID, false)
	if e != nil {
		return e
	}

	if err := clientset.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return app.ErrClusterOperationFailed
		}
	}
	return nil
}
