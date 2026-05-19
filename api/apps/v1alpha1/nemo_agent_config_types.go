/*
Copyright 2025.

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
	"fmt"
	"os"
	"path"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// NemoAgentConfigFileName is the key used for the rendered NeMo Agent Toolkit config.
	NemoAgentConfigFileName = "config.yml"
	// NemoAgentConfigVolumeName is the volume name used for the rendered config.
	NemoAgentConfigVolumeName = "agent-config"
	// NemoAgentConfigMountPath is the directory where the rendered config is mounted.
	NemoAgentConfigMountPath = "/config"
	// NemoAgentConfigContainerName is the default container name for the agent pod.
	NemoAgentConfigContainerName = "nemo-agent"

	// NemoAgentConfigConditionReady indicates that the NeMo Agent Toolkit config is ready.
	NemoAgentConfigConditionReady = "Ready"
	// NemoAgentConfigConditionFailed indicates that the NeMo Agent Toolkit config has failed validation or rendering.
	NemoAgentConfigConditionFailed = "Failed"

	// NemoAgentConfigStatusPending indicates that the config has not been processed.
	NemoAgentConfigStatusPending = "Pending"
	// NemoAgentConfigStatusReady indicates that the config is ready.
	NemoAgentConfigStatusReady = "Ready"
	// NemoAgentConfigStatusFailed indicates that the config failed validation or rendering.
	NemoAgentConfigStatusFailed = "Failed"
)

// NemoAgentConfigSpec defines a NeMo Agent Toolkit configuration.
type NemoAgentConfigSpec struct {
	// Functions configures named tools available to the agent.
	// +kubebuilder:validation:MinProperties=1
	Functions map[string]NemoAgentFunctionConfig `json:"functions"`

	// LLMs configures named large language model backends.
	// +kubebuilder:validation:MinProperties=1
	LLMs map[string]NemoAgentLLMConfig `json:"llms"`

	// Workflow configures the agent workflow.
	Workflow NemoAgentWorkflowConfig `json:"workflow"`

	// Pod configures the pod that runs the rendered NeMo Agent Toolkit config.
	Pod NemoAgentPodSpec `json:"pod"`
}

// NemoAgentPodSpec defines the pod that runs the NeMo Agent Toolkit config.
type NemoAgentPodSpec struct {
	// Name overrides the generated pod name. Defaults to the NemoAgentConfig name.
	Name string `json:"name,omitempty"`
	// ContainerName overrides the generated container name. Defaults to "nemo-agent".
	ContainerName string `json:"containerName,omitempty"`
	// Image is the container image that provides the nat CLI.
	Image Image `json:"image"`

	// Command overrides the default command. Defaults to ["nat"].
	Command []string `json:"command,omitempty"`
	// Args overrides the default args. Defaults to ["serve", "--config_file", "/config/config.yml"].
	Args []string `json:"args,omitempty"`
	// Env are additional environment variables for the agent pod.
	Env []corev1.EnvVar `json:"env,omitempty"`
	// RestartPolicy defines the pod restart policy. Defaults to Always.
	RestartPolicy corev1.RestartPolicy `json:"restartPolicy,omitempty"`
	// ConfigMount configures how the rendered config ConfigMap is mounted into the pod.
	ConfigMount NemoAgentConfigMount `json:"configMount,omitempty"`

	// Labels are additional labels applied to the agent pod.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations are additional annotations applied to the agent pod.
	Annotations        map[string]string            `json:"annotations,omitempty"`
	NodeSelector       map[string]string            `json:"nodeSelector,omitempty"`
	Tolerations        []corev1.Toleration          `json:"tolerations,omitempty"`
	Affinity           *corev1.Affinity             `json:"affinity,omitempty"`
	Resources          *corev1.ResourceRequirements `json:"resources,omitempty"`
	RuntimeClassName   string                       `json:"runtimeClassName,omitempty"`
	ServiceAccountName string                       `json:"serviceAccountName,omitempty"`
	UserID             *int64                       `json:"userID,omitempty"`
	GroupID            *int64                       `json:"groupID,omitempty"`
}

// NemoAgentConfigMount defines how the rendered config is projected into the pod.
type NemoAgentConfigMount struct {
	// Name is the volume and volumeMount name. Defaults to "agent-config".
	Name string `json:"name,omitempty"`
	// MountPath is the container mount path for the config volume. Defaults to "/config".
	MountPath string `json:"mountPath,omitempty"`
	// Key is the ConfigMap data key containing the rendered config. Defaults to "config.yml".
	Key string `json:"key,omitempty"`
	// Path is the filename used inside the mounted volume. Defaults to Key.
	Path string `json:"path,omitempty"`
	// ReadOnly controls whether the config volume mount is read-only. Defaults to false.
	ReadOnly bool `json:"readOnly,omitempty"`
}

// NemoAgentFunctionConfig defines a NeMo Agent Toolkit function/tool config.
//
// +kubebuilder:pruning:PreserveUnknownFields
type NemoAgentFunctionConfig struct {
	// Type is the NeMo Agent Toolkit function provider type.
	// +kubebuilder:validation:MinLength=1
	Type string `json:"_type"`

	// MaxResults limits the number of results returned by search-style tools.
	// +kubebuilder:validation:Minimum=1
	MaxResults *int32 `json:"max_results,omitempty"`
}

// NemoAgentLLMConfig defines a NeMo Agent Toolkit LLM config.
//
// +kubebuilder:pruning:PreserveUnknownFields
type NemoAgentLLMConfig struct {
	// Type is the NeMo Agent Toolkit LLM provider type.
	// +kubebuilder:validation:MinLength=1
	Type string `json:"_type"`

	// ModelName is the model served by the LLM provider.
	// +kubebuilder:validation:MinLength=1
	ModelName string `json:"model_name,omitempty"`

	// BaseURL is the OpenAI-compatible base URL for a NIM LLM endpoint.
	// +kubebuilder:validation:Pattern=`^https?:\/\/[^\s]+\/v1\/?$`
	// +kubebuilder:validation:Format=uri
	BaseURL string `json:"base_url,omitempty"`

	// Temperature controls sampling randomness.
	// +kubebuilder:validation:Minimum=0
	Temperature *float64 `json:"temperature,omitempty"`

	// ChatTemplateKwargs passes provider-specific chat template options.
	// +kubebuilder:pruning:PreserveUnknownFields
	ChatTemplateKwargs map[string]apiextensionsv1.JSON `json:"chat_template_kwargs,omitempty"`
}

// NemoAgentWorkflowConfig defines the workflow that drives the agent.
//
// +kubebuilder:pruning:PreserveUnknownFields
type NemoAgentWorkflowConfig struct {
	// Type is the NeMo Agent Toolkit workflow type.
	// +kubebuilder:validation:MinLength=1
	Type string `json:"_type"`

	// ToolNames lists named tools from Functions that the workflow may invoke.
	// +kubebuilder:validation:MinItems=1
	ToolNames []string `json:"tool_names,omitempty"`

	// LLMName references the named LLM from LLMs that the workflow should use.
	// +kubebuilder:validation:MinLength=1
	LLMName string `json:"llm_name,omitempty"`

	// Verbose enables verbose workflow logging.
	Verbose *bool `json:"verbose,omitempty"`

	// ParseAgentResponseMaxRetries configures response parsing retry attempts.
	// +kubebuilder:validation:Minimum=0
	ParseAgentResponseMaxRetries *int32 `json:"parse_agent_response_max_retries,omitempty"`
}

// NemoAgentConfigStatus defines the observed state of NemoAgentConfig.
type NemoAgentConfigStatus struct {
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	State      string             `json:"state,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.state`,priority=0
// +kubebuilder:printcolumn:name="Age",type="date",format="date-time",JSONPath=".metadata.creationTimestamp",priority=0

// NemoAgentConfig is the Schema for the nemoagentconfigs API.
type NemoAgentConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NemoAgentConfigSpec   `json:"spec,omitempty"`
	Status NemoAgentConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NemoAgentConfigList contains a list of NemoAgentConfig.
type NemoAgentConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NemoAgentConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NemoAgentConfig{}, &NemoAgentConfigList{})
}

// GetConfigMapName returns the name for the rendered config ConfigMap.
func (n *NemoAgentConfig) GetConfigMapName() string {
	return n.Name
}

// GetPodName returns the name for the agent pod.
func (n *NemoAgentConfig) GetPodName() string {
	if n.Spec.Pod.Name != "" {
		return n.Spec.Pod.Name
	}
	return n.Name
}

// GetImage returns the image for the agent pod.
func (n *NemoAgentConfig) GetImage() string {
	return fmt.Sprintf("%s:%s", n.Spec.Pod.Image.Repository, n.Spec.Pod.Image.Tag)
}

// GetCommand returns the command for the agent pod.
func (n *NemoAgentConfig) GetCommand() []string {
	if len(n.Spec.Pod.Command) > 0 {
		return n.Spec.Pod.Command
	}
	return []string{"nat"}
}

// GetContainerName returns the container name for the agent pod.
func (n *NemoAgentConfig) GetContainerName() string {
	if n.Spec.Pod.ContainerName != "" {
		return n.Spec.Pod.ContainerName
	}
	return NemoAgentConfigContainerName
}

// GetRestartPolicy returns the restart policy for the agent pod.
func (n *NemoAgentConfig) GetRestartPolicy() corev1.RestartPolicy {
	if n.Spec.Pod.RestartPolicy != "" {
		return n.Spec.Pod.RestartPolicy
	}
	return corev1.RestartPolicyAlways
}

// GetConfigVolumeName returns the config volume name.
func (n *NemoAgentConfig) GetConfigVolumeName() string {
	if n.Spec.Pod.ConfigMount.Name != "" {
		return n.Spec.Pod.ConfigMount.Name
	}
	return NemoAgentConfigVolumeName
}

// GetConfigMountPath returns the config volume mount path.
func (n *NemoAgentConfig) GetConfigMountPath() string {
	if n.Spec.Pod.ConfigMount.MountPath != "" {
		return n.Spec.Pod.ConfigMount.MountPath
	}
	return NemoAgentConfigMountPath
}

// GetConfigMapKey returns the ConfigMap data key for the rendered config.
func (n *NemoAgentConfig) GetConfigMapKey() string {
	if n.Spec.Pod.ConfigMount.Key != "" {
		return n.Spec.Pod.ConfigMount.Key
	}
	return NemoAgentConfigFileName
}

// GetConfigMountFileName returns the filename used in the mounted config volume.
func (n *NemoAgentConfig) GetConfigMountFileName() string {
	if n.Spec.Pod.ConfigMount.Path != "" {
		return n.Spec.Pod.ConfigMount.Path
	}
	return n.GetConfigMapKey()
}

// GetConfigFilePath returns the full path to the rendered config inside the container.
func (n *NemoAgentConfig) GetConfigFilePath() string {
	return path.Join(n.GetConfigMountPath(), n.GetConfigMountFileName())
}

// GetArgs returns the arguments for the agent pod.
func (n *NemoAgentConfig) GetArgs() []string {
	if len(n.Spec.Pod.Args) > 0 {
		return n.Spec.Pod.Args
	}
	return []string{"serve", "--config_file", n.GetConfigFilePath()}
}

// GetStandardSelectorLabels returns the labels used to identify the agent pod.
func (n *NemoAgentConfig) GetStandardSelectorLabels() map[string]string {
	return map[string]string{
		"app":                        n.Name,
		"app.kubernetes.io/name":     n.Name,
		"app.kubernetes.io/instance": n.Name,
	}
}

// GetStandardLabels returns the standard labels for NemoAgentConfig child resources.
func (n *NemoAgentConfig) GetStandardLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":             n.Name,
		"app.kubernetes.io/instance":         n.Name,
		"app.kubernetes.io/operator-version": os.Getenv("OPERATOR_VERSION"),
		"app.kubernetes.io/part-of":          "nemo-agent-toolkit",
		"app.kubernetes.io/managed-by":       "k8s-nim-operator",
	}
}
