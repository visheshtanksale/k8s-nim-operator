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

package controller

import (
	"context"
	"fmt"
	"reflect"
	"time"

	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	sigsyaml "sigs.k8s.io/yaml"

	appsv1alpha1 "github.com/NVIDIA/k8s-nim-operator/api/apps/v1alpha1"
	"github.com/NVIDIA/k8s-nim-operator/internal/conditions"
	"github.com/NVIDIA/k8s-nim-operator/internal/k8sutil"
	"github.com/NVIDIA/k8s-nim-operator/internal/utils"
)

const (
	// NemoAgentConfigFinalizer is the finalizer annotation.
	NemoAgentConfigFinalizer = "finalizer.nemoagentconfig.apps.nvidia.com"
	// NemoAgentConfigHashAnnotation is updated when config or pod spec changes.
	NemoAgentConfigHashAnnotation = "apps.nvidia.com/nemo-agent-config-hash"
)

// NemoAgentConfigReconciler reconciles a NemoAgentConfig object.
type NemoAgentConfigReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// NewNemoAgentConfigReconciler creates a new reconciler for NemoAgentConfig.
func NewNemoAgentConfigReconciler(client client.Client, scheme *runtime.Scheme) *NemoAgentConfigReconciler {
	return &NemoAgentConfigReconciler{
		Client: client,
		Scheme: scheme,
	}
}

// +kubebuilder:rbac:groups=apps.nvidia.com,resources=nemoagentconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.nvidia.com,resources=nemoagentconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.nvidia.com,resources=nemoagentconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=configmaps;pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;update;patch

