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

package workercluster

import (
	"slices"

	corev1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	ketchesversioned "github.com/ketches/ketches/pkg/generated/clientset/versioned"
	ketchesscheme "github.com/ketches/ketches/pkg/generated/clientset/versioned/scheme"
	"github.com/ketches/ketches/pkg/kube"
	velero "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	veleroversioned "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned"
	veleroscheme "github.com/vmware-tanzu/velero/pkg/generated/clientset/versioned/scheme"
	apiextclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apiextscheme "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/scheme"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapisv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayapiversioned "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
	gatewayapischeme "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned/scheme"
)

type Cluster interface {
	Resource() *corev1alpha1.Cluster
	Reset()

	Name() string
	Description() string
	KubeConfig() string
	RESTConfig() *rest.Config

	KubeClientset() kubernetes.Interface
	DynamicClient() dynamic.Interface
	APIExtClientset() apiextclientset.Interface
	KetchesClientset() ketchesversioned.Interface
	VeleroClientset() veleroversioned.Interface
	GatewayAPIClientset() gatewayapiversioned.Interface

	KubeRuntimeClient() client.Client
	APIExtRuntimeClient() client.Client
	KetchesRuntimeClient() client.Client
	GatewayAPIRuntimeClient() client.Client
	VeleroRuntimeClient() client.Client

	Store() StoreInterface
}

var _ Cluster = (*cluster)(nil)

type cluster struct {
	resource         *corev1alpha1.Cluster
	restConfig       *rest.Config
	kubeClient       kubernetes.Interface
	dynamicClient    dynamic.Interface
	ketchesClient    ketchesversioned.Interface
	apiextClient     apiextclientset.Interface
	gatewayapiClient gatewayapiversioned.Interface
	veleroClient     veleroversioned.Interface

	kubeRuntimeClient       client.Client
	ketchesRuntimeClient    client.Client
	apiextRuntimeClient     client.Client
	gatewayapiRuntimeClient client.Client
	veleroRuntimeClient     client.Client
}

func NewCluster(clusterResource *corev1alpha1.Cluster) Cluster {
	return &cluster{
		resource: clusterResource,
	}
}

func (c *cluster) Resource() *corev1alpha1.Cluster {
	if c == nil {
		return nil
	}
	return c.resource
}

func (c *cluster) Reset() {
	c = &cluster{
		resource: c.resource,
	}
}

func (c *cluster) Name() string {
	if c == nil {
		return ""
	}
	return c.resource.Name
}

func (c *cluster) Description() string {
	if c == nil {
		return ""
	}
	return c.resource.Spec.Description
}

func (c *cluster) KubeConfig() string {
	if c == nil {
		return ""
	}
	return c.resource.Spec.KubeConfig
}

func (c *cluster) RESTConfig() *rest.Config {
	if c == nil {
		return nil
	}
	if c.restConfig == nil {
		restConfig, err := kube.RESTConfigFromKubeConfig(c.resource.Spec.KubeConfig)
		if err != nil {
			return nil
		}
		c.restConfig = restConfig
	}
	return c.restConfig
}

func (c *cluster) KubeClientset() kubernetes.Interface {
	if c == nil {
		return nil
	}
	if c.kubeClient == nil {
		kubeClient, err := kubernetes.NewForConfig(c.RESTConfig())
		if err != nil {
			return nil
		}
		c.kubeClient = kubeClient
	}
	return c.kubeClient
}

func (c *cluster) DynamicClient() dynamic.Interface {
	if c == nil {
		return nil
	}
	if c.dynamicClient == nil {
		dynamicClient, err := dynamic.NewForConfig(c.RESTConfig())
		if err != nil {
			return nil
		}
		c.dynamicClient = dynamicClient
	}
	return c.dynamicClient
}

func (c *cluster) APIExtClientset() apiextclientset.Interface {
	if c == nil {
		return nil
	}

	if c.apiextClient == nil {
		cli, err := apiextclientset.NewForConfig(c.RESTConfig())
		if err != nil {
			return nil
		}
		c.apiextClient = cli
	}
	return c.apiextClient
}

