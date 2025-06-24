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
	"sync"

	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	networkingv1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
)

type StoreInterface interface {
	// DeploymentLister() appsv1.DeploymentLister
	// PodLister() listerscorev1.PodLister
	// ServiceLister() listerscorev1.ServiceLister
	// IngressLister() networkingv1.IngressLister
	IngressClassLister() networkingv1.IngressClassLister
	// HorizontalPodAutoscalerLister() autoscalingv1.HorizontalPodAutoscalerLister
	// ConfigMapLister() listerscorev1.ConfigMapLister
	// SecretLister() listerscorev1.SecretLister
}

type store struct {

	// deploymentLister              appsv1.DeploymentLister
	// podLister                     listerscorev1.PodLister
	// serviceLister                 listerscorev1.ServiceLister
	// ingressLister                 networkingv1.IngressLister
	ingressClassLister networkingv1.IngressClassLister
	// horizontalPodAutoscalerLister autoscalingv1.HorizontalPodAutoscalerLister
	// configMapLister               listerscorev1.ConfigMapLister
	// secretLister                  listerscorev1.SecretLister

	// namespaceLister      dynamiclister.GenericLister
	// virtualMachineLister dynamiclister.GenericLister
}

// func (s *store) DeploymentLister() appsv1.DeploymentLister {
// 	return s.deploymentLister
// }

// func (s *store) PodLister() listerscorev1.PodLister {
// 	return s.podLister
// }

// func (s *store) ServiceLister() listerscorev1.ServiceLister {
// 	return s.serviceLister
// }

// func (s *store) IngressLister() networkingv1.IngressLister {
// 	return s.ingressLister
// }

func (s *store) IngressClassLister() networkingv1.IngressClassLister {
	return s.ingressClassLister
}

// func (s *store) HorizontalPodAutoscalerLister() autoscalingv1.HorizontalPodAutoscalerLister {
// 	return s.horizontalPodAutoscalerLister
// }

// func (s *store) ConfigMapLister() listerscorev1.ConfigMapLister {
// 	return s.configMapLister
// }

// func (s *store) SecretLister() listerscorev1.SecretLister {
// 	return s.secretLister
// }

// func (s *store) NamespaceLister() dynamiclister.GenericLister {
// 	return s.namespaceLister
// }

// func (s *store) VirtualMachineLister() dynamiclister.GenericLister {
// 	return s.virtualMachineLister
// }

var once sync.Once
var cachedStore StoreInterface

func Store() StoreInterface {
	once.Do(loadStore)

	return cachedStore
}

func loadStore() {
	kubeInformerFactory := informers.NewSharedInformerFactory(Client(), 0)

	// deployment := kubeInformerFactory.Apps().V1().Deployments()
	// deploymentInformer := deployment.Informer()
	// pod := kubeInformerFactory.Core().V1().Pods()
	// podInformer := pod.Informer()
	// service := kubeInformerFactory.Core().V1().Services()
	// serviceInformer := service.Informer()
	// ingress := kubeInformerFactory.Networking().V1().Ingresses()
	// ingressInformer := ingress.Informer()
	ingressClasses := kubeInformerFactory.Networking().V1().IngressClasses()
	ingressClassesInformer := ingressClasses.Informer()
	// horizontalPodAutoscaler := kubeInformerFactory.Autoscaling().V1().HorizontalPodAutoscalers()
	// horizontalPodAutoscalerInformer := horizontalPodAutoscaler.Informer()
	// configMap := kubeInformerFactory.Core().V1().ConfigMaps()
	// configMapInformer := configMap.Informer()
	// secret := kubeInformerFactory.Core().V1().Secrets()
	// secretInformer := secret.Informer()

	kubeInformerFactory.Start(wait.NeverStop)

	sharedInformers := []cache.SharedInformer{
		// deploymentInformer,
		// podInformer,
		// serviceInformer,
		// ingressInformer,
		ingressClassesInformer,
		// horizontalPodAutoscalerInformer,
		// configMapInformer,
		// secretInformer,
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

	kubeInformerFactory.WaitForCacheSync(wait.NeverStop)

	// deploymentLister := deployment.Lister()
	// podLister := pod.Lister()
	// serviceLister := service.Lister()
	// ingressLister := ingress.Lister()
	ingressClassLister := ingressClasses.Lister()
	// horizontalpodautoscalerLister := horizontalPodAutoscaler.Lister()
	// configMapLister := configMap.Lister()
	// secretLister := secret.Lister()

	cachedStore = &store{
		// deploymentLister:              deploymentLister,
		// podLister:                     podLister,
		// serviceLister:                 serviceLister,
		// ingressLister:                 ingressLister,
		ingressClassLister: ingressClassLister,
		// horizontalPodAutoscalerLister: horizontalpodautoscalerLister,
		// configMapLister:               configMapLister,
		// secretLister:                  secretLister,
	}
}
