package controllers

import (
	"context"
	"time"

	gpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	configv1 "github.com/openshift/api/config/v1"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/scheme"
	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
	v1 "k8s.io/api/core/v1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
			Expect(common.SliceContainsString(g.ObjectMeta.Finalizers, common.GlobalConfig.AddonID)).To(BeTrue())
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

	Context("Delete Reconcile", func() {
		gpuAddon, r := prepareClusterForGPUAddonDeletionTest()
		It("Should not return an error when reconciling", func() {
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: gpuAddon.Namespace,
					Name:      gpuAddon.Name,
				},
			}

			_, err := r.Reconcile(context.TODO(), req)
			Expect(err).ShouldNot(HaveOccurred())
		})
		It("Should Delete GPUAddon CSV", func() {
			g := &operatorsv1alpha1.ClusterServiceVersion{}
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Name:      common.GlobalConfig.AddonID,
				Namespace: common.GlobalConfig.AddonNamespace,
			}, g)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})
		It("GPU Addon CR should be deleted now.", func() {
			// At this point the GPUAddon should be deleted.
			err := r.Client.Delete(context.TODO(), gpuAddon)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})
		XIt("Should Delete NFD CR as a result.", func() {
			nfd := &nfdv1.NodeFeatureDiscovery{}
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Name:      common.GlobalConfig.NfdCrName,
				Namespace: gpuAddon.Namespace,
			}, nfd)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
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
	gpuAddon.APIVersion = "v1alpha1"
	gpuAddon.Kind = "GPUAddon"
	gpuAddon.UID = types.UID("uid-uid")
	gpuAddon.Finalizers = []string{common.GlobalConfig.AddonID}
	now := metav1.NewTime(time.Now())
	gpuAddon.ObjectMeta.DeletionTimestamp = &now

	// This addon's CSV
	gpuaddoncsv := common.NewCsv(common.GlobalConfig.AddonNamespace, common.GlobalConfig.AddonID, "")

	ownerRef := []metav1.OwnerReference{}
	ownerRef = append(ownerRef, metav1.OwnerReference{
		UID:        gpuAddon.UID,
		Kind:       gpuAddon.Kind,
		APIVersion: gpuAddon.APIVersion,
		Name:       gpuAddon.Name,
	})
	// NFD
	nfdCr := &nfdv1.NodeFeatureDiscovery{}
	nfdCr.Name = common.GlobalConfig.NfdCrName
	nfdCr.Namespace = gpuAddon.Namespace
	nfdCr.ObjectMeta.OwnerReferences = ownerRef
	// ClusterPolicy
	clusterPolicy := &gpuv1.ClusterPolicy{}
	clusterPolicy.Name = common.GlobalConfig.ClusterPolicyName
	clusterPolicy.Namespace = gpuAddon.Namespace
	clusterPolicy.ObjectMeta.OwnerReferences = ownerRef

	r := newTestGPUAddonReconciler(gpuAddon, gpuaddoncsv, nfdCr, clusterPolicy)

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
