package common

import (
	"context"
	"fmt"
	"strings"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const almExamples = "alm-examples"

func GetCsvWithPrefix(c client.Client, namespace string, prefix string) (*operatorsv1alpha1.ClusterServiceVersion, error) {

	csvs := operatorsv1alpha1.ClusterServiceVersionList{}
	err := c.List(context.TODO(), &csvs, &client.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	for _, csv := range csvs.Items {
		if strings.HasPrefix(csv.Name, prefix) {
			return &csv, nil
		}
	}
	return nil, k8serrors.NewNotFound(schema.ParseGroupResource(""), fmt.Sprintf("%v/%v*", namespace, prefix))
}

func GetAlmExamples(csv *operatorsv1alpha1.ClusterServiceVersion) (string, error) {
	annotations := csv.ObjectMeta.GetAnnotations()
	if val, ok := annotations[almExamples]; ok {
		return val, nil
	}

	return "", fmt.Errorf("%s not found in given csv %v", almExamples, csv)
}
