package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
)

const (
	alertManagerName = "gpuaddon-alertmanager"

	alertManagerConfigName = "gpuaddon-alertmanager-config"

	pagerDutyKey = "PAGERDUTY_KEY"

	snitchURLKey = "SNITCH_URL"
)

var (
	pagerdutyAlerts = []string{
		"NVIDIAGPUAddonGPUOperatorSubscriptionInstallationPending",
	}
)

func (r *MonitoringReconciler) reconcileAlertManager(
	ctx context.Context,
	m *addonv1alpha1.Monitoring) error {

	logger := log.FromContext(ctx, "Reconcile Step", "AlertManager CR")

	logger.Info("Reconciling AlertManager")
	existingAlertManager := &promv1.Alertmanager{}

	err := r.Get(ctx, types.NamespacedName{
		Name:      alertManagerName,
		Namespace: m.Namespace,
	}, existingAlertManager)

	exists := !k8serrors.IsNotFound(err)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	alertManager := &promv1.Alertmanager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertManagerName,
			Namespace: m.Namespace,
		},
	}

	if exists {
		alertManager = existingAlertManager
	}

	res, err := controllerutil.CreateOrPatch(context.TODO(), r.Client, alertManager, func() error {
		return r.setDesiredAlertManager(r.Client, alertManager, m)
	})
	if err != nil {
		return err
	}

	logger.Info("AlertManager reconciled successfully",
		"name", alertManager.Name,
		"namespace", alertManager.Namespace,
		"result", res)

	return nil
}

func (r *MonitoringReconciler) setDesiredAlertManager(
	c client.Client,
	alertManager *promv1.Alertmanager,
	m *addonv1alpha1.Monitoring) error {

	if alertManager == nil {
		return errors.New("alertManager cannot be nil")
	}

	alertManager.Spec = promv1.AlertmanagerSpec{}

	replicas := int32(3)
	alertManager.Spec.Replicas = &replicas

	alertManager.Spec.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("100m"),
			"memory": resource.MustParse("200Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("100m"),
			"memory": resource.MustParse("200Mi"),
		},
	}

	alertManager.Spec.TopologySpreadConstraints = []corev1.TopologySpreadConstraint{
		{
			MaxSkew: 1,
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "alertmanager",
				},
			},
			WhenUnsatisfiable: corev1.ScheduleAnyway,
			TopologyKey:       "kubernetes.io/hostname",
		},
	}

	if err := ctrl.SetControllerReference(m, alertManager, c.Scheme()); err != nil {
		return err
	}

	return nil
}

func (r *MonitoringReconciler) deleteAlertManager(
	ctx context.Context,
	m *addonv1alpha1.Monitoring) error {

	am := &promv1.Alertmanager{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertManagerName,
			Namespace: m.Namespace,
		},
	}

	err := r.Delete(ctx, am)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete AlertManager %s in %s: %w", am.Name, am.Namespace, err)
	}

	return nil
}

func (r *MonitoringReconciler) reconcileAlertManagerConfig(
	ctx context.Context,
	m *addonv1alpha1.Monitoring) error {

	logger := log.FromContext(ctx, "Reconcile Step", "AlertManagerConfig CR")

	logger.Info("Reconciling AlertManagerConfig")
	existingAMC := &promv1alpha1.AlertmanagerConfig{}

	err := r.Get(ctx, types.NamespacedName{
		Name:      alertManagerConfigName,
		Namespace: m.Namespace,
	}, existingAMC)

	exists := !k8serrors.IsNotFound(err)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	alertManagerConfig := &promv1alpha1.AlertmanagerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertManagerConfigName,
			Namespace: m.Namespace,
		},
	}

	if exists {
		alertManagerConfig = existingAMC
	}

	if err := r.checkPagerDutyServiceKey(ctx, common.GlobalConfig.PagerDutySecretName, m.Namespace); err != nil {
		return err
	}

	deadMansSnitchURL, err := r.getDeadMansSnitchURL(ctx, common.GlobalConfig.DeadMansSnitchSecretName, m.Namespace)
	if err != nil {
		return err
	}

	pagerDutySecretName := common.GlobalConfig.PagerDutySecretName

	res, err := controllerutil.CreateOrPatch(context.TODO(), r.Client, alertManagerConfig, func() error {
		return r.setDesiredAlertManagerConfig(r.Client, alertManagerConfig, pagerDutySecretName, deadMansSnitchURL, m)
	})
	if err != nil {
		return err
	}

	logger.Info("AlertManagerConfig reconciled successfully",
		"name", alertManagerConfig.Name,
		"namespace", alertManagerConfig.Namespace,
		"result", res)

	return nil
}

