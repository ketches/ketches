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

package ci

import (
	"context"

	civ1alpha1 "github.com/ketches/ketches/api/ci/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// WorkflowReconciler reconciles a Workflow object
type WorkflowReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ci.ketches.io,resources=workflows,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ci.ketches.io,resources=workflows/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ci.ketches.io,resources=workflows/finalizers,verbs=update

// Reconcile reconciles Workflow objects
func (r *WorkflowReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Build instance
	var build civ1alpha1.Workflow
	if err := r.Get(ctx, req.NamespacedName, &build); err != nil {
		log.Error(err, "unable to fetch Build")
		return ctrl.Result{}, err
	}
	if build.Status.Phase == "" {
		build.Status.Phase = civ1alpha1.BuildPhasePending
		if err := r.Status().Update(ctx, &build); err != nil {
			log.Error(err, "unable to update Build")
			return ctrl.Result{}, err
		}
	}
	switch build.Status.Phase {
	case civ1alpha1.BuildPhasePending:
		// Create a new runner pod to build
		runnerPod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      build.Name,
				Namespace: build.Namespace,
				Labels: map[string]string{
					civ1alpha1.RequiredResourceLabelKey: civ1alpha1.RequiredResourceLabelValue,
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						// Image: "ketches/ketches-runner:latest",
						Image: "busybox",
						Name:  "ketches-runner",
						Command: []string{
							"/bin/sh",
							"-c",
							"echo 'Hello World'",
						},
						Env: []corev1.EnvVar{
							{
								Name:  "GIT_REPO",
								Value: build.Spec.GitRepo,
							}, {
								Name:  "GIT_BRANCH",
								Value: build.Spec.GitBranch,
							}, {
								Name:  "GIT_USERNAME",
								Value: build.Spec.User,
							}, {
								Name:  "GIT_TOKEN",
								Value: build.Spec.Token,
							},
						},
					},
				},
				ServiceAccountName: "ketches-runner",
				RestartPolicy:      corev1.RestartPolicyOnFailure,
			},
		}
		if err := r.Create(ctx, runnerPod); err != nil {
			log.Error(err, "unable to create runner pod")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WorkflowReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&civ1alpha1.Workflow{}).
		Complete(r)
}
