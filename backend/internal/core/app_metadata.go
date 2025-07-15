package core

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	gatewayapisv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayapisv1beta1 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

type AppMetadata struct {
	AppID            string                     `json:"appId"`
	AppSlug          string                     `json:"appSlug"`
	DisplayName      string                     `json:"displayName"`
	Description      string                     `json:"description"`
	AppType          string                     `json:"appType"`
	RequestCPU       int32                      `json:"requestCPU"`
	RequestMemory    int32                      `json:"requestMemory"`
	LimitCPU         int32                      `json:"limitCPU"`
	LimitMemory      int32                      `json:"limitMemory"`
	Replicas         int32                      `json:"replicas"`
	ContainerImage   string                     `json:"containerImage"`
	RegistryUsername string                     `json:"registryUsername"`
	RegistryPassword string                     `json:"registryPassword"`
	ContainerCommand string                     `json:"containerCommand"`
	EnvVars          []AppMetadataEnvVar        `json:"envVars,omitempty"`
	Volumes          []AppMetadataVolume        `json:"volumes,omitempty"`
	Gateways         []AppMetadataGateway       `json:"gateways,omitempty"`
	Probes           []AppMetadataProbe         `json:"probes,omitempty"`
	SchedulingRule   *AppMetadataSchedulingRule `json:"schedulingRule,omitempty"`
	Edition          string                     `json:"edition,omitempty"`
	EnvID            string                     `json:"envId,omitempty"`
	EnvSlug          string                     `json:"envSlug,omitempty"`
	ProjectID        string                     `json:"projectId,omitempty"`
	ProjectSlug      string                     `json:"projectSlug,omitempty"`
	ClusterNamespace string                     `json:"clusterNamespace"`
}

type AppMetadataEnvVar struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type AppMetadataVolume struct {
	Slug         string   `json:"slug"`
	MountPath    string   `json:"mountPath"`
	SubPath      string   `json:"subPath,omitempty"`
	StorageClass string   `json:"storageClass,omitempty"`
	AccessModes  []string `json:"accessModes,omitempty"`
	VolumeType   string   `json:"volumeType"`
	Capacity     int      `json:"capacity"`
	VolumeMode   string   `json:"volumeMode"`
}

type AppMetadataGateway struct {
	Port        int32  `json:"port"`
	Protocol    string `json:"protocol"`
	Exposed     bool   `json:"exposed"`
	Domain      string `json:"domain,omitempty"`
	Path        string `json:"path,omitempty"`
	GatewayIP   string `json:"gatewayIP,omitempty"`
	GatewayPort int32  `json:"gatewayPort,omitempty"`
}

type AppMetadataProbe struct {
	Type                string `json:"type"`
	ProbeMode           string `json:"probeMode"`
	HTTPGetPath         string `json:"httpGetPath,omitempty"`
	HTTPGetPort         int    `json:"httpGetPort,omitempty"`
	TCPSocketPort       int    `json:"tcpSocketPort,omitempty"`
	ExecCommand         string `json:"execCommand,omitempty"`
	InitialDelaySeconds int32  `json:"initialDelaySeconds"`
	PeriodSeconds       int32  `json:"periodSeconds"`
	TimeoutSeconds      int32  `json:"timeoutSeconds"`
	SuccessThreshold    int32  `json:"successThreshold"`
	FailureThreshold    int32  `json:"failureThreshold"`
}

type Toleration struct {
	Key      string `json:"key,omitempty"`
	Value    string `json:"value,omitempty"`
	Operator string `json:"operator,omitempty"` // e.g., "Equal", "Exists"
	Effect   string `json:"effect,omitempty"`   // e.g., "NoSchedule", "PreferNoSchedule", "NoExecute"
}

type AppMetadataSchedulingRule struct {
	RuleType     string       `json:"ruleType,omitempty"`
	NodeName     string       `json:"nodeName,omitempty"`
	NodeSelector []string     `json:"nodeSelector,omitempty"`
	NodeAffinity []string     `json:"nodeAffinity,omitempty"`
	Tolerations  []Toleration `json:"tolerations,omitempty"`
}