// Reconcile keeps the rendered ConfigMap and agent Pod aligned with NemoAgentConfig.
func (r *NemoAgentConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	nemoAgentConfig := &appsv1alpha1.NemoAgentConfig{}
	if err := r.Get(ctx, req.NamespacedName, nemoAgentConfig); err != nil {
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "unable to fetch NemoAgentConfig", "name", req.NamespacedName)
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling", "NemoAgentConfig", nemoAgentConfig.Name)

	if nemoAgentConfig.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(nemoAgentConfig, NemoAgentConfigFinalizer) {
			if err := k8sutil.RetryUpdate(ctx, r.Client, nemoAgentConfig, func(obj client.Object) {
				controllerutil.AddFinalizer(obj, NemoAgentConfigFinalizer)
			}); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		if controllerutil.ContainsFinalizer(nemoAgentConfig, NemoAgentConfigFinalizer) {
			if err := k8sutil.RetryUpdate(ctx, r.Client, nemoAgentConfig, func(obj client.Object) {
				controllerutil.RemoveFinalizer(obj, NemoAgentConfigFinalizer)
			}); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	if result, err := r.reconcileNemoAgentConfig(ctx, nemoAgentConfig); err != nil {
		logger.Error(err, "error reconciling NemoAgentConfig", "name", nemoAgentConfig.Name)
		if statusErr := r.setStatus(ctx, nemoAgentConfig, appsv1alpha1.NemoAgentConfigStatusFailed, conditions.Failed, conditions.Failed, err.Error()); statusErr != nil {
			logger.Error(statusErr, "unable to update NemoAgentConfig status")
			return result, statusErr
		}
		r.getEventRecorder().Eventf(nemoAgentConfig, corev1.EventTypeWarning, conditions.Failed,
			"NemoAgentConfig %s failed, msg: %s", nemoAgentConfig.Name, err.Error())
		return result, err
	}

	return ctrl.Result{}, nil
}

func (r *NemoAgentConfigReconciler) reconcileNemoAgentConfig(ctx context.Context, nemoAgentConfig *appsv1alpha1.NemoAgentConfig) (ctrl.Result, error) {
	configYAML, err := r.renderConfigYAML(ctx, nemoAgentConfig)
	if err != nil {
		return ctrl.Result{}, err
	}

	configHash := utils.CalculateSHA256(configYAML + utils.DeepHashObject(nemoAgentConfig.Spec.Pod))
	if err := r.reconcileConfigMap(ctx, nemoAgentConfig, configYAML, configHash); err != nil {
		return ctrl.Result{}, err
	}

	recreating, err := r.reconcilePod(ctx, nemoAgentConfig, configHash)
	if err != nil {
		return ctrl.Result{}, err
	}
	if recreating {
		if err := r.setStatus(ctx, nemoAgentConfig, appsv1alpha1.NemoAgentConfigStatusPending, conditions.NotReady, "PodRecreating", "agent pod is being recreated"); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
	}

	return r.updateStatusFromPod(ctx, nemoAgentConfig)
}

func (r *NemoAgentConfigReconciler) renderConfigYAML(ctx context.Context, nemoAgentConfig *appsv1alpha1.NemoAgentConfig) (string, error) {
	raw := &unstructured.Unstructured{}
	raw.SetGroupVersionKind(appsv1alpha1.SchemeGroupVersion.WithKind("NemoAgentConfig"))
	if err := r.Get(ctx, types.NamespacedName{Name: nemoAgentConfig.Name, Namespace: nemoAgentConfig.Namespace}, raw); err != nil {
		return "", err
	}

	spec, ok, err := unstructured.NestedMap(raw.Object, "spec")
	if err != nil {
		return "", fmt.Errorf("failed to read spec: %w", err)
	}
	if !ok {
		return "", fmt.Errorf("spec is required")
	}

	config := map[string]interface{}{}
	for _, key := range []string{"functions", "llms", "workflow"} {
		value, ok := spec[key]
		if !ok {
			return "", fmt.Errorf("spec.%s is required", key)
		}
		config[key] = value
	}

	configYAML, err := sigsyaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("failed to marshal NeMo Agent Toolkit config: %w", err)
	}
	return string(configYAML), nil
}

func (r *NemoAgentConfigReconciler) reconcileConfigMap(ctx context.Context, nemoAgentConfig *appsv1alpha1.NemoAgentConfig, configYAML, configHash string) error {
	desired := r.desiredConfigMap(nemoAgentConfig, configYAML, configHash)
	if err := controllerutil.SetControllerReference(nemoAgentConfig, desired, r.Scheme); err != nil {
		return err
	}

	current := &corev1.ConfigMap{}
	key := types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}
	err := r.Get(ctx, key, current)
	if err != nil && !apiErrors.IsNotFound(err) {
		return err
	}
	if err == nil {
		if owned, _ := controllerutil.HasOwnerReference(current.OwnerReferences, nemoAgentConfig, r.Scheme); !owned {
			return fmt.Errorf("ConfigMap %s already exists and is not owned by NemoAgentConfig %s", key.String(), nemoAgentConfig.Name)
		}
	}

	return k8sutil.SyncResource(ctx, r.Client, current, desired)
}

func (r *NemoAgentConfigReconciler) desiredConfigMap(nemoAgentConfig *appsv1alpha1.NemoAgentConfig, configYAML, configHash string) *corev1.ConfigMap {
	annotations := map[string]string{
		NemoAgentConfigHashAnnotation: configHash,
	}
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:        nemoAgentConfig.GetConfigMapName(),
			Namespace:   nemoAgentConfig.Namespace,
			Labels:      nemoAgentConfig.GetStandardLabels(),
			Annotations: annotations,
		},
		Data: map[string]string{
			nemoAgentConfig.GetConfigMapKey(): configYAML,
		},
	}
}

