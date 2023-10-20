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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	"github.com/ketches/ketches/pkg/clusterset"
	"github.com/ketches/ketches/pkg/extension/helm"
	"github.com/ketches/ketches/pkg/ketches"
	"github.com/ketches/ketches/pkg/kube"
)

// HelmRepositoryReconciler reconciles a HelmRepository object
type HelmRepositoryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.ketches.io,resources=helmrepositories,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.ketches.io,resources=helmrepositories/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.ketches.io,resources=helmrepositories/finalizers,verbs=update

// Reconcile reconciles HelmRepository objects
func (r *HelmRepositoryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	helmRepository := &corev1alpha1.HelmRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
	}
	if err := r.Get(ctx, req.NamespacedName, helmRepository); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if helmRepository.CheckOrSetRequiredLabels() {
		if err := kube.ApplyResource(ctx, r.Client, helmRepository); err != nil {
			log.Error(err, "failed to update HelmRepository labels")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if helmRepository.Status.Phase == "" {
		helmRepository.Status.Phase = corev1alpha1.HelmRepositoryPhasePending
		if err := kube.UpdateResourceStatus(ctx, r.Client, helmRepository); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	// update cluster status
	space := &corev1alpha1.Space{}
	err := r.Get(ctx, types.NamespacedName{Name: helmRepository.Namespace}, space)
	if err != nil {
		helmRepository.Status.Phase = corev1alpha1.HelmRepositoryPhasePending
		helmRepository.SetStatusCondition(corev1alpha1.HelmRepositoryConditionTypeClusterReady, err)
		if err := kube.UpdateResourceStatus(ctx, r.Client, helmRepository); err != nil {
			log.Error(err, "unable to update HelmRepository status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, fmt.Errorf("space %s not found", helmRepository.Namespace)
	}

	// check space status
	if space.Status.Phase != corev1alpha1.SpacePhaseReady {
		helmRepository.SetStatusCondition(corev1alpha1.HelmRepositoryConditionTypeSpaceReady, fmt.Errorf("space %s not ready", space.Name))
		err := kube.UpdateResourceStatus(ctx, r.Client, helmRepository)
		return ctrl.Result{}, err
	}

	workerCluster, ok := ketches.Store().Clusterset().Cluster(space.Spec.Cluster)
	if !ok {
		log.Error(err, "unable to get worker cluster")
		return ctrl.Result{RequeueAfter: time.Second * 1}, nil
	}
	if helmRepository.CheckOrSetRequiredLabels() {
		if err := kube.ApplyResource(ctx, r.Client, helmRepository); err != nil {
			log.Error(err, "failed to update HelmRepository labels")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if helmRepository.Status.Phase == "" {
		helmRepository.Status.Phase = corev1alpha1.HelmRepositoryPhasePending
		if err := kube.UpdateResourceStatus(ctx, r.Client, helmRepository); err != nil {
			log.Error(err, "unable to update HelmRepository status")
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	if helmRepository.GetDeletionTimestamp() != nil {
		return r.onHelmRepositoryDeleted(ctx, helmRepository, workerCluster)
	} else {
		if helmRepository.CheckOrSetFinalizers() {
			err := kube.ApplyResource(ctx, r.Client, helmRepository)
			if err != nil {
				log.Error(err, "unable to update HelmRepository finalizers")
				return ctrl.Result{}, err
			}
		}
	}

	switch helmRepository.Status.Phase {
	case corev1alpha1.HelmRepositoryPhasePending:
		addHelmRepository(ctx, helmRepository, workerCluster)
	case corev1alpha1.HelmRepositoryPhaseAdded:
		return ctrl.Result{}, nil
	case corev1alpha1.HelmRepositoryPhaseFailed:
		addHelmRepository(ctx, helmRepository, workerCluster)
	}

	err = kube.UpdateResourceStatus(ctx, r.Client, helmRepository)
	if err != nil {
		log.Error(err, "failed to update HelmRepository status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HelmRepositoryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.HelmRepository{}).
		Complete(r)
}

func (r *HelmRepositoryReconciler) onHelmRepositoryDeleted(ctx context.Context, helmRepository *corev1alpha1.HelmRepository, workerCluster clusterset.Cluster) (ctrl.Result, error) {
	err := helm.RepoRemove(helmRepository.Name)
	if err == nil {
		helmRepository.SetFinalizers(nil)
		err = kube.ApplyResource(ctx, r.Client, helmRepository)
	}
	return ctrl.Result{}, err
}

func addHelmRepository(ctx context.Context, helmRepository *corev1alpha1.HelmRepository, workerCluster clusterset.Cluster) error {
	err := helm.RepoAdd(helmRepository.Name, helmRepository.Spec.Url)
	setHelmRepositoryAddStatus(helmRepository, err)
	return err
}

func setHelmRepositoryAddStatus(helmRepository *corev1alpha1.HelmRepository, err error) {
	helmRepository.Status.Phase = corev1alpha1.HelmRepositoryPhaseAdded
	if err != nil {
		helmRepository.Status.Phase = corev1alpha1.HelmRepositoryPhaseFailed
	}
	helmRepository.SetStatusCondition(corev1alpha1.HelmRepositoryConditionTypeHelmRepositoryAdded, err)
}
