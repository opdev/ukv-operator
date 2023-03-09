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
type UKVSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	DBType                  string `json:"dbType,omitempty"` // UMem or UDisk
	DBContainerImage        string `json:"dbContainerImage,omitempty"`
	DBServiceName           string `json:"dbServiceName,omitempty"`
	DBServicePort           int    `json:"dbServicePort,omitempty"`
	PersistenceStorageClass string `json:"persistenceStorageClass,omitempty"` // For a UDisk type, provide the K8S Storage Class Name
	PersistenceSize         int    `json:"persistenceSize,omitempty"`
	DBConfigMapName         string `json:"dbConfigMapName,omitempty"`
	//+kubebuilder:default:=1
	NumOfInstances int32 `json:"numOfInstances,omitempty"`
}

// UKVStatus defines the observed state of UKV
type UKVStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	DeploymentStatus string `json:"deploymentStatus,omitempty"`
	DeploymentName   string `json:"deploymentName,omitempty"`
	ServiceStatus    string `json:"serviceStatus,omitempty"`
	ServiceName      string `json:"serviceName,omitempty"`
	ServicePort      int    `json:"servicePort,omitempty"`
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
