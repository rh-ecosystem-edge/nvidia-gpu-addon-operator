package common

import (
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ContainCondition(conditions []metav1.Condition, cond_type string, cond_status metav1.ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Type == cond_type {
			return condition.Status == cond_status
		}
	}
	return false
}

func NewCsv(namespace string, name string, almExample string) *operatorsv1alpha1.ClusterServiceVersion {
	csv := &operatorsv1alpha1.ClusterServiceVersion{}
	csv.Name = name
	csv.Namespace = namespace
	csv.ObjectMeta.Annotations = map[string]string{
		"alm-examples": almExample,
	}
	return csv
}
