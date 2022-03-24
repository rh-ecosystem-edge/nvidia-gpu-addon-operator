package common

import (
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func GetCRasUnstructuredObjectFromAlmExample(almExample string, logger logr.Logger) (*unstructured.Unstructured, error) {
	var crJson unstructured.UnstructuredList
	err := json.Unmarshal([]byte(almExample), &crJson.Items)
	if err != nil {
		logger.Error(err, "Failed to unmarshal alm example")
		return nil, err
	}

	if len(crJson.Items) < 1 {
		message := fmt.Sprintf("list crs is empty, something gone wrong. alm example %s", almExample)
		err = fmt.Errorf(message)
		logger.Error(err, message)
		return nil, fmt.Errorf(message)
	}

	return &crJson.Items[0], nil
}