type AppDeployOption struct {
	ZeroReplicas bool // If true, set replicas to 0 for initial deployment
	DebugMode    bool
}

func (a *AppMetadata) Deploy(ctx context.Context, cli client.Client, options *AppDeployOption) app.Error {
	if options != nil {
		if options.ZeroReplicas {
			a.Replicas = 0 // Set replicas to 0 for initial deployment
		}
		if options.DebugMode {
			a.ContainerCommand = "sleep infinity" // Set a debug command to keep the container running
		}
	}

	manifests, err := a.GetApplyManifests()
	if err != nil {
		return err
	}
	for _, resource := range manifests {
		if err := ApplyResource(ctx, cli, resource); err != nil {
			return err
		}
	}

	return nil
}

func (a *AppMetadata) Undeploy(ctx context.Context, cli client.Client) app.Error {
	manifests, err := a.GetApplyManifests()
	if err != nil {
		return err
	}
	for _, resource := range manifests {
		if err := DeleteResource(ctx, cli, resource); err != nil {
			return err
		}
	}

	return nil
}

func (a *AppMetadata) GetApplyManifests() ([]client.Object, app.Error) {
	switch a.AppType {
	case app.AppTypeDeployment:
		return a.deploymentManifests()
	case app.AppTypeStatefulSet:
		return a.statefulSetManifests()
	default:
		return nil, app.NewError(http.StatusBadRequest, "Not supported app type: "+a.AppType)
	}
}

func (a *AppMetadata) standardLabels() map[string]string {
	return map[string]string{
		"ketches.cn/owned":   "true",
		"ketches.cn/app":     a.AppSlug,
		"ketches.cn/id":      a.AppID,
		"ketches.cn/edition": a.Edition,
	}
}

func (a *AppMetadata) standardSelectorLabels() map[string]string {
	return map[string]string{
		"ketches.cn/owned": "true",
		"ketches.cn/app":   a.AppSlug,
	}
}

func (a *AppMetadata) standardAnnotations() map[string]string {
	return map[string]string{
		"ketches.cn/deployed-at": time.Now().Format(time.RFC3339),
	}
}