func (r *NemoAgentConfigReconciler) reconcilePod(ctx context.Context, nemoAgentConfig *appsv1alpha1.NemoAgentConfig, configHash string) (bool, error) {
	desired := r.desiredPod(nemoAgentConfig, configHash)
	if err := controllerutil.SetControllerReference(nemoAgentConfig, desired, r.Scheme); err != nil {
		return false, err
	}

	current := &corev1.Pod{}
	key := types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}
	err := r.Get(ctx, key, current)
	if apiErrors.IsNotFound(err) {
		return false, r.Create(ctx, desired)
	}
	if err != nil {
		return false, err
	}
	if owned, _ := controllerutil.HasOwnerReference(current.OwnerReferences, nemoAgentConfig, r.Scheme); !owned {
		return false, fmt.Errorf("Pod %s already exists and is not owned by NemoAgentConfig %s", key.String(), nemoAgentConfig.Name)
	}
	if !current.DeletionTimestamp.IsZero() {
		return true, nil
	}
	if current.Annotations[NemoAgentConfigHashAnnotation] == configHash {
		return false, nil
	}
	if err := r.Delete(ctx, current); err != nil && !apiErrors.IsNotFound(err) {
		return false, err
	}
	return true, nil
}

func (r *NemoAgentConfigReconciler) desiredPod(nemoAgentConfig *appsv1alpha1.NemoAgentConfig, configHash string) *corev1.Pod {
	labels := utils.MergeMaps(nemoAgentConfig.GetStandardLabels(), nemoAgentConfig.GetStandardSelectorLabels())
	labels = utils.MergeMaps(labels, nemoAgentConfig.Spec.Pod.Labels)
	annotations := utils.MergeMaps(map[string]string{
		NemoAgentConfigHashAnnotation: configHash,
	}, nemoAgentConfig.Spec.Pod.Annotations)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        nemoAgentConfig.GetPodName(),
			Namespace:   nemoAgentConfig.Namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			RestartPolicy:      nemoAgentConfig.GetRestartPolicy(),
			ServiceAccountName: nemoAgentConfig.Spec.Pod.ServiceAccountName,
			NodeSelector:       nemoAgentConfig.Spec.Pod.NodeSelector,
			Tolerations:        nemoAgentConfig.Spec.Pod.Tolerations,
			Affinity:           nemoAgentConfig.Spec.Pod.Affinity,
			Containers: []corev1.Container{
				{
					Name:            nemoAgentConfig.GetContainerName(),
					Image:           nemoAgentConfig.GetImage(),
					ImagePullPolicy: corev1.PullPolicy(nemoAgentConfig.Spec.Pod.Image.PullPolicy),
					Command:         nemoAgentConfig.GetCommand(),
					Args:            nemoAgentConfig.GetArgs(),
					Env:             nemoAgentConfig.Spec.Pod.Env,
					Resources:       derefResourceRequirements(nemoAgentConfig.Spec.Pod.Resources),
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      nemoAgentConfig.GetConfigVolumeName(),
							MountPath: nemoAgentConfig.GetConfigMountPath(),
							ReadOnly:  nemoAgentConfig.Spec.Pod.ConfigMount.ReadOnly,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: nemoAgentConfig.GetConfigVolumeName(),
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{Name: nemoAgentConfig.GetConfigMapName()},
							Items: []corev1.KeyToPath{
								{
									Key:  nemoAgentConfig.GetConfigMapKey(),
									Path: nemoAgentConfig.GetConfigMountFileName(),
								},
							},
						},
					},
				},
			},
		},
	}
	if nemoAgentConfig.Spec.Pod.RuntimeClassName != "" {
		pod.Spec.RuntimeClassName = &nemoAgentConfig.Spec.Pod.RuntimeClassName
	}
	for _, secret := range nemoAgentConfig.Spec.Pod.Image.PullSecrets {
		pod.Spec.ImagePullSecrets = append(pod.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: secret})
	}
	if nemoAgentConfig.Spec.Pod.UserID != nil || nemoAgentConfig.Spec.Pod.GroupID != nil {
		pod.Spec.SecurityContext = &corev1.PodSecurityContext{
			RunAsUser:  nemoAgentConfig.Spec.Pod.UserID,
			RunAsGroup: nemoAgentConfig.Spec.Pod.GroupID,
		}
	}
	return pod
}

