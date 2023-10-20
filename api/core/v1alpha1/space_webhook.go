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

package v1alpha1

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var spacelog = logf.Log.WithName("space-resource")

func (r *Space) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-core-ketches-io-v1alpha1-space,mutating=true,failurePolicy=fail,sideEffects=None,groups=core.ketches.io,resources=spaces,verbs=create;update,versions=v1alpha1,name=mspace.core.ketches.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Space{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Space) Default() {
	spacelog.Info("default", "name", r.Name)

	if r.Spec.DisplayName == "" {
		r.Spec.DisplayName = r.Name
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-core-ketches-io-v1alpha1-space,mutating=false,failurePolicy=fail,sideEffects=None,groups=core.ketches.io,resources=spaces,verbs=create;update;delete,versions=v1alpha1,name=vspace.core.ketches.io,admissionReviewVersions=v1

var _ webhook.Validator = &Space{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Space) ValidateCreate() (admission.Warnings, error) {
	spacelog.Info("validate create", "name", r.Name)

	if r.Spec.Cluster == "" {
		return nil, fmt.Errorf("cluster is required")
	}

	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Space) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	spacelog.Info("validate update", "name", r.Name)

	var allErrs field.ErrorList

	oldObj, _ := old.(*Space)

	if r.Spec.Cluster != oldObj.Spec.Cluster {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "cluster"), r.Spec.Cluster, "field is immutable"))
	}

	if len(allErrs) > 0 {
		return nil, allErrs.ToAggregate()
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Space) ValidateDelete() (admission.Warnings, error) {
	spacelog.Info("validate delete", "name", r.Name)

	if r.Status.ApplicationCount > 0 {
		return nil, fmt.Errorf("space still owns %d applications, can't be deleted", r.Status.ApplicationCount)
	}
	return nil, nil
}
