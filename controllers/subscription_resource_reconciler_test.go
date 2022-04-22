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

package controllers

import (
	"context"

	configv1 "github.com/openshift/api/config/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
)

var _ = Describe("Subscription Resource Reconcile", Ordered, func() {
	Context("Reconcile", func() {
		common.ProcessConfig()
		rrec := &SubscriptionResourceReconciler{}
		gpuAddon := addonv1alpha1.GPUAddon{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
		}
		clusterVersion := &configv1.ClusterVersion{
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

		scheme := scheme.Scheme
		Expect(operatorsv1alpha1.AddToScheme(scheme)).ShouldNot(HaveOccurred())
		Expect(configv1.AddToScheme(scheme)).ShouldNot(HaveOccurred())

		var s operatorsv1alpha1.Subscription

		It("should create the Subscription", func() {
			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(clusterVersion).
				Build()

			_, err := rrec.Reconcile(context.TODO(), c, &gpuAddon)
			Expect(err).ShouldNot(HaveOccurred())

			err = c.Get(context.TODO(), types.NamespacedName{
				Namespace: gpuAddon.Namespace,
				Name:      "gpu-operator-certified",
			}, &s)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