func (a *AppMetadata) deploymentManifests() ([]client.Object, app.Error) {
	var result []client.Object

	envs := make([]corev1.EnvVar, 0, len(a.EnvVars))
	for _, envVar := range a.EnvVars {
		envs = append(envs, corev1.EnvVar{
			Name:  envVar.Key,
			Value: envVar.Value,
		})
	}

	labels := a.standardLabels()
	annotations := a.standardAnnotations()
	selectorLabels := a.standardSelectorLabels()

	for _, pvc := range a.persistentVolumeClaimManifests() {
		result = append(result, &pvc)
	}

	volumeMounts := make([]corev1.VolumeMount, 0, len(a.Volumes))
	volumes := make([]corev1.Volume, 0, len(a.Volumes))
	for _, volume := range a.Volumes {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volume.Slug,
			MountPath: volume.MountPath,
			SubPath:   volume.SubPath,
		})
		volumes = append(volumes, corev1.Volume{
			Name: volume.Slug,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: volume.Slug,
				},
			},
		})
	}

	if len(a.Gateways) > 0 {
		result = append(result, a.serviceManifest()...)
		result = append(result, a.gatewayManifests()...)
	}

	var (
		command []string
		args    []string
	)
	if a.ContainerCommand != "" {
		labels["ketches.cn/debugging"] = "true" // Mark as debugging if command is set
		command = []string{"sh"}
		args = []string{"-c", a.ContainerCommand}
	}

	var livenessProbe, readinessProbe, startupProbe *corev1.Probe
	for _, probe := range a.Probes {
		p := &corev1.Probe{
			InitialDelaySeconds: probe.InitialDelaySeconds,
			TimeoutSeconds:      probe.TimeoutSeconds,
			PeriodSeconds:       probe.PeriodSeconds,
			SuccessThreshold:    probe.SuccessThreshold,
			FailureThreshold:    probe.FailureThreshold,
		}
		var probeHandler corev1.ProbeHandler
		switch probe.ProbeMode {
		case app.AppProbeModeHTTPGet:
			probeHandler = corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: probe.HTTPGetPath,
					Port: intstr.FromInt(probe.HTTPGetPort),
				},
			}
		case app.AppProbeModeTCPSocket:
			probeHandler = corev1.ProbeHandler{
				TCPSocket: &corev1.TCPSocketAction{
					Port: intstr.FromInt(probe.TCPSocketPort),
				},
			}
		case app.AppProbeModeExec:
			probeHandler = corev1.ProbeHandler{
				Exec: &corev1.ExecAction{
					Command: []string{"/bin/sh", "-c", probe.ExecCommand},
				},
			}
		}
		p.ProbeHandler = probeHandler
		switch probe.Type {
		case app.AppProbeTypeLiveness:
			livenessProbe = p
		case app.AppProbeTypeReadiness:
			readinessProbe = p
		case app.AppProbeTypeStartup:
			startupProbe = p
		}
	}

	var (
		schedulingRuleNodeName     string
		schedulingRuleNodeSelector map[string]string
		nodeAffinity               *corev1.NodeAffinity
		tolerations                []corev1.Toleration
	)
	if a.SchedulingRule != nil {
		switch a.SchedulingRule.RuleType {
		case app.SchedulingRuleTypeNodeName:
			schedulingRuleNodeName = a.SchedulingRule.NodeName
		case app.SchedulingRuleTypeNodeSelector:
			for _, selector := range a.SchedulingRule.NodeSelector {
				parts := strings.SplitN(selector, "=", 2)
				if len(parts) != 2 {
					log.Printf("invalid node selector format: %s", selector)
					continue
				}
				if schedulingRuleNodeSelector == nil {
					schedulingRuleNodeSelector = make(map[string]string)
				}
				schedulingRuleNodeSelector[parts[0]] = parts[1]
			}
		case app.SchedulingRuleTypeNodeAffinity:
			if len(a.SchedulingRule.NodeAffinity) > 0 {
				nodeAffinity = &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "kubernetes.io/hostname",
										Operator: corev1.NodeSelectorOpIn,
										Values:   a.SchedulingRule.NodeAffinity,
									},
								},
							},
						},
					},
				}
			}
		}

		for _, toleration := range a.SchedulingRule.Tolerations {
			tolerations = append(tolerations, corev1.Toleration{
				Key:      toleration.Key,
				Value:    toleration.Value,
				Operator: corev1.TolerationOperator(toleration.Operator),
				Effect:   corev1.TaintEffect(toleration.Effect),
			})
		}
	}

	result = append(result, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        a.AppSlug,
			Namespace:   a.ClusterNamespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &a.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					NodeName:     schedulingRuleNodeName,
					NodeSelector: schedulingRuleNodeSelector,
					Containers: []corev1.Container{
						{
							Name:            a.AppSlug,
							Image:           a.ContainerImage,
							ImagePullPolicy: corev1.PullAlways,
							Command:         command,
							Args:            args,
							Env:             envs,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", a.RequestCPU)),
									corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", a.RequestMemory)),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", a.LimitCPU)),
									corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", a.LimitMemory)),
								},
							},
							LivenessProbe:  livenessProbe,
							ReadinessProbe: readinessProbe,
							StartupProbe:   startupProbe,
							VolumeMounts:   volumeMounts,
						},
					},
					Volumes: volumes,
					Affinity: &corev1.Affinity{
						NodeAffinity: nodeAffinity,
					},
					Tolerations: tolerations,
				},
			},
		},
	})

	return result, nil
}

