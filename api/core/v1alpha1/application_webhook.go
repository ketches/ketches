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
var applicationlog = logf.Log.WithName("application-resource")

func (r *Application) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-core-ketches-io-v1alpha1-application,mutating=true,failurePolicy=fail,sideEffects=None,groups=core.ketches.io,resources=applications,verbs=create;update,versions=v1alpha1,name=mapplication.core.ketches.io,admissionReviewVersions=v1

var _ webhook.Defaulter = &Application{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Application) Default() {
	applicationlog.Info("default", "name", r.Name)

	if r.Spec.DesiredState == "" {
		r.Spec.DesiredState = DesiredStateRunning
	}

	if r.Spec.DisplayName == "" {
		r.Spec.DisplayName = r.Name
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-core-ketches-io-v1alpha1-application,mutating=false,failurePolicy=fail,sideEffects=None,groups=core.ketches.io,resources=applications,verbs=create;update,versions=v1alpha1,name=vapplication.core.ketches.io,admissionReviewVersions=v1

var _ webhook.Validator = &Application{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Application) ValidateCreate() (admission.Warnings, error) {
	applicationlog.Info("validate create", "name", r.Name)

	var allErrs field.ErrorList

	if r.Spec.Type == "" {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "type"), r.Spec.Type, "field is required"))
	} else if r.Spec.Type != WorkloadTypeDeployment && r.Spec.Type != WorkloadTypeStatefulSet && r.Spec.Type != WorkloadTypeJob && r.Spec.Type != WorkloadTypeCronJob {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "type"), r.Spec.Type, "field is not supported"))
	}

	if len(allErrs) > 0 {
		return nil, allErrs.ToAggregate()
	}
	return nil, nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Application) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	applicationlog.Info("validate update", "name", r.Name)

	var allErrs field.ErrorList

	oldObj, _ := old.(*Application)

	if r.Spec.Type != oldObj.Spec.Type {
		allErrs = append(allErrs, field.Invalid(field.NewPath("spec", "type"), r.Spec.Type, "field is immutable"))
	}

	if len(allErrs) > 0 {
		return nil, allErrs.ToAggregate()
	}
	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Application) ValidateDelete() (admission.Warnings, error) {
	applicationlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil, nil
}
