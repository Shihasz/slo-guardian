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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// SLOPolicySpec defines the desired state of SLOPolicy
type SLOPolicySpec struct {
	// TargetDeployment is the name of the Deployment this policy watches
	TargetDeployment string `json:"targetDeployment"`

	// TargetURL is the HTTP endpoint to health-check
	TargetURL string `json:"targetURL"`

	// SLOTargetPercent is the required availability, e.g. 99.9
	SLOTargetPercent float64 `json:"sloTargetPercent"`

	// CheckIntervalSeconds is how often to probe the target
	CheckIntervalSeconds int `json:"checkIntervalSeconds"`

	// RemediationAction is what to do when the error budget is exhausted
	// +kubebuilder:validation:Enum=None;RestartDeployment;ScaleUp
	RemediationAction string `json:"remediationAction"`
}

// SLOPolicyStatus defines the observed state of SLOPolicy
type SLOPolicyStatus struct {
	// CurrentAvailabilityPercent is the measured availability over the tracked window
	CurrentAvailabilityPercent float64 `json:"currentAvailabilityPercent,omitempty"`

	// ErrorBudgetRemainingPercent is how much budget is left before breach
	ErrorBudgetRemainingPercent float64 `json:"errorBudgetRemainingPercent,omitempty"`

	// TotalChecks is the number of health checks performed so far
	TotalChecks int64 `json:"totalChecks,omitempty"`

	// FailedChecks is the number of failed health checks so far
	FailedChecks int64 `json:"failedChecks,omitempty"`

	// LastCheckTime is when the last health check ran
	LastCheckTime metav1.Time `json:"lastCheckTime,omitempty"`

	// LastRemediationTime is when remediation was last triggered
	LastRemediationTime *metav1.Time `json:"lastRemediationTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SLOPolicy is the Schema for the slopolicies API
type SLOPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of SLOPolicy
	// +required
	Spec SLOPolicySpec `json:"spec"`

	// status defines the observed state of SLOPolicy
	// +optional
	Status SLOPolicyStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// SLOPolicyList contains a list of SLOPolicy
type SLOPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []SLOPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &SLOPolicy{}, &SLOPolicyList{})
		return nil
	})
}
