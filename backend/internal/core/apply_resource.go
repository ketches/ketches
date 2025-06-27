package core

import (
	"context"
	"log"
	"reflect"

	"github.com/ketches/ketches/internal/app"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ApplyResource(ctx context.Context, cli client.Client, obj client.Object) app.Error {
	kind := obj.GetObjectKind().GroupVersionKind().Kind
	switch kind {
	case "PersistentVolumeClaim":
		// Special handling for PVC cause it has immutable fields
		applyPVC(ctx, cli, obj.(*corev1.PersistentVolumeClaim))
	default:
		applyResource(ctx, cli, obj)
	}
	return nil
}

func newEmptyObjectFrom(obj client.Object) client.Object {
	t := reflect.TypeOf(obj).Elem()
	return reflect.New(t).Interface().(client.Object)
}

func applyResource(ctx context.Context, cli client.Client, obj client.Object) app.Error {
	log.Printf("Applying resource: %s/%s (%T)", obj.GetNamespace(), obj.GetName(), obj)
	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		got := newEmptyObjectFrom(obj)
		err := cli.Get(ctx, client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}, got)
		if err != nil {
			if k8serrors.IsNotFound(err) {
				obj.SetResourceVersion("")
				return cli.Create(ctx, obj)
			}
			return err
		}
		obj.SetResourceVersion(got.GetResourceVersion())
		return cli.Update(ctx, obj)
	}); err != nil {
		log.Println("failed to apply resource:", err)
		return app.ErrClusterOperationFailed
	}

	return nil
}

func applyPVC(ctx context.Context, cli client.Client, obj *corev1.PersistentVolumeClaim) app.Error {
	got := &corev1.PersistentVolumeClaim{}
	gotErr := cli.Get(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, got)
	if gotErr != nil {
		if k8serrors.IsNotFound(gotErr) {
			// PVC does not exist, create it
			createErr := cli.Create(ctx, obj)
			if createErr != nil {
				log.Println("Failed to create PVC:", createErr)
				return app.ErrClusterOperationFailed
			}
			return nil
		}
	}

	// PVC spec is immutable after creation except resources.requests
	// and volumeAttributesClassName for bound claims
	got.Labels = obj.Labels
	got.Annotations = obj.Annotations
	got.Spec.Resources.Requests = obj.Spec.Resources.Requests
	obj = got

	if err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		obj.ResourceVersion = got.ResourceVersion
		updateErr := cli.Update(ctx, got)
		if updateErr != nil {
			log.Println("Failed to update PVC:", updateErr)
			gotNewer := &corev1.PersistentVolumeClaim{}
			gotErr := cli.Get(ctx, client.ObjectKey{Name: obj.Name, Namespace: obj.Namespace}, gotNewer)
			if gotErr != nil {
				return gotErr
			}
			obj.ResourceVersion = gotNewer.ResourceVersion
			return updateErr
		}

		return nil
	}); err != nil {
		return app.ErrClusterOperationFailed
	}

	log.Printf("Successfully applied PVC: %s/%s", obj.Namespace, obj.Name)
	return nil
}
