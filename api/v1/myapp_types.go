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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type KubernetesDeployment struct {
	ObjectMeta KubernetesObjectMeta `json:"metadata"`
	Spec       KubernetesDeploySpec `json:"spec"`
}

type KubernetesObjectMeta struct {
	Name string `json:"name"`
	// +optional
	Namespace string `json:"namespace"`
	// +optional
	Labels map[string]string `json:"labels"`
	// +optional
	Annotations map[string]string `json:"annotations"`
	// +optional
	Node string `json:"node"`
}

type KubernetesDeploySpec struct {
	Replicas int32                    `json:"replicas"`
	Selector metav1.LabelSelector     `json:"selector"`
	Template KubernetesDeployTemplate `json:"template"`
	// +optional
	ImagePullSecrets []KubernetesImagePullSecret `json:"imagePullSecrets"`
}

type KubernetesImagePullSecret struct {
	// +optional
	Name string `json:"name"`
}

type KubernetesDeployTemplate struct {
	Labels map[string]string `json:"labels"`
	// +optional
	Volumes    []corev1.Volume       `json:"volumes"`
	Containers []KubernetesContainer `json:"containers"`
	// +optional
	NodeSelector map[string]string `json:"nodeSelector"`
}

type KubernetesContainer struct {
	Name  string `json:"name"`
	Image string `json:"image"`
	// +optional
	Ports []corev1.ContainerPort `json:"ports"`
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts"`
	// +optional
	Stdin bool `json:"stdin"`
	// +optional
	StdinOnce bool `json:"stdinOnce"`
	// +optional
	TTY bool `json:"tty"`
	// +optional
	Envs []corev1.EnvVar `json:"envs"`
	// +optional
	Resources corev1.ResourceRequirements `json:"resources"`
	// +optional
	Args []string `json:"args"`
	// +optional
	Command []string `json:"command"`
	// +optional
	MountPath string `json:"mountPath"`
	// +optional
	Config string `json:"config"`
	// +optional
	ConfigMountPath string `json:"configMountPath"`
}

// MyAppSpec defines the desired state of MyApp
type MyAppSpec struct {
	DeploymentTemplate KubernetesDeployment `json:"deploymentTemplate"`
}

// MyAppStatus defines the observed state of MyApp
type MyAppStatus struct {
	// AvailableReplicas    int32                        `json:"availableReplicas"`
	// UpdatedReplicas      int32                        `json:"updatedReplicas"`
	// ReadyReplicas        int32                        `json:"readyReplicas"`
	// DeploymentConditions []appsv1.DeploymentCondition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MyApp is the Schema for the myapps API
type MyApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MyAppSpec   `json:"spec,omitempty"`
	Status MyAppStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MyAppList contains a list of MyApp
type MyAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MyApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MyApp{}, &MyAppList{})
}
