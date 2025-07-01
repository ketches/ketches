package kube

import (
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	appsv1 "k8s.io/client-go/listers/apps/v1"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

type storeInterface interface {
	DeploymentLister() appsv1.DeploymentLister
	ReplicaSetLister() appsv1.ReplicaSetLister
	StatefulSetLister() appsv1.StatefulSetLister
	PodLister() listerscorev1.PodLister
	ServiceLister() listerscorev1.ServiceLister
	PersistentVolumeClaimLister() listerscorev1.PersistentVolumeClaimLister
	// IngressLister() networkingv1.IngressLister
	// IngressClassLister() networkingv1.IngressClassLister
	// HorizontalPodAutoscalerLister() autoscalingv1.HorizontalPodAutoscalerLister
	// ConfigMapLister() listerscorev1.ConfigMapLister
	// SecretLister() listerscorev1.SecretLister
}

type store struct {
	deploymentLister            appsv1.DeploymentLister
	replicaSetLister            appsv1.ReplicaSetLister
	statefulSetLister           appsv1.StatefulSetLister
	podLister                   listerscorev1.PodLister
	serviceLister               listerscorev1.ServiceLister
	persistentVolumeClaimLister listerscorev1.PersistentVolumeClaimLister
	// ingressLister                 networkingv1.IngressLister
	// ingressClassLister networkingv1.IngressClassLister
	// horizontalPodAutoscalerLister autoscalingv1.HorizontalPodAutoscalerLister
	// configMapLister               listerscorev1.ConfigMapLister
	// secretLister                  listerscorev1.SecretLister

	// namespaceLister      dynamiclister.GenericLister
	// virtualMachineLister dynamiclister.GenericLister
}

func (s *store) DeploymentLister() appsv1.DeploymentLister {
	return s.deploymentLister
}

func (s *store) ReplicaSetLister() appsv1.ReplicaSetLister {
	return s.replicaSetLister
}

func (s *store) StatefulSetLister() appsv1.StatefulSetLister {
	return s.statefulSetLister
}

func (s *store) PodLister() listerscorev1.PodLister {
	return s.podLister
}

func (s *store) ServiceLister() listerscorev1.ServiceLister {
	return s.serviceLister
}

func (s *store) PersistentVolumeClaimLister() listerscorev1.PersistentVolumeClaimLister {
	return s.persistentVolumeClaimLister
}

// func (s *store) IngressLister() networkingv1.IngressLister {
// 	return s.ingressLister
// }

// func (s *store) IngressClassLister() networkingv1.IngressClassLister {
// 	return s.ingressClassLister
// }

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

func loadStore(clientset kubernetes.Interface) storeInterface {
	kubeInformerFactory := informers.NewSharedInformerFactoryWithOptions(clientset, 0, informers.WithTweakListOptions(func(options *metav1.ListOptions) {
		options.LabelSelector = "ketches/owned=true"
	}))

	deployment := kubeInformerFactory.Apps().V1().Deployments()
	deploymentInformer := deployment.Informer()
	replicaSet := kubeInformerFactory.Apps().V1().ReplicaSets()
	replicaSetInformer := replicaSet.Informer()
	statefulSet := kubeInformerFactory.Apps().V1().StatefulSets()
	statefulSetInformer := statefulSet.Informer()
	pod := kubeInformerFactory.Core().V1().Pods()
	podInformer := pod.Informer()
	podInformer.AddEventHandler(handleAppRunningInfoSSE()) // Register a pod event handler for app running info SSE
	service := kubeInformerFactory.Core().V1().Services()
	serviceInformer := service.Informer()
	persistentVolumeClaim := kubeInformerFactory.Core().V1().PersistentVolumeClaims()
	persistentVolumeClaimInformer := persistentVolumeClaim.Informer()
	// ingress := kubeInformerFactory.Networking().V1().Ingresses()
	// ingressInformer := ingress.Informer()
	// ingressClasses := kubeInformerFactory.Networking().V1().IngressClasses()
	// ingressClassesInformer := ingressClasses.Informer()
	// horizontalPodAutoscaler := kubeInformerFactory.Autoscaling().V1().HorizontalPodAutoscalers()
	// horizontalPodAutoscalerInformer := horizontalPodAutoscaler.Informer()
	// configMap := kubeInformerFactory.Core().V1().ConfigMaps()
	// configMapInformer := configMap.Informer()
	// secret := kubeInformerFactory.Core().V1().Secrets()
	// secretInformer := secret.Informer()

	kubeInformerFactory.Start(wait.NeverStop)

	sharedInformers := []cache.SharedInformer{
		deploymentInformer,
		replicaSetInformer,
		statefulSetInformer,
		podInformer,
		serviceInformer,
		persistentVolumeClaimInformer,
		// ingressInformer,
		// ingressClassesInformer,
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

	deploymentLister := deployment.Lister()
	replicaSetLister := replicaSet.Lister()
	statefulSetLister := statefulSet.Lister()
	podLister := pod.Lister()
	serviceLister := service.Lister()
	persistentVolumeClaimLister := persistentVolumeClaim.Lister()
	// ingressLister := ingress.Lister()
	// ingressClassLister := ingressClasses.Lister()
	// horizontalpodautoscalerLister := horizontalPodAutoscaler.Lister()
	// configMapLister := configMap.Lister()
	// secretLister := secret.Lister()

	return &store{
		deploymentLister:            deploymentLister,
		replicaSetLister:            replicaSetLister,
		statefulSetLister:           statefulSetLister,
		podLister:                   podLister,
		serviceLister:               serviceLister,
		persistentVolumeClaimLister: persistentVolumeClaimLister,
		// ingressLister:                 ingressLister,
		// ingressClassLister: ingressClassLister,
		// horizontalPodAutoscalerLister: horizontalpodautoscalerLister,
		// configMapLister:               configMapLister,
		// secretLister:                  secretLister,
	}
}
