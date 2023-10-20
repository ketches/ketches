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

const (
	RequiredResourceLabel      = "ketches.io/owned=true"
	RequiredResourceLabelKey   = "ketches.io/owned"
	RequiredResourceLabelValue = "true"

	WorkflowLabelKey = "ketches.io/workflow"
)

func WorkflowRequiredLabelSet(workflow string) map[string]string {
	return map[string]string{
		RequiredResourceLabel: RequiredResourceLabelValue,
		WorkflowLabelKey:      workflow,
	}
}

// CheckOrSetRequiredLabels checks if the required labels are set on the Workflow object.
// If not, it sets the required labels and returns true.
func (workflow *Workflow) CheckOrSetRequiredLabels() bool {
	var result bool
	if workflow.Labels == nil {
		workflow.Labels = make(map[string]string)
	}
	if v, ok := workflow.Labels[RequiredResourceLabelKey]; !ok || v != RequiredResourceLabelValue {
		workflow.Labels[RequiredResourceLabelKey] = RequiredResourceLabelValue
		result = true
	}
	if v, ok := workflow.Labels[WorkflowLabelKey]; !ok || v != workflow.Name {
		workflow.Labels[WorkflowLabelKey] = workflow.Name
		result = true
	}
	return result
}
