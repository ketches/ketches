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

	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1alpha1 "github.com/ketches/ketches/api/core/v1alpha1"
	"github.com/ketches/ketches/pkg/extension/helm"
	"github.com/ketches/ketches/pkg/kube"
	"github.com/ketches/ketches/pkg/kube/incluster"
	"github.com/ketches/ketches/pkg/kube/workercluster"
)

// ExtensionReconciler reconciles a Extension object
type ExtensionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.ketches.io,resources=extensions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.ketches.io,resources=extensions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.ketches.io,resources=extensions/finalizers,verbs=update

// Reconcile reconciles Extension objects
func (r *ExtensionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	extension := &corev1alpha1.Extension{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
	}
	if err := r.Get(ctx, req.NamespacedName, extension); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if extension.CheckOrSetRequiredLabels() {
		if err := kube.ApplyResource(ctx, r.Client, extension); err != nil {
			log.Error(err, "failed to update Extension labels")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if extension.Status.Phase == "" {
		extension.Status.Phase = corev1alpha1.ExtensionPhasePending
		if err := kube.UpdateResourceStatus(ctx, r.Client, extension); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		return ctrl.Result{}, nil
	}

	space := &corev1alpha1.Space{}
	err := r.Get(ctx, types.NamespacedName{Name: extension.Namespace}, space)
	if err != nil {
		extension.Status.Phase = corev1alpha1.ExtensionPhasePending
		extension.SetStatusCondition(corev1alpha1.ExtensionConditionTypeSpaceReady, err)
		if err := kube.UpdateResourceStatus(ctx, r.Client, extension); err != nil {
			log.Error(err, "unable to update Extension status")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, fmt.Errorf("space %s not found", extension.Namespace)
	}

	if space.Status.Phase != corev1alpha1.SpacePhaseReady {
		extension.SetStatusCondition(corev1alpha1.ExtensionConditionTypeSpaceReady, fmt.Errorf("space %s not ready", space.Name))
		err := kube.UpdateResourceStatus(ctx, r.Client, extension)
		return ctrl.Result{}, err
	}

	workerCluster, ok := incluster.Store().Clusterset().Cluster(space.Spec.Cluster)
	if !ok {
		log.Error(err, "unable to get worker cluster")
		return ctrl.Result{RequeueAfter: time.Second * 1}, nil
	}

	if extension.GetDeletionTimestamp() != nil {
		return r.onExtensionDeleted(ctx, extension, workerCluster)
	} else {
		if extension.CheckOrSetFinalizers() {
			err := kube.ApplyResource(ctx, r.Client, extension)
			if err != nil {
				log.Error(err, "unable to update Extension finalizers")
				return ctrl.Result{}, err
			}
		}
	}

	if extension.Spec.DesiredState == corev1alpha1.ExtensionDesiredStateInstalled {
		switch extension.Status.Phase {
		case corev1alpha1.ExtensionPhasePending, corev1alpha1.ExtensionPhaseUninstalled:
			r.installExtension(ctx, extension, workerCluster)
		case corev1alpha1.ExtensionPhaseInstalled:
			// TODO: check vision and upgrade if needed?
			return ctrl.Result{}, nil
		case corev1alpha1.ExtensionPhaseFailed:
			// TODO: implement
		}
	} else {
		if extension.Status.Phase == corev1alpha1.ExtensionPhaseUninstalled {
			return ctrl.Result{}, nil
		}
		r.uninstallExtension(ctx, extension, workerCluster)
	}

	err = kube.UpdateResourceStatus(ctx, r.Client, extension)
	if err != nil {
		log.Error(err, "failed to update Extension status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ExtensionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Extension{}).
		Complete(r)
}

func (r *ExtensionReconciler) installExtension(ctx context.Context, extension *corev1alpha1.Extension, workerCluster workercluster.Cluster) error {
	var err error
	switch extension.Spec.InstallType {
	case corev1alpha1.ExtensionInstallTypeHelm:
		err = r.installHelmExtension(ctx, extension, workerCluster)
	case corev1alpha1.ExtensionInstallTypeKubeApply:
		err = r.installKubeApplyExtension(ctx, extension)
	}
	return err
}

func (r *ExtensionReconciler) uninstallExtension(ctx context.Context, extension *corev1alpha1.Extension, workerCluster workercluster.Cluster) error {
	var err error
	switch extension.Spec.InstallType {
	case corev1alpha1.ExtensionInstallTypeHelm:
		err = r.uninstallExtension(ctx, extension, workerCluster)
	case corev1alpha1.ExtensionInstallTypeKubeApply:
		err = r.uninstallKubeApplyExtension(ctx, extension)
	}
	return err
}

func (r *ExtensionReconciler) installHelmExtension(ctx context.Context, extension *corev1alpha1.Extension, workerCluster workercluster.Cluster) error {
	rel, existed := helm.Status(workerCluster.RESTConfig(), extension.Spec.HelmInstallation.Name, extension.Spec.TargetNamespace)
	if existed {
		setHelmChartInstalledStatus(extension, rel, nil)
		return nil
	}

	hr := &corev1alpha1.HelmRepository{}
	err := r.Get(ctx, types.NamespacedName{Namespace: adminSpaceName(workerCluster.Name()), Name: extension.Spec.HelmInstallation.Repository}, hr)
	if err != nil || hr.Status.Phase != corev1alpha1.HelmRepositoryPhaseAdded {
		extension.Status.Phase = corev1alpha1.ExtensionPhaseFailed
		extension.SetStatusCondition(corev1alpha1.ExtensionConditionTypeHelmRepositoryAdded, fmt.Errorf("helm repository %s is not added", extension.Spec.HelmInstallation.Repository))
		return fmt.Errorf("helm repository %s is not added", extension.Spec.HelmInstallation.Repository)
	}

	rel, err = helm.Install(workerCluster.RESTConfig(), extension.Spec.HelmInstallation.Name, extension.Spec.HelmInstallation.Chart, extension.Spec.TargetNamespace, extension.Spec.HelmInstallation.Version, extension.Spec.HelmInstallation.KeyVals)
	if err != nil {
		setHelmChartInstalledStatus(extension, rel, err)
		return err
	}

	setHelmChartInstalledStatus(extension, rel, nil)
	return nil
}

func (r *ExtensionReconciler) uninstallHelmExtension(ctx context.Context, extension *corev1alpha1.Extension, workerCluster workercluster.Cluster) error {
	rel, existed := helm.Status(workerCluster.RESTConfig(), extension.Spec.HelmInstallation.Name, extension.Spec.TargetNamespace)
	if !existed {
		setHelmChartUninstalledStatus(extension, rel, nil)
		return nil
	}

	hr := &corev1alpha1.HelmRepository{}
	err := r.Get(ctx, types.NamespacedName{Namespace: adminSpaceName(workerCluster.Name()), Name: extension.Spec.HelmInstallation.Repository}, hr)
	if err != nil || hr.Status.Phase != corev1alpha1.HelmRepositoryPhaseAdded {
		extension.Status.Phase = corev1alpha1.ExtensionPhaseFailed
		extension.SetStatusCondition(corev1alpha1.ExtensionConditionTypeHelmRepositoryAdded, fmt.Errorf("helm repository %s is not added", extension.Spec.HelmInstallation.Repository))
		return fmt.Errorf("helm repository %s is not added", extension.Spec.HelmInstallation.Repository)
	}

	rel, err = helm.Uninstall(workerCluster.RESTConfig(), extension.Spec.HelmInstallation.Name, extension.Spec.TargetNamespace)
	if err != nil {
		setHelmChartUninstalledStatus(extension, rel, err)
		return err
	}

	setHelmChartUninstalledStatus(extension, rel, nil)
	return nil
}

func setHelmChartInstalledStatus(extension *corev1alpha1.Extension, r *release.Release, err error) {
	extension.Status.Phase = corev1alpha1.ExtensionPhaseInstalled
	if err != nil {
		extension.Status.Phase = corev1alpha1.ExtensionPhaseFailed
	}
	extension.SetStatusCondition(corev1alpha1.ExtensionConditionTypeHelmChartInstalled, err)
	extension.Status.HelmRelease = parseHelmRelease(r)
}

func setHelmChartUninstalledStatus(extension *corev1alpha1.Extension, r *release.Release, err error) {
	extension.Status.Phase = corev1alpha1.ExtensionPhaseUninstalled
	if err != nil {
		extension.Status.Phase = corev1alpha1.ExtensionPhaseFailed
	}
	extension.SetStatusCondition(corev1alpha1.ExtensionConditionTypeHelmChartUninstalled, err)
	extension.Status.HelmRelease = parseHelmRelease(r)
}

func parseHelmRelease(r *release.Release) *corev1alpha1.HelmRelease {
	if r == nil {
		return nil
	}

	result := &corev1alpha1.HelmRelease{}
	if r.Info != nil {
		var resourcesCount int
		for _, resources := range r.Info.Resources {
			resourcesCount += len(resources)
		}
		result.Resources = resourcesCount
		result.Revision = r.Version
		result.Status = r.Info.Status.String()
	}
	if r.Chart != nil {
		result.AppVersion = r.Chart.AppVersion()
		result.Chart = r.Chart.Metadata.Name
	}

	return result
}

func (r *ExtensionReconciler) installKubeApplyExtension(ctx context.Context, extension *corev1alpha1.Extension) error {
	// TODO: implement
	return nil
}

func (r *ExtensionReconciler) uninstallKubeApplyExtension(ctx context.Context, extension *corev1alpha1.Extension) error {
	// TODO: implement
	return nil
}

func (r *ExtensionReconciler) onExtensionDeleted(ctx context.Context, extension *corev1alpha1.Extension, workerCluster workercluster.Cluster) (ctrl.Result, error) {
	var result ctrl.Result
	var err error
	switch extension.Spec.InstallType {
	case corev1alpha1.ExtensionInstallTypeHelm:
		result, err = onHelmExtensionDeleted(ctx, extension, workerCluster)
	case corev1alpha1.ExtensionInstallTypeKubeApply:
		result, err = onKubeApplyExtensionDeleted(ctx, extension)
	}
	if err == nil {
		extension.SetFinalizers(nil)
		err = kube.ApplyResource(ctx, r.Client, extension)
	}
	return result, err
}

func onHelmExtensionDeleted(ctx context.Context, extension *corev1alpha1.Extension, workerCluster workercluster.Cluster) (ctrl.Result, error) {
	_, err := helm.Uninstall(workerCluster.RESTConfig(), extension.Spec.HelmInstallation.Name, extension.Spec.TargetNamespace)
	return ctrl.Result{Requeue: err != nil}, err
}

func onKubeApplyExtensionDeleted(ctx context.Context, extension *corev1alpha1.Extension) (ctrl.Result, error) {
	// TODO: implement
	return ctrl.Result{}, nil
}

// constructExtensionOwnerServiceAccount constructs a service account that owned the derived resources.
// This service account is used as a agent owned resource in worker cluster, when the extension is added
// in the master cluster, the service account created in the worker cluster and all derived resources will be
// controlled by the service account. And when the extension is deleted the service account will be deleted,
// and then clean up all the resources created by the extension.
func constructExtensionOwnerServiceAccount(extension *corev1alpha1.Extension) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      extensionOwnerServiceAccountName(extension.Name),
			Namespace: extension.Spec.TargetNamespace,
			Labels:    extension.Labels,
		},
	}
}

func extensionOwnerServiceAccountName(extensionName string) string {
	return fmt.Sprintf("%s-extension-owner", extensionName)
}