func (c *cluster) KetchesClientset() ketchesversioned.Interface {
	if c == nil {
		return nil
	}
	if c.ketchesClient == nil {
		ketchesClient, err := ketchesversioned.NewForConfig(c.RESTConfig())
		if err != nil {
			return nil
		}
		c.ketchesClient = ketchesClient
	}
	return c.ketchesClient
}

func (c *cluster) GatewayAPIClientset() gatewayapiversioned.Interface {
	if c == nil {
		return nil
	}

	if c.gatewayapiClient == nil {
		cli, err := gatewayapiversioned.NewForConfig(c.RESTConfig())
		if err != nil {
			return nil
		}
		c.gatewayapiClient = cli
	}
	return c.gatewayapiClient
}

func (c *cluster) VeleroClientset() veleroversioned.Interface {
	if c == nil {
		return nil
	}

	if c.veleroClient == nil {
		cli, err := veleroversioned.NewForConfig(c.RESTConfig())
		if err != nil {
			return nil
		}
		c.veleroClient = cli
	}
	return c.veleroClient
}

func (c *cluster) KubeRuntimeClient() client.Client {
	if c == nil {
		return nil
	}
	if c.kubeRuntimeClient == nil {
		kubeRuntimeClient, err := client.New(c.RESTConfig(), client.Options{
			Scheme: kubescheme.Scheme,
		})
		if err != nil {
			return nil
		}
		c.kubeRuntimeClient = kubeRuntimeClient
	}
	return c.kubeRuntimeClient
}

func (c *cluster) APIExtRuntimeClient() client.Client {
	if c == nil {
		return nil
	}

	if c.apiextRuntimeClient == nil {
		cli, err := client.New(c.restConfig, client.Options{
			Scheme: apiextscheme.Scheme,
		})
		if err != nil {
			return nil
		}
		c.apiextRuntimeClient = cli
	}
	return c.apiextRuntimeClient
}

func (c *cluster) KetchesRuntimeClient() client.Client {
	if c == nil {
		return nil
	}

	if c.ketchesRuntimeClient == nil {
		cli, err := client.New(c.restConfig, client.Options{
			Scheme: ketchesscheme.Scheme,
		})
		if err != nil {
			return nil
		}
		c.ketchesRuntimeClient = cli
	}
	return c.ketchesRuntimeClient
}

func (c *cluster) APIGroups() []string {
	if c == nil {
		return nil
	}

	gl, err := c.KubeClientset().Discovery().ServerGroups()
	if err != nil {
		return nil
	}

	var result []string
	for _, g := range gl.Groups {
		result = append(result, g.Name)
	}
	return result
}

func (c *cluster) GatewayAPIRuntimeClient() client.Client {
	if c == nil {
		return nil
	}

	if !slices.Contains(c.APIGroups(), gatewayapisv1.GroupName) {
		return nil
	}

	if c.gatewayapiRuntimeClient == nil {
		cli, err := client.New(c.restConfig, client.Options{
			Scheme: gatewayapischeme.Scheme,
		})
		if err != nil {
			return nil
		}

		c.gatewayapiRuntimeClient = cli
	}
	return c.gatewayapiRuntimeClient
}

func (c *cluster) VeleroRuntimeClient() client.Client {
	if c == nil {
		return nil
	}

	if !slices.Contains(c.APIGroups(), velero.SchemeGroupVersion.Group) {
		return nil
	}

	if c.veleroRuntimeClient == nil {
		cli, err := client.New(c.restConfig, client.Options{
			Scheme: veleroscheme.Scheme,
		})
		if err != nil {
			return nil
		}
		c.veleroRuntimeClient = cli
	}
	return c.veleroRuntimeClient
}

func (c *cluster) Store() StoreInterface {
	if c == nil {
		return nil
	}

	return cachedStore(c)
}
