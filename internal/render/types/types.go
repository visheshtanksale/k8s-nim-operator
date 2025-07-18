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

package types

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

// DaemonsetParams holds the parameters for rendering a Daemonset template.
type DaemonsetParams struct {
	Name               string
	Namespace          string
	Labels             map[string]string
	Annotations        map[string]string
	Replicas           int
	ContainerName      string
	Args               []string
	Command            []string
	Image              string
	ImagePullSecrets   []string
	ImagePullPolicy    string
	Volumes            []corev1.Volume
	VolumeMounts       []corev1.VolumeMount
	Env                []corev1.EnvVar
	Resources          corev1.ResourceRequirements
	NodeSelector       map[string]string
	Tolerations        []corev1.Toleration
	Affinity           *corev1.PodAffinity
	LivenessProbe      *corev1.Probe
	ReadinessProbe     *corev1.Probe
	StartupProbe       *corev1.Probe
	ServiceAccountName string
	NIMCachePVC        string
}

// DeploymentParams holds the parameters for rendering a Deployment template.
type DeploymentParams struct {
	Name               string
	Namespace          string
	Labels             map[string]string
	Annotations        map[string]string
	PodAnnotations     map[string]string
	SelectorLabels     map[string]string
	Replicas           int
	ContainerName      string
	Args               []string
	Command            []string
	Image              string
	ImagePullSecrets   []string
	ImagePullPolicy    string
	SchedulerName      string
	Volumes            []corev1.Volume
	VolumeMounts       []corev1.VolumeMount
	Env                []corev1.EnvVar
	Resources          *corev1.ResourceRequirements
	NodeSelector       map[string]string
	Tolerations        []corev1.Toleration
	Affinity           *corev1.PodAffinity
	LivenessProbe      *corev1.Probe
	ReadinessProbe     *corev1.Probe
	StartupProbe       *corev1.Probe
	ServiceAccountName string
	NIMCachePVC        string
	UserID             *int64
	GroupID            *int64
	RuntimeClassName   string
	OrchestratorType   string
	Ports              []corev1.ContainerPort
	InitContainers     []corev1.Container
	PodResourceClaims  []corev1.PodResourceClaim
}

// LeaderWorkerSetParams holds the parameters for rendering a LeaderWorkerSet template.
type LeaderWorkerSetParams struct {
	Name               string
	Namespace          string
	Labels             map[string]string
	Annotations        map[string]string
	PodAnnotations     map[string]string
	SelectorLabels     map[string]string
	Replicas           int
	Size               int
	ContainerName      string
	Args               []string
	Command            []string
	Image              string
	ImagePullSecrets   []string
	ImagePullPolicy    string
	SchedulerName      string
	WorkerVolumes      []corev1.Volume
	LeaderVolumes      []corev1.Volume
	WorkerVolumeMounts []corev1.VolumeMount
	LeaderVolumeMounts []corev1.VolumeMount
	WorkerEnvs         []corev1.EnvVar
	LeaderEnvs         []corev1.EnvVar
	Resources          *corev1.ResourceRequirements
	NodeSelector       map[string]string
	Tolerations        []corev1.Toleration
	Affinity           *corev1.PodAffinity
	LivenessProbe      *corev1.Probe
	ReadinessProbe     *corev1.Probe
	StartupProbe       *corev1.Probe
	ServiceAccountName string
	NIMCachePVC        string
	UserID             *int64
	GroupID            *int64
	RuntimeClassName   string
	OrchestratorType   string
	Ports              []corev1.ContainerPort
	InitContainers     []corev1.Container
	PodResourceClaims  []corev1.PodResourceClaim
}

// StatefulSetParams holds the parameters for rendering a StatefulSet template.
type StatefulSetParams struct {
	Name               string
	Namespace          string
	Labels             map[string]string
	Annotations        map[string]string
	SelectorLabels     map[string]string
	Replicas           int
	ContainerName      string
	ServiceName        string
	Image              string
	ImagePullSecrets   []string
	ImagePullPolicy    string
	Args               []string
	Command            []string
	Volumes            []corev1.Volume
	VolumeMounts       []corev1.VolumeMount
	Env                []corev1.EnvVar
	Resources          *corev1.ResourceRequirements
	NodeSelector       map[string]string
	Tolerations        []corev1.Toleration
	Affinity           *corev1.PodAffinity
	ServiceAccountName string
	LivenessProbe      *corev1.Probe
	ReadinessProbe     *corev1.Probe
	StartupProbe       *corev1.Probe
	NIMCachePVC        string
	RuntimeClassName   string
	OrchestratorType   string
	InitContainers     []corev1.Container
}

