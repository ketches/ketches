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
	"time"

	"github.com/ketches/ketches/internal/global"
	"github.com/ketches/ketches/pkg/clusterset"
	"github.com/ketches/ketches/pkg/ketches"
	"github.com/ketches/ketches/pkg/kube"
	"github.com/ketches/ketches/util/conv"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/util/retry"
	gatewayapi "sigs.k8s.io/gateway-api/apis/v1beta1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.ketches.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.ketches.io,resources=clusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.ketches.io,resources=clusters/finalizers,verbs=update

// Reconcile reconciles Cluster objects
func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	cluster := &corev1alpha1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.Name,
		},
	}
	if err := r.Get(ctx, req.NamespacedName, cluster); err != nil {
		if errors.IsNotFound(err) {
			return r.onClusterDeleted(ctx, cluster)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if cluster.CheckOrSetRequiredLabels() {
		if err := kube.ApplyResource(ctx, r.Client, cluster); err != nil {
			log.Error(err, "failed to update Cluster labels")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if cluster.Status.Phase == "" {
		cluster.Status.Phase = corev1alpha1.ClusterPhaseConnecting
		if err := kube.UpdateResourceStatus(ctx, r.Client, cluster); err != nil {
			return ctrl.Result{}, err
		}
	}

	workerCluster, ok := ketches.Store().Clusterset().Cluster(cluster.Name)
	if !ok {
		return ctrl.Result{RequeueAfter: time.Second * 1}, nil
	}

	if err := r.applyClusterDerivedResources(ctx, cluster, workerCluster); err != nil {
		log.Error(err, "failed to apply derived resources")
		return ctrl.Result{}, err
	}

	r.completeClusterStatus(ctx, cluster, workerCluster)

	// update cluster status
	if err := kube.UpdateResourceStatus(ctx, r.Client, cluster); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Cluster{}).
		Owns(&corev1alpha1.Space{}).
		Complete(r)
}

func (r *ClusterReconciler) onClusterDeleted(ctx context.Context, cluster *corev1alpha1.Cluster) (ctrl.Result, error) {
	err := ketches.Client().CoreV1alpha1().Spaces().DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{LabelSelector: corev1alpha1.ClusterLabelKey + "=" + cluster.Name})
	if err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	return ctrl.Result{}, nil
}

func (r *ClusterReconciler) completeClusterStatus(ctx context.Context, cluster *corev1alpha1.Cluster, workerCluster clusterset.Cluster) {
	cluster.Status.Version = "unknown"

	err := r.pingWorkerCluster(ctx, workerCluster)
	cluster.SetStatusCondition(corev1alpha1.ClusterConditionTypePingPassed, err)
	if err != nil {
		cluster.Status.Phase = corev1alpha1.ClusterPhaseDisconnected
		return
	} else {
		cluster.Status.Phase = corev1alpha1.ClusterPhaseConnected
	}

	spaces := &corev1alpha1.SpaceList{}
	r.List(ctx, spaces, &client.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set{corev1alpha1.ClusterLabelKey: cluster.Name})})
	cluster.SetStatusSpaces(spaces)

	extensions := &corev1alpha1.ExtensionList{}
	r.List(ctx, extensions, &client.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set{corev1alpha1.ClusterLabelKey: cluster.Name})})
	cluster.SetStatusExtensions(extensions)

	cluster.Status.Server = workerCluster.RESTConfig().Host
	v, _ := workerCluster.KubeClientset().Discovery().ServerVersion()
	if v != nil {
		cluster.Status.Version = v.GitVersion
	}
}

func (r *ClusterReconciler) updateStatus(ctx context.Context, cluster *corev1alpha1.Cluster) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		err := r.Status().Update(ctx, cluster)
		if err != nil && errors.IsConflict(err) {
			current := cluster.DeepCopyObject().(client.Object)
			err = r.Get(ctx, client.ObjectKey{Name: cluster.Name}, current)
			if err != nil {
				return err
			}
			cluster.SetResourceVersion(current.GetResourceVersion())
		}
		return err
	})
}

func (r *ClusterReconciler) applyClusterDerivedResources(ctx context.Context, cluster *corev1alpha1.Cluster, workerCluster clusterset.Cluster) error {
	if workerCluster.GatewayAPIRuntimeClient() != nil {
		gateways := r.constructGateway(ctx, cluster, workerCluster.GatewayAPIRuntimeClient())
		for _, gateway := range gateways {
			err := kube.ApplyResource(ctx, workerCluster.GatewayAPIRuntimeClient(), gateway)
			cluster.SetStatusCondition(corev1alpha1.ClusterConditionTypeGatewayReady, err)
		}
	}
	return nil
}

func (r *ClusterReconciler) constructGateway(ctx context.Context, cluster *corev1alpha1.Cluster, gatewayClient client.Client) []*gatewayapi.Gateway {
	var result []*gatewayapi.Gateway
	gcl := &gatewayapi.GatewayList{}
	err := gatewayClient.List(ctx, gcl)
	if err != nil {
		return nil
	}

	for _, gc := range gcl.Items {
		name := gatewayName(gc.Name)

		var listeners []gatewayapi.Listener
		for _, domain := range cluster.Spec.WildCardDomains {
			listeners = append(listeners, gatewayapi.Listener{
				Name:     gatewayapi.SectionName(name),
				Hostname: conv.Ptr(gatewayapi.Hostname(domain)),
				Port:     gatewayapi.PortNumber(80),
				Protocol: gatewayapi.HTTPProtocolType,
				AllowedRoutes: &gatewayapi.AllowedRoutes{
					Namespaces: &gatewayapi.RouteNamespaces{
						From: conv.Ptr(gatewayapi.NamespacesFromAll),
					},
				},
			})
		}

		result = append(result, &gatewayapi.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: global.BuiltinNamespace,
				Labels:    cluster.Labels,
			},
			Spec: gatewayapi.GatewaySpec{
				GatewayClassName: gatewayapi.ObjectName(gc.Name),
				Listeners:        listeners,
			},
		})
	}
	return result
}

func gatewayName(gcName string) string {
	return "ketches-gateway-" + gcName
}

func (r *ClusterReconciler) pingWorkerCluster(ctx context.Context, workerCluster clusterset.Cluster) error {
	content, err := workerCluster.KubeClientset().Discovery().RESTClient().Get().AbsPath("/livez").DoRaw(ctx)
	if err != nil {
		return err
	}
	if string(content) != "ok" {
		return fmt.Errorf("ping response is not ok, got %s", string(content))
	}
	return nil
}