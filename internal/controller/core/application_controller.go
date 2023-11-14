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

package core

import (
	"context"
	"fmt"
	"path"
	"time"

	corev1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	"github.com/ketches/ketches/pkg/kube"
	"github.com/ketches/ketches/pkg/kube/incluster"
	"github.com/ketches/ketches/pkg/kube/workercluster"
	"github.com/ketches/ketches/util/conv"
	appsv1 "k8s.io/api/apps/v1"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"
	gatewayapisv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// ApplicationReconciler reconciles an Application object
type ApplicationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.ketches.io,resources=applications,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.ketches.io,resources=applications/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.ketches.io,resources=applications/finalizers,verbs=update

// Reconcile reconciles Application objects
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)

	var app = &corev1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
	}

	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var space = &corev1alpha1.Space{}
	err := r.Get(ctx, types.NamespacedName{Name: app.Namespace}, space)
	if err != nil {
		app.Status.Phase = corev1alpha1.ApplicationPhasePending
		app.SetStatusCondition(corev1alpha1.ApplicationConditionTypeSpaceReady, err)
		return r.updateStatus(ctx, app, nil)
	}

	// check space status
	if space.Status.Phase != corev1alpha1.SpacePhaseReady {
		app.SetStatusCondition(corev1alpha1.ApplicationConditionTypeSpaceReady, fmt.Errorf("space %s not ready", space.Name))
		return r.updateStatus(ctx, app, nil)
	}

	workerCluster, ok := incluster.Store().Clusterset().Cluster(space.Spec.Cluster)
	if !ok {
		log.Error(err, "unable to get worker cluster")
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}
	if app.CheckOrSetRequiredLabels() {
		if err := kube.ApplyResource(ctx, r.Client, app); err != nil {
			log.Error(err, "failed to update Application labels")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if app.Status.Phase == "" {
		app.Status.Phase = corev1alpha1.ApplicationPhasePending
		return r.updateStatus(ctx, app, nil)
	}

	if app.GetDeletionTimestamp() != nil {
		return r.onApplicationDeleted(ctx, app, workerCluster)
	} else {
		if app.CheckOrSetFinalizers() {
			err := kube.ApplyResource(ctx, r.Client, app)
			if err != nil {
				log.Error(err, "unable to update Application finalizers")
				return ctrl.Result{}, err
			}
		}
	}

	err = r.applyApplicationDerivedResources(ctx, workerCluster, app)
	if err != nil {
		log.Error(err, "failed to apply resources")
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	r.completeApplicationStatus(ctx, app, workerCluster)

	return r.updateStatus(ctx, app, nil)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Application{}).
		Complete(r)
}

// onApplicationDeleted handles Application deletion
func (r *ApplicationReconciler) onApplicationDeleted(ctx context.Context, app *corev1alpha1.Application, workerCluster workercluster.Cluster) (ctrl.Result, error) {
	err := workerCluster.KubeClientset().CoreV1().ServiceAccounts(app.Namespace).Delete(ctx, applicationOwnerServiceAccountName(app.Name), metav1.DeleteOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return ctrl.Result{Requeue: true}, err
	} else {
		app.SetFinalizers(nil)
		err = kube.ApplyResource(ctx, r.Client, app)
	}
	return ctrl.Result{}, err
}

func (r *ApplicationReconciler) applyApplicationDerivedResources(ctx context.Context, workerCluster workercluster.Cluster, app *corev1alpha1.Application) error {
	// apply owner service account
	ownerServiceAccount, err := workerCluster.KubeClientset().CoreV1().ServiceAccounts(app.Namespace).Get(ctx, applicationOwnerServiceAccountName(app.Name), metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		if ownerServiceAccount, err = workerCluster.KubeClientset().CoreV1().ServiceAccounts(app.Namespace).Create(ctx, constructApplicationOwnerServiceAccount(app), metav1.CreateOptions{}); err != nil {
			return err
		}
	}
	ownerReference := *metav1.NewControllerRef(ownerServiceAccount, corev1.SchemeGroupVersion.WithKind("ServiceAccount"))

	// delete obsolete configmaps
	configMaps := &corev1.ConfigMapList{}
	workerCluster.KubeRuntimeClient().List(ctx, configMaps, client.InNamespace(app.Namespace), client.MatchingLabels(app.Labels))
	for _, configmap := range configMaps.Items {
		var found bool
		for _, mountFile := range app.Spec.MountFiles {
			if configMapName(app, mountFile) == configmap.Name {
				found = true
				break
			}
		}
		if !found {
			kube.DeleteResource(ctx, workerCluster.KubeRuntimeClient(), &configmap)
		}
	}

	// apply the configmaps
	for _, mountFile := range app.Spec.MountFiles {
		configMap := constructConfigMap(app, mountFile, ownerReference)
		if err := kube.ApplyResource(ctx, workerCluster.KubeRuntimeClient(), configMap); err != nil {
			return err
		}
	}

	// delete obsolete pvcs
	pvcs := &corev1.PersistentVolumeClaimList{}
	workerCluster.KubeRuntimeClient().List(ctx, pvcs, client.InNamespace(app.Namespace), client.MatchingLabels(app.Labels))
	for _, pvc := range pvcs.Items {
		var found bool
		for _, md := range app.Spec.MountDirectories {
			if persistentVolumeClaimName(app, md) == pvc.Name {
				found = true
				break
			}
		}
		if !found {
			kube.DeleteResource(ctx, workerCluster.KubeRuntimeClient(), &pvc)
		}
	}

	// apply the pvcs and pvs
	for _, md := range app.Spec.MountDirectories {
		pvc := constructPersistentVolumeClaim(app, md, ownerReference)
		if _, err := workerCluster.KubeClientset().CoreV1().PersistentVolumeClaims(app.Namespace).Get(ctx, pvc.Name, metav1.GetOptions{}); err != nil {
			if errors.IsNotFound(err) {
				if _, err = workerCluster.KubeClientset().CoreV1().PersistentVolumeClaims(app.Namespace).Create(ctx, pvc, metav1.CreateOptions{}); err != nil {
					return err
				}
			} else {
				return err
			}
		}
		// if app.Spec.Type == corev1alpha1.WorkloadTypeDeployment {
		// if err = kube.ApplyResource(ctx, workerCluster.KubeRuntimeClient(), pvc); err != nil {
		// 	return err
		// }
		// }

		// pv := app.ConstructPersistentVolume(md)
		// if err := kube.ApplyResource(ctx, workerCluster.KubeRuntimeClient(), pv); err != nil {
		// 	// return err
		// }
	}

	// apply the workload
	workload := constructWorkload(app, ownerReference)
	switch workloadType(app) {
	case corev1alpha1.WorkloadTypeDeployment:
		if err := kube.ApplyResource(ctx, workerCluster.KubeRuntimeClient(), workload); err != nil {
			return err
		}
		kube.DeleteResource(ctx, workerCluster.KubeRuntimeClient(), &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app.Name,
				Namespace: app.Namespace,
			},
		})
	case corev1alpha1.WorkloadTypeStatefulSet:
		if err := kube.ApplyResource(ctx, workerCluster.KubeRuntimeClient(), workload); err != nil {
			return err
		}
		kube.DeleteResource(ctx, workerCluster.KubeRuntimeClient(), &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app.Name,
				Namespace: app.Namespace,
			},
		})
	default:
		return fmt.Errorf("unknown workload type: %s", workloadType(app))
	}

	// apply the autoscaler
	if app.Spec.Autoscaler != nil {
		autoscaler := constructHorizontalPodAutoscaler(app, workload, ownerReference)
		if err := kube.ApplyResource(ctx, workerCluster.KubeRuntimeClient(), autoscaler); err != nil {
			return err
		}
	} else {
		// delete the autoscaler
		autoscaler := &autoscalingv1.HorizontalPodAutoscaler{
			ObjectMeta: metav1.ObjectMeta{
				Name:      app.Name,
				Namespace: app.Namespace,
			},
		}
		if err := kube.DeleteResource(ctx, workerCluster.KubeRuntimeClient(), autoscaler); err != nil {
			return err
		}
	}

	// delete obsolete services and ingresses
	services := &corev1.ServiceList{}
	err = workerCluster.KubeRuntimeClient().List(ctx, services, client.InNamespace(app.Namespace), client.MatchingLabels(app.Labels))
	if err != nil {
		return err
	}
	for _, service := range services.Items {
		var found bool
		for _, port := range app.Spec.Ports {
			if service.Name == fmt.Sprintf("%s-%d", app.Name, port.Number) {
				found = true
				break
			}
		}
		if !found {
			if err := workerCluster.KubeRuntimeClient().Delete(ctx, &service); err != nil {
				return err
			}
		}
	}

	// if workerCluster.GatewayAPIRuntimeClient() != nil {
	// 	gateways := &gatewayapi.GatewayList{}
	// 	err = workerCluster.GatewayAPIRuntimeClient().List(ctx, gateways, client.InNamespace(app.Namespace), client.MatchingLabels(app.Labels))
	// 	if err != nil {
	// 		return err
	// 	}
	// 	for _, gateway := range gateways.Items {
	// 		var found bool
	// 		for _, port := range app.Spec.Ports {
	// 			for _, gw := range port.Gateways {
	// 				if gw.Type == corev1alpha1.GatewayTypeHTTP && gateway.Name == gatewayName(app, port.Number, gw) {
	// 					found = true
	// 					break
	// 				}
	// 			}
	// 		}
	// 		if !found {
	// 			if err := workerCluster.KubeRuntimeClient().Delete(ctx, &gateway); err != nil {
	// 				return err
	// 			}
	// 		}
	// 	}
	// }

	// apply the service or ingress
	for _, port := range app.Spec.Ports {
		svc := constructService(app, port, ownerReference)
		for _, gw := range port.Gateways {
			switch gw.Type {
			case corev1alpha1.GatewayTypeTCP:
				svc.Spec.Type = corev1.ServiceTypeNodePort
				svc.Spec.Ports[0].NodePort = gw.NodePort
				if err := kube.ApplyResource(ctx, workerCluster.KubeRuntimeClient(), svc); err != nil {
					return err
				}
			case corev1alpha1.GatewayTypeHTTP:
				if err := kube.ApplyResource(ctx, workerCluster.KubeRuntimeClient(), svc); err != nil {
					return err
				}
				if len(gw.Host) > 0 && workerCluster.GatewayAPIRuntimeClient() != nil {
					// gateway api integration
					// gateways := constructGateway(app, port, ownerReference)
					// for _, gateway := range gateways {
					// 	if err := kube.ApplyResource(ctx, workerCluster.GatewayAPIRuntimeClient(), gateway); err != nil {
					// 		return err
					// 	}
					// }
					httpRoutes := constructHTTPRoute(app, port, ownerReference)
					for _, httpRoute := range httpRoutes {
						if err := kube.ApplyResource(ctx, workerCluster.GatewayAPIRuntimeClient(), httpRoute); err != nil {
							return err
						}
					}
				}
			}
		}
	}

	return nil
}

