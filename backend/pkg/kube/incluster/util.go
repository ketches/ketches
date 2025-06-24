/*
Copyright 2025 The Ketches Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package incluster

import (
	"context"
	"reflect"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

func ListIngressClasses() map[string]bool {
	ics, _ := Store().IngressClassLister().List(labels.Everything())
	result := make(map[string]bool, len(ics))
	for _, ic := range ics {
		v, ok := ic.Annotations[networkingv1.AnnotationIsDefaultIngressClass]
		result[ic.Name] = ok && v == "true"
	}
	return result
}

func DefaultIngressClass() string {
	var result string
	ics, _ := Store().IngressClassLister().List(labels.Everything())
	for _, ic := range ics {
		if val, ok := ic.Annotations[networkingv1.AnnotationIsDefaultIngressClass]; ok && val == "true" {
			result = ic.Name
			break
		}
		result = ic.Name
	}
	return result
}

func DefaultGatewayClass(gatewayAPIClient versioned.Interface) string {
	var result string
	gcs, _ := gatewayAPIClient.GatewayV1beta1().GatewayClasses().List(context.Background(), metav1.ListOptions{})
	if len(gcs.Items) > 0 {
		result = gcs.Items[0].Name
	}
	return result
}

// newEmptyObjectFrom returns a new empty object of the same type as the given object.
// It's a bit faster than obj.DeepCopyObject().(client.Object) by benchmarks.
func newEmptyObjectFrom(obj client.Object) client.Object {
	// without Elem(), t will be a pointer to the type. For example, *corev1.Pod, not corev1.Pod
	t := reflect.TypeOf(obj).Elem()
	return reflect.New(t).Interface().(client.Object)
}

func ApplyResource(ctx context.Context, cli client.Client, obj client.Object) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		got := newEmptyObjectFrom(obj)
		err := cli.Get(ctx, client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}, got)
		if err != nil {
			if errors.IsNotFound(err) {
				obj.SetResourceVersion("")
				return cli.Create(ctx, obj)
			}
			return err
		}
		obj.SetResourceVersion(got.GetResourceVersion())
		return cli.Update(ctx, obj)
	})
}

func PatchResource(ctx context.Context, cli client.Client, old, new client.Object) error {
	return cli.Patch(ctx, new, client.MergeFrom(old))
}

func DeleteResource(ctx context.Context, cli client.Client, obj client.Object) error {
	if err := cli.Delete(ctx, obj); err != nil && !errors.IsNotFound(err) {
		return err
	}
	return nil
}

func UpdateResourceStatus(ctx context.Context, cli client.Client, obj client.Object) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		err := cli.Status().Update(ctx, obj)
		if err != nil && errors.IsConflict(err) {
			current := obj.DeepCopyObject().(client.Object)
			err = cli.Get(ctx, client.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}, current)
			if err != nil {
				return err
			}
			obj.SetResourceVersion(current.GetResourceVersion())
		}
		return err
	})
}
