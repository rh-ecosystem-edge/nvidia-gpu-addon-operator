package common

import (
	"testing"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/scheme"
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