func (r *ApplicationReconciler) completeApplicationStatus(ctx context.Context, app *corev1alpha1.Application, workerCluster workercluster.Cluster) {
	switch workloadType(app) {
	case corev1alpha1.WorkloadTypeDeployment:
		workload := &appsv1.Deployment{}
		err := workerCluster.KubeRuntimeClient().Get(ctx, client.ObjectKey{Namespace: app.Namespace, Name: app.Name}, workload)
		if err != nil {
			break
		}
		app.Status.DeploymentConditions = workload.Status.Conditions
		switch app.Spec.DesiredState {
		case corev1alpha1.ApplicationDesiredStateRunning:
			if workload.Spec.Replicas == conv.Ptr(int32(0)) {
				if workload.Status.Replicas == 0 {
					app.Status.Phase = corev1alpha1.ApplicationPhaseStopped
				} else {
					app.Status.Phase = corev1alpha1.ApplicationPhaseStopping
				}
			} else {
				if workload.Status.Replicas > *workload.Spec.Replicas {
					app.Status.Phase = corev1alpha1.ApplicationPhaseRolling
				} else if workload.Status.AvailableReplicas == *workload.Spec.Replicas {
					app.Status.Phase = corev1alpha1.ApplicationPhaseRunning
				} else {
					app.Status.Phase = corev1alpha1.ApplicationPhaseStarting
				}
			}
		case corev1alpha1.ApplicationDesiredStateStopped:
			if workload.Spec.Replicas == conv.Ptr(int32(0)) {
				if workload.Status.Replicas == 0 {
					app.Status.Phase = corev1alpha1.ApplicationPhaseStopped
				} else {
					app.Status.Phase = corev1alpha1.ApplicationPhaseStopping
				}
			} else {
				app.Status.Phase = corev1alpha1.ApplicationPhaseStopping
			}
		}
	case corev1alpha1.WorkloadTypeStatefulSet:
		workload := &appsv1.StatefulSet{}
		err := workerCluster.KubeRuntimeClient().Get(ctx, client.ObjectKey{Namespace: app.Namespace, Name: app.Name}, workload)
		if err != nil {
			break
		}
		app.Status.StatefulSetConditions = workload.Status.Conditions
		switch app.Spec.DesiredState {
		case corev1alpha1.ApplicationDesiredStateRunning:
			if workload.Spec.Replicas == conv.Ptr(int32(0)) {
				if workload.Status.Replicas == 0 {
					app.Status.Phase = corev1alpha1.ApplicationPhasePending
				} else {
					app.Status.Phase = corev1alpha1.ApplicationPhaseStopping
				}
			} else {
				if workload.Status.AvailableReplicas == *workload.Spec.Replicas {
					app.Status.Phase = corev1alpha1.ApplicationPhaseRunning
				} else if workload.Status.Replicas > *workload.Spec.Replicas {
					app.Status.Phase = corev1alpha1.ApplicationPhaseRolling
				} else {
					app.Status.Phase = corev1alpha1.ApplicationPhaseStarting
				}
			}
		case corev1alpha1.ApplicationDesiredStateStopped:
			if workload.Spec.Replicas == conv.Ptr(int32(0)) {
				if workload.Status.Replicas == 0 {
					app.Status.Phase = corev1alpha1.ApplicationPhaseStopped
				} else {
					app.Status.Phase = corev1alpha1.ApplicationPhaseStopping
				}
			} else {
				app.Status.Phase = corev1alpha1.ApplicationPhaseStopping
			}
		}
	}
	app.Status.Edition = app.Labels[corev1alpha1.ApplicationEditionLabelKey]
}

