/*
Copyright 2022.

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

	v1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Workflow is the Schema for the workflows API
//+kubebuilder:subresource:status
type Workflow struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkflowSpec   `json:"spec,omitempty"`
	Status WorkflowStatus `json:"status,omitempty"`
}

// InitStatus init status fields if nil
func (w *Workflow) InitStatus() {
	if w.Status.State == "" {
		w.Status.State = WorkflowStateInit
	}

	if w.Status.StepStatuses == nil {
		w.Status.StepStatuses = make(map[string]*StepStatus, len(w.Spec.Steps))
	}

	stepStatuses := w.Status.StepStatuses
	for stepIndex, stepSpec := range w.Spec.Steps {
		stepIndexName := GetIndexedName(stepIndex, stepSpec.Name)
		if stepStatuses[stepIndexName] == nil {
			stepStatuses[stepIndexName] = &StepStatus{
				Name:           stepSpec.Name,
				State:          StepStateInit,
				ActionStatuses: make(map[string]*ActionStatus, len(stepSpec.Actions)),
			}
		}

		actionStatuses := stepStatuses[stepIndexName].ActionStatuses
		for _, actionSpec := range stepSpec.Actions {
			if actionStatuses[actionSpec.Name] == nil {
				actionStatuses[actionSpec.Name] = &ActionStatus{
					Name:      actionSpec.Name,
					State:     ActionStateInit,
					JobStatus: v1.JobStatus{},
				}
			}
		}
	}
}

// GetIndexedName
func GetIndexedName(index int, name string) string {
	return fmt.Sprintf("%02d-%s", index, name)
}

//+kubebuilder:object:root=true

// WorkflowList contains a list of Workflow
type WorkflowList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workflow `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workflow{}, &WorkflowList{})
}
