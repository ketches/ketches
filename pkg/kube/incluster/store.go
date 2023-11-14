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

package incluster

import (
	"log"
	"sync"

	corev1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	"github.com/ketches/ketches/pkg/generated/informers/externalversions"
	"github.com/ketches/ketches/pkg/generated/listers/core/v1alpha1"
	"github.com/ketches/ketches/pkg/global"
	"github.com/ketches/ketches/pkg/kube/workercluster"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	networkingv1 "k8s.io/client-go/listers/networking/v1"
	"k8s.io/client-go/tools/cache"
)

type StoreInterface interface {
	ClusterLister() v1alpha1.ClusterLister
	SpaceLister() v1alpha1.SpaceLister
	ExtensionLister() v1alpha1.ExtensionLister
	ApplicationLister() v1alpha1.ApplicationLister
	UserLister() v1alpha1.UserLister
	RoleLister() v1alpha1.RoleLister
	AuditLister() v1alpha1.AuditLister

	Clusterset() workercluster.Clusterset

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
	clusterLister        v1alpha1.ClusterLister
	spaceLister          v1alpha1.SpaceLister
	extensionLister      v1alpha1.ExtensionLister
	helmRepositoryLister v1alpha1.HelmRepositoryLister
	applicationLister    v1alpha1.ApplicationLister
	userLister           v1alpha1.UserLister
	roleLister           v1alpha1.RoleLister
	auditLister          v1alpha1.AuditLister

	clusterset workercluster.Clusterset

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

func (s *store) ClusterLister() v1alpha1.ClusterLister {
	return s.clusterLister
}

func (s *store) SpaceLister() v1alpha1.SpaceLister {
	return s.spaceLister
}

func (s *store) ExtensionLister() v1alpha1.ExtensionLister {
	return s.extensionLister
}

func (s *store) ApplicationLister() v1alpha1.ApplicationLister {
	return s.applicationLister
}

func (s *store) UserLister() v1alpha1.UserLister {
	return s.userLister
}

func (s *store) RoleLister() v1alpha1.RoleLister {
	return s.roleLister
}

func (s *store) AuditLister() v1alpha1.AuditLister {
	return s.auditLister
}

func (s *store) Clusterset() workercluster.Clusterset {
	return s.clusterset
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

	ketchesInformerFactory := externalversions.NewSharedInformerFactoryWithOptions(KetchesClient(), 0, externalversions.WithTweakListOptions(func(options *metav1.ListOptions) {
		options.LabelSelector = global.OwnedResourceLabel
	}))

	cluster := ketchesInformerFactory.Core().V1alpha1().Clusters()
	clusterInformer := cluster.Informer()
	space := ketchesInformerFactory.Core().V1alpha1().Spaces()
	spaceInformer := space.Informer()
	extension := ketchesInformerFactory.Core().V1alpha1().Extensions()
	extensionInformer := extension.Informer()
	application := ketchesInformerFactory.Core().V1alpha1().Applications()
	applicationInformer := application.Informer()
	user := ketchesInformerFactory.Core().V1alpha1().Users()
	userInformer := user.Informer()
	role := ketchesInformerFactory.Core().V1alpha1().Roles()
	roleInformer := role.Informer()
	audit := ketchesInformerFactory.Core().V1alpha1().Audits()
	auditInformer := audit.Informer()

	cs := workercluster.NewClusterset()
	clusterInformer.AddEventHandler(clusterEventHandler(cs))

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
	ketchesInformerFactory.Start(wait.NeverStop)

	sharedInformers := []cache.SharedInformer{
		clusterInformer,
		spaceInformer,
		extensionInformer,
		applicationInformer,
		userInformer,
		roleInformer,
		auditInformer,

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
	ketchesInformerFactory.WaitForCacheSync(wait.NeverStop)

	// deploymentLister := deployment.Lister()
	// podLister := pod.Lister()
	// serviceLister := service.Lister()
	// ingressLister := ingress.Lister()
	ingressClassLister := ingressClasses.Lister()
	// horizontalpodautoscalerLister := horizontalPodAutoscaler.Lister()
	// configMapLister := configMap.Lister()
	// secretLister := secret.Lister()

	cachedStore = &store{
		clusterLister:     cluster.Lister(),
		extensionLister:   extension.Lister(),
		spaceLister:       space.Lister(),
		applicationLister: application.Lister(),
		userLister:        user.Lister(),
		roleLister:        role.Lister(),
		auditLister:       audit.Lister(),

		clusterset: cs,

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

var clusterEventHandler = func(cs workercluster.Clusterset) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			c := obj.(*corev1alpha1.Cluster)

			cluster := workercluster.NewCluster(c)
			if cluster != nil {
				cs.Set(c.Name, cluster)
				log.Printf("cluster %s loaded", c.Name)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newc := newObj.(*corev1alpha1.Cluster)
			oldc := oldObj.(*corev1alpha1.Cluster)
			if newc.ResourceVersion == oldc.ResourceVersion {
				return
			}

			if newc.Spec.KubeConfig == oldc.Spec.KubeConfig {
				return
			}

			cluster, ok := cs.Cluster(newc.Name)
			if !ok {
				cluster = workercluster.NewCluster(newc)
			}
			cluster.Reset()
			if cluster != nil {
				cs.Set(newc.Name, cluster)
				log.Printf("cluster %s resynced", newc.Name)
			}
		},
		DeleteFunc: func(obj interface{}) {
			c := obj.(*corev1alpha1.Cluster)
			cs.Forget(c.Name)
			log.Printf("cluster %s discarded", c.Name)
		},
	}
}
