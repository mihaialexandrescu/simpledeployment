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

package v0

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SimpleDeploymentSpec defines the desired state of SimpleDeployment
type SimpleDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Container image name used by the underlying Deployment resource.
	//+kubebuilder:validation:MinLength=0
	Image string `json:"image"`

	// Number of replicas used by the underlying Deployment resource.
	// It is a pointer so that users can set it to 0.
	//+kubebuilder:default=1
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:validation:Maximum=10
	//+optional
	Replicas *int32 `json:"replicas,omitempty"`

	// Info on where the application will be published.
	// Assumption: a single ingress rule will be used and there is already an Ingress Controller deployed.
	//+optional
	IngressInfo `json:"ingressInfo,omitempty"`
}

// This just assumes a single ingress rule.
type IngressInfo struct {
	// IngressControllerType currently only supports nginx.
	//+optional
	//+kubebuilder:default=nginx
	//+kubebuilder:validation:Enum=nginx
	IngressControllerType string `json:"ingressControllerType,omitempty"`

	// IngressClassName is used to select which IngressController will be used by the Ingress the operator defines.
	// Can be optional if there is a default Ingress Controller defined in the K8s cluster.
	//+optional
	//+kubebuilder:default=nginx
	IngressClassName string `json:"ingressClassName,omitempty"`

	// Port opened for this app on the Ingress Controller (prerequisite).
	// This is NOT the port specified in the ingress rule for the backend service.
	//+kubebuilder:validation:Minimum=0
	//+kubebuilder:default=443
	//+optional
	PublicPort int32 `json:"publicPort,omitempty"`

	// Host part of link. Goes into host field in ingress rule.
	//+kubebuilder:validation:MinLength=0
	//+optional
	Host string `json:"host,omitempty"`

	// Path part of link. Goes into path field in ingress rule.
	// Should start with "/".
	//+optional
	//+kubebuilder:default=/
	Path string `json:"path,omitempty"`

	// Rewrite-target function.
	//+optional
	RWTarget string `json:"rewriteTarget,omitempty"`
}

// SimpleDeploymentStatus defines the observed state of SimpleDeployment
type SimpleDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Deployment string `json:"deployment,omitempty"`
	URL        string `json:"url,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SimpleDeployment is the Schema for the simpledeployments API
//+kubebuilder:printcolumn:name="CfgImage",type=string,JSONPath=`.spec.image`
//+kubebuilder:printcolumn:name="CfgReplicas",type=integer,JSONPath=`.spec.replicas`
//+kubebuilder:printcolumn:name="Deployment",type=string,JSONPath=`.status.deployment`
//+kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.status.url`
//+kubebuilder:printcolumn:name="Age",type="date",JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=sd;sds
type SimpleDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SimpleDeploymentSpec   `json:"spec,omitempty"`
	Status SimpleDeploymentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SimpleDeploymentList contains a list of SimpleDeployment
type SimpleDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SimpleDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SimpleDeployment{}, &SimpleDeploymentList{})
}