func derefResourceRequirements(resources *corev1.ResourceRequirements) corev1.ResourceRequirements {
	if resources == nil {
		return corev1.ResourceRequirements{}
	}
	return *resources.DeepCopy()
}

func (r *NemoAgentConfigReconciler) updateStatusFromPod(ctx context.Context, nemoAgentConfig *appsv1alpha1.NemoAgentConfig) (ctrl.Result, error) {
	pod := &corev1.Pod{}
	key := types.NamespacedName{Name: nemoAgentConfig.GetPodName(), Namespace: nemoAgentConfig.Namespace}
	if err := r.Get(ctx, key, pod); err != nil {
		if apiErrors.IsNotFound(err) {
			return ctrl.Result{}, r.setStatus(ctx, nemoAgentConfig, appsv1alpha1.NemoAgentConfigStatusPending, conditions.NotReady, "PodPending", "agent pod has not been created")
		}
		return ctrl.Result{}, err
	}

	switch pod.Status.Phase {
	case corev1.PodFailed:
		return ctrl.Result{}, r.setStatus(ctx, nemoAgentConfig, appsv1alpha1.NemoAgentConfigStatusFailed, conditions.Failed, "PodFailed", "agent pod failed")
	case corev1.PodSucceeded:
		return ctrl.Result{}, r.setStatus(ctx, nemoAgentConfig, appsv1alpha1.NemoAgentConfigStatusReady, conditions.Ready, "PodSucceeded", "agent pod completed successfully")
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
			return ctrl.Result{}, r.setStatus(ctx, nemoAgentConfig, appsv1alpha1.NemoAgentConfigStatusReady, conditions.Ready, "PodReady", "agent pod is ready")
		}
	}

	return ctrl.Result{RequeueAfter: 10 * time.Second}, r.setStatus(ctx, nemoAgentConfig, appsv1alpha1.NemoAgentConfigStatusPending, conditions.NotReady, "PodNotReady", "agent pod is not ready")
}

func (r *NemoAgentConfigReconciler) setStatus(ctx context.Context, nemoAgentConfig *appsv1alpha1.NemoAgentConfig, state string, conditionType string, reason string, message string) error {
	return k8sutil.RetryStatusUpdate(ctx, r.Client, nemoAgentConfig, func(obj client.Object) {
		cr := obj.(*appsv1alpha1.NemoAgentConfig) //nolint:forcetypeassert
		meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
			Type:    conditionType,
			Status:  metav1.ConditionTrue,
			Reason:  reason,
			Message: message,
		})
		if conditionType == conditions.Ready {
			meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
				Type:   conditions.Failed,
				Status: metav1.ConditionFalse,
				Reason: conditions.Ready,
			})
		}
		if conditionType == conditions.Failed {
			meta.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
				Type:   conditions.Ready,
				Status: metav1.ConditionFalse,
				Reason: conditions.Failed,
			})
		}
		cr.Status.State = state
	})
}

func (r *NemoAgentConfigReconciler) getEventRecorder() record.EventRecorder {
	if r.recorder == nil {
		return record.NewFakeRecorder(1)
	}
	return r.recorder
}

// SetupWithManager sets up the controller with the Manager.
func (r *NemoAgentConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("nemo-agent-config-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1alpha1.NemoAgentConfig{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Pod{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc: func(e event.UpdateEvent) bool {
				if oldConfig, ok := e.ObjectOld.(*appsv1alpha1.NemoAgentConfig); ok {
					newConfig, ok := e.ObjectNew.(*appsv1alpha1.NemoAgentConfig)
					if ok {
						if !newConfig.DeletionTimestamp.IsZero() {
							return true
						}
						return !reflect.DeepEqual(oldConfig.Spec, newConfig.Spec)
					}
				}
				return true
			},
		}).
		Complete(r)
}
