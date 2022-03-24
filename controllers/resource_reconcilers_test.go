package controllers

import (
	"context"
	"fmt"

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("ResourceReconciler common functions", Ordered, func() {
	BeforeAll(func() {
		common.ProcessConfig()
	})

	Context("createOwnedCrFromCsvAlmExample tests", func() {
		owner := &addonv1alpha1.GPUAddon{}
		owner.UID = types.UID("uid")
		csv_base := &operatorsv1alpha1.ClusterServiceVersion{}
		csv_base.Name = "foo"
		csv_base.Namespace = "bar"
		It("Happy flow", func() {
			csv := csv_base.DeepCopy()
			csv.ObjectMeta.Annotations = map[string]string{
				"alm-examples": `[
					{
						"apiVersion": "v1",
						"kind": "Pod",
						"metadata": {
							"name": "gpu-addon"
						}
					}
				]`,
			}
			c := newTestClientWith(csv)
			err := createOwnedCrFromCsvAlmExample(context.TODO(), c, crFromCsvOptions{
				csvPrefix:    "foo",
				csvNamespace: "bar",
				crName:       "test-pod",
				crNamespace:  "default",
				owner:        *owner,
			})
			Expect(err).ShouldNot(HaveOccurred())
			createdObj := &v1.Pod{}
			err = c.Get(context.TODO(), types.NamespacedName{
				Name:      "test-pod",
				Namespace: "default",
			}, createdObj)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(owner.GetUID()).To(Equal(createdObj.ObjectMeta.GetOwnerReferences()[0].UID))
		})
		It("Should return error when no CSV found", func() {
			c := newTestClientWith()
			err := createOwnedCrFromCsvAlmExample(context.TODO(), c, crFromCsvOptions{
				csvPrefix:    "foo",
				csvNamespace: "bar",
				crName:       "test-pod",
				crNamespace:  "default",
				owner:        *owner,
			})
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})
		It("Should error when CSV ALM example is missing", func() {
			csv := csv_base.DeepCopy()
			c := newTestClientWith(csv)
			err := createOwnedCrFromCsvAlmExample(context.TODO(), c, crFromCsvOptions{
				csvPrefix:    "foo",
				csvNamespace: "bar",
				crName:       "test-pod",
				crNamespace:  "default",
				owner:        *owner,
			})
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeFalse())
		})
		It("Should error when CSV ALM example is invalid", func() {
			csv := csv_base.DeepCopy()
			csv.ObjectMeta.Annotations = map[string]string{
				"alm-examples": "",
			}
			c := newTestClientWith(csv)
			err := createOwnedCrFromCsvAlmExample(context.TODO(), c, crFromCsvOptions{
				csvPrefix:    "foo",
				csvNamespace: "bar",
				crName:       "test-pod",
				crNamespace:  "default",
				owner:        *owner,
			})
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeFalse())
		})
	})
})

type resourceReconcilerTestConfig struct {
	rr                       ResourceReconciler
	client                   client.Client
	gpuAddon                 *addonv1alpha1.GPUAddon
	errorExpected            bool
	expectedConditionsStates map[string]metav1.ConditionStatus
}

var _ = Describe("ResourceReconciler instance tests", func() {
	common.ProcessConfig()
	gpuAddon := &addonv1alpha1.GPUAddon{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gpu-addon-test",
			Namespace: common.GlobalConfig.AddonNamespace,
		},
	}
	Context("NFDRecourceReconciler", func() {
		nfdCsv := common.NewCsv(common.GlobalConfig.NfdCsvNamespace, fmt.Sprintf("%v-test-csv", common.GlobalConfig.NfdCsvPrefix), common.NfdAlmExample)
		tests := []TableEntry{
			Entry("Happy Flow", resourceReconcilerTestConfig{
				client:        newTestClientWith(nfdCsv),
				rr:            &NFDResourceReconciler{},
				errorExpected: false,
				expectedConditionsStates: map[string]metav1.ConditionStatus{
					"NodeFeatureDiscoveryDeployed": "True",
				},
			}),
			Entry("Error when no CSV in namespace", resourceReconcilerTestConfig{
				client:        newTestClientWith(),
				rr:            &NFDResourceReconciler{},
				errorExpected: true,
				gpuAddon:      gpuAddon,
				expectedConditionsStates: map[string]metav1.ConditionStatus{
					"NodeFeatureDiscoveryDeployed": "False",
				},
			}),
		}

		DescribeTable("Reconcile Flows", func(test resourceReconcilerTestConfig) {
			runTestCase(test, gpuAddon)
		}, tests)
	})

	Context("ClusterPolicyRecourceReconciler tests", func() {
		tests := []TableEntry{
			Entry("Error when no CSV in namespace", resourceReconcilerTestConfig{
				client:        newTestClientWith(),
				rr:            &ClusterPolicyResourceReconciler{},
				errorExpected: true,
				gpuAddon:      gpuAddon,
				expectedConditionsStates: map[string]metav1.ConditionStatus{
					"ClusterPolicyDeployed": "False",
				},
			}),
		}

		DescribeTable("Reconcile Flows", func(test resourceReconcilerTestConfig) {
			runTestCase(test, gpuAddon)
		}, tests)
	})
})

func runTestCase(test resourceReconcilerTestConfig, gpuAddon *addonv1alpha1.GPUAddon) {
	conditions, err := test.rr.Reconcile(context.TODO(), test.client, gpuAddon)
	Expect(len(conditions)).To(Equal(len(test.expectedConditionsStates)), "Returned conditions are not equal to expected length")
	for cond_type, cond_status := range test.expectedConditionsStates {
		debugMsg := fmt.Sprintf("FAIL: Condition '%v':'%v' was not met", cond_type, cond_status)
		Expect(common.ContainCondition(conditions, cond_type, cond_status)).To(BeTrue(), debugMsg)
	}
	if test.errorExpected {
		Expect(err).To(HaveOccurred())
	} else {
		Expect(err).ToNot(HaveOccurred())
	}
}

func newTestClientWith(objs ...runtime.Object) client.Client {
	s := scheme.Scheme
	Expect(v1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(nfdv1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(gpuv1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(operatorsv1alpha1.AddToScheme(s)).ShouldNot(HaveOccurred())
	return fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
}
