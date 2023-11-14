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
	"sync"

	"github.com/ketches/ketches/pkg/global"
	"github.com/ketches/ketches/pkg/kube/dynamiclister"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	appsv1listers "k8s.io/client-go/listers/apps/v1"
	autoscalingv1listers "k8s.io/client-go/listers/autoscaling/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	networkingv1listers "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type cachedStores map[string]StoreInterface

var lock sync.Mutex
var _cachedStores cachedStores = make(cachedStores)

func cachedStore(c *cluster) StoreInterface {
	store, ok := _cachedStores[c.Name()]
	if !ok {
		lock.Lock()
		store = getCachedStore(c.KubeClientset(), c.DynamicClient())
		lock.Unlock()
	}
	return store
}

type StoreInterface interface {
	DeploymentLister() appsv1listers.DeploymentLister
	PodLister() corev1listers.PodLister
	ServiceLister() corev1listers.ServiceLister
	HorizontalPodAutoscalerLister() autoscalingv1listers.HorizontalPodAutoscalerLister
	ConfigMapLister() corev1listers.ConfigMapLister
	SecretLister() corev1listers.SecretLister

	VirtualMachineLister() dynamiclister.GenericLister
}

type store struct {
	restConfig *rest.Config

	deploymentLister              appsv1listers.DeploymentLister
	podLister                     corev1listers.PodLister
	serviceLister                 corev1listers.ServiceLister
	ingressLister                 networkingv1listers.IngressLister
	ingressClassLister            networkingv1listers.IngressClassLister
	horizontalPodAutoscalerLister autoscalingv1listers.HorizontalPodAutoscalerLister
	configMapLister               corev1listers.ConfigMapLister
	secretLister                  corev1listers.SecretLister

	virtualMachineLister dynamiclister.GenericLister
}

func (s *store) DeploymentLister() appsv1listers.DeploymentLister {
	return s.deploymentLister
}

func (s *store) PodLister() corev1listers.PodLister {
	return s.podLister
}

func (s *store) ServiceLister() corev1listers.ServiceLister {
	return s.serviceLister
}

func (s *store) HorizontalPodAutoscalerLister() autoscalingv1listers.HorizontalPodAutoscalerLister {
	return s.horizontalPodAutoscalerLister
}

func (s *store) ConfigMapLister() corev1listers.ConfigMapLister {
	return s.configMapLister
}

func (s *store) SecretLister() corev1listers.SecretLister {
	return s.secretLister
}

func (s *store) VirtualMachineLister() dynamiclister.GenericLister {
	return s.virtualMachineLister
}

// getCachedStore initializes a new store with the given kubeClient and dynamicClient. In this method, we create
// informers for all the resources we need to watch, and wait for them to sync. No need to call this method
// more than once if your kubeClient and dynamicClient are the same.
func getCachedStore(kubeClient kubernetes.Interface, dynamicClient dynamic.Interface) StoreInterface {
	// informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, 0, informers.WithTweakListOptions(func(options *metav1.ListOptions) {
	// 	options.LabelSelector = global.OwnedResourceLabel
	// }))

	informerFactory := informers.NewSharedInformerFactoryWithOptions(kubeClient, 0)

	dynamicInformerFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(dynamicClient, 0, metav1.NamespaceAll, func(options *metav1.ListOptions) {
		options.LabelSelector = global.OwnedResourceLabel
	})

	kubeFactory := informers.NewSharedInformerFactory(kubeClient, 0)

	deployment := informerFactory.Apps().V1().Deployments()
	deploymentInformer := deployment.Informer()
	pod := informerFactory.Core().V1().Pods()
	podInformer := pod.Informer()
	service := informerFactory.Core().V1().Services()
	serviceInformer := service.Informer()
	horizontalPodAutoscaler := informerFactory.Autoscaling().V1().HorizontalPodAutoscalers()
	horizontalPodAutoscalerInformer := horizontalPodAutoscaler.Informer()
	configMap := informerFactory.Core().V1().ConfigMaps()
	configMapInformer := configMap.Informer()
	secret := informerFactory.Core().V1().Secrets()
	secretInformer := secret.Informer()

	virtualMachine := dynamicInformerFactory.ForResource(schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	})
	virtualMachineInformer := virtualMachine.Informer()

	informerFactory.Start(wait.NeverStop)
	dynamicInformerFactory.Start(wait.NeverStop)
	kubeFactory.Start(wait.NeverStop)

	sharedInformers := []cache.SharedInformer{
		deploymentInformer,
		podInformer,
		serviceInformer,
		horizontalPodAutoscalerInformer,
		configMapInformer,
		secretInformer,

		virtualMachineInformer,
	}
	var wg sync.WaitGroup
	wg.Add(len(sharedInformers))
	for _, si := range sharedInformers {
		go func(si cache.SharedInformer) {
			if !cache.WaitForCacheSync(wait.NeverStop, si.HasSynced) {
				panic("timed out waiting for caches to sync")
			}
			wg.Done()
		}(si)
	}
	wg.Wait()

	informerFactory.WaitForCacheSync(wait.NeverStop)
	dynamicInformerFactory.WaitForCacheSync(wait.NeverStop)
	kubeFactory.WaitForCacheSync(wait.NeverStop)

	deploymentLister := deployment.Lister()
	podLister := pod.Lister()
	serviceLister := service.Lister()
	horizontalpodautoscalerLister := horizontalPodAutoscaler.Lister()
	configMapLister := configMap.Lister()
	secretLister := secret.Lister()

	virtualMachineLister := virtualMachine.Lister()

	return &store{
		deploymentLister:              deploymentLister,
		podLister:                     podLister,
		serviceLister:                 serviceLister,
		horizontalPodAutoscalerLister: horizontalpodautoscalerLister,
		configMapLister:               configMapLister,
		secretLister:                  secretLister,

		virtualMachineLister: dynamiclister.NewGenericLister(virtualMachineLister),
	}
}
