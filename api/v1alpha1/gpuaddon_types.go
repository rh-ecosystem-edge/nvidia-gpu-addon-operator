/*
Copyright 2022.

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

// GPUAddonSpec defines the desired state of GPUAddon
type GPUAddonSpec struct {

	//+kubebuilder:default:=true
	// If enabled, addon will deploy the GPU console plugin.
	ConsolePluginEnabled bool `json:"console_plugin_enabled,omitempty"`
	// Optional NVAIE pullsecret
	NVAIEPullSecret string `json:"nvaie_pullsecret,omitempty"`
}

// GPUAddonStatus defines the observed state of GPUAddon
type GPUAddonStatus struct {
	// The state of the addon operator
	Phase GPUAddonState `json:"phase"`
	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:validation:Enum=Idle;Installing;Ready;Updating;Uninstalling
type GPUAddonState string

const (
	GPUAddonStateIdle         GPUAddonState = "Idle"
	GPUAddonStateInstalling   GPUAddonState = "Installing"
	GPUAddonStateReady        GPUAddonState = "Ready"
	GPUAddonStateUpdating     GPUAddonState = "Updating"
	GPUAddonStateUninstalling GPUAddonState = "Uninstalling"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.addon_state`
//+kubebuilder:printcolumn:name="Console Plugin",type=boolean,JSONPath=`.spec.console_plugin_enabled`
//+kubebuilder:printcolumn:name="NVAIE State",type=string,JSONPath=`.status.nvaie_state`

// GPUAddon is the Schema for the gpuaddons API
type GPUAddon struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GPUAddonSpec   `json:"spec,omitempty"`
	Status GPUAddonStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// GPUAddonList contains a list of GPUAddon
type GPUAddonList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GPUAddon `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GPUAddon{}, &GPUAddonList{})
}