func (r *ApplicationReconciler) updateStatus(ctx context.Context, app *corev1alpha1.Application, err error) (ctrl.Result, error) {
	if updateErr := kube.UpdateResourceStatus(ctx, r.Client, app); updateErr != nil {
		return ctrl.Result{}, updateErr
	}
	if err != nil {
		return ctrl.Result{}, err
	}
	if app.Status.Phase != corev1alpha1.ApplicationPhaseRunning && app.Status.Phase != corev1alpha1.ApplicationPhaseStopped {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}
	return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
}

func constructWorkload(app *corev1alpha1.Application, owners ...metav1.OwnerReference) client.Object {
	objectMeta := metav1.ObjectMeta{
		Name:            app.Name,
		Namespace:       app.Namespace,
		Labels:          app.Labels,
		OwnerReferences: owners,
	}

	mainContainer := corev1.Container{
		Name:           app.Name,
		Image:          app.Spec.Image,
		Resources:      app.Spec.Resources,
		Command:        app.Spec.Command,
		Args:           app.Spec.Args,
		Env:            app.Spec.Env,
		LivenessProbe:  app.Spec.Healthz,
		ReadinessProbe: app.Spec.Healthz,
		SecurityContext: &corev1.SecurityContext{
			Privileged: &app.Spec.Privileged,
		},
	}

	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: app.Labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				mainContainer,
			},
			ImagePullSecrets: func(secrets []string) []corev1.LocalObjectReference {
				res := make([]corev1.LocalObjectReference, len(secrets))
				for i, v := range secrets {
					res[i] = corev1.LocalObjectReference{Name: v}
				}
				return res
			}(app.Spec.ImagePullSecrets),
		},
	}

	if app.Spec.Sidecars != nil {
		for _, sidecar := range app.Spec.Sidecars {
			switch sidecar.Type {
			case corev1alpha1.SidecarTypeInitRun:
				podTemplate.Spec.InitContainers = append(podTemplate.Spec.InitContainers, constructContainer(sidecar))
			case corev1alpha1.SidecarTypePreRun:
				podTemplate.Spec.Containers = append([]corev1.Container{constructContainer(sidecar)}, podTemplate.Spec.Containers...)
			case corev1alpha1.SidecarTypePostRun:
				podTemplate.Spec.Containers = append(podTemplate.Spec.Containers, constructContainer(sidecar))
			}
		}
	}

	for _, mf := range app.Spec.MountFiles {
		cm := configMapName(app, mf)
		podTemplate.Spec.Volumes = append(podTemplate.Spec.Volumes, corev1.Volume{
			Name: cm,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: cm},
					Items: []corev1.KeyToPath{
						{
							Key:  fileName(mf),
							Path: fileName(mf),
							Mode: fileMode(mf),
						},
					},
				},
			},
		})
		for _, c := range podTemplate.Spec.InitContainers {
			c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
				Name:      cm,
				MountPath: mf.Path,
				SubPath:   fileName(mf),
			})
		}
		for _, c := range podTemplate.Spec.Containers {
			c.VolumeMounts = append(c.VolumeMounts, corev1.VolumeMount{
				Name:      cm,
				MountPath: mf.Path,
			})
		}
	}

	// var volTmpls []corev1.PersistentVolumeClaim
	var volMnts []corev1.VolumeMount
	var vols []corev1.Volume
	for _, md := range app.Spec.MountDirectories {
		// volTmpls = append(volTmpls, app.ConstructPersistentVolumeClaim(md))
		volMnts = append(volMnts, corev1.VolumeMount{
			Name:      persistentVolumeClaimName(app, md),
			MountPath: md.Path,
		})
		vols = append(vols, corev1.Volume{
			Name: persistentVolumeClaimName(app, md),
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: persistentVolumeClaimName(app, md),
					ReadOnly:  md.ReadOnly,
				},
			},
		})
	}
	replicas := app.Spec.Replicas
	if app.Spec.DesiredState == corev1alpha1.ApplicationDesiredStateStopped {
		replicas = 0
	}
	switch app.Spec.Type {
	case corev1alpha1.WorkloadTypeDeployment:
		dep := &appsv1.Deployment{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Deployment",
				APIVersion: appsv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: objectMeta,
			Spec: appsv1.DeploymentSpec{
				Replicas: conv.Ptr(replicas),
				Selector: &metav1.LabelSelector{
					MatchLabels: app.StableLabelSet(),
				},
				Template: podTemplate,
			},
		}
		if len(app.Spec.MountDirectories) > 0 {
			dep.Spec.Template.Spec.Volumes = vols
			for i := range dep.Spec.Template.Spec.InitContainers {
				dep.Spec.Template.Spec.InitContainers[i].VolumeMounts = append(dep.Spec.Template.Spec.InitContainers[i].VolumeMounts, volMnts...)
			}
			for i := range dep.Spec.Template.Spec.Containers {
				dep.Spec.Template.Spec.Containers[i].VolumeMounts = append(dep.Spec.Template.Spec.Containers[i].VolumeMounts, volMnts...)
			}
		}
		return dep
	case corev1alpha1.WorkloadTypeStatefulSet:
		sts := &appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StatefulSet",
				APIVersion: appsv1.SchemeGroupVersion.Identifier(),
			},
			ObjectMeta: objectMeta,
			Spec: appsv1.StatefulSetSpec{
				Replicas: conv.Ptr(replicas),
				Selector: &metav1.LabelSelector{
					MatchLabels: app.StableLabelSet(),
				},
				Template: podTemplate,
			},
		}
		if len(app.Spec.MountDirectories) > 0 {
			sts.Spec.Template.Spec.Volumes = vols
			// sts.Spec.VolumeClaimTemplates = volTmpls
			for i := range sts.Spec.Template.Spec.InitContainers {
				sts.Spec.Template.Spec.InitContainers[i].VolumeMounts = append(sts.Spec.Template.Spec.InitContainers[i].VolumeMounts, volMnts...)
			}
			for i := range sts.Spec.Template.Spec.Containers {
				sts.Spec.Template.Spec.Containers[i].VolumeMounts = append(sts.Spec.Template.Spec.Containers[i].VolumeMounts, volMnts...)
			}
		}
		return sts
	case corev1alpha1.WorkloadTypeCronJob:
		cj := &batchv1.CronJob{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CronJob",
				APIVersion: batchv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: objectMeta,
			Spec: batchv1.CronJobSpec{
				Schedule: app.Spec.CronSchedule,
				JobTemplate: batchv1.JobTemplateSpec{
					Spec: batchv1.JobSpec{
						Selector: &metav1.LabelSelector{
							MatchLabels: app.StableLabelSet(),
						},
						Template: podTemplate,
						Suspend:  conv.Ptr(replicas == 0),
					},
				},
			},
		}
		return cj
	case corev1alpha1.WorkloadTypeJob:
		j := &batchv1.Job{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Job",
				APIVersion: batchv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: objectMeta,
			Spec: batchv1.JobSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: app.StableLabelSet(),
				},
				Template: podTemplate,
				Suspend:  conv.Ptr(replicas == 0),
			},
		}
		return j
	default:
		return nil
	}
}

