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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var helmrepositorylog = logf.Log.WithName("helmrepository-resource")

func (r *HelmRepository) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-core-ketches-io-v1alpha1-helmrepository,mutating=true,failurePolicy=fail,sideEffects=None,groups=core.ketches.io,resources=helmrepositories,verbs=create;update,versions=v1alpha1,name=mhelmrepository.core.ketches.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &HelmRepository{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *HelmRepository) Default() {
	helmrepositorylog.Info("default", "name", r.Name)
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-core-ketches-io-v1alpha1-helmrepository,mutating=false,failurePolicy=fail,sideEffects=None,groups=core.ketches.io,resources=helmrepositories,verbs=create;update,versions=v1alpha1,name=vhelmrepository.core.ketches.io,admissionReviewVersions=v1

var _ webhook.Validator = &HelmRepository{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *HelmRepository) ValidateCreate() (admission.Warnings, error) {
	helmrepositorylog.Info("validate create", "name", r.Name)

	var allErrs field.ErrorList

	if r.Spec.Url == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "url"), r.Spec.Url, "field is required"))
	}

	if len(allErrs) > 0 {
		return nil, allErrs.ToAggregate()
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *HelmRepository) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	helmrepositorylog.Info("validate update", "name", r.Name)

	var allErrs field.ErrorList

	oldObj, _ := old.(*HelmRepository)
	if r.Spec.Url != oldObj.Spec.Url {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "url"), r.Spec.Url, "field is immutable"))
	}

	if len(allErrs) > 0 {
		return nil, allErrs.ToAggregate()
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *HelmRepository) ValidateDelete() (admission.Warnings, error) {
	helmrepositorylog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
