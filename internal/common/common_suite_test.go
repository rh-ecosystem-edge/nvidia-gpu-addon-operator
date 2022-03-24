package common

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/scheme"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
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

	Context("CSV ALM Examples", func() {
		It("Happy Flow", func() {
			csv := operatorsv1alpha1.ClusterServiceVersion{}
			csv.Namespace = "test-ns"
			csv.Name = "test-2345433"
			csv.ObjectMeta.Annotations = map[string]string{
				"alm-examples": "[{}]",
			}
			example, err := GetAlmExamples(&csv)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(example).To(Equal("[{}]"))
		})

		It("Should return An error with invalid CSV examples", func() {
			csv := operatorsv1alpha1.ClusterServiceVersion{}
			csv.Namespace = "test-ns"
			csv.Name = "test-2345433"
			_, err := GetAlmExamples(&csv)
			Expect(err).Should(HaveOccurred())
		})
	})
})

var _ = Describe("CR Utils", func() {
	Context("Unstructured Object parsing", func() {
		DescribeTable("Should return error when invalid input",
			func(almExample string) {
				logger := ctrl.Log.WithName("Test")
				_, err := GetCRasUnstructuredObjectFromAlmExample(almExample, logger)
				Expect(err).Should(HaveOccurred())
			},
			Entry("Empty string", ""),
			Entry("Not a JSON array", "{}"),
			Entry("Empty JSON array", "[]"),
		)

		It("Happy Flow", func() {
			logger := ctrl.Log.WithName("Test")
			obj, err := GetCRasUnstructuredObjectFromAlmExample(`[
		{
          "apiVersion": "v1",
          "kind": "Namespace",
          "metadata": {
            "name": "gpu-addon"
        	}
		}

	]`, logger)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(obj.GetName()).To(Equal("gpu-addon"))
			Expect(obj.GetKind()).To(Equal("Namespace"))
		})
	})
})

func newClientWith(objs ...runtime.Object) client.Client {
	s := scheme.Scheme
	Expect(operatorsv1alpha1.AddToScheme(s)).ShouldNot(HaveOccurred())
	return fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()
}
