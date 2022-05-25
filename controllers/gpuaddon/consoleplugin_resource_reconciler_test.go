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

package gpuaddon

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	consolev1alpha1 "github.com/openshift/api/console/v1alpha1"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/scheme"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
)

var _ = Describe("ConsolePlugin Resource Reconcile", Ordered, func() {
	Context("Reconcile", func() {
		common.ProcessConfig()
		rrec := &ConsolePluginResourceReconciler{}
		gpuAddon := addonv1alpha1.GPUAddon{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
		}
		console := &operatorv1.Console{
			ObjectMeta: metav1.ObjectMeta{
				Name: "cluster",
			},
			Spec: operatorv1.ConsoleSpec{},
		}
		clusterVersion := &configv1.ClusterVersion{
			ObjectMeta: metav1.ObjectMeta{
				Name: "version",
			},
			Status: configv1.ClusterVersionStatus{
				History: []configv1.UpdateHistory{
					{
						State:   configv1.CompletedUpdate,
						Version: "4.10.1",
					},
				},
			},
		}

		scheme := scheme.Scheme
		Expect(consolev1alpha1.AddToScheme(scheme)).ShouldNot(HaveOccurred())
		Expect(operatorv1.AddToScheme(scheme)).ShouldNot(HaveOccurred())
		Expect(appsv1.AddToScheme(scheme)).ShouldNot(HaveOccurred())
		Expect(corev1.AddToScheme(scheme)).ShouldNot(HaveOccurred())
		Expect(configv1.AddToScheme(scheme)).ShouldNot(HaveOccurred())

		var cp consolev1alpha1.ConsolePlugin
		var dp appsv1.Deployment
		var s corev1.Service

		Context("when OpenShift version is less than 4.10", func() {
			unsupportedClusterVersion := &configv1.ClusterVersion{
				ObjectMeta: metav1.ObjectMeta{
					Name: "version",
				},
				Status: configv1.ClusterVersionStatus{
					History: []configv1.UpdateHistory{
						{
							State:   configv1.CompletedUpdate,
							Version: "4.9.7",
						},
					},
				},
			}

			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(unsupportedClusterVersion).
				Build()

			It("should not reconcile the ConsolePlugin components", func() {
				conditions, err := rrec.Reconcile(context.TODO(), c, &gpuAddon)
				Expect(err).ShouldNot(HaveOccurred())

				err = c.Get(context.TODO(), types.NamespacedName{
					Namespace: gpuAddon.Namespace,
					Name:      "console-plugin-nvidia-gpu",
				}, &dp)
				Expect(err).Should(HaveOccurred())
				Expect(k8serrors.IsNotFound(err)).To(BeTrue())

				err = c.Get(context.TODO(), types.NamespacedName{
					Namespace: gpuAddon.Namespace,
					Name:      "console-plugin-nvidia-gpu",
				}, &s)
				Expect(err).Should(HaveOccurred())
				Expect(k8serrors.IsNotFound(err)).To(BeTrue())

				err = c.Get(context.TODO(), client.ObjectKey{
					Name: "console-plugin-nvidia-gpu",
				}, &cp)
				Expect(err).Should(HaveOccurred())
				Expect(k8serrors.IsNotFound(err)).To(BeTrue())

				Expect(conditions).To(HaveLen(1))
				Expect(conditions[0].Reason).To(Equal("NotSupported"))
			})
		})

		Context("when enabled", func() {
			It("should create the ConsolePlugin components", func() {
				gpuAddon.Spec = addonv1alpha1.GPUAddonSpec{
					ConsolePluginEnabled: true,
				}

				c := fake.
					NewClientBuilder().
					WithScheme(scheme).
					WithRuntimeObjects(clusterVersion, console).
					Build()

				conditions, err := rrec.Reconcile(context.TODO(), c, &gpuAddon)
				Expect(err).ShouldNot(HaveOccurred())

				err = c.Get(context.TODO(), types.NamespacedName{
					Namespace: gpuAddon.Namespace,
					Name:      "console-plugin-nvidia-gpu",
				}, &dp)
				Expect(err).ShouldNot(HaveOccurred())

				err = c.Get(context.TODO(), types.NamespacedName{
					Namespace: gpuAddon.Namespace,
					Name:      "console-plugin-nvidia-gpu",
				}, &s)
				Expect(err).ShouldNot(HaveOccurred())

				err = c.Get(context.TODO(), client.ObjectKey{
					Name: "console-plugin-nvidia-gpu",
				}, &cp)
				Expect(err).ShouldNot(HaveOccurred())

				Expect(conditions).To(HaveLen(1))
				Expect(conditions[0].Reason).To(Equal("Success"))

				err = c.Get(context.TODO(), client.ObjectKey{
					Name: "cluster",
				}, console)
				Expect(err).ShouldNot(HaveOccurred())

				Expect(console.Spec.Plugins).To(HaveLen(1))
				Expect(console.Spec.Plugins[0]).To(Equal("console-plugin-nvidia-gpu"))
			})
		})

		Context("when disabled", func() {
			It("should not create the ConsolePlugin components", func() {
				gpuAddon.Spec = addonv1alpha1.GPUAddonSpec{
					ConsolePluginEnabled: false,
				}

				c := fake.
					NewClientBuilder().
					WithScheme(scheme).
					WithRuntimeObjects(clusterVersion).
					Build()

				conditions, err := rrec.Reconcile(context.TODO(), c, &gpuAddon)
				Expect(err).ShouldNot(HaveOccurred())

				err = c.Get(context.TODO(), types.NamespacedName{
					Namespace: gpuAddon.Namespace,
					Name:      "console-plugin-nvidia-gpu",
				}, &dp)
				Expect(err).Should(HaveOccurred())
				Expect(k8serrors.IsNotFound(err)).To(BeTrue())

				err = c.Get(context.TODO(), types.NamespacedName{
					Namespace: gpuAddon.Namespace,
					Name:      "console-plugin-nvidia-gpu",
				}, &s)
				Expect(err).Should(HaveOccurred())
				Expect(k8serrors.IsNotFound(err)).To(BeTrue())

				err = c.Get(context.TODO(), client.ObjectKey{
					Name: "console-plugin-nvidia-gpu",
				}, &cp)
				Expect(err).Should(HaveOccurred())
				Expect(k8serrors.IsNotFound(err)).To(BeTrue())

				Expect(conditions).To(HaveLen(1))
				Expect(conditions[0].Reason).To(Equal("Success"))
			})

			It("should delete the ConsolePlugin components when previously enabled", func() {
				gpuAddon.Spec = addonv1alpha1.GPUAddonSpec{
					ConsolePluginEnabled: false,
				}

				cp := &consolev1alpha1.ConsolePlugin{
					ObjectMeta: metav1.ObjectMeta{
						Name: "console-plugin-nvidia-gpu",
					},
				}
				dp := &appsv1.Deployment{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "console-plugin-nvidia-gpu",
						Namespace: common.GlobalConfig.AddonNamespace,
					},
				}
				s := &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "console-plugin-nvidia-gpu",
						Namespace: common.GlobalConfig.AddonNamespace,
					},
				}

				c := fake.
					NewClientBuilder().
					WithScheme(scheme).
					WithRuntimeObjects(cp, dp, s, clusterVersion).
					Build()

				conditions, err := rrec.Reconcile(context.TODO(), c, &gpuAddon)
				Expect(err).ShouldNot(HaveOccurred())

				err = c.Get(context.TODO(), client.ObjectKey{
					Name: "console-plugin-nvidia-gpu",
				}, cp)
				Expect(err).Should(HaveOccurred())
				Expect(k8serrors.IsNotFound(err)).To(BeTrue())

				err = c.Get(context.TODO(), types.NamespacedName{
					Namespace: gpuAddon.Namespace,
					Name:      "console-plugin-nvidia-gpu",
				}, s)
				Expect(err).Should(HaveOccurred())
				Expect(k8serrors.IsNotFound(err)).To(BeTrue())

				err = c.Get(context.TODO(), types.NamespacedName{
					Namespace: gpuAddon.Namespace,
					Name:      "console-plugin-nvidia-gpu",
				}, dp)
				Expect(err).Should(HaveOccurred())
				Expect(k8serrors.IsNotFound(err)).To(BeTrue())

				Expect(conditions).To(HaveLen(1))
				Expect(conditions[0].Reason).To(Equal("Success"))
			})
		})
	})

	Context("Delete", func() {
		common.ProcessConfig()
		rrec := &ConsolePluginResourceReconciler{}

		cp := &consolev1alpha1.ConsolePlugin{
			ObjectMeta: metav1.ObjectMeta{
				Name: "console-plugin-nvidia-gpu",
			},
		}
		dp := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "console-plugin-nvidia-gpu",
				Namespace: common.GlobalConfig.AddonNamespace,
			},
		}
		s := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "console-plugin-nvidia-gpu",
				Namespace: common.GlobalConfig.AddonNamespace,
			},
		}

		scheme := scheme.Scheme
		Expect(consolev1alpha1.AddToScheme(scheme)).ShouldNot(HaveOccurred())

		It("should delete the ConsolePlugin components", func() {
			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(cp, dp, s).
				Build()

			err := rrec.Delete(context.TODO(), c)
			Expect(err).ShouldNot(HaveOccurred())

			err = c.Get(context.TODO(), client.ObjectKey{
				Name: cp.Name,
			}, cp)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())

			err = c.Get(context.TODO(), types.NamespacedName{
				Name:      dp.Name,
				Namespace: dp.Namespace,
			}, dp)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())

			err = c.Get(context.TODO(), types.NamespacedName{
				Name:      s.Name,
				Namespace: s.Namespace,
			}, s)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})
	})
})
