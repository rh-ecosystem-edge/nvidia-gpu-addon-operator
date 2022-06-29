package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
)

const (
	prometheusName = "gpuaddon-prometheus"

	prometheusServingCertSecretName = "prometheus-serving-cert-secret"

	prometheusKubeRBACProxyConfigMapName = "prometheus-kube-rbac-proxy-config"

	kubeRBACProxyPort = 9339

	prometheusServiceName = "gpuaddon-prometheus-service"
)

func (r *MonitoringReconciler) reconcilePrometheus(
	ctx context.Context,
	m *addonv1alpha1.Monitoring) error {

	logger := log.FromContext(ctx, "Reconcile Step", "Prometheus CR")
	logger.Info("Reconciling Prometheus")

	existingPrometheus := &promv1.Prometheus{}

	err := r.Get(ctx, types.NamespacedName{
		Name:      prometheusName,
		Namespace: m.Namespace,
	}, existingPrometheus)

	exists := !k8serrors.IsNotFound(err)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	prometheus := &promv1.Prometheus{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheusName,
			Namespace: m.Namespace,
		},
	}

	if exists {
		prometheus = existingPrometheus
	}

	res, err := controllerutil.CreateOrPatch(context.TODO(), r.Client, prometheus, func() error {
		return r.setDesiredPrometheus(r.Client, prometheus, m)
	})
	if err != nil {
		return err
	}

	logger.Info("Prometheus reconciled successfully",
		"name", prometheus.Name,
		"namespace", prometheus.Namespace,
		"result", res)

	return nil
}

func (r *MonitoringReconciler) setDesiredPrometheus(
	c client.Client,
	prometheus *promv1.Prometheus,
	m *addonv1alpha1.Monitoring) error {

	if prometheus == nil {
		return errors.New("prometheus cannot be nil")
	}

	selector := metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{Key: "app", Operator: metav1.LabelSelectorOpExists},
		},
	}

	prometheus.Spec = promv1.PrometheusSpec{
		CommonPrometheusFields: promv1.CommonPrometheusFields{
			ServiceAccountName:     "prometheus-k8s",
			ServiceMonitorSelector: &selector,
			PodMonitorSelector:     &selector,
			EnableAdminAPI:         false,
			ListenLocal:            true,
		},
		RuleNamespaceSelector: &selector,
	}

	prometheus.Spec.Alerting = &promv1.AlertingSpec{
		Alertmanagers: []promv1.AlertmanagerEndpoints{{
			Namespace: prometheus.Namespace,
			Name:      "alertmanager-operated",
			Port:      intstr.FromString("web"),
		}},
	}

	prometheus.Spec.Resources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("1"),
			"memory": resource.MustParse("250Mi"),
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("1"),
			"memory": resource.MustParse("250Mi"),
		},
	}

	prometheus.Spec.Containers = []corev1.Container{
		{
			// TODO: adjust image tag to openshift version
			Image: "quay.io/openshift/origin-kube-rbac-proxy:4.10.0",
			Name:  "kube-rbac-proxy",
			Args: []string{
				fmt.Sprintf("--secure-listen-address=0.0.0.0:%d", kubeRBACProxyPort),
				"--upstream=http://127.0.0.1:9090/",
				"--logtostderr=true",
				"--v=10",
				"--tls-cert-file=/etc/tls-secret/tls.crt",
				"--tls-private-key-file=/etc/tls-secret/tls.key",
				"--client-ca-file=/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt",
				"--config-file=/etc/kube-rbac-config/config-file.json",
			},
			Ports: []corev1.ContainerPort{{
				Name:          "https",
				ContainerPort: kubeRBACProxyPort,
			}},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "serving-cert",
					MountPath: "/etc/tls-secret",
				},
				{
					Name:      "kube-rbac-config",
					MountPath: "/etc/kube-rbac-config",
				},
			},
		},
	}

	prometheus.Spec.Volumes = []corev1.Volume{
		{
			Name: "serving-cert",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: prometheusServingCertSecretName,
				},
			},
		},
		{
			Name: "kube-rbac-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: prometheusKubeRBACProxyConfigMapName,
					},
				},
			},
		},
	}

	// r.prometheus.Spec.AdditionalAlertRelabelConfigs = &corev1.SecretKeySelector{
	// 	LocalObjectReference: corev1.LocalObjectReference{
	// 		Name: alertRelabelConfigSecretName,
	// 	},
	// 	Key: alertRelabelConfigSecretKey,
	// }

	if err := ctrl.SetControllerReference(m, prometheus, c.Scheme()); err != nil {
		return err
	}

	return nil
}

func (r *MonitoringReconciler) deletePrometheus(
	ctx context.Context,
	m *addonv1alpha1.Monitoring) error {

	p := &promv1.Prometheus{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheusName,
			Namespace: m.Namespace,
		},
	}

	err := r.Delete(ctx, p)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete Prometheus %s in %s: %w", p.Name, p.Namespace, err)
	}

	return nil
}

