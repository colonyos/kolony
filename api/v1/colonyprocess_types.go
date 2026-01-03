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

// ProcessState represents the state of a ColonyOS process
// +kubebuilder:validation:Enum=Pending;Waiting;Running;Success;Failed
type ProcessState string

const (
	ProcessStatePending ProcessState = "Pending"
	ProcessStateWaiting ProcessState = "Waiting"
	ProcessStateRunning ProcessState = "Running"
	ProcessStateSuccess ProcessState = "Success"
	ProcessStateFailed  ProcessState = "Failed"
)

// ColonyProcessSpec defines the desired state of ColonyProcess
type ColonyProcessSpec struct {
	// FuncName is the name of the function to execute
	// +kubebuilder:validation:Required
	FuncName string `json:"funcName"`

	// ExecutorType is the type of executor to run this process
	// +kubebuilder:validation:Required
	ExecutorType string `json:"executorType"`

	// Args are positional arguments passed to the function
	// +optional
	Args []string `json:"args,omitempty"`

	// Kwargs are keyword arguments passed to the function
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Kwargs *runtime.RawExtension `json:"kwargs,omitempty"`

	// Env are environment variables to set for the process
	// +optional
	Env map[string]string `json:"env,omitempty"`

	// Conditions specify requirements for executor selection
	// +optional
	Conditions *ProcessConditions `json:"conditions,omitempty"`

	// MaxWaitTime is the maximum time in seconds to wait in queue
	// +optional
	// +kubebuilder:default=600
	MaxWaitTime int `json:"maxWaitTime,omitempty"`

	// MaxExecTime is the maximum execution time in seconds
	// +optional
	// +kubebuilder:default=3600
	MaxExecTime int `json:"maxExecTime,omitempty"`

	// MaxRetries is the maximum number of retry attempts
	// +optional
	// +kubebuilder:default=0
	MaxRetries int `json:"maxRetries,omitempty"`
}

// ProcessConditions specify requirements for executor selection
type ProcessConditions struct {
	// Nodes is the number of nodes required
	// +optional
	Nodes int `json:"nodes,omitempty"`

	// ProcessesPerNode is the number of processes per node (required by container-executor)
	// +optional
	// +kubebuilder:default=1
	ProcessesPerNode int `json:"processesPerNode,omitempty"`

	// WallTime is the maximum wall clock time in seconds (required by container-executor)
	// +optional
	WallTime int64 `json:"wallTime,omitempty"`

	// CPU is the number of CPU cores required
	// +optional
	CPU int `json:"cpu,omitempty"`

	// Memory is the amount of memory required in MB
	// +optional
	Memory int `json:"memory,omitempty"`

	// GPU specifies GPU requirements
	// +optional
	GPU *GPURequirements `json:"gpu,omitempty"`
}

// GPURequirements specifies GPU requirements
type GPURequirements struct {
	// Count is the number of GPUs required
	// +optional
	Count int `json:"count,omitempty"`

	// Name is the GPU model name (e.g., "nvidia-a100")
	// +optional
	Name string `json:"name,omitempty"`

	// Memory is the GPU memory required in MB
	// +optional
	Memory int `json:"memory,omitempty"`
}

// ColonyProcessStatus defines the observed state of ColonyProcess
type ColonyProcessStatus struct {
	// ProcessID is the ID of the process in ColonyOS
	// +optional
	ProcessID string `json:"processId,omitempty"`

	// State is the current state of the process
	// +optional
	State ProcessState `json:"state,omitempty"`

	// AssignedExecutor is the name of the executor running this process
	// +optional
	AssignedExecutor string `json:"assignedExecutor,omitempty"`

	// SubmitTime is when the process was submitted to ColonyOS
	// +optional
	SubmitTime *metav1.Time `json:"submitTime,omitempty"`

	// StartTime is when the process started executing
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is when the process completed
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// Retries is the number of retry attempts made
	// +optional
	Retries int `json:"retries,omitempty"`

	// Output contains the process output
	// +optional
	Output []string `json:"output,omitempty"`

	// Errors contains any error messages
	// +optional
	Errors []string `json:"errors,omitempty"`

	// Conditions represent the current state of the ColonyProcess resource
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Function",type=string,JSONPath=`.spec.funcName`
// +kubebuilder:printcolumn:name="Executor",type=string,JSONPath=`.spec.executorType`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Assigned",type=string,JSONPath=`.status.assignedExecutor`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ColonyProcess is the Schema for the colonyprocesses API
type ColonyProcess struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ColonyProcessSpec   `json:"spec,omitempty"`
	Status ColonyProcessStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ColonyProcessList contains a list of ColonyProcess
type ColonyProcessList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ColonyProcess `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ColonyProcess{}, &ColonyProcessList{})
}
