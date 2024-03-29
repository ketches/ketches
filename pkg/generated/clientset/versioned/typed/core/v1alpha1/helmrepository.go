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
// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	"time"

	v1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	scheme "github.com/ketches/ketches/pkg/generated/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// HelmRepositoriesGetter has a method to return a HelmRepositoryInterface.
// A group's client should implement this interface.
type HelmRepositoriesGetter interface {
	HelmRepositories(namespace string) HelmRepositoryInterface
}

// HelmRepositoryInterface has methods to work with HelmRepository resources.
type HelmRepositoryInterface interface {
	Create(ctx context.Context, helmRepository *v1alpha1.HelmRepository, opts v1.CreateOptions) (*v1alpha1.HelmRepository, error)
	Update(ctx context.Context, helmRepository *v1alpha1.HelmRepository, opts v1.UpdateOptions) (*v1alpha1.HelmRepository, error)
	UpdateStatus(ctx context.Context, helmRepository *v1alpha1.HelmRepository, opts v1.UpdateOptions) (*v1alpha1.HelmRepository, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.HelmRepository, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.HelmRepositoryList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.HelmRepository, err error)
	HelmRepositoryExpansion
}

// helmRepositories implements HelmRepositoryInterface
type helmRepositories struct {
	client rest.Interface
	ns     string
}

// newHelmRepositories returns a HelmRepositories
func newHelmRepositories(c *CoreV1alpha1Client, namespace string) *helmRepositories {
	return &helmRepositories{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the helmRepository, and returns the corresponding helmRepository object, and an error if there is any.
func (c *helmRepositories) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.HelmRepository, err error) {
	result = &v1alpha1.HelmRepository{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("helmrepositories").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of HelmRepositories that match those selectors.
func (c *helmRepositories) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.HelmRepositoryList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1alpha1.HelmRepositoryList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("helmrepositories").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested helmRepositories.
func (c *helmRepositories) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("helmrepositories").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a helmRepository and creates it.  Returns the server's representation of the helmRepository, and an error, if there is any.
func (c *helmRepositories) Create(ctx context.Context, helmRepository *v1alpha1.HelmRepository, opts v1.CreateOptions) (result *v1alpha1.HelmRepository, err error) {
	result = &v1alpha1.HelmRepository{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("helmrepositories").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(helmRepository).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a helmRepository and updates it. Returns the server's representation of the helmRepository, and an error, if there is any.
func (c *helmRepositories) Update(ctx context.Context, helmRepository *v1alpha1.HelmRepository, opts v1.UpdateOptions) (result *v1alpha1.HelmRepository, err error) {
	result = &v1alpha1.HelmRepository{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("helmrepositories").
		Name(helmRepository.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(helmRepository).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *helmRepositories) UpdateStatus(ctx context.Context, helmRepository *v1alpha1.HelmRepository, opts v1.UpdateOptions) (result *v1alpha1.HelmRepository, err error) {
	result = &v1alpha1.HelmRepository{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("helmrepositories").
		Name(helmRepository.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(helmRepository).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the helmRepository and deletes it. Returns an error if one occurs.
func (c *helmRepositories) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("helmrepositories").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *helmRepositories) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("helmrepositories").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched helmRepository.
func (c *helmRepositories) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.HelmRepository, err error) {
	result = &v1alpha1.HelmRepository{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("helmrepositories").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
