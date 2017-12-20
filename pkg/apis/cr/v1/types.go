/*
Copyright 2017 The Kubernetes Authors.

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
	"github.com/upmc-enterprises/kong-operator/pkg/kong"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Cluster is a specification for an Cluster resource
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   ClusterSpec   `json:"spec"`
	Status ClusterStatus `json:"status,omitempty"`
}

// ClusterSpec defines cluster options
type ClusterSpec struct {
	// Name is the cluster name
	Name string `json:"name"`

	// Replicas allows user to override the base image
	Replicas int32 `json:"replicas"`

	// BaseImage allows user to override the base image
	BaseImage string `json:"base-image"`

	// UseSamplePostgres defines if sample postgres db should be deployed
	UseSamplePostgres bool `json:"useSamplePostgres"`

	// Apis defines list of api's to configure in kong
	Apis []kong.Data `json:"apis"`

	// Plugins defines the list of plugins to enable
	Plugins []kong.Plugin `json:"plugins"`

	// Consumers define the users
	Consumers []kong.ConsumerTPR `json:"consumers"`
}

// ClusterStatus is the status for an Example resource
type ClusterStatus struct {
	State   ClusterState `json:"state,omitempty"`
	Message string       `json:"message,omitempty"`
}

type ClusterState string

const (
	ClusterStateAdded ClusterState = "ADDED"
	ClusterStateDeleted ClusterState = "DELETED"
	ClusterStateModified ClusterState = "MODIFIED"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ClusterList is a list of Cluster resources
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Cluster `json:"items"`
}