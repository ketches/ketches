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

package dynamiclister

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
)

type GenericObjectResult struct {
	obj runtime.Object
	err error
}

func (o *GenericObjectResult) ToObject(obj runtime.Object) error {
	if o.err != nil {
		return o.err
	}
	if o.obj == nil {
		return o.err
	}

	unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o.obj)
	if err != nil {
		return err
	}

	return runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredMap, obj)
}

type GenericObjectListResult struct {
	objs []runtime.Object
	err  error
}

func (o *GenericObjectListResult) ToObjectList(obj runtime.Object) error {
	if o.err != nil {
		return o.err
	}
	if o.objs == nil {
		return o.err
	}

	var unstructuredList unstructured.UnstructuredList

	for _, item := range o.objs {
		unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(item)
		if err != nil {
			return err
		}

		unstructuredList.Items = append(unstructuredList.Items, unstructured.Unstructured{Object: unstructuredMap})
	}

	return runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredList.UnstructuredContent(), obj)
}

type GenericLister interface {
	List(selector labels.Selector) *GenericObjectListResult
	Get(name string) *GenericObjectResult
	ByNamespace(namespace string) GenericNamespaceLister
}

type GenericNamespaceLister interface {
	List(selector labels.Selector) *GenericObjectListResult
	Get(name string) *GenericObjectResult
}
