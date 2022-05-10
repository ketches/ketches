package k8s

const (
	// High enough QPS models fit all expected use cases.
	KubeDefaultQPS = 1e6
	// High enough Burst models fit all expected use cases.
	KubeDefaultBurst = 1e6
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
