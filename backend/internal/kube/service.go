package kube

import (
	"context"
	"net/http"

	"github.com/ketches/ketches/internal/app"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
)

func CreateService(ctx context.Context, clusterID string, service *corev1.Service) (*corev1.Service, app.Error) {
	clientset, e := ClusterClientset(ctx, clusterID, false)
	if e != nil {
		return nil, e
	}

	newService, err := clientset.CoreV1().Services(service.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil, app.NewError(http.StatusConflict, "Service already exists")
		}
		return nil, app.ErrClusterOperationFailed
	}
	return newService, nil
}

func UpdateService(ctx context.Context, clusterID string, service *corev1.Service) (*corev1.Service, app.Error) {
	clientset, err := ClusterClientset(ctx, clusterID, false)
	if err != nil {
		return nil, err
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := clientset.CoreV1().Services(service.Namespace).Get(context.TODO(), service.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		service.ResourceVersion = current.ResourceVersion

		_, err = clientset.CoreV1().Services(service.Namespace).Update(context.TODO(), service, metav1.UpdateOptions{})
		return err
	}); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "Service not found")
		}
		return nil, app.ErrClusterOperationFailed
	}
	return service, nil
}

func ListServices(ctx context.Context, clusterID, namespace, appName string) ([]*corev1.Service, app.Error) {
	store, e := ClusterStore(ctx, clusterID)
	if e != nil {
		return nil, e
	}

	services, err := store.ServiceLister().Services(namespace).List(labels.SelectorFromSet(labels.Set{"ketches/app": appName}))
	if err != nil {
		return nil, app.ErrClusterOperationFailed
	}

	return services, nil
}

func GetService(ctx context.Context, clusterID, namespace, name string) (*corev1.Service, app.Error) {
	store, e := ClusterStore(ctx, clusterID)
	if e != nil {
		return nil, e
	}

	service, err := store.ServiceLister().Services(namespace).Get(name)
	if err != nil {
		return nil, app.NewError(http.StatusNotFound, "Service not found")
	}

	return service, nil
}

func DeleteService(ctx context.Context, clusterID, namespace, name string) app.Error {
	clientset, err := ClusterClientset(ctx, clusterID, false)
	if err != nil {
		return err
	}

	if err := clientset.CoreV1().Services(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return app.ErrClusterOperationFailed
		}
	}
	return nil
}
