package core

import (
	"context"
	"errors"

	"github.com/ketches/ketches/internal/kube"
	"github.com/ketches/ketches/internal/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"
)

func GetAppTopology(ctx context.Context, client *kubernetes.Clientset, appCtx *models.AppContext) (*models.AppTopologyResponse, error) {
	ns := appCtx.Env.ClusterNamespace

	var nodes []models.AppTopologyNode
	var edges []models.AppTopologyEdge

	nodes = append(nodes, models.AppTopologyNode{
		ID:   "app-" + appCtx.App.ID,
		Type: "Application",
		Name: appCtx.App.Name,
	})

	workloadID := ""
	workloadType := appCtx.App.AppType
	if workloadType == "" {
		workloadType = "Deployment"
	}
	workloadID = "workload-" + appCtx.App.Slug
	nodes = append(nodes, models.AppTopologyNode{
		ID:   workloadID,
		Type: workloadType,
		Name: appCtx.App.Slug,
	})
	edges = append(edges, models.AppTopologyEdge{
		Source: "app-" + appCtx.App.ID,
		Target: workloadID,
	})

	pods, _ := client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{
		LabelSelector: "app=" + appCtx.App.Slug,
	})
	for _, pod := range pods.Items {
		podID := "pod-" + pod.Name
		nodes = append(nodes, models.AppTopologyNode{
			ID:     podID,
			Type:   "Pod",
			Name:   pod.Name,
			Status: string(pod.Status.Phase),
		})
		edges = append(edges, models.AppTopologyEdge{
			Source: workloadID,
			Target: podID,
		})
	}

	if svc, err := client.CoreV1().Services(ns).Get(ctx, appCtx.App.Slug, metav1.GetOptions{}); err == nil {
		svcID := "svc-" + svc.Name
		nodes = append(nodes, models.AppTopologyNode{
			ID:   svcID,
			Type: "Service",
			Name: svc.Name,
		})
		edges = append(edges, models.AppTopologyEdge{
			Source: workloadID,
			Target: svcID,
		})

		gwClient, err := kube.GlobalClusterStore.GetGatewayClient(appCtx.Env.ClusterID)
		if err == nil {
			routes, _ := gwClient.GatewayV1().HTTPRoutes(ns).List(ctx, metav1.ListOptions{})
			for _, route := range routes.Items {
				isMatched := false
				for _, rule := range route.Spec.Rules {
					for _, backend := range rule.BackendRefs {
						if string(backend.Name) == svc.Name {
							isMatched = true
							break
						}
					}
				}
				if isMatched {
					routeID := "route-" + route.Name
					nodes = append(nodes, models.AppTopologyNode{
						ID:   routeID,
						Type: "HTTPRoute",
						Name: route.Name,
					})
					edges = append(edges, models.AppTopologyEdge{
						Source: svcID,
						Target: routeID,
					})
				}
			}
		}
	}

	if len(appCtx.ConfigFiles) > 0 {
		cmName := appCtx.App.Slug + "-config"
		cmID := "cm-" + cmName
		nodes = append(nodes, models.AppTopologyNode{
			ID:   cmID,
			Type: "ConfigMap",
			Name: cmName,
		})
		edges = append(edges, models.AppTopologyEdge{
			Source: workloadID,
			Target: cmID,
		})
	}

	if appCtx.App.RegistryUsername != "" {
		secretName := appCtx.App.Slug + "-registry"
		secretID := "secret-" + secretName
		nodes = append(nodes, models.AppTopologyNode{
			ID:   secretID,
			Type: "Secret",
			Name: secretName,
		})
		edges = append(edges, models.AppTopologyEdge{
			Source: workloadID,
			Target: secretID,
		})
	}

	for _, v := range appCtx.Volumes {
		if v.VolumeType == "pvc" {
			pvcID := "pvc-" + v.Slug
			nodes = append(nodes, models.AppTopologyNode{
				ID:   pvcID,
				Type: "PVC",
				Name: v.Slug,
			})
			edges = append(edges, models.AppTopologyEdge{
				Source: workloadID,
				Target: pvcID,
			})
			if pvc, err := client.CoreV1().PersistentVolumeClaims(ns).Get(ctx, v.Slug, metav1.GetOptions{}); err == nil && pvc.Spec.VolumeName != "" {
				pvName := pvc.Spec.VolumeName
				pvID := "pv-" + pvName
				nodes = append(nodes, models.AppTopologyNode{
					ID:   pvID,
					Type: "PV",
					Name: pvName,
				})
				edges = append(edges, models.AppTopologyEdge{
					Source: pvcID,
					Target: pvID,
				})
			}
		}
	}

	hpaList, _ := client.AutoscalingV2().HorizontalPodAutoscalers(ns).List(ctx, metav1.ListOptions{})
	for _, hpa := range hpaList.Items {
		ref := hpa.Spec.ScaleTargetRef
		if ref.Name != appCtx.App.Slug {
			continue
		}
		if ref.Kind != "Deployment" && ref.Kind != "StatefulSet" {
			continue
		}
		hpaID := "hpa-" + hpa.Name
		nodes = append(nodes, models.AppTopologyNode{
			ID:   hpaID,
			Type: "HPA",
			Name: hpa.Name,
		})
		edges = append(edges, models.AppTopologyEdge{
			Source: workloadID,
			Target: hpaID,
		})
	}

	return &models.AppTopologyResponse{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func GetAppTopologyResourceYaml(ctx context.Context, client *kubernetes.Clientset, appCtx *models.AppContext, nodeID string) (string, error) {
	dynClient, err := kube.GlobalClusterStore.GetDynamicClient(appCtx.Env.ClusterID)
	if err != nil {
		return "", err
	}

	ns := appCtx.Env.ClusterNamespace

	var gvr schema.GroupVersionResource
	var name string

	switch {
	case len(nodeID) > 8 && nodeID[:8] == "workload":
		name = appCtx.App.Slug
		if appCtx.App.AppType == "StatefulSet" {
			gvr = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}
		} else {
			gvr = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
		}
	case len(nodeID) > 4 && nodeID[:4] == "pod-":
		name = nodeID[4:]
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}
	case len(nodeID) > 4 && nodeID[:4] == "svc-":
		name = nodeID[4:]
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}
	case len(nodeID) > 6 && nodeID[:6] == "route-":
		name = nodeID[6:]
		gvr = schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "httproutes"}
	case len(nodeID) > 3 && nodeID[:3] == "cm-":
		name = nodeID[3:]
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}
	case len(nodeID) > 7 && nodeID[:7] == "secret-":
		name = nodeID[7:]
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	case len(nodeID) > 4 && nodeID[:4] == "pvc-":
		name = nodeID[4:]
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumeclaims"}
	case len(nodeID) > 4 && nodeID[:4] == "pv-":
		name = nodeID[4:]
		gvr = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "persistentvolumes"}
	case len(nodeID) > 4 && nodeID[:4] == "hpa-":
		name = nodeID[4:]
		gvr = schema.GroupVersionResource{Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"}
	default:
		return "", errors.New("unsupported topology node type")
	}

	var obj *unstructured.Unstructured
	if gvr.Resource == "persistentvolumes" {
		obj, err = dynClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	} else {
		obj, err = dynClient.Resource(gvr).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	}
	if err != nil {
		return "", err
	}

	obj.SetManagedFields(nil)
	obj.Object["status"] = nil

	raw, err := yaml.Marshal(obj.Object)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
