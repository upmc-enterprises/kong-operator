/*
Copyright (c) 2017, UPMC Enterprises
All rights reserved.
Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:
    * Redistributions of source code must retain the above copyright
      notice, this list of conditions and the following disclaimer.
    * Redistributions in binary form must reproduce the above copyright
      notice, this list of conditions and the following disclaimer in the
      documentation and/or other materials provided with the distribution.
    * Neither the name UPMC Enterprises nor the
      names of its contributors may be used to endorse or promote products
      derived from this software without specific prior written permission.
THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL UPMC ENTERPRISES BE LIABLE FOR ANY
DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
*/

package tpr

import (
	"encoding/json"

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/meta"
	"k8s.io/client-go/pkg/api/unversioned"
)

// KongCluster defines the cluster
type KongCluster struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	APIVersion string      `json:"apiVersion"`
	Type       string      `json:"type"`
	Kind       string      `json:"kind"`
	Spec       ClusterSpec `json:"spec"`
}

// KongClusterTPR defines the cluster
type KongClusterTPR struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	APIVersion string         `json:"apiVersion"`
	Type       string         `json:"type"`
	Kind       string         `json:"kind"`
	Spec       ClusterSpecTPR `json:"spec"`
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
	Apis map[string]*API `json:"apis"`
}

// ClusterSpecTPR defines cluster options from TPR
type ClusterSpecTPR struct {
	// Name is the cluster name
	Name string `json:"name"`

	// Replicas allows user to override the base image
	Replicas int32 `json:"replicas"`

	// BaseImage allows user to override the base image
	BaseImage string `json:"base-image"`

	// UseSamplePostgres defines if sample postgres db should be deployed
	UseSamplePostgres bool `json:"useSamplePostgres"`

	// Apis defines list of api's to configure in kong
	Apis []*API `json:"apis"`
}

// API defines a kong api
type API struct {
	// Name defines api name
	Name string `json:"name"`

	// Hosts defines lists of kong hosts
	Hosts []string `json:"hosts"`

	// UpstreamURL defines api upstream url
	UpstreamURL string `json:"upstream_url"`
}

// Required to satisfy Object interface
func (e *KongCluster) GetObjectKind() unversioned.ObjectKind {
	return &e.TypeMeta
}

// Required to satisfy ObjectMetaAccessor interface
func (e *KongCluster) GetObjectMeta() meta.Object {
	return &e.Metadata
}

func (e *KongCluster) UnmarshalJSON(data []byte) error {
	// *f = make(map[string]*foo) // required since map is not initialized
	tmp := KongClusterTPR{}
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	tmp2 := KongCluster{
		APIVersion: tmp.APIVersion,
		Kind:       tmp.Kind,
		Metadata:   tmp.Metadata,
		Type:       tmp.Type,
		TypeMeta:   tmp.TypeMeta,
		Spec: ClusterSpec{
			Name:              tmp.Spec.Name,
			Replicas:          tmp.Spec.Replicas,
			BaseImage:         tmp.Spec.BaseImage,
			UseSamplePostgres: tmp.Spec.UseSamplePostgres,
		},
	}

	tmp2.Spec.Apis = make(map[string]*API)
	for i := 0; i < len(tmp.Spec.Apis); i++ {
		tprAPI := tmp.Spec.Apis[i]
		tmp2.Spec.Apis[tprAPI.Name] = tprAPI
	}

	*e = tmp2

	return nil
}
