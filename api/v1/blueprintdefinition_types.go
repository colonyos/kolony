/*
Copyright 2026.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BlueprintDefinitionSpec defines the desired state of BlueprintDefinition
type BlueprintDefinitionSpec struct {
	// Kind is the type of resource this definition describes (e.g., "ExecutorDeployment", "GPUCluster")
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// Names contains the naming conventions for this kind
	// +optional
	Names BlueprintDefinitionNames `json:"names,omitempty"`

	// Handler specifies how blueprints of this kind are reconciled
	// +optional
	Handler BlueprintDefinitionHandler `json:"handler,omitempty"`
}

// BlueprintDefinitionNames contains naming conventions
type BlueprintDefinitionNames struct {
	// Singular is the singular name for the kind
	// +optional
	Singular string `json:"singular,omitempty"`

	// Plural is the plural name for the kind
	// +optional
	Plural string `json:"plural,omitempty"`
}

// BlueprintDefinitionHandler specifies reconciliation settings
type BlueprintDefinitionHandler struct {
	// ExecutorType is the type of executor that handles reconciliation
	// +kubebuilder:validation:Required
	ExecutorType string `json:"executorType"`

	// FunctionName is the function to call for reconciliation
	// +optional
	FunctionName string `json:"functionName,omitempty"`

	// ReconcileInterval is the interval in seconds between reconciliations
	// +optional
	// +kubebuilder:default=60
	ReconcileInterval int `json:"reconcileInterval,omitempty"`
}

// BlueprintDefinitionStatus defines the observed state of BlueprintDefinition
type BlueprintDefinitionStatus struct {
	// DefinitionID is the ID of the BlueprintDefinition in ColonyOS
	// +optional
	DefinitionID string `json:"definitionId,omitempty"`

	// Registered indicates whether the definition is registered in ColonyOS
	// +optional
	Registered bool `json:"registered,omitempty"`

	// LastSyncTime is the last time the definition was synced with ColonyOS
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Conditions represent the current state of the BlueprintDefinition resource
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Kind",type=string,JSONPath=`.spec.kind`
// +kubebuilder:printcolumn:name="ExecutorType",type=string,JSONPath=`.spec.handler.executorType`
// +kubebuilder:printcolumn:name="Registered",type=boolean,JSONPath=`.status.registered`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// BlueprintDefinition is the Schema for the blueprintdefinitions API
type BlueprintDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BlueprintDefinitionSpec   `json:"spec,omitempty"`
	Status BlueprintDefinitionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BlueprintDefinitionList contains a list of BlueprintDefinition
type BlueprintDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BlueprintDefinition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BlueprintDefinition{}, &BlueprintDefinitionList{})
}
