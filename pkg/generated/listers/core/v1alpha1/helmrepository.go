/*
Copyright 2023 The Ketches Authors.

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
// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	v1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/cache"
)

// HelmRepositoryLister helps list HelmRepositories.
// All objects returned here must be treated as read-only.
type HelmRepositoryLister interface {
	// List lists all HelmRepositories in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.HelmRepository, err error)
	// HelmRepositories returns an object that can list and get HelmRepositories.
	HelmRepositories(namespace string) HelmRepositoryNamespaceLister
	HelmRepositoryListerExpansion
}

// helmRepositoryLister implements the HelmRepositoryLister interface.
type helmRepositoryLister struct {
	indexer cache.Indexer
}

// NewHelmRepositoryLister returns a new HelmRepositoryLister.
func NewHelmRepositoryLister(indexer cache.Indexer) HelmRepositoryLister {
	return &helmRepositoryLister{indexer: indexer}
}

// List lists all HelmRepositories in the indexer.
func (s *helmRepositoryLister) List(selector labels.Selector) (ret []*v1alpha1.HelmRepository, err error) {
	err = cache.ListAll(s.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.HelmRepository))
	})
	return ret, err
}

// HelmRepositories returns an object that can list and get HelmRepositories.
func (s *helmRepositoryLister) HelmRepositories(namespace string) HelmRepositoryNamespaceLister {
	return helmRepositoryNamespaceLister{indexer: s.indexer, namespace: namespace}
}

// HelmRepositoryNamespaceLister helps list and get HelmRepositories.
// All objects returned here must be treated as read-only.
type HelmRepositoryNamespaceLister interface {
	// List lists all HelmRepositories in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*v1alpha1.HelmRepository, err error)
	// Get retrieves the HelmRepository from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*v1alpha1.HelmRepository, error)
	HelmRepositoryNamespaceListerExpansion
}

// helmRepositoryNamespaceLister implements the HelmRepositoryNamespaceLister
// interface.
type helmRepositoryNamespaceLister struct {
	indexer   cache.Indexer
	namespace string
}

// List lists all HelmRepositories in the indexer for a given namespace.
func (s helmRepositoryNamespaceLister) List(selector labels.Selector) (ret []*v1alpha1.HelmRepository, err error) {
	err = cache.ListAllByNamespace(s.indexer, s.namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*v1alpha1.HelmRepository))
	})
	return ret, err
}

// Get retrieves the HelmRepository from the indexer for a given namespace and name.
func (s helmRepositoryNamespaceLister) Get(name string) (*v1alpha1.HelmRepository, error) {
	obj, exists, err := s.indexer.GetByKey(s.namespace + "/" + name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(v1alpha1.Resource("helmrepository"), name)
	}
	return obj.(*v1alpha1.HelmRepository), nil
}
