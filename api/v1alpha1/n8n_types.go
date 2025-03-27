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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type Postgres struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Host string `json:"host,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Port uint32 `json:"port,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Database string `json:"database,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	User string `json:"user,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Password string `json:"password,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Ssl bool `json:"sst,omitempty"`
}

type Database struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Postgres Postgres `json:"postgres,omitempty"`
}

// N8nSpec defines the desired state of N8n
type N8nSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Database Database `json:"database,omitempty"`
}

// N8nStatus defines the observed state of N8n
type N8nStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// N8n is the Schema for the n8ns API
type N8n struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   N8nSpec   `json:"spec,omitempty"`
	Status N8nStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// N8nList contains a list of N8n
type N8nList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []N8n `json:"items"`
}

func init() {
	SchemeBuilder.Register(&N8n{}, &N8nList{})
}