func constructService(app *corev1alpha1.Application, port *corev1alpha1.Port, owners ...metav1.OwnerReference) *corev1.Service {
	name := portName(app, port)
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       app.Namespace,
			Labels:          app.Labels,
			OwnerReferences: owners,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: app.StableLabelSet(),
			Ports: []corev1.ServicePort{
				{
					Name:       name,
					Protocol:   corev1.ProtocolTCP,
					Port:       port.Number,
					TargetPort: intstr.FromInt32(port.Target),
				},
			},
		},
	}
}

// func constructIngress(app *corev1alpha1.Application, port *corev1alpha1.Port, owners ...metav1.OwnerReference) []*networkingv1.Ingress {
// 	var result []*networkingv1.Ingress
// 	for _, gateway := range port.Gateways {
// 		var httpPath = gateway.Path
// 		if len(httpPath) == 0 {
// 			httpPath = "/"
// 		}
// 		name := gatewayName(app, port.Number, gateway)
//
// 		result = append(result, &networkingv1.Ingress{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:            name,
// 				Namespace:       app.Namespace,
// 				Labels:          app.Labels,
// 				OwnerReferences: owners,
// 			},
// 			Spec: networkingv1.IngressSpec{
// 				IngressClassName: conv.Ptr(gatewayClassName(app, gateway)),
// 				DefaultBackend: &networkingv1.IngressBackend{
// 					Service: &networkingv1.IngressServiceBackend{
// 						Name: name,
// 						Port: networkingv1.ServiceBackendPort{
// 							Number: port.Number,
// 						},
// 					},
// 				},
// 				Rules: []networkingv1.IngressRule{
// 					{
// 						Host: gateway.Host,
// 						IngressRuleValue: networkingv1.IngressRuleValue{
// 							HTTP: &networkingv1.HTTPIngressRuleValue{
// 								Paths: []networkingv1.HTTPIngressPath{
// 									{
// 										Path:     httpPath,
// 										PathType: conv.Ptr(networkingv1.PathTypePrefix),
// 										Backend: networkingv1.IngressBackend{
// 											Service: &networkingv1.IngressServiceBackend{
// 												Name: name,
// 												Port: networkingv1.ServiceBackendPort{
// 													Number: port.Number,
// 												},
// 											},
// 										},
// 									},
// 								},
// 							},
// 						},
// 					},
// 				},
// 			},
// 		})
// 	}
// 	return result
// }