func (a *AppMetadata) statefulSetManifests() ([]client.Object, app.Error) {
	var result []client.Object

	envs := make([]corev1.EnvVar, 0, len(a.EnvVars))
	for _, envVar := range a.EnvVars {
		envs = append(envs, corev1.EnvVar{
			Name:  envVar.Key,
			Value: envVar.Value,
		})
	}

	labels := a.standardLabels()
	selectorLabels := a.standardSelectorLabels()
	annotations := a.standardAnnotations()

	var (
		volumeClaims = a.persistentVolumeClaimManifests()
		volumeMounts = make([]corev1.VolumeMount, 0, len(a.Volumes))
		volumes      = make([]corev1.Volume, 0, len(a.Volumes))
	)
	for _, volume := range a.Volumes {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volume.Slug,
			MountPath: volume.MountPath,
			SubPath:   volume.SubPath,
		})

		volumes = append(volumes, corev1.Volume{
			Name: volume.Slug,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: volume.Slug,
				},
			},
		})
	}

	if len(a.Gateways) > 0 {
		result = append(result, a.serviceManifest()...)
		result = append(result, a.gatewayManifests()...)
	}

	var (
		command []string
		args    []string
	)
	if a.ContainerCommand != "" {
		command = []string{"sh"}
		args = []string{"-c", a.ContainerCommand}
	}

	var (
		schedulingRuleNodeName     string
		schedulingRuleNodeSelector map[string]string
		nodeAffinity               *corev1.NodeAffinity
		tolerations                []corev1.Toleration
	)
	if a.SchedulingRule != nil {
		switch a.SchedulingRule.RuleType {
		case app.SchedulingRuleTypeNodeName:
			schedulingRuleNodeName = a.SchedulingRule.NodeName
		case app.SchedulingRuleTypeNodeSelector:
			for _, selector := range a.SchedulingRule.NodeSelector {
				parts := strings.SplitN(selector, "=", 2)
				if len(parts) != 2 {
					log.Printf("invalid node selector format: %s", selector)
					continue
				}
				if schedulingRuleNodeSelector == nil {
					schedulingRuleNodeSelector = make(map[string]string)
				}
				schedulingRuleNodeSelector[parts[0]] = parts[1]
			}
		case app.SchedulingRuleTypeNodeAffinity:
			if len(a.SchedulingRule.NodeAffinity) > 0 {
				nodeAffinity = &corev1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
						NodeSelectorTerms: []corev1.NodeSelectorTerm{
							{
								MatchExpressions: []corev1.NodeSelectorRequirement{
									{
										Key:      "kubernetes.io/hostname",
										Operator: corev1.NodeSelectorOpIn,
										Values:   a.SchedulingRule.NodeAffinity,
									},
								},
							},
						},
					},
				}
			}
		}

		for _, toleration := range a.SchedulingRule.Tolerations {
			tolerations = append(tolerations, corev1.Toleration{
				Key:      toleration.Key,
				Value:    toleration.Value,
				Operator: corev1.TolerationOperator(toleration.Operator),
				Effect:   corev1.TaintEffect(toleration.Effect),
			})
		}
	}

	result = append(result, &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.AppSlug,
			Namespace: a.ClusterNamespace,
			Labels:    labels,
		},
		Spec: appsv1.StatefulSetSpec{
			VolumeClaimTemplates: volumeClaims,
			ServiceName:          a.AppSlug,
			Replicas:             &a.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					NodeName:     schedulingRuleNodeName,
					NodeSelector: schedulingRuleNodeSelector,
					Containers: []corev1.Container{
						{
							Name:            a.AppSlug,
							Image:           a.ContainerImage,
							ImagePullPolicy: corev1.PullAlways,
							Command:         command,
							Args:            args,
							Env:             envs,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", a.RequestCPU)),
									corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", a.RequestMemory)),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%dm", a.LimitCPU)),
									corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%dMi", a.LimitMemory)),
								},
							},
							VolumeMounts: volumeMounts,
						},
					},
					Volumes: volumes,
					Affinity: &corev1.Affinity{
						NodeAffinity: nodeAffinity,
					},
					Tolerations: tolerations,
				},
			},
			PersistentVolumeClaimRetentionPolicy: &appsv1.StatefulSetPersistentVolumeClaimRetentionPolicy{
				WhenDeleted: appsv1.RetainPersistentVolumeClaimRetentionPolicyType,
			},
		},
	})

	return result, nil
}

