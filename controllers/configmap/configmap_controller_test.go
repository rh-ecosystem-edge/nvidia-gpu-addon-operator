package configmap

import (
	"context"

	gpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/scheme"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
)

var _ = Describe("ConfigMapReconciler", func() {
	Describe("Reconcile", func() {
		common.ProcessConfig()

		configmap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.GlobalConfig.AddonID,
				Namespace: common.GlobalConfig.AddonNamespace,
			},
		}

		gpuaddon := &addonv1alpha1.GPUAddon{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "TestAddon",
				Namespace: common.GlobalConfig.AddonNamespace,
			},
		}

		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      common.GlobalConfig.AddonID,
				Namespace: common.GlobalConfig.AddonNamespace,
			},
		}

		g := &addonv1alpha1.GPUAddon{}

		Context("with a valid configmap label", func() {
			BeforeEach(func() {
				configmap.Labels = map[string]string{
					getAddonDeleteLabel(): "true",
				}
			})

			It("should delete the GPUAddon CR", func() {
				r := newTestConfigReconciler(configmap, gpuaddon)

				_, err := r.Reconcile(context.TODO(), req)
				Expect(err).ShouldNot(HaveOccurred())

				err = r.Get(context.TODO(), types.NamespacedName{Name: gpuaddon.Name, Namespace: gpuaddon.Namespace}, g)
				Expect(err).Should(HaveOccurred())
				Expect(k8serrors.IsNotFound(err)).To(BeTrue())
			})
		})

		Context("with an invalid configmap label", func() {
			BeforeEach(func() {
				configmap.Labels = map[string]string{
					getAddonDeleteLabel(): "invalidBool",
				}
			})

			It("should not do anything", func() {
				r := newTestConfigReconciler(configmap, gpuaddon)

				_, err := r.Reconcile(context.TODO(), req)
				Expect(err).ShouldNot(HaveOccurred())

				err = r.Client.Get(context.TODO(), types.NamespacedName{Name: gpuaddon.Name, Namespace: gpuaddon.Namespace}, g)
				Expect(err).ShouldNot(HaveOccurred())
			})
		})
	})
})

func newTestConfigReconciler(objs ...runtime.Object) *ConfigMapReconciler {
	s := scheme.Scheme

	Expect(operatorsv1alpha1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(corev1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(addonv1alpha1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(gpuv1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(nfdv1.AddToScheme(s)).ShouldNot(HaveOccurred())

	c := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	return &ConfigMapReconciler{
		Client: c,
		Scheme: s,
	}
}