// func constructGateway(app *corev1alpha1.Application, port *corev1alpha1.Port, owners ...metav1.OwnerReference) []*gatewayapi.Gateway {
// 	var result []*gatewayapi.Gateway
// 	for _, gateway := range port.Gateways {
// 		name := gatewayName(app, port.Number, gateway)
// 		result = append(result, &gatewayapi.Gateway{
// 			ObjectMeta: metav1.ObjectMeta{
// 				Name:            name,
// 				Namespace:       app.Namespace,
// 				Labels:          app.Labels,
// 				OwnerReferences: owners,
// 			},
// 			Spec: gatewayapi.GatewaySpec{
// 				GatewayClassName: gatewayapi.ObjectName(gatewayClassName(app, gateway)),
// 				Listeners: []gatewayapi.Listener{
// 					{
// 						Name:     gatewayapi.SectionName(name),
// 						Hostname: conv.Ptr(gatewayapi.Hostname(gateway.Host)),
// 						Port:     gatewayapi.PortNumber(port.Number),
// 						Protocol: gatewayapi.HTTPProtocolType,
// 					},
// 				},
// 			},
// 		})
// 	}
// 	return result
// }

func constructHTTPRoute(app *corev1alpha1.Application, port *corev1alpha1.Port, owners ...metav1.OwnerReference) []*gatewayapisv1.HTTPRoute {
	var result []*gatewayapisv1.HTTPRoute
	for _, gateway := range port.Gateways {
		// name := gatewayName(gateway.ClassName)

		var httpPath = gateway.Path
		if len(httpPath) == 0 {
			httpPath = "/"
		}
		result = append(result, &gatewayapisv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:            gateway.Name,
				Namespace:       app.Namespace,
				Labels:          app.Labels,
				OwnerReferences: owners,
			},
			Spec: gatewayapisv1.HTTPRouteSpec{
				CommonRouteSpec: gatewayapisv1.CommonRouteSpec{
					ParentRefs: []gatewayapisv1.ParentReference{
						{
							Name: gatewayapisv1.ObjectName(gateway.Name),
						},
					},
				},
				Hostnames: []gatewayapisv1.Hostname{
					gatewayapisv1.Hostname(gateway.Host),
				},
				Rules: []gatewayapisv1.HTTPRouteRule{
					{
						Matches: []gatewayapisv1.HTTPRouteMatch{
							{
								Path: &gatewayapisv1.HTTPPathMatch{
									Type:  conv.Ptr(gatewayapisv1.PathMatchPathPrefix),
									Value: conv.Ptr(httpPath),
								},
							},
						},
						BackendRefs: []gatewayapisv1.HTTPBackendRef{
							{
								BackendRef: gatewayapisv1.BackendRef{
									BackendObjectReference: gatewayapisv1.BackendObjectReference{
										Name: gatewayapisv1.ObjectName(portName(app, port)),
										Port: conv.Ptr(gatewayapisv1.PortNumber(port.Number)),
									},
								},
							},
						},
					},
				},
			},
		})
	}
	return result
}

