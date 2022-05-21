package kube

import (
	"flag"
	"os"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	clientset       *kubernetes.Clientset
	restConfig      *rest.Config
	restClient      *rest.RESTClient
	discoveryClient *discovery.DiscoveryClient
	dynamicClient   dynamic.Interface
)

type ResourceKind string

const (
	ResourceKindCluster            ResourceKind = "Cluster"
	ResourceKindNode               ResourceKind = "Node"
	ResourceKindNamespace          ResourceKind = "Namespace"
	ResourceKindDeployment         ResourceKind = "Deployment"
	ResourceKindPod                ResourceKind = "Pod"
	ResourceKindService            ResourceKind = "Service"
	ResourceKindIngress            ResourceKind = "Ingress"
	ResourceKindJob                ResourceKind = "Job"
	ResourceKindCronJob            ResourceKind = "CronJob"
	ResourceKindStatefulSet        ResourceKind = "StatefulSet"
	ResourceKindServiceAccount     ResourceKind = "ServiceAccount"
	ResourceKindHPA                ResourceKind = "HorizontalPodAutoscaler"
	ResourceKindClusterRoleBinding ResourceKind = "ClusterRoleBinding"
	ResourceKindRoleBinding        ResourceKind = "RoleBinding"
)

func (r ResourceKind) String() string { return string(r) }

func Clientset() *kubernetes.Clientset {
	if clientset == nil {
		var err error
		clientset, err = kubernetes.NewForConfig(RestConfig())
		if err != nil {
			panic(err)
		}
	}
	return clientset
}

func RestConfig() *rest.Config {
	if restConfig == nil {
		var kubeconfig string
		flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kubeconfig file with authorization information")
		flag.Parse()

		if kubeconfig != "" {
			restConfig = restconfigFromPath(kubeconfig)
			return restConfig
		}

		kubeconfigenv := os.Getenv("KUBE_CONFIG")
		if kubeconfigenv != "" {
			return restconfigFromPath(kubeconfigenv)
		}
		var err error
		restConfig, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}

	return restConfig
}

func RESTClient() *rest.RESTClient {
	if restClient == nil {
		var err error
		restClient, err = rest.RESTClientFor(RestConfig())
		if err != nil {
			panic(err)
		}
	}
	return restClient
}

func DiscoveryClient() *discovery.DiscoveryClient {
	if discoveryClient == nil {
		var err error
		discoveryClient, err = discovery.NewDiscoveryClientForConfig(RestConfig())
		if err != nil {
			panic(err)
		}
	}
	return discoveryClient
}

func DynamicClient() dynamic.Interface {
	if dynamicClient == nil {
		var err error
		dynamicClient, err = dynamic.NewForConfig(RestConfig())
		if err != nil {
			panic(err)
		}
	}
	return dynamicClient
}

func AllGroupVersionResources() []*schema.GroupVersionResource {
	_, apiResourceList, err := DiscoveryClient().ServerGroupsAndResources()
	if err != nil {
		return nil
	}
	var res []*schema.GroupVersionResource
	for _, apiResource := range apiResourceList {
		for _, resource := range apiResource.APIResources {
			if strings.Contains(resource.Name, "/") {
				continue
			}
			gv := strings.Split(apiResource.GroupVersion, "/")
			var group, version string
			if len(gv) == 2 {
				group = gv[0]
				version = gv[1]
			} else {
				version = apiResource.GroupVersion
			}

			res = append(res, &schema.GroupVersionResource{
				Resource: resource.Name,
				Group:    group,
				Version:  version,
			})
		}
	}

	return res
}

func AllGroupVersionKinds() []*schema.GroupVersionKind {
	_, apiResourceList, err := DiscoveryClient().ServerGroupsAndResources()
	if err != nil {
		return nil
	}
	var res []*schema.GroupVersionKind
	for _, apiResource := range apiResourceList {
		for _, r := range apiResource.APIResources {
			gv := strings.Split(apiResource.GroupVersion, "/")
			var group, version string
			if len(gv) == 2 {
				group = gv[0]
				version = gv[1]
			} else {
				version = apiResource.GroupVersion
			}

			res = append(res, &schema.GroupVersionKind{
				Kind:    r.Kind,
				Group:   group,
				Version: version,
			})
		}
	}

	return res
}

func restconfigFromPath(kubeconfig string) *rest.Config {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		panic(err)
	}
	return config
}
