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
	"maps"
	"reflect"
	"time"

	civ1alpha1 "github.com/ketches/ketches/api/ci/v1alpha1"
	corev1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	"github.com/ketches/ketches/pkg/clusterset"
	"github.com/ketches/ketches/pkg/ketches"
	"github.com/ketches/ketches/pkg/kube"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// SpaceReconciler reconciles a Space object
type SpaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.ketches.io,resources=spaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.ketches.io,resources=spaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.ketches.io,resources=spaces/finalizers,verbs=update

// Reconcile reconciles Space object
func (r *SpaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx).WithValues("Space", req.NamespacedName)

	var space = &corev1alpha1.Space{
		ObjectMeta: metav1.ObjectMeta{
			Name: req.Name,
		},
	}

	if err := r.Get(ctx, req.NamespacedName, space); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if space.CheckOrSetRequiredLabels() {
		if err := kube.ApplyResource(ctx, r.Client, space); err != nil {
			log.Error(err, "failed to update Space labels")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if space.Status.Phase == "" {
		space.Status.Phase = corev1alpha1.SpacePhaseNotReady
		if err := kube.UpdateResourceStatus(ctx, r.Client, space); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	// update cluster status
	cluster := &corev1alpha1.Cluster{}
	err := r.Get(ctx, types.NamespacedName{Name: space.Spec.Cluster}, cluster)
	if err != nil {
		space.SetStatusCondition(corev1alpha1.SpaceConditionTypeClusterReady, fmt.Errorf("cluster %s not found", space.Spec.Cluster))
		return ctrl.Result{}, err
	}

	// check cluster status
	if cluster.Status.Phase != corev1alpha1.ClusterPhaseConnected {
		space.SetStatusCondition(corev1alpha1.SpaceConditionTypeClusterReady, fmt.Errorf("cluster %s is not connected", space.Spec.Cluster))
		err := kube.UpdateResourceStatus(ctx, r.Client, space)
		return ctrl.Result{}, err
	}

	workerCluster, ok := ketches.Store().Clusterset().Cluster(space.Spec.Cluster)
	if !ok {
		clusterNotFountErr := fmt.Errorf("cluster %s not found", space.Spec.Cluster)
		space.Status.Phase = corev1alpha1.SpacePhaseNotReady
		space.SetStatusCondition(corev1alpha1.SpaceConditionTypeClusterReady, clusterNotFountErr)
		if err := r.Status().Update(ctx, space); err != nil {
			log.Error(err, "unable to update Space status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, clusterNotFountErr
	}

	if space.GetDeletionTimestamp() != nil {
		return r.onSpaceDeleted(ctx, space, workerCluster)
	} else {
		if space.CheckOrSetFinalizers() {
			err := kube.ApplyResource(ctx, r.Client, space)
			if err != nil {
				log.Error(err, "unable to update Space finalizers")
				return ctrl.Result{}, err
			}
		}
	}

	if err := r.applyResources(ctx, workerCluster, space); err != nil {
		log.Error(err, "unable to apply resources")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}
	apps := &corev1alpha1.ApplicationList{}
	r.List(ctx, apps, &client.ListOptions{LabelSelector: labels.SelectorFromSet(labels.Set{corev1alpha1.SpaceLabelKey: space.Name})})
	space.SetStatusApplications(apps)

	// update space status
	space.Status.Phase = corev1alpha1.SpacePhaseReady
	if err = kube.UpdateResourceStatus(ctx, r.Client, space); err != nil {
		log.Error(err, "unable to update Space status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SpaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Space{}).
		Owns(&corev1.ResourceQuota{}).
		Owns(&corev1.LimitRange{}).
		Owns(&corev1alpha1.Application{}).
		Complete(r)
}

func (r *SpaceReconciler) onSpaceDeleted(ctx context.Context, space *corev1alpha1.Space, workerCluster clusterset.Cluster) (ctrl.Result, error) {
	var recycleNamespace = func(namespace string) error {
		nsInMaster := &corev1.Namespace{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: namespace}, nsInMaster)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}

		if nsInMaster.Labels[corev1alpha1.RequiredResourceLabelKey] == corev1alpha1.LabelTrueValue {
			if err := r.Client.Delete(ctx, nsInMaster); err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
			}
		} else {
			// delete resources owned by ketches space
			err := r.deleteSpaceOwnedResources(ctx, space)
			if err != nil {
				return err
			}
		}

		nsInWorker := &corev1.Namespace{}
		err = workerCluster.KubeRuntimeClient().Get(ctx, types.NamespacedName{Name: namespace}, nsInWorker)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil
			}
			return err
		}
		if nsInWorker.Labels[corev1alpha1.RequiredResourceLabelKey] == corev1alpha1.LabelTrueValue {
			if err := workerCluster.KubeRuntimeClient().Delete(ctx, nsInWorker); err != nil {
				if !errors.IsNotFound(err) {
					return err
				}
			}
		}

		return nil
	}

	err := recycleNamespace(space.Name)
	if err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Second}, err
	}

	space.SetFinalizers(nil)
	err = kube.ApplyResource(ctx, r.Client, space)
	return ctrl.Result{}, err
}