func constructHorizontalPodAutoscaler(app *corev1alpha1.Application, workload client.Object, owners ...metav1.OwnerReference) *autoscalingv1.HorizontalPodAutoscaler {
	return &autoscalingv1.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:            workload.GetName(),
			Namespace:       app.Namespace,
			Labels:          app.Labels,
			OwnerReferences: owners,
		},
		Spec: autoscalingv1.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv1.CrossVersionObjectReference{
				Kind:       workloadTypeString(app),
				Name:       workload.GetName(),
				APIVersion: workload.GetObjectKind().GroupVersionKind().GroupVersion().String(),
			},
			MinReplicas:                    &app.Spec.Autoscaler.MinReplicas,
			MaxReplicas:                    app.Spec.Autoscaler.MaxReplicas,
			TargetCPUUtilizationPercentage: &app.Spec.Autoscaler.TargetCPUUtilizationPercentage,
		},
	}
}

func constructConfigMap(app *corev1alpha1.Application, mountFile *corev1alpha1.MountFile, owners ...metav1.OwnerReference) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:            configMapName(app, mountFile),
			Namespace:       app.Namespace,
			Labels:          app.Labels,
			OwnerReferences: owners,
		},
		Data: map[string]string{fileName(mountFile): mountFile.Content},
	}
	return configMap
}

