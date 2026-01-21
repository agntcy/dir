// Copyright AGNTCY Contributors (https://github.com/agntcy)
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	runtimev1 "github.com/agntcy/dir/runtime/api/runtime/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DiscoveredWorkloadStatus defines the observed state of DiscoveredWorkload.
type DiscoveredWorkloadStatus struct {
	// Phase indicates the current phase of the workload
	// +kubebuilder:validation:Enum=Discovered;Processing;Ready;Error
	// +optional
	Phase string `json:"phase,omitempty"`

	// LastSeen is the timestamp when the workload was last observed
	// +optional
	LastSeen *metav1.Time `json:"lastSeen,omitempty"`

	// Message provides additional information about the current state
	// +optional
	Message string `json:"message,omitempty"`

	// Conditions represent the latest available observations of the workload's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=dw;dws
// +kubebuilder:printcolumn:name="Runtime",type=string,JSONPath=`.spec.runtime`
// +kubebuilder:printcolumn:name="Name",type=string,JSONPath=`.spec.name`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.workload_type`
// +kubebuilder:printcolumn:name="Addresses",type=string,JSONPath=`.spec.addresses`
// +kubebuilder:printcolumn:name="Ports",type=string,JSONPath=`.spec.ports`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DiscoveredWorkload is the Schema for the discoveredworkloads API.
type DiscoveredWorkload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   *runtimev1.Workload      `json:"spec,omitempty"`
	Status DiscoveredWorkloadStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DiscoveredWorkloadList contains a list of DiscoveredWorkload.
type DiscoveredWorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DiscoveredWorkload `json:"items"`
}
