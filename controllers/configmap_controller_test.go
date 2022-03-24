package controllers

import (
	"context"

	gpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/scheme"
	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("ConfigMap Controller Tests", func() {
	It("deleteGouAddon function test", func() {
		gpuaddon := &addonv1alpha1.GPUAddon{}
		gpuaddon.Name = "TestAddon"
		gpuaddon.Namespace = common.GlobalConfig.AddonNamespace

		g := &addonv1alpha1.GPUAddon{}
		reconciler := newTestConfigReconciler(gpuaddon)

		err := reconciler.Client.Get(context.TODO(), types.NamespacedName{Name: "TestAddon", Namespace: gpuaddon.Namespace}, g)
		Expect(err).ShouldNot(HaveOccurred())

		err = reconciler.deleteGpuAddonCr(context.TODO(), gpuaddon.Namespace)
		Expect(err).ShouldNot(HaveOccurred())

		err = reconciler.Client.Get(context.TODO(), types.NamespacedName{Name: "TestAddon", Namespace: gpuaddon.Namespace}, g)
		Expect(k8serrors.IsNotFound(err)).To(BeTrue())
	})
	Context("Reconcile tests", func() {
		configmap := &v1.ConfigMap{}
		configmap.Name = common.GlobalConfig.AddonID
		configmap.Namespace = common.GlobalConfig.AddonNamespace
		configmap.Labels = map[string]string{}

		gpuaddon := &addonv1alpha1.GPUAddon{}
		gpuaddon.Name = "TestAddon"
		gpuaddon.Namespace = configmap.Namespace
		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      common.GlobalConfig.AddonID,
				Namespace: common.GlobalConfig.AddonNamespace,
			},
		}
		g := &addonv1alpha1.GPUAddon{}
		It("Happy flow", func() {
			// Delete Addon CR
			configmap.Labels[AddonDeleteLabel] = "true"

			r := newTestConfigReconciler(configmap, gpuaddon)

			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).ShouldNot(HaveOccurred())
			err = r.Client.Get(context.TODO(), types.NamespacedName{Name: "TestAddon", Namespace: configmap.Namespace}, g)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())

		})

		It("Should not do anything when no delete label, or invalid value", func() {
			configmap.Labels = map[string]string{}
			r := newTestConfigReconciler(configmap, gpuaddon)
			// Dont return error and ignore when invalid value or non relevant configmap
			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).ShouldNot(HaveOccurred())

			configmap.Labels[AddonDeleteLabel] = "invalidBool"
			_, err = r.Reconcile(context.TODO(), req)
			Expect(err).ShouldNot(HaveOccurred())

			// GPU addon should still exists
			err = r.Client.Get(context.TODO(), types.NamespacedName{Name: "TestAddon", Namespace: configmap.Namespace}, g)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})

func newTestConfigReconciler(objs ...runtime.Object) *ConfigMapReconciler {
	s := scheme.Scheme

	Expect(operatorsv1alpha1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(v1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(addonv1alpha1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(gpuv1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(nfdv1.AddToScheme(s)).ShouldNot(HaveOccurred())

	c := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	return &ConfigMapReconciler{
		Client: c,
		Scheme: s,
	}
}
