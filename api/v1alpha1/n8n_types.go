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
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Host string `json:"host"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port uint32 `json:"port"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Database string `json:"database"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	User string `json:"user"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Password string `json:"password"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Ssl bool `json:"ssl,omitempty"`
}

type Database struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Postgres Postgres `json:"postgres"`
}

// IngressConfig defines the configuration for Kubernetes Ingress
type IngressConfig struct {
	// Enable indicates whether to create an Ingress resource
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	Enable bool `json:"enable"`
	// IngressClassName is the name of the IngressClass to use
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	IngressClassName string `json:"ingressClassName,omitempty"`
	// TLS configuration for the Ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	TLS []IngressTLS `json:"tls,omitempty"`
}

// IngressTLS defines TLS configuration for Ingress
type IngressTLS struct {
	// Hosts are the hosts included in the TLS certificate
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Hosts []string `json:"hosts,omitempty"`
	// SecretName is the name of the secret containing TLS credentials
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	SecretName string `json:"secretName,omitempty"`
}

// HTTPRouteConfig defines the configuration for Gateway API HTTPRoute
// +kubebuilder:validation:XValidation:rule="!self.enable || has(self.gatewayRef)",message="gatewayRef is required when enable is true"
type HTTPRouteConfig struct {
	// Enable indicates whether to create an HTTPRoute resource
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	Enable bool `json:"enable"`
	// GatewayRef is the name of the Gateway to attach to
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	GatewayRef GatewayRef `json:"gatewayRef,omitempty"`
}

// GatewayRef defines the reference to a Gateway
type GatewayRef struct {
	// Name of the gateway
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Namespace of the gateway
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Namespace string `json:"namespace,omitempty"`
}

// PersistentStorageConfig defines the configuration for persistent storage
type PersistentStorageConfig struct {
	// Enable indicates whether to create a PVC for n8n data
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	Enable bool `json:"enable"`
	// StorageClassName is the name of the StorageClass to use
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	StorageClassName string `json:"storageClassName,omitempty"`
	// Size is the size of the volume (e.g., "10Gi")
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:default="10Gi"
	Size string `json:"size,omitempty"`
}

// Metrics defines the configuration for metrics
type MetricsConfig struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	Enable bool `json:"enable"`
}

// +kubebuilder:validation:XValidation:rule="!self.enable || has(self.url)",message="url is required when enable is true"
type HostnameConfig struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Required
	Enable bool `json:"enable"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +kubebuilder:validation:MinLength=1
	Url string `json:"url,omitempty"`
}

// N8nSpec defines the desired state of N8n
type N8nSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Database Database `json:"database"`

	// Ingress configuration for the N8n instance
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Ingress *IngressConfig `json:"ingress,omitempty"`

	// HTTPRoute configuration for the N8n instance
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	HTTPRoute *HTTPRouteConfig `json:"httpRoute,omitempty"`

	// PersistentStorage configuration for n8n data
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	PersistentStorage *PersistentStorageConfig `json:"persistentStorage,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Metrics *MetricsConfig `json:"metrics,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Hostname *HostnameConfig `json:"hostname,omitempty"`
}

// N8nStatus defines the observed state of N8n
type N8nStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:validation:XValidation:rule="!(has(self.spec.ingress) && has(self.spec.ingress.enable) && self.spec.ingress.enable && has(self.spec.httpRoute) && has(self.spec.httpRoute.enable) && self.spec.httpRoute.enable)",message="Ingress and HTTPRoute cannot both be enabled"
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
	metav1.ListMeta `json:"metadata"`
	Items           []N8n `json:"items"`
}

func init() {
	SchemeBuilder.Register(&N8n{}, &N8nList{})
}
