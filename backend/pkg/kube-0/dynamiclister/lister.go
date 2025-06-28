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
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

type genericLister struct {
	cacheGenericLister cache.GenericLister
}

type genericNamespaceLister struct {
	cacheGenericNamespaceLister cache.GenericNamespaceLister
}

func NewGenericLister(lister cache.GenericLister) GenericLister {
	return &genericLister{
		cacheGenericLister: lister,
	}
}

var _ GenericLister = &genericLister{}

func (l *genericLister) List(selector labels.Selector) *GenericObjectListResult {
	objs, err := l.cacheGenericLister.List(selector)
	return &GenericObjectListResult{
		objs: objs,
		err:  err,
	}
}

func (l *genericLister) Get(name string) *GenericObjectResult {
	obj, err := l.cacheGenericLister.Get(name)
	return &GenericObjectResult{
		obj: obj,
		err: err,
	}
}

func (l *genericLister) ByNamespace(namespace string) GenericNamespaceLister {
	return &genericNamespaceLister{
		cacheGenericNamespaceLister: l.cacheGenericLister.ByNamespace(namespace),
	}
}

func (l *genericNamespaceLister) List(selector labels.Selector) *GenericObjectListResult {
	objs, err := l.cacheGenericNamespaceLister.List(selector)
	return &GenericObjectListResult{
		objs: objs,
		err:  err,
	}
}

func (l *genericNamespaceLister) Get(name string) *GenericObjectResult {
	obj, err := l.cacheGenericNamespaceLister.Get(name)
	return &GenericObjectResult{
		obj: obj,
		err: err,
	}
}
