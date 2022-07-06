package main

import (
	"context"
	"fmt"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/version"
)

func TestMain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Main Suite")
}

var _ = Describe("Jumpstart GPUAddon", func() {
	common.ProcessConfig()
	It("Should create GPUAddon CR if not exists", func() {
		c := newTestClientWith()
		Expect(jumpstartAddon(c)).ShouldNot(HaveOccurred())
		g := &addonv1alpha1.GPUAddon{}
		Expect(c.Get(context.TODO(), types.NamespacedName{
			Namespace: common.GlobalConfig.AddonNamespace,
			Name:      common.GlobalConfig.AddonID,
		}, g)).ShouldNot(HaveOccurred())
	})
	It("Should not fail if a GPUAddon CR already Exists", func() {
		g := &addonv1alpha1.GPUAddon{}
		g.Name = common.GlobalConfig.AddonID
		g.Namespace = common.GlobalConfig.AddonNamespace
		c := newTestClientWith(g)
		Expect(jumpstartAddon(c)).ShouldNot(HaveOccurred())
		Expect(c.Get(context.TODO(), types.NamespacedName{
			Namespace: common.GlobalConfig.AddonNamespace,
			Name:      common.GlobalConfig.AddonID,
		}, g)).ShouldNot(HaveOccurred())
		versionLabel := fmt.Sprintf("%v-version", common.GlobalConfig.AddonID)
		Expect(g.ObjectMeta.Labels[versionLabel]).To(Equal(version.Version()))
	})
})

var _ = Describe("Jumpstart Monitoring", func() {
	common.ProcessConfig()
	It("should create the Monitoring CR if not exists", func() {
		c := newTestClientWith()
		Expect(jumpstartMonitoring(c)).ShouldNot(HaveOccurred())
		m := &addonv1alpha1.Monitoring{}
		Expect(c.Get(context.TODO(), types.NamespacedName{
			Namespace: common.GlobalConfig.AddonNamespace,
			Name:      common.GlobalConfig.AddonID,
		}, m)).ShouldNot(HaveOccurred())
	})
	It("should not fail if a Monitoring CR already exists", func() {
		m := &addonv1alpha1.Monitoring{}
		m.Name = common.GlobalConfig.AddonID
		m.Namespace = common.GlobalConfig.AddonNamespace
		c := newTestClientWith(m)
		Expect(jumpstartMonitoring(c)).ShouldNot(HaveOccurred())
		Expect(c.Get(context.TODO(), types.NamespacedName{
			Namespace: common.GlobalConfig.AddonNamespace,
			Name:      common.GlobalConfig.AddonID,
		}, m)).ShouldNot(HaveOccurred())
	})
})

func newTestClientWith(objs ...runtime.Object) client.Client {
	s := clientgoscheme.Scheme
	Expect(addonv1alpha1.AddToScheme(s)).ShouldNot(HaveOccurred())
	return fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
}
