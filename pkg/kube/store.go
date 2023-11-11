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

package kube

import (
	"github.com/ketches/ketches/pkg/kube/dynamic"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"sync"

	"github.com/ketches/ketches/pkg/global"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	appsv1 "k8s.io/client-go/listers/apps/v1"
	autoscalingv1 "k8s.io/client-go/listers/autoscaling/v1"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	networkingv1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"

	corev1 "k8s.io/api/core/v1"
)

type StoreInterface interface {
	DeploymentLister() appsv1.DeploymentLister
	PodLister() listerscorev1.PodLister
	ServiceLister() listerscorev1.ServiceLister
	IngressLister() networkingv1.IngressLister
	IngressClassLister() networkingv1.IngressClassLister
	HorizontalPodAutoscalerLister() autoscalingv1.HorizontalPodAutoscalerLister
	ConfigMapLister() listerscorev1.ConfigMapLister
	SecretLister() listerscorev1.SecretLister

	NamespaceLister() dynamic.GenericLister
	VirtualMachineLister() dynamic.GenericLister
}

type store struct {
	deploymentLister              appsv1.DeploymentLister
	podLister                     listerscorev1.PodLister
	serviceLister                 listerscorev1.ServiceLister
	ingressLister                 networkingv1.IngressLister
	ingressClassLister            networkingv1.IngressClassLister
	horizontalPodAutoscalerLister autoscalingv1.HorizontalPodAutoscalerLister
	configMapLister               listerscorev1.ConfigMapLister
	secretLister                  listerscorev1.SecretLister

	namespaceLister      dynamic.GenericLister
	virtualMachineLister dynamic.GenericLister
}

func (s *store) DeploymentLister() appsv1.DeploymentLister {
	return s.deploymentLister
}

func (s *store) PodLister() listerscorev1.PodLister {
	return s.podLister
}

func (s *store) ServiceLister() listerscorev1.ServiceLister {
	return s.serviceLister
}

func (s *store) IngressLister() networkingv1.IngressLister {
	return s.ingressLister
}

func (s *store) IngressClassLister() networkingv1.IngressClassLister {
	return s.ingressClassLister
}

func (s *store) HorizontalPodAutoscalerLister() autoscalingv1.HorizontalPodAutoscalerLister {
	return s.horizontalPodAutoscalerLister
}

func (s *store) ConfigMapLister() listerscorev1.ConfigMapLister {
	return s.configMapLister
}

func (s *store) SecretLister() listerscorev1.SecretLister {
	return s.secretLister
}

func (s *store) NamespaceLister() dynamic.GenericLister {
	return s.namespaceLister
}

func (s *store) VirtualMachineLister() dynamic.GenericLister {
	return s.virtualMachineLister
}

var once sync.Once
var cachedStore StoreInterface

func Store() StoreInterface {
	once.Do(loadStore)

	return cachedStore
}

func loadStore() {
	informerFactory := informers.NewSharedInformerFactoryWithOptions(Client(), 0, informers.WithTweakListOptions(func(options *metav1.ListOptions) {
		options.LabelSelector = global.OwnedResourceLabel
	}))

	dynamicInformerFactory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(DynamicClient(), 0, metav1.NamespaceAll, func(options *metav1.ListOptions) {
		// options.LabelSelector = global.OwnedResourceLabel
	})

	kubeFactory := informers.NewSharedInformerFactory(Client(), 0)

	deployment := informerFactory.Apps().V1().Deployments()
	deploymentInformer := deployment.Informer()
	pod := informerFactory.Core().V1().Pods()
	podInformer := pod.Informer()
	service := informerFactory.Core().V1().Services()
	serviceInformer := service.Informer()
	ingress := informerFactory.Networking().V1().Ingresses()
	ingressInformer := ingress.Informer()
	ingressClass := kubeFactory.Networking().V1().IngressClasses()
	ingressClassInformer := ingressClass.Informer()
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
	namespace := dynamicInformerFactory.ForResource(corev1.SchemeGroupVersion.WithResource("namespaces"))
	virtualMachineInformer := virtualMachine.Informer()
	namespaceInformer := namespace.Informer()

	informerFactory.Start(wait.NeverStop)
	dynamicInformerFactory.Start(wait.NeverStop)
	kubeFactory.Start(wait.NeverStop)

	sharedInformers := []cache.SharedInformer{
		deploymentInformer,
		podInformer,
		serviceInformer,
		ingressInformer,
		ingressClassInformer,
		horizontalPodAutoscalerInformer,
		configMapInformer,
		secretInformer,

		namespaceInformer,
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
	ingressLister := ingress.Lister()
	ingressClassLister := ingressClass.Lister()
	horizontalpodautoscalerLister := horizontalPodAutoscaler.Lister()
	configMapLister := configMap.Lister()
	secretLister := secret.Lister()

	namespaceLister := namespace.Lister()
	virtualMachineLister := virtualMachine.Lister()

	cachedStore = &store{
		deploymentLister:              deploymentLister,
		podLister:                     podLister,
		serviceLister:                 serviceLister,
		ingressLister:                 ingressLister,
		ingressClassLister:            ingressClassLister,
		horizontalPodAutoscalerLister: horizontalpodautoscalerLister,
		configMapLister:               configMapLister,
		secretLister:                  secretLister,

		namespaceLister:      dynamic.NewGenericLister(namespaceLister),
		virtualMachineLister: dynamic.NewGenericLister(virtualMachineLister),
	}
}
