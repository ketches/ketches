package kube

type Model struct {
	Name        string      `json:"name"`
	Labels      Labels      `json:"labels"`
	Annotations Annotations `json:"annotations"`
}

type NamespacedModel struct {
	Model
	Namespace string `json:"namespace"`
}

type Namespace struct {
	Model
}

type keyvals map[string]string

type Labels keyvals

type Annotations keyvals

type Deployment struct {
	NamespacedModel
	Containers []Container `json:"containers"`
}

type Pod struct {
	NamespacedModel
	ServiceAccount string      `json:"serviceAccount"`
	Containers     []Container `json:"containers"`
}

type Container struct {
	Name    string          `json:"name"`
	Image   string          `json:"image"`
	Args    string          `json:"args"`
	Envs    []keyvals       `json:"envs"`
	Ports   []ContainerPort `json:"ports"`
	Configs []keyvals       `json:"configs"`
	Volumes []Volume        `json:"volumes"`
}

type ContainerPort struct {
	Protocol string `json:"protocol"`
	Name     string `json:"name"`
	Port     int32  `json:"port"`
}

type Volume struct {
}

type PodDisruptionBudget struct {
	NamespacedModel
	MinAvailablePodReplicas int32  `json:"minAvailablePodReplicas"`
	SelectorMatchLabels     Labels `json:"selectorMatchLabels"`
}

type HorizontalPodAutoscaler struct {
	NamespacedModel
	MaxReplicas                 int32        `json:"maxReplicas"`
	MinReplicas                 int32        `json:"minReplicas"`
	ScaleTargetKind             ResourceKind `json:"scaleTargetKind"`
	ScaleTargetName             string       `json:"scaleTargetName"`
	TargetCPUAverageUtilization int32        `json:"targetCPUAverageUtilization"`
}

type Service struct {
	NamespacedModel
	SelectorMatchLabels Labels        `json:"selectorMatchLabels"`
	Ports               []ServicePort `json:"ports"`
	Type                ServiceType   `json:"type"`
}

type ServiceType string

const (
	ClusterID    ServiceType = "ClusterID"
	NodePort     ServiceType = "NodePort"
	LoadBalancer ServiceType = "LoadBalancer"
	ExternalName ServiceType = "ExternalName"
)

type ServicePort struct {
	Name       string `json:"name"`
	Port       int32  `json:"port"`
	TargetPort string `json:"targetPort"`
	NodePort   int32  `json:"nodePort"`
	Protocol   string `json:"protocol"`
}

type ServiceAccount struct {
	NamespacedModel
	RoleName      string `json:"roleName"`
	IsBindingRole bool   `json:"isBindingRole"`
	IsClusterRole bool   `json:"isClusterRole"`
}
