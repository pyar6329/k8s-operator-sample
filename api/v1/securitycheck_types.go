/*
Copyright 2025.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SecurityCheckSpec defines the desired state of SecurityCheck
type SecurityCheckSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// TargetNamespace specifies which namespace to monitor
	TargetNamespace string `json:"targetNamespace,omitempty"`

	// Rules defines security rules to check
	Rules []SecurityRule `json:"rules,omitempty"`
}

// SecurityRule defines a single security rule
type SecurityRule struct {
	// Name of the rule
	Name string `json:"name"`

	// Enabled indicates if this rule is active
	Enabled *bool `json:"enabled,omitempty"`
}

// SecurityCheckStatus defines the observed state of SecurityCheck.
type SecurityCheckStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// TotalPods is the number of pods checked
	TotalPods int32 `json:"totalPods,omitempty"`

	// ViolationsCount is the number of violations found
	ViolationsCount int32 `json:"violationsCount,omitempty"`

	// LastCheckTime is the timestamp of the last check
	LastCheckTime *metav1.Time `json:"lastCheckTime,omitempty"`

	// Conditions represent the latest available observations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Namespace",type="string",JSONPath=".spec.targetNamespace"
//+kubebuilder:printcolumn:name="Total Pods",type="integer",JSONPath=".status.totalPods"
//+kubebuilder:printcolumn:name="Violations",type="integer",JSONPath=".status.violationsCount"
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// SecurityCheck is the Schema for the securitychecks API
type SecurityCheck struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of SecurityCheck
	// +required
	Spec SecurityCheckSpec `json:"spec"`

	// status defines the observed state of SecurityCheck
	// +optional
	Status SecurityCheckStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// SecurityCheckList contains a list of SecurityCheck
type SecurityCheckList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecurityCheck `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecurityCheck{}, &SecurityCheckList{})
}