func (a *AppMetadata) persistentVolumeClaimManifests() []corev1.PersistentVolumeClaim {
	var result []corev1.PersistentVolumeClaim

	for _, volume := range a.Volumes {
		result = append(result, corev1.PersistentVolumeClaim{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PersistentVolumeClaim", // Specify the kind explicitly, used in apply logic
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      volume.Slug,
				Namespace: a.ClusterNamespace,
				Labels:    a.standardLabels(),
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				VolumeMode:  utils.Ptr(corev1.PersistentVolumeMode(volume.VolumeMode)),
				AccessModes: pvcAccessModes(volume.AccessModes),
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dMi", volume.Capacity)),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(fmt.Sprintf("%dMi", volume.Capacity)),
					},
				},
				StorageClassName: &volume.StorageClass,
			},
		})
	}

	return result
}

func (a *AppMetadata) serviceManifest() []client.Object {
	gateways := make(map[string]AppMetadataGateway)
	for _, gateway := range a.Gateways {
		key := fmt.Sprintf("%s-%s-%d", a.AppSlug, gateway.Protocol, gateway.Port)
		if _, ok := gateways[key]; !ok {
			gateways[key] = gateway
		}
	}

	result := make([]client.Object, 0, len(gateways)+1)

	labels := a.standardLabels()
	selectorLabels := a.standardSelectorLabels()

	ports := make([]corev1.ServicePort, 0, len(a.Gateways))
	for _, g := range gateways {
		// All ports in one service
		ports = append(ports, corev1.ServicePort{
			Protocol: corev1.ProtocolTCP,
			Port:     g.Port,
			TargetPort: intstr.IntOrString{
				IntVal: g.Port,
			},
		})

		// Create a service for each gateway
		result = append(result, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s-%d", a.AppSlug, g.Protocol, g.Port),
				Namespace: a.ClusterNamespace,
				Labels:    labels,
			},
			Spec: corev1.ServiceSpec{
				Selector: selectorLabels,
				Ports: []corev1.ServicePort{
					{
						Protocol: corev1.ProtocolTCP,
						Port:     g.Port,
						TargetPort: intstr.IntOrString{
							IntVal: g.Port,
						},
					},
				},
			},
		})
	}

	result = append(result, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      a.AppSlug,
			Namespace: a.ClusterNamespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports:    ports,
		},
	})

	return result
}