func (r *MonitoringReconciler) reconcilePrometheusKubeRBACProxyConfigMap(
	ctx context.Context,
	m *addonv1alpha1.Monitoring) error {

	logger := log.FromContext(ctx, "Reconcile Step", "Prometheus KubeRBACProxy ConfigMap")
	logger.Info("Reconciling Prometheus KubeRBACProxy ConfigMap")

	existingCM := &corev1.ConfigMap{}

	err := r.Get(ctx, types.NamespacedName{
		Name:      prometheusKubeRBACProxyConfigMapName,
		Namespace: m.Namespace,
	}, existingCM)

	exists := !k8serrors.IsNotFound(err)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheusKubeRBACProxyConfigMapName,
			Namespace: m.Namespace,
		},
	}

	if exists {
		cm = existingCM
	}

	res, err := controllerutil.CreateOrPatch(context.TODO(), r.Client, cm, func() error {
		return r.setDesiredPrometheusKubeRBACProxyConfigMap(r.Client, cm, m)
	})
	if err != nil {
		return err
	}

	logger.Info("Prometheus KubeRBACProxy ConfigMap reconciled successfully",
		"name", cm.Name,
		"namespace", cm.Namespace,
		"result", res)

	return nil
}

func (r *MonitoringReconciler) setDesiredPrometheusKubeRBACProxyConfigMap(
	c client.Client,
	cm *corev1.ConfigMap,
	m *addonv1alpha1.Monitoring) error {

	if cm == nil {
		return errors.New("configmap cannot be nil")
	}

	cm.Data = map[string]string{
		"config-file.json": (func() string {
			config := struct {
				Authorization struct {
					Static [2]struct {
						Path            string `json:"path"`
						ResourceRequest bool   `json:"resourceRequest"`
						Verb            string `json:"verb"`
					} `json:"static"`
				} `json:"authorization"`
			}{}

			item := &config.Authorization.Static[0]
			item.Verb = "get"
			item.Path = "/metrics"
			item.ResourceRequest = false

			item = &config.Authorization.Static[1]
			item.Verb = "get"
			item.Path = "/federate"
			item.ResourceRequest = false

			raw, _ := json.Marshal(config)
			return string(raw)
		})(),
	}

	if err := ctrl.SetControllerReference(m, cm, c.Scheme()); err != nil {
		return err
	}

	return nil
}

func (r *MonitoringReconciler) deletePrometheusKubeRBACProxyConfigMap(
	ctx context.Context,
	m *addonv1alpha1.Monitoring) error {

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheusKubeRBACProxyConfigMapName,
			Namespace: m.Namespace,
		},
	}

	err := r.Delete(ctx, cm)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete Prometheus KubeRBACProxy ConfigMap %s in %s: %w", cm.Name, cm.Namespace, err)
	}

	return nil
}

func (r *MonitoringReconciler) reconcilePrometheusService(
	ctx context.Context,
	m *addonv1alpha1.Monitoring) error {

	logger := log.FromContext(ctx, "Reconcile Step", "Prometheus Service")
	logger.Info("Reconciling Prometheus Service")

	existingService := &corev1.Service{}

	err := r.Get(ctx, types.NamespacedName{
		Name:      prometheusServiceName,
		Namespace: m.Namespace,
	}, existingService)

	exists := !k8serrors.IsNotFound(err)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheusServiceName,
			Namespace: m.Namespace,
		},
	}

	if exists {
		s = existingService
	}

	res, err := controllerutil.CreateOrPatch(context.TODO(), r.Client, s, func() error {
		return r.setDesiredPrometheusService(r.Client, s, m)
	})
	if err != nil {
		return err
	}

	logger.Info("Prometheus Service reconciled successfully",
		"name", s.Name,
		"namespace", s.Namespace,
		"result", res)

	return nil
}

func (r *MonitoringReconciler) setDesiredPrometheusService(
	c client.Client,
	s *corev1.Service,
	m *addonv1alpha1.Monitoring) error {

	if s == nil {
		return errors.New("service cannot be nil")
	}

	s.Spec.Ports = []corev1.ServicePort{
		{
			Name:       "https",
			Protocol:   corev1.ProtocolTCP,
			Port:       int32(kubeRBACProxyPort),
			TargetPort: intstr.FromString("https"),
		},
	}

	s.Spec.Selector = map[string]string{
		"app.kubernetes.io/name": "prometheus",
	}

	annotations := s.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
		s.SetAnnotations(annotations)
	}

	annotations["service.beta.openshift.io/serving-cert-secret-name"] = prometheusServingCertSecretName
	annotations["service.alpha.openshift.io/serving-cert-secret-name"] = prometheusServingCertSecretName

	if err := ctrl.SetControllerReference(m, s, c.Scheme()); err != nil {
		return err
	}

	return nil
}

func (r *MonitoringReconciler) deletePrometheusService(
	ctx context.Context,
	m *addonv1alpha1.Monitoring) error {

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      prometheusServiceName,
			Namespace: m.Namespace,
		},
	}

	err := r.Delete(ctx, s)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete Prometheus Service %s in %s: %w", s.Name, s.Namespace, err)
	}

	return nil
}
