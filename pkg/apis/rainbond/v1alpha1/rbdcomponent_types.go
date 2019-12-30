package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RbdComponentType is the type of rainbond component
type RbdComponentType string

const (
	// RbdComponentTypeAPI rbd-api
	RbdComponentTypeAPI RbdComponentType = "rbd-api"
	// RbdComponentTypeWorker rbd-worker
	RbdComponentTypeWorker RbdComponentType = "rbd-worker"
)

// LogLevel -
type LogLevel string

const (
	// LogLevelDebug -
	LogLevelDebug RbdComponentType = "debug"
	// LogLevelInfo -
	LogLevelInfo RbdComponentType = "info"
	// LogLevelWarning -
	LogLevelWarning RbdComponentType = "warning"
	// LogLevelError -
	LogLevelError RbdComponentType = "error"
)

// RbdComponentSpec defines the desired state of RbdComponent
type RbdComponentSpec struct {
	// type of rainbond component
	Type RbdComponentType `json:"type"`
	// version of rainbond component
	Version  string   `json:"version"`
	LogLevel LogLevel `json:"logLevel,omitempty"`
}

// RbdComponentStatus defines the observed state of RbdComponent
type RbdComponentStatus struct {
	PodStatus PodStatus `json:"podStatus"`
	Message   string    `json:"message"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RbdComponent is the Schema for the rbdcomponents API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=rbdcomponents,scope=Namespaced
type RbdComponent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RbdComponentSpec   `json:"spec,omitempty"`
	Status RbdComponentStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// RbdComponentList contains a list of RbdComponent
type RbdComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RbdComponent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RbdComponent{}, &RbdComponentList{})
}

type PodStatus struct {
	// Ready are the component pods that are ready to serve requests
	// The pod names are the same as the component pod names
	Ready []string `json:"ready,omitempty"`
	// Unready are the components not ready to serve requests
	Unready []string `json:"unready,omitempty"`
	// Healthy are the component pods that pass the liveness.
	Healthy []string `json:"healthy,omitempty"`
	// Healthy are the component pods that pass the de liveness.
	UnHealthy []string `json:"unHealthy,omitempty"`
}
