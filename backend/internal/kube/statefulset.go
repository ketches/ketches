package kube

import (
	"context"
	"net/http"

	"github.com/ketches/ketches/internal/app"
	appsv1 "k8s.io/api/apps/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

func CreateStatefulSet(ctx context.Context, clusterID string, statefulSet *appsv1.StatefulSet) (*appsv1.StatefulSet, app.Error) {
	clientset, e := ClusterClientset(ctx, clusterID, false)
	if e != nil {
		return nil, e
	}

	newStatefulSet, err := clientset.AppsV1().StatefulSets(statefulSet.Namespace).Create(ctx, statefulSet, metav1.CreateOptions{})
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			return nil, app.NewError(http.StatusConflict, "StatefulSet already exists")
		}
		return nil, app.ErrClusterOperationFailed
	}
	return newStatefulSet, nil
}

func UpdateStatefulSet(ctx context.Context, clusterID string, statefulSet *appsv1.StatefulSet) (*appsv1.StatefulSet, app.Error) {
	clientset, err := ClusterClientset(ctx, clusterID, false)
	if err != nil {
		return nil, err
	}

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		current, err := clientset.AppsV1().StatefulSets(statefulSet.Namespace).Get(ctx, statefulSet.Name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		statefulSet.ResourceVersion = current.ResourceVersion

		_, err = clientset.AppsV1().StatefulSets(statefulSet.Namespace).Update(ctx, statefulSet, metav1.UpdateOptions{})
		return err
	}); err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, app.NewError(http.StatusNotFound, "StatefulSet not found")
		}
		return nil, app.ErrClusterOperationFailed
	}
	return statefulSet, nil
}

func GetStatefulSet(ctx context.Context, clusterID, namespace, name string) (*appsv1.StatefulSet, app.Error) {
	store, e := ClusterStore(ctx, clusterID)
	if e != nil {
		return nil, e
	}

	statefulSet, err := store.StatefulSetLister().StatefulSets(namespace).Get(name)
	if err != nil {
		return nil, app.NewError(http.StatusNotFound, "StatefulSet not found")
	}

	return statefulSet, nil
}

func DeleteStatefulSet(ctx context.Context, clusterID, namespace, name string) app.Error {
	clientset, err := ClusterClientset(ctx, clusterID, false)
	if err != nil {
		return err
	}

	if err := clientset.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		if !k8serrors.IsNotFound(err) {
			return app.ErrClusterOperationFailed
		}
	}
	return nil
}
