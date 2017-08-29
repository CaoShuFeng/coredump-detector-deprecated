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

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const CoredumpResourcePlural = "coredumps"
const CoredumpQuotaResourcePlural = "coredumpquotas"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Coredump struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              CoredumpSpec   `json:"spec"`
	Status            CoredumpStatus `json:"status,omitempty"`
}

type CoredumpSpec struct {
	ContainerName string    `json:"containerName"`
	Pod           string    `json:"pod"`
	Uid           types.UID `json:"uid"`
	Pid           int       `json:"pid"`
	Filename      string    `json:"filename"`
	// Time is the kernel time when coredump happens.
	Time metav1.Time `json:"dumptime"`
	// Volume is the persistent volume, where coredump file is saved.
	Volume string `json:"volume"`
	// Size of coredump file
	Size *resource.Quantity `json:"size"`
}

type CoredumpStatus struct {
	State   CoredumpState `json:"state,omitempty"`
	Message string        `json:"message,omitempty"`
}

type CoredumpState string

const (
	// The initial state.
	CoredumpStateCreated CoredumpState = "Created"
	// The controller denied to save this coredump to persistent volume.
	// The coredump will deleted from host cache in this state.
	CoredumpStateDenied CoredumpState = "Denied"
	// The controller allowed to save this coredump to persistent volume.
	CoredumpStateStateAllowed CoredumpState = "Allowed"
	// The coredump has been saved to persistent volume successfully.
	CoredumpStateProcessed CoredumpState = "Saved"
	// Failed to save the coredump file for some reason.
	CoredumpStateFailed CoredumpState = "FailedToSave"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CoredumpList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []Coredump `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CoredumpQuota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              QuotaSpec   `json:"spec"`
	Status            QuotaStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type CoredumpQuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []CoredumpQuota `json:"items"`
}

type QuotaSpec struct {
	Hard *resource.Quantity `json:"hard"`
}

type QuotaStatus struct {
	Used *resource.Quantity `json:"used"`
	Hard *resource.Quantity `json:"hard"`
}