func (a *AppMetadata) gatewayManifests() []client.Object {
	var result []client.Object
	for _, gateway := range a.Gateways {
		if !gateway.Exposed {
			continue // Skip gateways that are not exposed
		}

		switch gateway.Protocol {
		case app.AppGatewayProtocolHTTP, app.AppGatewayProtocolHTTPS:
			if gateway.Domain == "" {
				continue // Skip if domain is not set for HTTP/HTTPS gateways
			}
			if gateway.Path == "" {
				gateway.Path = "/" // Default path for HTTP/HTTPS gateways
			}

			gatewayName := a.ClusterNamespace // Gateway auto generated for each env, so use namespace as class name

			// Gateway
			// result = append(result, &gatewayapisv1.Gateway{
			// 	ObjectMeta: metav1.ObjectMeta{
			// 		Name:      gatewayName,
			// 		Namespace: a.ClusterNamespace,
			// 		Labels:    a.standardLabels(),
			// 	},
			// 	Spec: gatewayapisv1.GatewaySpec{
			// 		GatewayClassName: gatewayapisv1.ObjectName(a.ClusterNamespace),
			// 		Listeners: []gatewayapisv1.Listener{
			// 			{
			// 				Name:     "http",
			// 				Protocol: gatewayapisv1.HTTPProtocolType,
			// 				Port:     gatewayapisv1.PortNumber(80),
			// 				AllowedRoutes: &gatewayapisv1.AllowedRoutes{
			// 					Namespaces: &gatewayapisv1.RouteNamespaces{
			// 						From: utils.Ptr(gatewayapisv1.NamespacesFromSame),
			// 					},
			// 					Kinds: []gatewayapisv1.RouteGroupKind{
			// 						{
			// 							Kind: "HTTPRoute",
			// 						},
			// 					},
			// 				},
			// 			},
			// 		},
			// 	},
			// })

			// HTTPRoute
			result = append(result, &gatewayapisv1.HTTPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      gatewayName,
					Namespace: a.ClusterNamespace,
					Labels:    a.standardLabels(),
				},
				Spec: gatewayapisv1.HTTPRouteSpec{
					CommonRouteSpec: gatewayapisv1.CommonRouteSpec{
						ParentRefs: []gatewayapisv1.ParentReference{
							{
								Name: gatewayapisv1.ObjectName(gatewayName),
							},
						},
					},
					Hostnames: []gatewayapisv1.Hostname{
						gatewayapisv1.Hostname(gateway.Domain),
					},
					Rules: []gatewayapisv1.HTTPRouteRule{
						{
							Matches: []gatewayapisv1.HTTPRouteMatch{
								{
									Path: &gatewayapisv1.HTTPPathMatch{
										Type:  utils.Ptr(gatewayapisv1.PathMatchPathPrefix),
										Value: utils.Ptr(gateway.Path),
									},
								},
							},
							BackendRefs: []gatewayapisv1.HTTPBackendRef{
								{
									BackendRef: gatewayapisv1.BackendRef{
										BackendObjectReference: gatewayapisv1.BackendObjectReference{
											Name: gatewayapisv1.ObjectName(a.AppSlug),
											Port: utils.Ptr(gatewayapisv1.PortNumber(gateway.Port)),
										},
									},
								},
							},
						},
					},
				},
			})
		case app.AppGatewayProtocolTCP, app.AppGatewayProtocolUDP:
			if gateway.GatewayPort == 0 {
				continue // Skip if GatewayPort is not set
			}

			gatewayName := fmt.Sprintf("%s-%s-%d", a.AppSlug, gateway.Protocol, gateway.Port)

			// Gateway
			result = append(result, &gatewayapisv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      gatewayName,
					Namespace: a.ClusterNamespace,
					Labels:    a.standardLabels(),
				},
				Spec: gatewayapisv1.GatewaySpec{
					GatewayClassName: gatewayapisv1.ObjectName("nginx"),
					Listeners: []gatewayapisv1.Listener{
						{
							Name:     "tcp",
							Protocol: gatewayapisv1.TCPProtocolType,
							Port:     gatewayapisv1.PortNumber(gateway.GatewayPort),
							AllowedRoutes: &gatewayapisv1.AllowedRoutes{
								Namespaces: &gatewayapisv1.RouteNamespaces{
									From: utils.Ptr(gatewayapisv1.NamespacesFromSame),
								},
								Kinds: []gatewayapisv1.RouteGroupKind{
									{
										Kind: "TCPRoute",
									},
								},
							},
						},
					},
				},
			})

			// TCPRoute
			result = append(result, &gatewayapisv1beta1.TCPRoute{
				ObjectMeta: metav1.ObjectMeta{
					Name:      gatewayName,
					Namespace: a.ClusterNamespace,
					Labels:    a.standardLabels(),
				},
				Spec: gatewayapisv1beta1.TCPRouteSpec{
					CommonRouteSpec: gatewayapisv1beta1.CommonRouteSpec{
						ParentRefs: []gatewayapisv1beta1.ParentReference{
							{
								Name:        gatewayapisv1beta1.ObjectName(gatewayName),
								SectionName: utils.Ptr(gatewayapisv1.SectionName("tcp")),
							},
						},
					},
					Rules: []gatewayapisv1beta1.TCPRouteRule{
						{
							BackendRefs: []gatewayapisv1beta1.BackendRef{
								{
									BackendObjectReference: gatewayapisv1beta1.BackendObjectReference{
										Name: gatewayapisv1beta1.ObjectName(a.AppSlug),
										Port: utils.Ptr(gatewayapisv1beta1.PortNumber(gateway.Port)),
									},
								},
							},
						},
					},
				},
			})
		}

		var httpPath = gateway.Path
		if len(httpPath) == 0 {
			httpPath = "/"
		}

	}
	return result
}

func pvcAccessModes(accessModes []string) []corev1.PersistentVolumeAccessMode {
	var kubeAccessModes []corev1.PersistentVolumeAccessMode
	for _, mode := range accessModes {
		kubeAccessModes = append(kubeAccessModes, corev1.PersistentVolumeAccessMode(mode))
	}
	return kubeAccessModes
}