func constructPersistentVolumeClaim(app *corev1alpha1.Application, md *corev1alpha1.MountDirectory, owners ...metav1.OwnerReference) *corev1.PersistentVolumeClaim {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:            persistentVolumeClaimName(app, md),
			Namespace:       app.Namespace,
			Labels:          app.StableLabelSet(),
			OwnerReferences: owners,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: md.StorageCapacity,
				},
			},
			StorageClassName: md.StorageClassName,
			VolumeMode:       conv.Ptr(corev1.PersistentVolumeFilesystem),
		},
	}
	return pvc
}

// func constructPersistentVolume(app *corev1alpha1.Application, md *corev1alpha1.MountDirectory) *corev1.PersistentVolume {
// 	var pvs corev1.PersistentVolumeSource
// 	if md.Local {
// 		pvs = corev1.PersistentVolumeSource{
// 			HostPath: &corev1.HostPathVolumeSource{
// 				Type: conv.Ptr(corev1.HostPathDirectoryOrCreate),
// 				Path: md.Path,
// 			},
// 		}
// 	}
//
// 	pv := &corev1.PersistentVolume{
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:   persistentVolumeName(app, md),
// 			Labels: app.Labels,
// 		},
// 		Spec: corev1.PersistentVolumeSpec{
// 			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
// 			Capacity:    corev1.ResourceList{corev1.ResourceStorage: md.StorageCapacity},
// 			ClaimRef: &corev1.ObjectReference{
// 				Namespace: app.Namespace,
// 				Name:      persistentVolumeClaimName(app, md),
// 			},
// 			PersistentVolumeSource: pvs,
// 			StorageClassName:       *md.StorageClassName,
// 			VolumeMode:             conv.Ptr(corev1.PersistentVolumeFilesystem),
// 		},
// 	}
// 	return pv
// }

