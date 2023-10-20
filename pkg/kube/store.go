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
	"sync"

	corev1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	appsv1 "k8s.io/client-go/listers/apps/v1"
	autoscalingv1 "k8s.io/client-go/listers/autoscaling/v1"
	corev1 "k8s.io/client-go/listers/core/v1"
	networkingv1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
)

type StoreInterface interface {
	DeploymentLister() appsv1.DeploymentLister
	PodLister() corev1.PodLister
	ServiceLister() corev1.ServiceLister
	IngressLister() networkingv1.IngressLister
	IngressClassLister() networkingv1.IngressClassLister
	HorizontalPodAutoscalerLister() autoscalingv1.HorizontalPodAutoscalerLister
	ConfigMapLister() corev1.ConfigMapLister
	SecretLister() corev1.SecretLister
}

type store struct {
	deploymentLister              appsv1.DeploymentLister
	podLister                     corev1.PodLister
	serviceLister                 corev1.ServiceLister
	ingressLister                 networkingv1.IngressLister
	ingressClassLister            networkingv1.IngressClassLister
	horizontalPodAutoscalerLister autoscalingv1.HorizontalPodAutoscalerLister
	configMapLister               corev1.ConfigMapLister
	secretLister                  corev1.SecretLister
}

func (s *store) DeploymentLister() appsv1.DeploymentLister {
	return s.deploymentLister
}

func (s *store) PodLister() corev1.PodLister {
	return s.podLister
}

func (s *store) ServiceLister() corev1.ServiceLister {
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

func (s *store) ConfigMapLister() corev1.ConfigMapLister {
	return s.configMapLister
}

func (s *store) SecretLister() corev1.SecretLister {
	return s.secretLister
}

var once sync.Once
var cachedStore StoreInterface

func Store() StoreInterface {
	once.Do(loadStore)

	return cachedStore
}

func loadStore() {
	informerFactory := informers.NewSharedInformerFactoryWithOptions(Client(), 0, informers.WithTweakListOptions(func(options *metav1.ListOptions) {
		options.LabelSelector = corev1alpha1.RequiredResourceLabel
	}))

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
	horizontalpodautoscaler := informerFactory.Autoscaling().V1().HorizontalPodAutoscalers()
	horizontalpodautoscalerInformer := horizontalpodautoscaler.Informer()
	configMap := informerFactory.Core().V1().ConfigMaps()
	configMapInformer := configMap.Informer()
	secret := informerFactory.Core().V1().Secrets()
	secretInformer := secret.Informer()
	informerFactory.Start(wait.NeverStop)

	sharedInformers := []cache.SharedInformer{
		deploymentInformer,
		podInformer,
		serviceInformer,
		ingressInformer,
		ingressClassInformer,
		horizontalpodautoscalerInformer,
		configMapInformer,
		secretInformer,
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

	deploymentLister := deployment.Lister()
	podLister := pod.Lister()
	serviceLister := service.Lister()
	ingressLister := ingress.Lister()
	ingressClassLister := ingressClass.Lister()
	horizontalpodautoscalerLister := horizontalpodautoscaler.Lister()
	configMapLister := configMap.Lister()
	secretLister := secret.Lister()

	cachedStore = &store{
		deploymentLister:              deploymentLister,
		podLister:                     podLister,
		serviceLister:                 serviceLister,
		ingressLister:                 ingressLister,
		ingressClassLister:            ingressClassLister,
		horizontalPodAutoscalerLister: horizontalpodautoscalerLister,
		configMapLister:               configMapLister,
		secretLister:                  secretLister,
	}
}
