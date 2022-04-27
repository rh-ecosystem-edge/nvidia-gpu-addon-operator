package common

import (
	"context"
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
	configv1 "github.com/openshift/api/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type config struct {
	NfdCsvPrefix      string `envconfig:"NFD_CSV_PREFIX" default:"nfd"`
	NfdCsvNamespace   string `envconfig:"NFD_CSV_NAMESPACE" default:"redhat-nvidia-gpu-addon"`
	GpuCsvPrefix      string `envconfig:"GPU_CSV_PREFIX" default:"gpu-operator-certified"`
	GpuCsvNamespace   string `envconfig:"GPU_CSV_NAMESPACE" default:"redhat-nvidia-gpu-addon"`
	AddonNamespace    string `envconfig:"WATCH_NAMESPACE" default:"redhat-nvidia-gpu-addon"`
	AddonID           string `envconfig:"ADDON_ID" default:"nvidia-gpu-addon"`
	AddonLabel        string `envconfig:"ADDON_LABEL" default:"api.openshift.com/addon-nvidia-gpu-operator"`
	ClusterPolicyName string `envconfig:"CLUSTER_POLICY_NAME" default:"ocp-gpu-addon"`
	NfdCrName         string `envconfig:"NFD_CR_NAME" default:"ocp-gpu-addon"`
}

var GlobalConfig config

func ProcessConfig() {
	_ = envconfig.Process("nvidia-gpu-addon", &GlobalConfig)
}

func SliceContainsString(slice []string, str string) bool {
	for _, element := range slice {
		if element == str {
			return true
		}
	}
	return false
}

func SliceRemoveString(slice []string, str string) []string {
	result := []string{}
	for _, element := range slice {
		if element != str {
			result = append(result, element)
		}
	}
	return result
}

func NewCondition(cond_type string, cond_status metav1.ConditionStatus, reason string, message string) metav1.Condition {
	return metav1.Condition{
		Type:               cond_type,
		Status:             cond_status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.NewTime(time.Now()),
	}
}

func GetOpenShiftVersion(client client.Client) (string, error) {
	clusterVersion := &configv1.ClusterVersion{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: "version"}, clusterVersion)
	if err != nil {
		return "", err
	}

	for _, condition := range clusterVersion.Status.History {
		if condition.State != "Completed" {
			continue
		}

		ocpVersion, err := utilversion.ParseGeneric(condition.Version)
		if err != nil {
			return "", err
		}

		return fmt.Sprintf("%d.%d", ocpVersion.Major(), ocpVersion.Minor()), nil
	}

	return "", fmt.Errorf("failed to find Completed Cluster Version")
}