// constructApplicationOwnerServiceAccount constructs a service account that owned the derived resources.
// This service account is used as a agent owned resource in worker cluster, when the application is added
// in the master cluster, the service account created in the worker cluster and all derived resources will be
// controlled by the service account. And when the application is deleted the service account will be deleted,
// and then clean up all the resources created by the application.
func constructApplicationOwnerServiceAccount(app *corev1alpha1.Application) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      applicationOwnerServiceAccountName(app.Name),
			Namespace: app.Namespace,
			Labels:    app.Labels,
		},
	}
}

func applicationOwnerServiceAccountName(appName string) string {
	return fmt.Sprintf("%s-application-owner", appName)
}

func configMapName(app *corev1alpha1.Application, mf *corev1alpha1.MountFile) string {
	return fmt.Sprintf("cm-%s-%s", app.Name, mf.Name)
}

func persistentVolumeClaimName(app *corev1alpha1.Application, md *corev1alpha1.MountDirectory) string {
	return fmt.Sprintf("pvc-%s-%s", app.Name, md.Name)
}

// func persistentVolumeName(app *corev1alpha1.Application, md *corev1alpha1.MountDirectory) string {
// 	return fmt.Sprintf("pv-%s-%s", app.Name, md.Name)
// }

func portName(app *corev1alpha1.Application, port *corev1alpha1.Port) string {
	return fmt.Sprintf("%s-port-%d", app.Name, port.Number)
}

// func gatewayName(app *corev1alpha1.Application, portNumber int32, gateway corev1alpha1.Gateway) string {
// 	return fmt.Sprintf("%s-gateway-%s-%d", app.Name, gateway.Name, portNumber)
// }

func gatewayClassName(app *corev1alpha1.Application, gateway corev1alpha1.Gateway) string {
	return gateway.ClassName
}

// func ingressClassName(app *corev1alpha1.Application, gateway corev1alpha1.Gateway) string {
// 	return kube.DefaultIngressClass()
// }

func constructContainer(sc *corev1alpha1.Sidecar) corev1.Container {
	c := corev1.Container{
		Name:      sc.Name,
		Image:     sc.Image,
		Resources: sc.Resources,
		Command:   sc.Command,
		Args:      sc.Args,
		Env:       sc.Env,
	}
	if sc.Privileged {
		c.SecurityContext = &corev1.SecurityContext{
			Privileged: &sc.Privileged,
		}
	}
	return c
}

func fileMode(file *corev1alpha1.MountFile) *int32 {
	if file.Mode == nil {
		return conv.Ptr(corev1.ConfigMapVolumeSourceDefaultMode)
	}
	return file.Mode
}

func fileName(file *corev1alpha1.MountFile) string {
	return path.Base(file.Path)
}

func workloadType(app *corev1alpha1.Application) corev1alpha1.WorkloadType {
	if app.Spec.Type == "" {
		return corev1alpha1.WorkloadTypeDeployment
	}
	return app.Spec.Type
}

func workloadTypeString(app *corev1alpha1.Application) string {
	return string(workloadType(app))
}
