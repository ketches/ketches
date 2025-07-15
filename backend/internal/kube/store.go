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
	// Platform generated resource listers
	DeploymentLister() appsv1.DeploymentLister
	ReplicaSetLister() appsv1.ReplicaSetLister
	StatefulSetLister() appsv1.StatefulSetLister
	PodLister() listerscorev1.PodLister
	ServiceLister() listerscorev1.ServiceLister
	ConfigMapLister() listerscorev1.ConfigMapLister
	PersistentVolumeClaimLister() listerscorev1.PersistentVolumeClaimLister

	// Kubernetes resource listers
	NodeLister() listerscorev1.NodeLister
}

type store struct {
	deploymentLister            appsv1.DeploymentLister
	replicaSetLister            appsv1.ReplicaSetLister
	statefulSetLister           appsv1.StatefulSetLister
	podLister                   listerscorev1.PodLister
	serviceLister               listerscorev1.ServiceLister
	configMapLister             listerscorev1.ConfigMapLister
	persistentVolumeClaimLister listerscorev1.PersistentVolumeClaimLister

	nodeLister listerscorev1.NodeLister
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

func (s *store) ConfigMapLister() listerscorev1.ConfigMapLister {
	return s.configMapLister
}

func (s *store) PersistentVolumeClaimLister() listerscorev1.PersistentVolumeClaimLister {
	return s.persistentVolumeClaimLister
}

func (s *store) NodeLister() listerscorev1.NodeLister {
	return s.nodeLister
}

func loadStore(clientset kubernetes.Interface) storeInterface {
	ketchesOwnedResourceInformerFactory := informers.NewSharedInformerFactoryWithOptions(clientset, 0, informers.WithTweakListOptions(func(options *metav1.ListOptions) {
		options.LabelSelector = "ketches.cn/owned=true"
	}))

	kubeInformerFactory := informers.NewSharedInformerFactory(clientset, 0)

	deployment := ketchesOwnedResourceInformerFactory.Apps().V1().Deployments()
	deploymentInformer := deployment.Informer()
	replicaSet := ketchesOwnedResourceInformerFactory.Apps().V1().ReplicaSets()
	replicaSetInformer := replicaSet.Informer()
	statefulSet := ketchesOwnedResourceInformerFactory.Apps().V1().StatefulSets()
	statefulSetInformer := statefulSet.Informer()
	pod := ketchesOwnedResourceInformerFactory.Core().V1().Pods()
	podInformer := pod.Informer()
	podInformer.AddEventHandler(handleAppRunningInfoSSE())
	service := ketchesOwnedResourceInformerFactory.Core().V1().Services()
	serviceInformer := service.Informer()
	configMap := ketchesOwnedResourceInformerFactory.Core().V1().ConfigMaps()
	configMapInformer := configMap.Informer()
	persistentVolumeClaim := ketchesOwnedResourceInformerFactory.Core().V1().PersistentVolumeClaims()
	persistentVolumeClaimInformer := persistentVolumeClaim.Informer()

	node := kubeInformerFactory.Core().V1().Nodes()
	nodeInformer := node.Informer()

	ketchesOwnedResourceInformerFactory.Start(wait.NeverStop)
	kubeInformerFactory.Start(wait.NeverStop)

	sharedInformers := []cache.SharedInformer{
		deploymentInformer,
		replicaSetInformer,
		statefulSetInformer,
		podInformer,
		serviceInformer,
		configMapInformer,
		persistentVolumeClaimInformer,

		nodeInformer,
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

	ketchesOwnedResourceInformerFactory.WaitForCacheSync(wait.NeverStop)
	kubeInformerFactory.WaitForCacheSync(wait.NeverStop)

	deploymentLister := deployment.Lister()
	replicaSetLister := replicaSet.Lister()
	statefulSetLister := statefulSet.Lister()
	podLister := pod.Lister()
	serviceLister := service.Lister()
	configMapLister := configMap.Lister()
	persistentVolumeClaimLister := persistentVolumeClaim.Lister()

	nodeLister := node.Lister()

	return &store{
		deploymentLister:            deploymentLister,
		replicaSetLister:            replicaSetLister,
		statefulSetLister:           statefulSetLister,
		podLister:                   podLister,
		serviceLister:               serviceLister,
		configMapLister:             configMapLister,
		persistentVolumeClaimLister: persistentVolumeClaimLister,

		nodeLister: nodeLister,
	}
}
