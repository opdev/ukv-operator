/*
Copyright 2023.

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

// UKVSpec defines the desired state of UKV
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.dbType) || has(self.dbType)", message="DB Type value is required once set"
type UKVSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// DB Type defines the type of DB from a list of supported types. This is mandatory and immutable once set.
	// +kubebuilder:validation:Enum:="leveldb";"rocksdb";"udisk";"umem";
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="Value is immutable"
	DBType string `json:"dbType,omitempty"`

	// DB Config Map name is required.
	// +kubebuilder:validation:Required
	DBConfigMapName string `json:"dbConfigMapName,omitempty"`

	// DB Port to connect clients.
	DBServicePort int `json:"dbServicePort,omitempty"`

	// List of persistent volumes to be attached. Required by some DB Types.
	Volumes []Persistence `json:"volumes,omitempty"`
	// +kubebuilder:default:=1
	NumOfInstances int32 `json:"numOfInstances,omitempty"`

	// Memory limit for this UKV.
	// +kubebuilder:validation:Pattern:="^[1-9][0-9]{0,3}[KMG]{1}i"
	MemoryLimit string `json:"memoryLimit,omitempty"` // memory limit on pod.
	// Concurrency (cores) limit for this UKV.
	ConcurrencyLimit string `json:"concurrencyLimit,omitempty"`

	// Optionally define labels for an affinity to run UKV on specific cluster nodes.
	NodeAffinityLabels []NodeAffinityLabel `json:"nodeAffinityLabels,omitempty"`
}

// Defines a persistence used by the DB
type Persistence struct {
	// Size of the requested volume in Gi, Mi, Ti etc'
	// +kubebuilder:validation:Pattern:="^[1-9][0-9]{0,3}[KMGTPE]{1}i"
	Size string `json:"size,omitempty"`
	// Path to mount inside UKV container. This must correspond with the data path in config map.
	MountPath string `json:"mountPath,omitempty"`
	// +kubebuilder:validation:Enum:="ReadWriteOnce";"ReadWriteMany"
	AccessMode string `json:"accessMode,omitempty"`
}

// Defines affinity used by UKV. learn more in https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/
type NodeAffinityLabel struct {
	// Label key of the cluster nodes to match
	Label string `json:"label,omitempty"`
	// Label value of the cluster nodes to match
	Value string `json:"value,omitempty"`
	// Weight of this preference in the range 1-100
	Weight int32 `json:"weight,omitempty"`
}

// UKVStatus defines the observed state of UKV
type UKVStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	DeploymentStatus string `json:"deploymentStatus,omitempty"`
	DeploymentName   string `json:"deploymentName,omitempty"`
	ServiceStatus    string `json:"serviceStatus,omitempty"`
	ServiceUrl       string `json:"serviceUrl,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// UKV is the Schema for the ukvs API
type UKV struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UKVSpec   `json:"spec,omitempty"`
	Status UKVStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UKVList contains a list of UKV
type UKVList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UKV `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UKV{}, &UKVList{})
}
