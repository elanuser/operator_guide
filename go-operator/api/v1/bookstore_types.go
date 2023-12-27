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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type BookApp struct {
	Repository      string             `json:"repository,omitempty"`
	Tag             string             `json:"tag,omitempty"`
	ImagePullPolicy corev1.PullPolicy  `json:"imagePullPolicy,omitempty"`
	Replicas        int32              `json:"replicas,omitempty"`
	Port            int32              `json:"port,omitempty"`
	TargetPort      int                `json:"targetPort,omitempty"`
	ServiceType     corev1.ServiceType `json:"serviceType,omitempty"`
}

type BookDB struct {
	Repository      string            `json:"repository,omitempty"`
	Tag             string            `json:"tag,omitempty"`
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	Replicas        int32             `json:"replicas,omitempty"`
	Port            int32             `json:"port,omitempty"`
	StorageClass    string            `json:"storageClass,omitempty"`
	DBSize          resource.Quantity `json:"dbSize,omitempty"`
}

// BookStoreSpec defines the desired state of BookStore
type BookStoreSpec struct {
	BookApp BookApp `json:"bookApp,omitempty"`
	BookDB  BookDB  `json:"bookDB,omitempty"`
}

// BookStoreStatus defines the observed state of BookStore
type BookStoreStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// BookStore is the Schema for the bookstores API
type BookStore struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BookStoreSpec   `json:"spec,omitempty"`
	Status BookStoreStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// BookStoreList contains a list of BookStore
type BookStoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BookStore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BookStore{}, &BookStoreList{})
}