// ServiceParams holds the parameters for rendering a Service template.
type ServiceParams struct {
	Name      string
	Namespace string
	// Defines multiple ports for the service
	Ports          []corev1.ServicePort
	Type           string
	Labels         map[string]string
	Annotations    map[string]string
	SelectorLabels map[string]string
}

// ServiceAccountParams holds the parameters for rendering a ServiceAccount template.
type ServiceAccountParams struct {
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
}

// RoleParams holds the parameters for rendering a Role template.
type RoleParams struct {
	Name      string
	Namespace string
	Rules     []rbacv1.PolicyRule
}

// RoleBindingParams holds the parameters for rendering a RoleBinding template.
type RoleBindingParams struct {
	Name               string
	Namespace          string
	RoleName           string
	ServiceAccountName string
}

// SCCParams holds the parameters for rendering a SecurityContextConstraints template.
type SCCParams struct {
	Name               string
	ServiceAccountName string
}

// IngressParams holds the parameters for rendering an Ingress template.
type IngressParams struct {
	Enabled     bool
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
	Spec        networkingv1.IngressSpec
}

// IngressHost defines attributes for ingress host.
type IngressHost struct {
	Host  string
	Paths []IngressPath
}

// IngressPath defines attributes for ingress paths.
type IngressPath struct {
	Path        string
	PathType    networkingv1.PathType
	ServiceType string
}

// HPAParams holds the parameters for rendering a HorizontalPodAutoscaler template.
type HPAParams struct {
	Enabled     bool
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
	HPASpec     autoscalingv2.HorizontalPodAutoscalerSpec
}

// ServiceMonitorParams holds the parameters for rendering a ServiceMonitor template.
type ServiceMonitorParams struct {
	Enabled       bool
	Name          string
	Namespace     string
	Labels        map[string]string
	Annotations   map[string]string
	MatchLabels   map[string]string
	Port          int32
	Path          string
	Interval      int32
	ScrapeTimeout int32
	SMSpec        monitoringv1.ServiceMonitorSpec
}

// ConfigMapParams holds the parameters for rendering a ConfigMap template.
type ConfigMapParams struct {
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
	// Key-value pairs representing filenames and their content
	ConfigMapData map[string]string
}

// SecretParams holds the parameters for rendering a Secret template.
type SecretParams struct {
	Name        string
	Namespace   string
	Labels      map[string]string
	Annotations map[string]string
	// Key-value pairs representing filenames and their content
	SecretMapData map[string]string
}

// InferenceServiceParams holds the parameters for rendering a InferenceService template.
type InferenceServiceParams struct {
	Name               string
	Namespace          string
	Labels             map[string]string
	Annotations        map[string]string
	PodAnnotations     map[string]string
	SelectorLabels     map[string]string
	MinReplicas        *int32
	MaxReplicas        *int32
	ScaleMetricType    string
	ScaleMetric        string
	ScaleTarget        *int32
	ContainerName      string
	Args               []string
	Command            []string
	Image              string
	ImagePullSecrets   []string
	ImagePullPolicy    string
	SchedulerName      string
	Volumes            []corev1.Volume
	VolumeMounts       []corev1.VolumeMount
	Env                []corev1.EnvVar
	Resources          *corev1.ResourceRequirements
	NodeSelector       map[string]string
	Tolerations        []corev1.Toleration
	Affinity           *corev1.PodAffinity
	LivenessProbe      *corev1.Probe
	ReadinessProbe     *corev1.Probe
	StartupProbe       *corev1.Probe
	ServiceAccountName string
	NIMCachePVC        string
	UserID             *int64
	GroupID            *int64
	RuntimeClassName   string
	OrchestratorType   string
	Ports              []corev1.ContainerPort
	InitContainers     []corev1.Container
	PodResourceClaims  []corev1.PodResourceClaim
	DeploymentMode     string
}
