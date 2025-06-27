package kube

import (
	"context"
	"log"
	"net/http"

	"github.com/ketches/ketches/internal/app"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func CreatePersistentVolumeClaim(ctx context.Context, clusterID string, pvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, app.Error) {
	clientset, e := ClusterClientset(ctx, clusterID, false)
	if e != nil {
		return nil, e
	}

	newPVC, err := clientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil, app.NewError(http.StatusConflict, "PersistentVolumeClaim already exists")
		}
		log.Printf("failed to create PersistentVolumeClaim %s/%s: %v", pvc.Namespace, pvc.Name, err)
		return nil, app.ErrClusterOperationFailed
	}
	return newPVC, nil
}

func UpdatePersistentVolumeClaim(ctx context.Context, clusterID string, pvc *corev1.PersistentVolumeClaim) (*corev1.PersistentVolumeClaim, app.Error) {
	clientset, err := ClusterClientset(ctx, clusterID, false)
	if err != nil {
		return nil, err
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := clientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Get(ctx, pvc.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		pvc.ResourceVersion = current.ResourceVersion

		_, err = clientset.CoreV1().PersistentVolumeClaims(pvc.Namespace).Update(ctx, pvc, metav1.UpdateOptions{})
		return err
	}); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "PersistentVolumeClaim not found")
		}
		return nil, app.ErrClusterOperationFailed
	}
	return pvc, nil
}

func GetPersistentVolumeClaim(ctx context.Context, clusterID, namespace, name string) (*corev1.PersistentVolumeClaim, app.Error) {
	store, e := ClusterStore(ctx, clusterID)
	if e != nil {
		return nil, e
	}

	pvc, err := store.PersistentVolumeClaimLister().PersistentVolumeClaims(namespace).Get(name)
	if err != nil {
		return nil, app.NewError(http.StatusNotFound, "PersistentVolumeClaim not found")
	}

	return pvc, nil
}

func DeletePersistentVolumeClaim(ctx context.Context, clusterID, namespace, name string) app.Error {
	clientset, err := ClusterClientset(ctx, clusterID, false)
	if err != nil {
		return err
	}

	if err := clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return app.ErrClusterOperationFailed
		}
	}
	return nil
}

func DeletePersistentVolumeClaimsByApp(ctx context.Context, clusterID, namespace, appName string) app.Error {
	clientset, err := ClusterClientset(ctx, clusterID, false)
	if err != nil {
		return err
	}

	if err := clientset.CoreV1().PersistentVolumeClaims(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{
		LabelSelector: "ketches/app=" + appName,
	}); err != nil {
		return app.ErrClusterOperationFailed
	}

	return nil
}
