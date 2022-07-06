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
	// NFD_CSV_PREFIX
	NfdCsvPrefix string `envconfig:"NFD_CSV_PREFIX" default:"nfd"`

	// NFD_CSV_NAMESPACE
	NfdCsvNamespace string `envconfig:"NFD_CSV_NAMESPACE" default:"redhat-nvidia-gpu-addon"`

	// GPU_CSV_PREFIX
	GpuCsvPrefix string `envconfig:"GPU_CSV_PREFIX" default:"gpu-operator-certified"`

	// GPU_CSV_NAMESPACE
	GpuCsvNamespace string `envconfig:"GPU_CSV_NAMESPACE" default:"redhat-nvidia-gpu-addon"`

	// WATCH_NAMESPACE
	AddonNamespace string `envconfig:"WATCH_NAMESPACE" default:"redhat-nvidia-gpu-addon"`

	// ADDON_ID
	AddonID string `envconfig:"ADDON_ID" default:"nvidia-gpu-addon"`

	// CLUSTER_POLICY_NAME
	ClusterPolicyName string `envconfig:"CLUSTER_POLICY_NAME" default:"ocp-gpu-addon"`

	// NFD_CR_NAME
	NfdCrName string `envconfig:"NFD_CR_NAME" default:"ocp-gpu-addon"`

	// RELATED_IMAGE_PLUGIN_IMAGE
	ConsolePluginImage string `envconfig:"RELATED_IMAGE_CONSOLE_PLUGIN" default:"quay.io/edge-infrastructure/console-plugin-nvidia-gpu@sha256:cec17462944cb2f800e7477101e0470c5f7a07998c012ef7470e14993ebebf40"`

	// PAGER_DUTY_SECRET_NAME
	PagerDutySecretName string `envconfig:"PAGER_DUTY_SECRET_NAME" default:"pagerduty"`

	// DEAD_MANS_SNITCH_SECRET_NAME
	DeadMansSnitchSecretName string `envconfig:"PAGE_RDUTY_SECRET_NAME" default:"deadmanssnitch"`
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

func IsOpenShiftVersionAtLeast(client client.Client, v string) (bool, error) {
	version, err := utilversion.ParseGeneric(v)
	if err != nil {
		return false, err
	}

	clusterVersion := &configv1.ClusterVersion{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: "version"}, clusterVersion)
	if err != nil {
		return false, err
	}

	for _, condition := range clusterVersion.Status.History {
		if condition.State != "Completed" {
			continue
		}

		ocpVersion, err := utilversion.ParseGeneric(condition.Version)
		if err != nil {
			return false, err
		}

		return ocpVersion.AtLeast(version), nil
	}

	return false, fmt.Errorf("failed to find Completed Cluster Version")
}