func (r *SpaceReconciler) deleteSpaceOwnedResources(ctx context.Context, space *corev1alpha1.Space) error {
	if err := r.Client.DeleteAllOf(ctx, &corev1alpha1.Application{}, client.InNamespace(space.Name)); err != nil {
		return err
	}
	if err := r.Client.DeleteAllOf(ctx, &corev1alpha1.ApplicationGroup{}, client.InNamespace(space.Name)); err != nil {
		return err
	}
	if err := r.Client.DeleteAllOf(ctx, &corev1alpha1.HelmRepository{}, client.InNamespace(space.Name)); err != nil {
		return err
	}
	if err := r.Client.DeleteAllOf(ctx, &corev1alpha1.Audit{}, client.InNamespace(space.Name)); err != nil {
		return err
	}
	if err := r.Client.DeleteAllOf(ctx, &civ1alpha1.Workflow{}, client.InNamespace(space.Name)); err != nil {
		return err
	}
	return nil
}

func (r *SpaceReconciler) updateStatus(ctx context.Context, space *corev1alpha1.Space) error {
	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		err := r.Status().Update(ctx, space)
		if err != nil && errors.IsConflict(err) {
			current := space.DeepCopyObject().(client.Object)
			err = r.Get(ctx, client.ObjectKey{Name: space.Name}, current)
			if err != nil {
				return err
			}
			space.SetResourceVersion(current.GetResourceVersion())
		}
		return err
	})
}

func (r *SpaceReconciler) applyResources(ctx context.Context, workerCluster clusterset.Cluster, space *corev1alpha1.Space) error {
	// apply namespace in master cluster
	namespace := r.constructNamespace(space)
	nsInMaster := &corev1.Namespace{}
	nsInWorker := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKeyFromObject(namespace), nsInMaster); err != nil {
		if errors.IsNotFound(err) {
			err = r.Create(ctx, namespace)
			space.SetStatusCondition(corev1alpha1.SpaceConditionTypeNamespaceReady, err)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if nsInMaster.Labels[corev1alpha1.RequiredResourceLabelKey] != corev1alpha1.LabelTrueValue {
			return fmt.Errorf("namespace %s in master cluster is not managed by ketches", namespace.Name)
		}
	}

	// apply namespace in worker cluster
	if err := workerCluster.KubeRuntimeClient().Get(ctx, client.ObjectKeyFromObject(namespace), nsInWorker); err != nil {
		if errors.IsNotFound(err) {
			err = workerCluster.KubeRuntimeClient().Create(ctx, namespace)
			space.SetStatusCondition(corev1alpha1.SpaceConditionTypeNamespaceReady, err)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if nsInWorker.Labels[corev1alpha1.RequiredResourceLabelKey] != corev1alpha1.LabelTrueValue {
			return fmt.Errorf("namespace %s in worker cluster is not managed by ketches", namespace.Name)
		}
	}

	// apply resource quota in worker cluster
	quota := r.constructResourceQuota(space)
	if space.Spec.ResourceQuota != nil {
		oldQuota := &corev1.ResourceQuota{}
		if err := workerCluster.KubeRuntimeClient().Get(ctx, client.ObjectKeyFromObject(quota), oldQuota); err != nil {
			if errors.IsNotFound(err) {
				err := workerCluster.KubeRuntimeClient().Create(ctx, quota)
				space.SetStatusCondition(corev1alpha1.SpaceConditionTypeResourceQuotaReady, err)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			// TODO: handle update conflict
			quotaChanged := func(old, new *corev1.ResourceQuota) bool {
				if !maps.Equal(old.Labels, new.Labels) {
					return true
				}

				if len(old.Spec.Hard) != len(new.Spec.Hard) {
					return true
				}
				for k, ov := range old.Spec.Hard {
					if nv, ok := new.Spec.Hard[k]; !ok || !ov.Equal(nv) {
						return true
					}
				}
				return false
			}
			if quotaChanged(oldQuota, quota) {
				err := workerCluster.KubeRuntimeClient().Update(ctx, quota)
				space.SetStatusCondition(corev1alpha1.SpaceConditionTypeResourceQuotaReady, err)
				if err != nil {
					return err
				}
			}
		}
	} else {
		// delete existing quota
		if err := workerCluster.KubeRuntimeClient().Delete(ctx, quota); err != nil {
			if !errors.IsNotFound(err) {
				space.SetStatusCondition(corev1alpha1.SpaceConditionTypeResourceQuotaReady, err)
				return err
			}
		}

		space.Status.Conditions = corev1alpha1.DeleteStatusCondition(space.Status.Conditions, corev1alpha1.SpaceConditionTypeResourceQuotaReady)
	}

	// apply limit range in worker cluster
	lr := r.constructLimitRange(space)
	if space.Spec.LimitRange != nil {
		old := &corev1.LimitRange{}
		if err := workerCluster.KubeRuntimeClient().Get(ctx, client.ObjectKeyFromObject(lr), old); err != nil {
			if errors.IsNotFound(err) {
				err := workerCluster.KubeRuntimeClient().Create(ctx, lr)
				space.SetStatusCondition(corev1alpha1.SpaceConditionTypeLimitRangeReady, err)
				if err != nil {
					return err
				}
			} else {
				return err
			}
		} else {
			// TODO: handle update conflict
			limitRangeChanged := func(old, new *corev1.LimitRange) bool {
				return !maps.Equal(old.Labels, new.Labels) || !reflect.DeepEqual(old.Spec.Limits, new.Spec.Limits)
			}
			if limitRangeChanged(old, lr) {
				err := workerCluster.KubeRuntimeClient().Update(ctx, quota)
				space.SetStatusCondition(corev1alpha1.SpaceConditionTypeLimitRangeReady, err)
				if err != nil {
					return err
				}
			}
		}
	} else {
		// delete existing limit range
		if err := workerCluster.KubeRuntimeClient().Delete(ctx, lr); err != nil {
			if !errors.IsNotFound(err) {
				space.SetStatusCondition(corev1alpha1.SpaceConditionTypeLimitRangeReady, err)
				return err
			}
		}
		space.Status.Conditions = corev1alpha1.DeleteStatusCondition(space.Status.Conditions, corev1alpha1.SpaceConditionTypeLimitRangeReady)
	}
	return nil
}

func (r *SpaceReconciler) constructNamespace(space *corev1alpha1.Space) *corev1.Namespace {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   space.Name,
			Labels: corev1alpha1.SpaceRequiredLabelSet(space.Name),
		},
	}
	return namespace
}

