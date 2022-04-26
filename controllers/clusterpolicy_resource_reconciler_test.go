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

	gpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
)

var _ = Describe("ClusterPolicy Resource Reconcile", Ordered, func() {
	Context("Reconcile", func() {
		common.ProcessConfig()
		rrec := &ClusterPolicyResourceReconciler{}
		gpuAddon := addonv1alpha1.GPUAddon{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		}
		scheme := scheme.Scheme
		Expect(gpuv1.AddToScheme(scheme)).ShouldNot(HaveOccurred())

		var cp gpuv1.ClusterPolicy

		It("should create the ClusterPolicy", func() {
			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects().
				Build()

			cond, err := rrec.Reconcile(context.TODO(), c, &gpuAddon)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(cond).To(HaveLen(1))
			Expect(cond[0].Type).To(Equal(ClusterPolicyDeployedCondition))
			Expect(cond[0].Status).To(Equal(metav1.ConditionTrue))

			err = c.Get(context.TODO(), types.NamespacedName{
				Name: common.GlobalConfig.ClusterPolicyName,
			}, &cp)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})
