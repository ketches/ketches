package kube

import (
	"context"
	"log"
	"reflect"

	"github.com/ketches/ketches/internal/app"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newEmptyObjectFrom(obj client.Object) client.Object {
	t := reflect.TypeOf(obj).Elem()
	return reflect.New(t).Interface().(client.Object)
}

func ApplyResource(ctx context.Context, cli client.Client, obj client.Object) app.Error {
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

func DeleteResource(ctx context.Context, cli client.Client, obj client.Object) app.Error {
	if err := cli.Delete(ctx, obj); err != nil {
		if k8serrors.IsNotFound(err) {
			// Resource already deleted
			return nil
		}
		log.Println("failed to delete resource:", err)
		return app.ErrClusterOperationFailed
	}
	return nil
}
