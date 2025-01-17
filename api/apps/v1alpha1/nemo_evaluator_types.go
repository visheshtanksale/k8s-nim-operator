/*
Copyright 2024.

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
	"maps"
	"os"

	rendertypes "github.com/NVIDIA/k8s-nim-operator/internal/render/types"
	utils "github.com/NVIDIA/k8s-nim-operator/internal/utils"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// NemoEvaluatorConditionReady indicates that the NEMO EvaluatorService is ready.
	NemoEvaluatorConditionReady = "Ready"
	// NemoEvaluatorConditionFailed indicates that the NEMO EvaluatorService has failed.
	NemoEvaluatorConditionFailed = "Failed"

	// NemoEvaluatorStatusPending indicates that NEMO EvaluatorService is in pending state
	NemoEvaluatorStatusPending = "Pending"
	// NemoEvaluatorStatusNotReady indicates that NEMO EvaluatorService is not ready
	NemoEvaluatorStatusNotReady = "NotReady"
	// NemoEvaluatorStatusReady indicates that NEMO EvaluatorService is ready
	NemoEvaluatorStatusReady = "Ready"
	// NemoEvaluatorStatusFailed indicates that NEMO EvaluatorService has failed
	NemoEvaluatorStatusFailed = "Failed"
)

// NemoEvaluatorSpec defines the desired state of NemoEvaluator
type NemoEvaluatorSpec struct {
	Image   Image           `json:"image,omitempty"`
	Command []string        `json:"command,omitempty"`
	Args    []string        `json:"args,omitempty"`
	Env     []corev1.EnvVar `json:"env,omitempty"`
	// The name of an secret that contains authn for the NGC NIM service API
	Labels         map[string]string            `json:"labels,omitempty"`
	Annotations    map[string]string            `json:"annotations,omitempty"`
	NodeSelector   map[string]string            `json:"nodeSelector,omitempty"`
	Tolerations    []corev1.Toleration          `json:"tolerations,omitempty"`
	PodAffinity    *corev1.PodAffinity          `json:"podAffinity,omitempty"`
	Resources      *corev1.ResourceRequirements `json:"resources,omitempty"`
	Expose         Expose                       `json:"expose,omitempty"`
	LivenessProbe  Probe                        `json:"livenessProbe,omitempty"`
	ReadinessProbe Probe                        `json:"readinessProbe,omitempty"`
	StartupProbe   Probe                        `json:"startupProbe,omitempty"`
	Scale          Autoscaling                  `json:"scale,omitempty"`
	Metrics        Metrics                      `json:"metrics,omitempty"`
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:default:=1
	Replicas     int    `json:"replicas,omitempty"`
	UserID       *int64 `json:"userID,omitempty"`
	GroupID      *int64 `json:"groupID,omitempty"`
	RuntimeClass string `json:"runtimeClass,omitempty"`

	Mongodb       *Mongodb       `json:"mongodb,omitempty"`
	ArgoWorkFlows *ArgoWorkFlows `json:"argoWorkFlows,omitempty"`
	Milvus        *Milvus        `json:"milvus,omitempty"`
	DataStore     *DataStore     `json:"dataStore,omitempty"`
}

// NemoEvaluatorStatus defines the observed state of NemoEvaluator
type NemoEvaluatorStatus struct {
	Conditions        []metav1.Condition `json:"conditions,omitempty"`
	AvailableReplicas int32              `json:"availableReplicas,omitempty"`
	State             string             `json:"state,omitempty"`
}

// +genclient
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.state`,priority=0
// +kubebuilder:printcolumn:name="Age",type="date",format="date-time",JSONPath=".metadata.creationTimestamp",priority=0

// NemoEvaluator is the Schema for the NemoEvaluator API
type NemoEvaluator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NemoEvaluatorSpec   `json:"spec,omitempty"`
	Status NemoEvaluatorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// NemoEvaluatorList contains a list of NemoEvaluator
type NemoEvaluatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NemoEvaluator `json:"items"`
}

// GetStandardSelectorLabels returns the standard selector labels for the NemoEvaluator deployment
func (n *NemoEvaluator) GetStandardSelectorLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name": n.Name,
	}
}

// GetStandardLabels returns the standard set of labels for NemoEvaluator resources
func (n *NemoEvaluator) GetStandardLabels() map[string]string {
	return map[string]string{
		"app.kubernetes.io/name":             n.Name,
		"app.kubernetes.io/instance":         n.Name,
		"app.kubernetes.io/operator-version": os.Getenv("OPERATOR_VERSION"),
		"app.kubernetes.io/part-of":          "nemo-evaluator-service",
		"app.kubernetes.io/managed-by":       "k8s-nim-operator",
	}
}

// GetStandardEnv returns the standard set of env variables for the NemoEvaluator container
func (n *NemoEvaluator) GetStandardEnv() []corev1.EnvVar {
	// add standard env required for NIM service

	envVars := []corev1.EnvVar{
		{
			Name:  "NAMESPACE",
			Value: n.Namespace,
		},
		{
			Name: "HOST_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					APIVersion: "v1",
					FieldPath:  "status.hostIP",
				},
			},
		},
		{
			Name:  "EVALUATOR_HOST",
			Value: "0.0.0.0",
		},
		{
			Name:  "EVALUATOR_PORT",
			Value: "7331",
		},
		{
			Name:  "DATABASE_URI",
			Value: n.Spec.Mongodb.Endpoint,
		},
		{
			Name:  "DATABASE_NAME",
			Value: "evaluations",
		},
		{
			Name:  "ARGO_HOST",
			Value: n.Spec.ArgoWorkFlows.Endpoint,
		},
		{
			Name:  "MILVUS_URL",
			Value: n.Spec.Milvus.Endpoint,
		},
		{
			Name:  "SERVICE_ACCOUNT",
			Value: n.Spec.ArgoWorkFlows.ServiceAccount,
		},
		{
			Name:  "DATA_STORE_HOST",
			Value: n.Spec.DataStore.Endpoint,
		},
		{
			Name:  "DATABASE_USERNAME",
			Value: "root",
		},
		{
			Name: "DATABASE_PASSWORD",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					Key:                  "mongodb-root-password",
					LocalObjectReference: corev1.LocalObjectReference{Name: "myrelease-mongodb"},
				},
			},
		},
		{
			Name:  "EVAL_CONTAINER",
			Value: n.GetImage(),
		},
		{
			Name:  "EVAL_ENABLE_VALIDATION",
			Value: "True",
		},
		{
			Name:  "OTEL_TRACES_EXPORTER",
			Value: "none",
		},
		{
			Name:  "OTEL_METRICS_EXPORTER",
			Value: "none",
		},
		{
			Name:  "OTEL_LOGS_EXPORTER",
			Value: "none",
		},
		{
			Name:  "OTEL_PYTHON_LOGGING_AUTO_INSTRUMENTATION_ENABLED",
			Value: "false",
		},
		{
			Name:  "LOG_HANDLERS",
			Value: "console",
		},
		{
			Name:  "CONSOLE_LOG_LEVEL",
			Value: "INFO",
		},
		{
			Name:  "EVAL_LOG_LEVEL",
			Value: "INFO",
		},
	}

	return envVars
}

// GetStandardAnnotations returns default annotations to apply to the NemoEvaluator instance
func (n *NemoEvaluator) GetStandardAnnotations() map[string]string {
	standardAnnotations := map[string]string{
		"openshift.io/scc": "nonroot",
	}
	return standardAnnotations
}

// GetNemoEvaluatorAnnotations returns annotations to apply to the NemoEvaluator instance
func (n *NemoEvaluator) GetNemoEvaluatorAnnotations() map[string]string {
	standardAnnotations := n.GetStandardAnnotations()

	if n.Spec.Annotations != nil {
		return utils.MergeMaps(standardAnnotations, n.Spec.Annotations)
	}

	return standardAnnotations
}

// GetServiceLabels returns merged labels to apply to the NemoEvaluator instance
func (n *NemoEvaluator) GetServiceLabels() map[string]string {
	standardLabels := n.GetStandardLabels()

	if n.Spec.Labels != nil {
		return utils.MergeMaps(standardLabels, n.Spec.Labels)
	}
	return standardLabels
}

// GetSelectorLabels returns standard selector labels to apply to the NemoEvaluator instance
func (n *NemoEvaluator) GetSelectorLabels() map[string]string {
	// TODO: add custom ones
	return n.GetStandardSelectorLabels()
}

// GetNodeSelector returns node selector labels for the NemoEvaluator instance
func (n *NemoEvaluator) GetNodeSelector() map[string]string {
	return n.Spec.NodeSelector
}

// GetTolerations returns tolerations for the NemoEvaluator instance
func (n *NemoEvaluator) GetTolerations() []corev1.Toleration {
	return n.Spec.Tolerations
}

// GetPodAffinity returns pod affinity for the NemoEvaluator instance
func (n *NemoEvaluator) GetPodAffinity() *corev1.PodAffinity {
	return n.Spec.PodAffinity
}

// GetContainerName returns name of the container for NemoEvaluator deployment
func (n *NemoEvaluator) GetContainerName() string {
	return fmt.Sprintf("%s-ctr", n.Name)
}

// GetCommand return command to override for the NemoEvaluator container
func (n *NemoEvaluator) GetCommand() []string {
	return n.Spec.Command
}

// GetArgs return arguments for the NemoEvaluator container
func (n *NemoEvaluator) GetArgs() []string {
	return n.Spec.Args
}

// GetEnv returns merged slice of standard and user specified env variables
func (n *NemoEvaluator) GetEnv() []corev1.EnvVar {
	return utils.MergeEnvVars(n.GetStandardEnv(), n.Spec.Env)
}

// GetImage returns container image for the NemoEvaluator
func (n *NemoEvaluator) GetImage() string {
	return fmt.Sprintf("%s:%s", n.Spec.Image.Repository, n.Spec.Image.Tag)
}

// GetImagePullSecrets returns the image pull secrets for the NIM container
func (n *NemoEvaluator) GetImagePullSecrets() []string {
	return n.Spec.Image.PullSecrets
}

// GetImagePullPolicy returns the image pull policy for the NIM container
func (n *NemoEvaluator) GetImagePullPolicy() string {
	return n.Spec.Image.PullPolicy
}

// GetResources returns resources to allocate to the NemoEvaluator container
func (n *NemoEvaluator) GetResources() *corev1.ResourceRequirements {
	return n.Spec.Resources
}

// GetLivenessProbe returns liveness probe for the NemoEvaluator container
func (n *NemoEvaluator) GetLivenessProbe() *corev1.Probe {
	if n.Spec.LivenessProbe.Probe == nil {
		return n.GetDefaultLivenessProbe()
	}
	return n.Spec.LivenessProbe.Probe
}

// GetDefaultLivenessProbe returns the default liveness probe for the NemoEvaluator container
func (n *NemoEvaluator) GetDefaultLivenessProbe() *corev1.Probe {
	probe := corev1.Probe{
		FailureThreshold: 3,
		PeriodSeconds:    10,
		SuccessThreshold: 1,
		TimeoutSeconds:   1,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/health",
				Port: intstr.IntOrString{
					Type:   intstr.Type(1),
					StrVal: "http",
				},
				Scheme: "HTTP",
			},
		},
	}
	return &probe
}

// GetReadinessProbe returns readiness probe for the NemoEvaluator container
func (n *NemoEvaluator) GetReadinessProbe() *corev1.Probe {
	if n.Spec.ReadinessProbe.Probe == nil {
		return n.GetDefaultReadinessProbe()
	}
	return n.Spec.ReadinessProbe.Probe
}

// GetDefaultReadinessProbe returns the default readiness probe for the NemoEvaluator container
func (n *NemoEvaluator) GetDefaultReadinessProbe() *corev1.Probe {
	probe := corev1.Probe{
		FailureThreshold: 3,
		PeriodSeconds:    10,
		SuccessThreshold: 1,
		TimeoutSeconds:   1,
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/health",
				Port: intstr.IntOrString{
					Type:   intstr.Type(1),
					StrVal: "http",
				},
			},
		},
	}

	return &probe
}

// GetStartupProbe returns startup probe for the NemoEvaluator container
func (n *NemoEvaluator) GetStartupProbe() *corev1.Probe {
	return n.Spec.StartupProbe.Probe
}

// GetServiceAccountName returns service account name for the NemoEvaluator deployment
func (n *NemoEvaluator) GetServiceAccountName() string {
	return n.Name
}

// GetRuntimeClass return the runtime class name for the NemoEvaluator deployment
func (n *NemoEvaluator) GetRuntimeClass() string {
	return n.Spec.RuntimeClass
}

// GetHPA returns the HPA spec for the NemoEvaluator deployment
func (n *NemoEvaluator) GetHPA() HorizontalPodAutoscalerSpec {
	return n.Spec.Scale.HPA
}

// GetServiceMonitor returns the Service Monitor details for the NemoEvaluator deployment
func (n *NemoEvaluator) GetServiceMonitor() ServiceMonitor {
	return n.Spec.Metrics.ServiceMonitor
}

// GetReplicas returns replicas for the NemoEvaluator deployment
func (n *NemoEvaluator) GetReplicas() int {
	if n.IsAutoScalingEnabled() {
		return 0
	}
	return n.Spec.Replicas
}

// GetDeploymentKind returns the kind of deployment for NemoEvaluator
func (n *NemoEvaluator) GetDeploymentKind() string {
	return "Deployment"
}

// IsAutoScalingEnabled returns true if autoscaling is enabled for NemoEvaluator deployment
func (n *NemoEvaluator) IsAutoScalingEnabled() bool {
	return n.Spec.Scale.Enabled != nil && *n.Spec.Scale.Enabled
}

// IsIngressEnabled returns true if ingress is enabled for NemoEvaluator deployment
func (n *NemoEvaluator) IsIngressEnabled() bool {
	return n.Spec.Expose.Ingress.Enabled != nil && *n.Spec.Expose.Ingress.Enabled
}

// GetIngressSpec returns the Ingress spec NemoEvaluator deployment
func (n *NemoEvaluator) GetIngressSpec() networkingv1.IngressSpec {
	return n.Spec.Expose.Ingress.Spec
}

// IsServiceMonitorEnabled returns true if servicemonitor is enabled for NemoEvaluator deployment
func (n *NemoEvaluator) IsServiceMonitorEnabled() bool {
	return n.Spec.Metrics.Enabled != nil && *n.Spec.Metrics.Enabled
}

// GetServicePort returns the service port for the NemoEvaluator deployment
func (n *NemoEvaluator) GetServicePort() int32 {
	return n.Spec.Expose.Service.Port
}

// GetServiceType returns the service type for the NemoEvaluator deployment
func (n *NemoEvaluator) GetServiceType() string {
	return string(n.Spec.Expose.Service.Type)
}

// GetUserID returns the user ID for the NemoEvaluator deployment
func (n *NemoEvaluator) GetUserID() *int64 {
	return n.Spec.UserID

}

// GetGroupID returns the group ID for the NemoEvaluator deployment
func (n *NemoEvaluator) GetGroupID() *int64 {
	return n.Spec.GroupID

}

// GetServiceAccountParams return params to render ServiceAccount from templates
func (n *NemoEvaluator) GetServiceAccountParams() *rendertypes.ServiceAccountParams {
	params := &rendertypes.ServiceAccountParams{}

	// Set metadata
	params.Name = n.GetName()
	params.Namespace = n.GetNamespace()
	params.Labels = n.GetServiceLabels()
	params.Annotations = n.GetNemoEvaluatorAnnotations()
	return params
}

// GetDeploymentParams returns params to render Deployment from templates
func (n *NemoEvaluator) GetDeploymentParams() *rendertypes.DeploymentParams {
	params := &rendertypes.DeploymentParams{}

	// Set metadata
	params.Name = n.GetName()
	params.Namespace = n.GetNamespace()
	params.Labels = n.GetServiceLabels()
	params.Annotations = n.GetNemoEvaluatorAnnotations()

	// Set template spec
	if !n.IsAutoScalingEnabled() {
		params.Replicas = n.GetReplicas()
	}
	params.NodeSelector = n.GetNodeSelector()
	params.Tolerations = n.GetTolerations()
	params.Affinity = n.GetPodAffinity()
	params.ImagePullSecrets = n.GetImagePullSecrets()
	params.ImagePullPolicy = n.GetImagePullPolicy()

	// Set labels and selectors
	params.SelectorLabels = n.GetSelectorLabels()

	// Set container spec
	params.ContainerName = n.GetContainerName()
	params.Env = n.GetEnv()
	params.Args = n.GetArgs()
	params.Command = n.GetCommand()
	params.Resources = n.GetResources()
	params.Image = n.GetImage()

	// Set container probes
	if IsProbeEnabled(n.Spec.LivenessProbe) {
		params.LivenessProbe = n.GetLivenessProbe()
	}
	if IsProbeEnabled(n.Spec.ReadinessProbe) {
		params.ReadinessProbe = n.GetReadinessProbe()
	}
	if IsProbeEnabled(n.Spec.StartupProbe) {
		params.StartupProbe = n.GetStartupProbe()
	}
	params.UserID = n.GetUserID()
	params.GroupID = n.GetGroupID()

	// Set service account
	params.ServiceAccountName = n.GetServiceAccountName()

	// Set runtime class
	params.RuntimeClassName = n.GetRuntimeClass()

	params.Ports = []corev1.ContainerPort{{Name: "http", Protocol: corev1.ProtocolTCP, ContainerPort: 7331}}
	return params
}

// GetStatefulSetParams returns params to render StatefulSet from templates
func (n *NemoEvaluator) GetStatefulSetParams() *rendertypes.StatefulSetParams {

	params := &rendertypes.StatefulSetParams{}

	// Set metadata
	params.Name = n.GetName()
	params.Namespace = n.GetNamespace()
	params.Labels = n.GetServiceLabels()
	params.Annotations = n.GetNemoEvaluatorAnnotations()

	// Set template spec
	if !n.IsAutoScalingEnabled() {
		params.Replicas = n.GetReplicas()
	}
	params.ServiceName = n.GetName()
	params.NodeSelector = n.GetNodeSelector()
	params.Tolerations = n.GetTolerations()
	params.Affinity = n.GetPodAffinity()
	params.ImagePullSecrets = n.GetImagePullSecrets()
	params.ImagePullPolicy = n.GetImagePullPolicy()

	// Set labels and selectors
	params.SelectorLabels = n.GetSelectorLabels()

	// Set container spec
	params.ContainerName = n.GetContainerName()
	params.Env = n.GetEnv()
	params.Args = n.GetArgs()
	params.Command = n.GetCommand()
	params.Resources = n.GetResources()
	params.Image = n.GetImage()

	// Set container probes
	params.LivenessProbe = n.GetLivenessProbe()
	params.ReadinessProbe = n.GetReadinessProbe()
	params.StartupProbe = n.GetStartupProbe()

	// Set service account
	params.ServiceAccountName = n.GetServiceAccountName()

	// Set runtime class
	params.RuntimeClassName = n.GetRuntimeClass()
	return params
}

// GetServiceParams returns params to render Service from templates
func (n *NemoEvaluator) GetServiceParams() *rendertypes.ServiceParams {
	params := &rendertypes.ServiceParams{}

	// Set metadata
	params.Name = n.GetName()
	params.Namespace = n.GetNamespace()
	params.Labels = n.GetServiceLabels()
	params.Annotations = n.GetServiceAnnotations()

	// Set service selector labels
	params.SelectorLabels = n.GetSelectorLabels()

	// Set service type
	params.Type = "ClusterIP"

	// Set service ports
	params.Port = 7331
	params.TargetPort = 7331
	params.PortName = "http"
	return params
}

// GetIngressParams returns params to render Ingress from templates
func (n *NemoEvaluator) GetIngressParams() *rendertypes.IngressParams {
	params := &rendertypes.IngressParams{}

	params.Enabled = n.IsIngressEnabled()
	// Set metadata
	params.Name = n.GetName()
	params.Namespace = n.GetNamespace()
	params.Labels = n.GetServiceLabels()
	params.Annotations = n.GetIngressAnnotations()
	params.Spec = n.GetIngressSpec()
	return params
}

// GetRoleParams returns params to render Role from templates
func (n *NemoEvaluator) GetRoleParams() *rendertypes.RoleParams {
	params := &rendertypes.RoleParams{}

	// Set metadata
	params.Name = n.GetName()
	params.Namespace = n.GetNamespace()

	// Set rules to use SCC
	params.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"create"},
		},
	}

	return params
}

// GetRoleBindingParams returns params to render RoleBinding from templates
func (n *NemoEvaluator) GetRoleBindingParams() *rendertypes.RoleBindingParams {
	params := &rendertypes.RoleBindingParams{}

	// Set metadata
	params.Name = n.GetName()
	params.Namespace = n.GetNamespace()

	params.ServiceAccountName = n.GetServiceAccountName()
	params.RoleName = n.GetName()
	return params
}

// GetHPAParams returns params to render HPA from templates
func (n *NemoEvaluator) GetHPAParams() *rendertypes.HPAParams {
	params := &rendertypes.HPAParams{}

	params.Enabled = n.IsAutoScalingEnabled()

	// Set metadata
	params.Name = n.GetName()
	params.Namespace = n.GetNamespace()
	params.Labels = n.GetServiceLabels()
	params.Annotations = n.GetHPAAnnotations()

	// Set HPA spec
	hpa := n.GetHPA()
	hpaSpec := autoscalingv2.HorizontalPodAutoscalerSpec{
		ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
			Kind:       n.GetDeploymentKind(),
			Name:       n.GetName(),
			APIVersion: "apps/v1",
		},
		MinReplicas: hpa.MinReplicas,
		MaxReplicas: hpa.MaxReplicas,
		Metrics:     hpa.Metrics,
		Behavior:    hpa.Behavior,
	}
	params.HPASpec = hpaSpec
	return params
}

// GetSCCParams return params to render SCC from templates
func (n *NemoEvaluator) GetSCCParams() *rendertypes.SCCParams {
	params := &rendertypes.SCCParams{}
	// Set metadata
	params.Name = "nemo-evaluators-scc"

	params.ServiceAccountName = n.GetServiceAccountName()
	return params
}

// GetServiceMonitorParams return params to render Service Monitor from templates
func (n *NemoEvaluator) GetServiceMonitorParams() *rendertypes.ServiceMonitorParams {
	params := &rendertypes.ServiceMonitorParams{}
	serviceMonitor := n.GetServiceMonitor()
	params.Enabled = n.IsServiceMonitorEnabled()
	params.Name = n.GetName()
	params.Namespace = n.GetNamespace()
	svcLabels := n.GetServiceLabels()
	maps.Copy(svcLabels, serviceMonitor.AdditionalLabels)
	params.Labels = svcLabels
	params.Annotations = n.GetServiceMonitorAnnotations()

	// Set Service Monitor spec
	smSpec := monitoringv1.ServiceMonitorSpec{
		NamespaceSelector: monitoringv1.NamespaceSelector{MatchNames: []string{n.Namespace}},
		Selector:          metav1.LabelSelector{MatchLabels: n.GetServiceLabels()},
		Endpoints:         []monitoringv1.Endpoint{{Port: "service-port", ScrapeTimeout: serviceMonitor.ScrapeTimeout, Interval: serviceMonitor.Interval}},
	}
	params.SMSpec = smSpec
	return params
}

func (n *NemoEvaluator) GetIngressAnnotations() map[string]string {
	NemoEvaluatorAnnotations := n.GetNemoEvaluatorAnnotations()

	if n.Spec.Expose.Ingress.Annotations != nil {
		return utils.MergeMaps(NemoEvaluatorAnnotations, n.Spec.Expose.Ingress.Annotations)
	}
	return NemoEvaluatorAnnotations
}

func (n *NemoEvaluator) GetServiceAnnotations() map[string]string {
	NemoEvaluatorAnnotations := n.GetNemoEvaluatorAnnotations()

	if n.Spec.Expose.Service.Annotations != nil {
		return utils.MergeMaps(NemoEvaluatorAnnotations, n.Spec.Expose.Service.Annotations)
	}
	return NemoEvaluatorAnnotations
}

func (n *NemoEvaluator) GetHPAAnnotations() map[string]string {
	NemoEvaluatorAnnotations := n.GetNemoEvaluatorAnnotations()

	if n.Spec.Scale.Annotations != nil {
		return utils.MergeMaps(NemoEvaluatorAnnotations, n.Spec.Scale.Annotations)
	}
	return NemoEvaluatorAnnotations
}

func (n *NemoEvaluator) GetServiceMonitorAnnotations() map[string]string {
	NemoEvaluatorAnnotations := n.GetNemoEvaluatorAnnotations()

	if n.Spec.Metrics.ServiceMonitor.Annotations != nil {
		return utils.MergeMaps(NemoEvaluatorAnnotations, n.Spec.Metrics.ServiceMonitor.Annotations)
	}
	return NemoEvaluatorAnnotations
}

func init() {
	SchemeBuilder.Register(&NemoEvaluator{}, &NemoEvaluatorList{})
}