func (r *SpaceReconciler) constructResourceQuota(space *corev1alpha1.Space) *corev1.ResourceQuota {
	hard := r.quotaHard(space)

	quota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.quotaName(space.Name),
			Namespace: space.Name,
			Labels:    corev1alpha1.SpaceRequiredLabelSet(space.Name),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(space, corev1alpha1.SchemeGroupVersion.WithKind(space.Kind)),
			},
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: hard,
		},
	}

	return quota
}

func (r *SpaceReconciler) quotaName(space string) string {
	return space + "-quota"
}

func (r *SpaceReconciler) quotaHard(space *corev1alpha1.Space) corev1.ResourceList {
	if space.Spec.ResourceQuota == nil {
		return nil
	}

	requests := space.Spec.ResourceQuota.Requests
	limits := space.Spec.ResourceQuota.Limits

	hard := make(corev1.ResourceList)

	// TODO: validate resource value
	if requests != nil {
		hard["requests.cpu"] = requests[corev1.ResourceCPU]
		hard["requests.memory"] = requests[corev1.ResourceMemory]
	}
	if limits != nil {
		hard["limits.cpu"] = limits[corev1.ResourceCPU]
		hard["limits.memory"] = limits[corev1.ResourceMemory]
	}
	return hard
}

func (r *SpaceReconciler) constructLimitRange(space *corev1alpha1.Space) *corev1.LimitRange {
	lr := &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.limitRangeName(space.Name),
			Namespace: space.Name,
			Labels:    corev1alpha1.SpaceRequiredLabelSet(space.Name),
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(space, corev1alpha1.SchemeGroupVersion.WithKind(space.Kind)),
			},
		},
	}

	if space.Spec.LimitRange != nil {
		lr.Spec = corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypeContainer,
					Default: corev1.ResourceList{
						corev1.ResourceCPU:    space.Spec.LimitRange.CPU,
						corev1.ResourceMemory: space.Spec.LimitRange.Memory,
					},
				},
			},
		}
	}

	return lr
}

func (r *SpaceReconciler) limitRangeName(space string) string {
	return space + "-limit-range"
}
