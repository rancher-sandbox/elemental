/*
Copyright © 2022 SUSE LLC

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
	fleet "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"github.com/rancher/wrangler/pkg/genericcondition"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineRegistration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineRegistrationSpec   `json:"spec"`
	Status MachineRegistrationStatus `json:"status"`
}

type MachineRegistrationSpec struct {
	MachineName                 string            `json:"machineName,omitempty"`
	MachineInventoryLabels      map[string]string `json:"machineInventoryLabels,omitempty"`
	MachineInventoryAnnotations map[string]string `json:"machineInventoryAnnotations,omitempty"`
	CloudConfig                 *fleet.GenericMap `json:"cloudConfig,omitempty"`
}

type MachineRegistrationStatus struct {
	Conditions        []genericcondition.GenericCondition `json:"conditions,omitempty"`
	RegistrationURL   string                              `json:"registrationURL,omitempty"`
	RegistrationToken string                              `json:"registrationToken,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MachineInventory struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MachineInventorySpec   `json:"spec"`
	Status MachineInventoryStatus `json:"status"`
}

type MachineInventorySpec struct {
	TPMHash                string               `json:"tpmHash,omitempty"`
	SMBIOS                 *fleet.GenericMap    `json:"smbios,omitempty"`
	ClusterName            string               `json:"clusterName"`
	MachineTokenSecretName string               `json:"machineTokenSecretName,omitempty"`
	Config                 MachineRuntimeConfig `json:"config,omitempty"`
}

type MachineRuntimeConfig struct {
	Role            string            `json:"role"`
	NodeName        string            `json:"nodeName,omitempty"`
	Address         string            `json:"address,omitempty"`
	InternalAddress string            `json:"internalAddress,omitempty"`
	Taints          []corev1.Taint    `json:"taints,omitempty"`
	Labels          map[string]string `json:"labels"`
}

type MachineInventoryStatus struct {
	ClusterRegistrationTokenNamespace string `json:"clusterRegistrationTokenNamespace,omitempty"`
	ClusterRegistrationTokenName      string `json:"clusterRegistrationTokenName,omitempty"`
}
