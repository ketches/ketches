package kube

import (
	"context"
	"log"
	"sort"
	"sync"

	"github.com/ketches/ketches/internal/models"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

func handleAppRunningInfoSSE() cache.ResourceEventHandlerFuncs {
	handlePodAdd := func(obj interface{}) {
		pod := obj.(*corev1.Pod)
		processPodForAppRunningInfoSSE(pod)
	}

	handlePodUpdate := func(_, newObj interface{}) {
		pod := newObj.(*corev1.Pod)
		processPodForAppRunningInfoSSE(pod)
	}

	handlePodDelete := func(obj interface{}) {
		var pod *corev1.Pod
		switch t := obj.(type) {
		case *corev1.Pod:
			pod = t
		case cache.DeletedFinalStateUnknown:
			pod = t.Obj.(*corev1.Pod)
		default:
			return
		}

		appID := pod.Labels["ketches.cn/id"]
		if appID == "" {
			return
		}

		GetAppInstanceSSEClients().removeAppInstance(appID, pod.Name)
		broadcastPodList(appID)
	}

	return cache.ResourceEventHandlerFuncs{
		AddFunc:    handlePodAdd,
		UpdateFunc: handlePodUpdate,
		DeleteFunc: handlePodDelete,
	}
}

func processPodForAppRunningInfoSSE(pod *corev1.Pod) {
	appID := pod.Labels["ketches.cn/id"]
	if appID == "" {
		return
	}
	appSlug := pod.Labels["ketches.cn/app"]
	if appSlug == "" {
		return
	}

	mainContainer := MainContainer(appSlug, pod)
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
		Edition:        pod.Labels["ketches.cn/edition"],
	}

	GetAppInstanceSSEClients().saveAppInstance(appID, instance)
	broadcastPodList(appID)
}

func broadcastPodList(appID string) {
	sseClients := GetAppInstanceSSEClients()
	instances := sseClients.GetAppInstances(appID)

	sseClients.RLock()
	defer sseClients.RUnlock()

	for client := range sseClients.List {
		if client.AppID != appID {
			continue
		}
		select {
		case client.MsgChan <- instances:
		default:
			log.Printf("app instance sse client channel for app id %s is full, skipping broadcast", appID)
		}
	}
}

type appInstanceCache struct {
	sync.RWMutex
	Data map[string]*AppInstanceGroup
}

type AppInstanceGroup struct {
	sync.RWMutex
	Instances []*models.AppInstanceModel
}

type AppInstanceSSEClient struct {
	AppID   string
	Ctx     context.Context
	MsgChan chan []*models.AppInstanceModel
}

type AppInstanceSSEClients struct {
	sync.RWMutex
	List  map[*AppInstanceSSEClient]struct{}
	Cache *appInstanceCache
}

var (
	once                  sync.Once
	appInstanceSSEClients AppInstanceSSEClients
)

func GetAppInstanceSSEClients() *AppInstanceSSEClients {
	once.Do(initAppInstanceSSEClients)
	return &appInstanceSSEClients
}

// Initialize the appInstanceSSEClients map
func initAppInstanceSSEClients() {
	appInstanceSSEClients = AppInstanceSSEClients{
		RWMutex: sync.RWMutex{},
		List:    make(map[*AppInstanceSSEClient]struct{}),
		Cache: &appInstanceCache{
			Data: make(map[string]*AppInstanceGroup),
		},
	}
}

func (c *AppInstanceSSEClients) Register(ctx context.Context, appID string) *AppInstanceSSEClient {
	c.Lock()
	defer c.Unlock()

	client := &AppInstanceSSEClient{
		AppID:   appID,
		Ctx:     ctx,
		MsgChan: make(chan []*models.AppInstanceModel, 1), // Buffered channel
	}

	// Add the client to the map
	c.List[client] = struct{}{}
	return client
}

func (c *AppInstanceSSEClients) Remove(client *AppInstanceSSEClient) {
	c.Lock()
	defer c.Unlock()

	// Remove the client from the map
	if _, exists := c.List[client]; exists {
		close(client.MsgChan) // Close the channel to signal no more messages
		delete(c.List, client)
	}
}

func (c *AppInstanceSSEClients) GetAppInstances(appID string) []*models.AppInstanceModel {
	c.Cache.RLock()
	group, exists := c.Cache.Data[appID]
	c.Cache.RUnlock()

	if !exists {
		return nil
	}

	group.RLock()
	defer group.RUnlock()

	instances := make([]*models.AppInstanceModel, len(group.Instances))
	copy(instances, group.Instances)
	return instances
}

func (c *AppInstanceSSEClients) saveAppInstance(appID string, pod *models.AppInstanceModel) {
	c.Cache.Lock()
	group, exists := c.Cache.Data[appID]
	if !exists {
		group = &AppInstanceGroup{}
		c.Cache.Data[appID] = group
	}
	c.Cache.Unlock()

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

func (c *AppInstanceSSEClients) removeAppInstance(appID, podName string) {
	c.Cache.RLock()
	group, exists := c.Cache.Data[appID]
	c.Cache.RUnlock()

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
