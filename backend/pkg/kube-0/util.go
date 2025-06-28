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

package kube

import (
	"bytes"
	"context"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func RESTConfigFromKubeConfig(kubeConfig string) (*rest.Config, error) {
	return clientcmd.RESTConfigFromKubeConfig([]byte(kubeConfig))
}

func ClientForConfig(config *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(config)
}

func DynamicClientForConfig(config *rest.Config) (dynamic.Interface, error) {
	return dynamic.NewForConfig(config)
}

type GetContainerOptions struct {
	Follow    bool
	TailLines *int64
	Previous  bool
}

func GetContainerLogs(ctx context.Context, kubeClient kubernetes.Interface, namespace, pod, container string, opt GetContainerOptions) ([]byte, error) {
	rc, err := kubeClient.CoreV1().Pods(namespace).GetLogs(pod, &corev1.PodLogOptions{
		Container: container,
		Follow:    opt.Follow,
		TailLines: opt.TailLines,
		Previous:  opt.Previous,
	}).Stream(ctx)
	if err != nil {
		return nil, err
	}

	defer rc.Close()
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(rc)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
