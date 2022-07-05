package gpuaddon

import (
	"context"
	"reflect"
	"time"

	gpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	configv1 "github.com/openshift/api/config/v1"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/scheme"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
)

var _ = Describe("GPUAddon Reconcile", Ordered, func() {
	Context("Creation Reconcile", func() {
		common.ProcessConfig()
		gpuAddon, r := prepareClusterForGPUAddonCreateTest()

		var g = &addonv1alpha1.GPUAddon{}

		It("First reconcile should not error", func() {
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: gpuAddon.Namespace,
					Name:      gpuAddon.Name,
				},
			}

			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).ShouldNot(HaveOccurred())

		})
		It("Get the GPUAddon CR from API - No Error", func() {
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Namespace: gpuAddon.Namespace,
				Name:      gpuAddon.Name,
			}, g)
			Expect(err).ShouldNot(HaveOccurred())

		})

		// Check finilizer
		It("Should add finilizers", func() {
			Expect(controllerutil.ContainsFinalizer(g, common.GlobalConfig.AddonID)).To(BeTrue())
		})
		Context("NFD related tests", func() {
			It("Should Have created an NFD CR and mark as owner", func() {
				nfdCr := &nfdv1.NodeFeatureDiscovery{}
				err := r.Client.Get(context.TODO(), types.NamespacedName{
					Name:      common.GlobalConfig.NfdCrName,
					Namespace: common.GlobalConfig.AddonNamespace,
				}, nfdCr)
				Expect(err).ShouldNot(HaveOccurred())

				ownerRef := nfdCr.ObjectMeta.OwnerReferences[0]
				Expect(ownerRef.UID).To(Equal(g.UID))
			})
			It("Should contain condition", func() {
				Expect(common.ContainCondition(g.Status.Conditions, "NodeFeatureDiscoveryDeployed", "True")).To(BeTrue())
			})
		})
		Context("ClusterPolicy related tests", func() {
			It("Should Create gpu-operator ClusterPolicy CR", func() {
				clusterPolicyCr := &gpuv1.ClusterPolicy{}
				err := r.Client.Get(context.TODO(), types.NamespacedName{
					Name: common.GlobalConfig.ClusterPolicyName,
				}, clusterPolicyCr)
				Expect(err).ShouldNot(HaveOccurred())
			})
			It("Should contain condition", func() {
				Expect(common.ContainCondition(g.Status.Conditions, "ClusterPolicyDeployed", "True")).To(BeTrue())
			})
		})
	})

	Context("Delete reconcile", func() {
		gpuAddon, r := prepareClusterForGPUAddonDeletionTest()

		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: gpuAddon.Namespace,
				Name:      gpuAddon.Name,
			},
		}

		_, err := r.Reconcile(context.TODO(), req)

		It("should return an error", func() {
			Expect(err).Should(HaveOccurred())
			Expect(err.Error()).Should(ContainSubstring("not all resources have been deleted"))
		})

		It("should delete the GPUAddon CSV", func() {
			g := &operatorsv1alpha1.ClusterServiceVersion{}
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Name:      common.GlobalConfig.AddonID,
				Namespace: common.GlobalConfig.AddonNamespace,
			}, g)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should already find the GPUAddon CR deleted", func() {
			err := r.Client.Delete(context.TODO(), gpuAddon)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should delete the ClusterPolicy CR", func() {
			cp := &gpuv1.ClusterPolicy{}
			err := r.Get(context.TODO(), client.ObjectKey{
				Name: common.GlobalConfig.ClusterPolicyName,
			}, cp)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should delete the Subscription CR", func() {
			s := &operatorsv1alpha1.Subscription{}
			err := r.Get(context.TODO(), client.ObjectKey{
				Name:      "gpu-operator-certified",
				Namespace: gpuAddon.Namespace,
			}, s)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should delete the NFD CR", func() {
			nfd := &nfdv1.NodeFeatureDiscovery{}
			err := r.Get(context.TODO(), types.NamespacedName{
				Name:      common.GlobalConfig.NfdCrName,
				Namespace: gpuAddon.Namespace,
			}, nfd)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		Context("a second time", func() {
			_, err := r.Reconcile(context.TODO(), req)

			It("should not return an error", func() {
				Expect(err).ShouldNot(HaveOccurred())
			})
		})
	})
})

func prepareClusterForGPUAddonCreateTest() (*addonv1alpha1.GPUAddon, *GPUAddonReconciler) {
	gpuAddon := &addonv1alpha1.GPUAddon{}
	gpuAddon.Name = "TestAddon"
	gpuAddon.Namespace = common.GlobalConfig.AddonNamespace
	gpuAddon.APIVersion = "v1alpha1"
	gpuAddon.UID = types.UID("uid-uid")
	gpuAddon.Kind = "GPUAddon"

	r := newTestGPUAddonReconciler(gpuAddon)

	return gpuAddon, r
}

func prepareClusterForGPUAddonDeletionTest() (*addonv1alpha1.GPUAddon, *GPUAddonReconciler) {
	gpuAddon := &addonv1alpha1.GPUAddon{}
	gpuAddon.Name = "TestAddon"
	gpuAddon.Namespace = common.GlobalConfig.AddonNamespace
	gpuAddon.UID = types.UID("uid-uid")
	gpuAddon.Finalizers = []string{common.GlobalConfig.AddonID}
	now := metav1.NewTime(time.Now())
	gpuAddon.ObjectMeta.DeletionTimestamp = &now

	kind := reflect.TypeOf(gpuv1.ClusterPolicy{}).Name()
	gvk := gpuv1.GroupVersion.WithKind(kind)

	// This addon's CSV
	gpuaddoncsv := common.NewCsv(common.GlobalConfig.AddonNamespace, common.GlobalConfig.AddonID, "")

	ownerRef := metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               gpuAddon.Name,
		UID:                gpuAddon.UID,
		BlockOwnerDeletion: pointer.BoolPtr(true),
		Controller:         pointer.BoolPtr(true),
	}

	// NFD
	nfd := &nfdv1.NodeFeatureDiscovery{
		ObjectMeta: metav1.ObjectMeta{
			Name:            common.GlobalConfig.NfdCrName,
			Namespace:       gpuAddon.Namespace,
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
	}

	// ClusterPolicy
	clusterPolicy := &gpuv1.ClusterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:            common.GlobalConfig.ClusterPolicyName,
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
	}

	// Subscription
	subscription := &operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "gpu-operator-certified",
			Namespace:       gpuAddon.Namespace,
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
	}

	// GPU Operator CSV
	csv := &operatorsv1alpha1.ClusterServiceVersion{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gpu-operator-certified.v1.10.1",
			Namespace: gpuAddon.Namespace,
		},
	}

	r := newTestGPUAddonReconciler(gpuAddon, gpuaddoncsv, nfd, subscription, csv, clusterPolicy)

	return gpuAddon, r
}

func newTestGPUAddonReconciler(objs ...runtime.Object) *GPUAddonReconciler {
	s := scheme.Scheme

	Expect(operatorsv1alpha1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(operatorsv1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(v1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(addonv1alpha1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(gpuv1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(nfdv1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(configv1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(appsv1.AddToScheme(s)).ShouldNot(HaveOccurred())

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

	objs = append(objs, clusterVersion)

	c := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	return &GPUAddonReconciler{
		Client: c,
		Scheme: s,
	}
}
