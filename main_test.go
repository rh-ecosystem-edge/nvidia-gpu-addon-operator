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

var _ = Describe("Jumpstart tests", func() {
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
		versionLabel := fmt.Sprintf("%v-version", common.GlobalConfig.AddonLabel)
		Expect(g.ObjectMeta.Labels[versionLabel]).To(Equal(version.Version()))
	})

})

func newTestClientWith(objs ...runtime.Object) client.Client {
	s := clientgoscheme.Scheme
	Expect(addonv1alpha1.AddToScheme(s)).ShouldNot(HaveOccurred())
	return fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
}
