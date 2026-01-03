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
	"k8s.io/apimachinery/pkg/runtime"
)

// BlueprintSpec defines the desired state of Blueprint
type BlueprintSpec struct {
	// Kind is the type of blueprint (must match a BlueprintDefinition)
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// Data contains the blueprint specification as arbitrary key-value pairs
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Data *runtime.RawExtension `json:"data,omitempty"`
}

// BlueprintStatus defines the observed state of Blueprint
type BlueprintStatus struct {
	// BlueprintID is the ID of the Blueprint in ColonyOS
	// +optional
	BlueprintID string `json:"blueprintId,omitempty"`

	// Generation is the current generation number in ColonyOS
	// +optional
	Generation int64 `json:"generation,omitempty"`

	// Synced indicates whether spec matches status in ColonyOS
	// +optional
	Synced bool `json:"synced,omitempty"`

	// CurrentData contains the current state as reported by the reconciler
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	CurrentData *runtime.RawExtension `json:"currentData,omitempty"`

	// LastReconcileTime is the last time the blueprint was reconciled
	// +optional
	LastReconcileTime *metav1.Time `json:"lastReconcileTime,omitempty"`

	// LastSyncTime is the last time the blueprint was synced with ColonyOS
	// +optional
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Conditions represent the current state of the Blueprint resource
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Kind",type=string,JSONPath=`.spec.kind`
// +kubebuilder:printcolumn:name="Synced",type=boolean,JSONPath=`.status.synced`
// +kubebuilder:printcolumn:name="Generation",type=integer,JSONPath=`.status.generation`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Blueprint is the Schema for the blueprints API
type Blueprint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BlueprintSpec   `json:"spec,omitempty"`
	Status BlueprintStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BlueprintList contains a list of Blueprint
type BlueprintList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Blueprint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Blueprint{}, &BlueprintList{})
}
