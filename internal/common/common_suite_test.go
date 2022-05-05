package common

import (
	"testing"

	configv1 "github.com/openshift/api/config/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestCommon(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Common Suite")
}

var _ = BeforeSuite(func() {
	ProcessConfig()
})

var _ = Describe("common.go | Common utils", func() {

	Context("SliceContainsString tests", func() {
		slice := []string{"foo", "bar"}
		It("Should return false when slice does not contain string", func() {
			res := SliceContainsString(slice, "baz")
			Expect(res).To(BeFalse())
		})
		It("Should Return true when string found in slice", func() {
			res := SliceContainsString(slice, "bar")
			Expect(res).To(BeTrue())
		})
	})

	Context("SliceRemoveString tests", func() {
		slice := []string{"foo", "bar"}
		sliceOriginalLen := len(slice)
		It("Should be non destructive to original slice", func() {
			res := SliceRemoveString(slice, "foo")
			Expect(len(slice)).To(Equal(sliceOriginalLen))
			Expect(len(res)).To(Equal(sliceOriginalLen - 1))
			Expect(res[0]).To(Equal("bar"))
			res = SliceRemoveString(res, "bar")
			Expect(len(res)).To(Equal(0))
		})
		It("Return a copy of same slice if nothing was removed", func() {
			res := SliceRemoveString(slice, "baz")
			Expect(slice).To(Equal(res))
			Expect(len(slice)).To(Equal(sliceOriginalLen))
		})
	})

	Context("NewCondition function tests", func() {
		It("Should populate time and data", func() {
			condition := NewCondition("TestCondition", "True", "", "")
			Expect(condition.LastTransitionTime.IsZero()).To(BeFalse())
			Expect(condition.Type).To(Equal("TestCondition"))
		})
	})

	Context("GetOpenshiftVersion", func() {
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
		Expect(configv1.AddToScheme(scheme)).ShouldNot(HaveOccurred())

		It("should return the 'major.minor' version of the OpenShift cluster", func() {
			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(clusterVersion).
				Build()

			v, err := GetOpenShiftVersion(c)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(v).To(Equal("4.9"))
		})
	})

	Context("IsOpenshiftVersionAtLeast", func() {
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
		Expect(configv1.AddToScheme(scheme)).ShouldNot(HaveOccurred())

		Context("when OpenShift version is greater or equal than the requested one", func() {
			It("should return true", func() {
				c := fake.
					NewClientBuilder().
					WithScheme(scheme).
					WithRuntimeObjects(clusterVersion).
					Build()

				v, err := IsOpenShiftVersionAtLeast(c, "4.9")
				Expect(err).ShouldNot(HaveOccurred())
				Expect(v).To(BeTrue())
			})
		})

		Context("when OpenShift version is less than the requested one", func() {
			It("should return false", func() {
				c := fake.
					NewClientBuilder().
					WithScheme(scheme).
					WithRuntimeObjects(clusterVersion).
					Build()

				v, err := IsOpenShiftVersionAtLeast(c, "4.10")
				Expect(err).ShouldNot(HaveOccurred())
				Expect(v).To(BeFalse())
			})
		})

		Context("when the requested version is not valid", func() {
			It("should return false", func() {
				c := fake.
					NewClientBuilder().
					WithScheme(scheme).
					WithRuntimeObjects(clusterVersion).
					Build()

				v, err := IsOpenShiftVersionAtLeast(c, "4.invalid")
				Expect(err).Should(HaveOccurred())
				Expect(v).To(BeFalse())
			})
		})
	})
})

var _ = Describe("CSV Utils", func() {
	Context("Fetching CSV", func() {
		It("Should return an error when not found", func() {
			c := newClientWith()
			output, err := GetCsvWithPrefix(c, "test-namespace", "test")
			Expect(output).To(BeNil())
			Expect(err).Should(HaveOccurred())
		})
		It("Happy flow", func() {
			csv := operatorsv1alpha1.ClusterServiceVersion{}
			csv.Namespace = "test-ns"
			csv.Name = "test-2345433"
			c := newClientWith(&csv)
			output, err := GetCsvWithPrefix(c, "test-ns", "test")
			Expect(err).ShouldNot(HaveOccurred())
			Expect(output.Name).To(Equal(csv.Name))
		})
	})
})

func newClientWith(objs ...runtime.Object) client.Client {
	s := scheme.Scheme
	Expect(operatorsv1alpha1.AddToScheme(s)).ShouldNot(HaveOccurred())
	return fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
}