func (r *MonitoringReconciler) setDesiredAlertManagerConfig(
	c client.Client,
	alertManagerConfig *promv1alpha1.AlertmanagerConfig,
	pagerDutySecretName string,
	deadMansSnitchURL string,
	m *addonv1alpha1.Monitoring) error {

	if alertManagerConfig == nil {
		return errors.New("alertManagerConfig cannot be nil")
	}

	pagerDutyRoute, err := convertToApiExtV1JSON(promv1alpha1.Route{
		GroupBy:        []string{"alertname"},
		GroupWait:      "30s",
		GroupInterval:  "5m",
		RepeatInterval: "12h",
		Matchers: []promv1alpha1.Matcher{
			{
				Name:      "alertname",
				Value:     getRegexMatcher(pagerdutyAlerts),
				MatchType: promv1alpha1.MatchRegexp,
			},
		},
		Receiver: "pagerduty",
	})
	if err != nil {
		return err
	}

	deadMansSnitchRoute, err := convertToApiExtV1JSON(promv1alpha1.Route{
		GroupBy:        []string{"alertname"},
		GroupWait:      "30s",
		GroupInterval:  "5m",
		RepeatInterval: "5m",
		Matchers: []promv1alpha1.Matcher{
			{
				Name:      "alertname",
				Value:     "DeadMansSnitch",
				MatchType: promv1alpha1.MatchEqual,
			},
		},
		Receiver: "DeadMansSnitch",
	})
	if err != nil {
		return err
	}

	alertManagerConfig.Spec = promv1alpha1.AlertmanagerConfigSpec{}
	alertManagerConfig.Spec.Route = &promv1alpha1.Route{
		Receiver: "null",
		Routes: []apiextensionsv1.JSON{
			pagerDutyRoute,
			deadMansSnitchRoute,
		},
	}

	alertManagerConfig.Spec.Receivers = []promv1alpha1.Receiver{
		{
			Name: "null",
		},
		{
			Name: "pagerduty",
			PagerDutyConfigs: []promv1alpha1.PagerDutyConfig{{
				ServiceKey: &corev1.SecretKeySelector{
					Key:                  pagerDutyKey,
					LocalObjectReference: corev1.LocalObjectReference{Name: pagerDutySecretName},
				},
			}},
		},
		{
			Name:           "DeadMansSnitch",
			WebhookConfigs: []promv1alpha1.WebhookConfig{{URL: &deadMansSnitchURL}},
		},
	}

	if err := ctrl.SetControllerReference(m, alertManagerConfig, c.Scheme()); err != nil {
		return err
	}

	return nil
}

func (r *MonitoringReconciler) deleteAlertManagerConfig(
	ctx context.Context,
	m *addonv1alpha1.Monitoring) error {

	amc := &promv1alpha1.AlertmanagerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      alertManagerConfigName,
			Namespace: m.Namespace,
		},
	}

	err := r.Delete(ctx, amc)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete AlertManagerConfig %s in %s: %w", amc.Name, amc.Namespace, err)
	}

	return nil
}

func (r *MonitoringReconciler) checkPagerDutyServiceKey(ctx context.Context, secretName, namespace string) error {
	pagerDutySecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: namespace,
	}, pagerDutySecret); err != nil {
		return fmt.Errorf("unable to get PagerDuty secret %s in %s: %w", secretName, namespace, err)
	}

	if pagerDutySecret.Data == nil {
		return fmt.Errorf("no data in PagerDuty secret %s in %s", secretName, namespace)
	}

	_, exists := pagerDutySecret.Data[pagerDutyKey]
	if !exists {
		return fmt.Errorf("entry PAGERDUTY_KEY is missing from PagerDuty secret %s in %s", secretName, namespace)
	}

	return nil
}

func (r *MonitoringReconciler) getDeadMansSnitchURL(ctx context.Context, secretName, namespace string) (string, error) {
	deadMansSnitchSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      secretName,
		Namespace: namespace,
	}, deadMansSnitchSecret); err != nil {
		return "", fmt.Errorf("unable to get DeadMan's Snitch secret %s in %s: %w", secretName, namespace, err)
	}

	if deadMansSnitchSecret.Data == nil {
		return "", fmt.Errorf("no data in DeadMansSnitch secret %s in %s", secretName, namespace)
	}

	deadMansSnitchURL, exists := deadMansSnitchSecret.Data[snitchURLKey]
	if !exists {
		return "", fmt.Errorf("entry SNITCH_URL is missing from DeadMan's Snitch secret %s in %s", secretName, namespace)
	}

	return string(deadMansSnitchURL), nil
}

func convertToApiExtV1JSON(value interface{}) (apiextensionsv1.JSON, error) {
	out := apiextensionsv1.JSON{}

	raw, err := json.Marshal(value)
	if err != nil {
		return out, err
	}

	out.Raw = raw

	return out, nil
}

func getRegexMatcher(alerts []string) string {
	return "^" + strings.Join(alerts, "$|^") + "$"
}
