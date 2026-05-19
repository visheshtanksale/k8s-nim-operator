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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	appsv1alpha1 "github.com/NVIDIA/k8s-nim-operator/api/apps/v1alpha1"
)

var _ = Describe("NemoAgentConfig Controller", func() {
	var (
		cli        client.Client
		reconciler *NemoAgentConfigReconciler
		scheme     *runtime.Scheme
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		Expect(appsv1alpha1.AddToScheme(scheme)).To(Succeed())
		Expect(corev1.AddToScheme(scheme)).To(Succeed())

		cli = fake.NewClientBuilder().
			WithScheme(scheme).
			WithStatusSubresource(&appsv1alpha1.NemoAgentConfig{}).
			Build()
		reconciler = &NemoAgentConfigReconciler{
			Client:   cli,
			Scheme:   scheme,
			recorder: record.NewFakeRecorder(100),
		}
	})

	It("should create a ConfigMap and Pod from the CR", func() {
		ctx := context.Background()
		nemoAgentConfig := &appsv1alpha1.NemoAgentConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: appsv1alpha1.SchemeGroupVersion.String(),
				Kind:       "NemoAgentConfig",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "nemo-agent-config",
				Namespace: "default",
			},
			Spec: appsv1alpha1.NemoAgentConfigSpec{
				Functions: map[string]appsv1alpha1.NemoAgentFunctionConfig{
					"wikipedia_search": {
						Type:       "wiki_search",
						MaxResults: ptr.To[int32](2),
					},
				},
				LLMs: map[string]appsv1alpha1.NemoAgentLLMConfig{
					"nim_llm": {
						Type:        "nim",
						ModelName:   "nvidia/nemotron-3-nano",
						BaseURL:     "http://llama-3-1-70b-instruct:8000/v1",
						Temperature: ptr.To(0.0),
						ChatTemplateKwargs: map[string]apiextensionsv1.JSON{
							"enable_thinking": {Raw: []byte("false")},
						},
					},
				},
				Workflow: appsv1alpha1.NemoAgentWorkflowConfig{
					Type:                         "react_agent",
					ToolNames:                    []string{"wikipedia_search"},
					LLMName:                      "nim_llm",
					Verbose:                      ptr.To(true),
					ParseAgentResponseMaxRetries: ptr.To[int32](3),
				},
				Pod: appsv1alpha1.NemoAgentPodSpec{
					Name:          "sleep-pod-1",
					ContainerName: "curl",
					RestartPolicy: corev1.RestartPolicyNever,
					ConfigMount: appsv1alpha1.NemoAgentConfigMount{
						Name:      "cm-storage",
						MountPath: "/model-store",
						Key:       "config.yml",
						Path:      "workflow.yml",
					},
					Image: appsv1alpha1.Image{
						Repository: "localhost:5000/nvstaging/nvidia-nat",
						Tag:        "v1.6.0",
						PullPolicy: string(corev1.PullAlways),
					},
					Command: []string{"sleep"},
					Args:    []string{"360000"},
				},
			},
		}
		Expect(cli.Create(ctx, nemoAgentConfig)).To(Succeed())

		_, err := reconciler.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: nemoAgentConfig.Name, Namespace: nemoAgentConfig.Namespace}})
		Expect(err).NotTo(HaveOccurred())

		cm := &corev1.ConfigMap{}
		Expect(cli.Get(ctx, types.NamespacedName{Name: nemoAgentConfig.Name, Namespace: nemoAgentConfig.Namespace}, cm)).To(Succeed())
		Expect(cm.Data).To(HaveKey(appsv1alpha1.NemoAgentConfigFileName))
		Expect(cm.Data[appsv1alpha1.NemoAgentConfigFileName]).To(ContainSubstring("wikipedia_search:"))
		Expect(cm.Data[appsv1alpha1.NemoAgentConfigFileName]).To(ContainSubstring("nim_llm:"))
		Expect(cm.Data[appsv1alpha1.NemoAgentConfigFileName]).To(ContainSubstring("workflow:"))

		pod := &corev1.Pod{}
		Expect(cli.Get(ctx, types.NamespacedName{Name: "sleep-pod-1", Namespace: nemoAgentConfig.Namespace}, pod)).To(Succeed())
		Expect(pod.Spec.RestartPolicy).To(Equal(corev1.RestartPolicyNever))
		Expect(pod.Spec.Containers).To(HaveLen(1))
		Expect(pod.Spec.Containers[0].Name).To(Equal("curl"))
		Expect(pod.Spec.Containers[0].Image).To(Equal("localhost:5000/nvstaging/nvidia-nat:v1.6.0"))
		Expect(pod.Spec.Containers[0].ImagePullPolicy).To(Equal(corev1.PullAlways))
		Expect(pod.Spec.Containers[0].Command).To(Equal([]string{"sleep"}))
		Expect(pod.Spec.Containers[0].Args).To(Equal([]string{"360000"}))
		Expect(pod.Spec.Containers[0].VolumeMounts).To(ContainElement(corev1.VolumeMount{
			Name:      "cm-storage",
			MountPath: "/model-store",
		}))
		Expect(pod.Spec.Volumes).To(HaveLen(1))
		Expect(pod.Spec.Volumes[0].ConfigMap.Name).To(Equal(nemoAgentConfig.Name))
		Expect(pod.Spec.Volumes[0].ConfigMap.Items).To(Equal([]corev1.KeyToPath{
			{
				Key:  "config.yml",
				Path: "workflow.yml",
			},
		}))
	})
})
