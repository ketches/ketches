package kube

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/ketches/ketches/internal/models"
	corev1 "k8s.io/api/core/v1"
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
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    handlePodAdd,
		UpdateFunc: handlePodUpdate,
		DeleteFunc: handlePodDelete,
	})
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

func handlePodAdd(obj interface{}) {
	pod := obj.(*corev1.Pod)
	processPod(pod)
}

func handlePodUpdate(_, newObj interface{}) {
	pod := newObj.(*corev1.Pod)
	processPod(pod)
}

func handlePodDelete(obj interface{}) {
	var pod *corev1.Pod
	switch t := obj.(type) {
	case *corev1.Pod:
		pod = t
	case cache.DeletedFinalStateUnknown:
		pod = t.Obj.(*corev1.Pod)
	default:
		return
	}

	slug := pod.Labels["ketches/app"]
	if slug == "" {
		return
	}

	key := cacheKey(pod.Namespace, slug)
	deletePodCache(key, pod.Name)
	broadcastPodList(key)
}

func processPod(pod *corev1.Pod) {
	slug := pod.Labels["ketches/app"]
	if slug == "" {
		return
	}

	mainContainer := MainContainer(slug, pod)
	containers := make([]*models.AppInstanceContainerModel, 0, len(pod.Spec.Containers))
	initContainers := make([]*models.AppInstanceContainerModel, 0, len(pod.Spec.InitContainers))
	for _, container := range pod.Status.ContainerStatuses {
		containers = append(containers, &models.AppInstanceContainerModel{
			ContainerName: container.Name,
			Image:         container.Image,
			Status:        GetContainerStatus(&container),
		})
	}
	// let mainContainer always be the first in the list
	sort.Slice(containers, func(i, j int) bool {
		return containers[i].ContainerName == mainContainer.Name
	})
	for _, container := range pod.Status.InitContainerStatuses {
		initContainers = append(initContainers, &models.AppInstanceContainerModel{
			ContainerName: container.Name,
			Image:         container.Image,
			Status:        GetContainerStatus(&container),
		})
	}
	instance := &models.AppInstanceModel{
		InstanceName:   pod.Name,
		Status:         GetPodStatus(pod),
		CreatedAt:      pod.CreationTimestamp.Time,
		InstanceIP:     pod.Status.PodIP,
		Containers:     containers,
		InitContainers: initContainers,
		NodeName:       pod.Spec.NodeName,
		NodeIP:         pod.Status.HostIP,
		ContainerCount: len(pod.Status.ContainerStatuses),
		RequestCPU:     mainContainer.Resources.Requests.Cpu().String(),
		RequestMemory:  mainContainer.Resources.Requests.Memory().String(),
		LimitCPU:       mainContainer.Resources.Limits.Cpu().String(),
		LimitMemory:    mainContainer.Resources.Limits.Memory().String(),
		Edition:        pod.Labels["ketches/edition"],
	}

	key := cacheKey(pod.Namespace, slug)
	updatePodCache(key, instance)
	broadcastPodList(key)
}

func broadcastPodList(key string) {
	instances := GetPodList(key)
	AppInstanceSSEClients.RLock()
	defer AppInstanceSSEClients.RUnlock()

	for client := range AppInstanceSSEClients.List {
		if client.Key != key {
			continue
		}
		select {
		case client.MsgChan <- instances:
		default:
			log.Printf("AppInstanceSSEClient channel for key %s is full, skipping broadcast", key)
		}
	}
}

func GetPodList(key string) []*models.AppInstanceModel {
	AppInstanceCache.RLock()
	group, exists := AppInstanceCache.Data[key]
	AppInstanceCache.RUnlock()

	if !exists {
		return nil
	}

	group.RLock()
	defer group.RUnlock()

	instances := make([]*models.AppInstanceModel, len(group.Instances))
	copy(instances, group.Instances)
	return instances
}

func cacheKey(namespace, slug string) string {
	return fmt.Sprintf("%s/%s", namespace, slug)
}

var AppInstanceCache = struct {
	sync.RWMutex
	Data map[string]*AppInstanceGroup
}{Data: map[string]*AppInstanceGroup{}}

type AppInstanceGroup struct {
	sync.RWMutex
	Instances []*models.AppInstanceModel
}

type AppInstanceSSEClient struct {
	Key string
	Ctx context.Context
	// Writer  http.ResponseWriter
	// Flusher http.Flusher
	MsgChan chan []*models.AppInstanceModel
}

var AppInstanceSSEClients = struct {
	sync.RWMutex
	List map[*AppInstanceSSEClient]struct{}
}{List: map[*AppInstanceSSEClient]struct{}{}}

func updatePodCache(key string, pod *models.AppInstanceModel) {
	AppInstanceCache.Lock()
	group, exists := AppInstanceCache.Data[key]
	if !exists {
		group = &AppInstanceGroup{}
		AppInstanceCache.Data[key] = group
	}
	AppInstanceCache.Unlock()

	group.Lock()
	defer group.Unlock()

	// 查找是否存在
	found := false
	for i, p := range group.Instances {
		if p.InstanceName == pod.InstanceName {
			group.Instances[i] = pod
			found = true
			break
		}
	}
	if !found {
		group.Instances = append(group.Instances, pod)
	}
}

func deletePodCache(key, podName string) {
	AppInstanceCache.RLock()
	group, exists := AppInstanceCache.Data[key]
	AppInstanceCache.RUnlock()

	if !exists {
		return
	}

	group.Lock()
	defer group.Unlock()

	pods := group.Instances
	updated := []*models.AppInstanceModel{}
	for _, p := range pods {
		if p.InstanceName != podName {
			updated = append(updated, p)
		}
	}
	group.Instances = updated
}
