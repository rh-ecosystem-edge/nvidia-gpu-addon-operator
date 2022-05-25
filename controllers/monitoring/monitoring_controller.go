/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package monitoring

import (
	"context"
	"fmt"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
)

// MonitoringReconciler reconciles the monitoring stack used by the add-on operator.
type MonitoringReconciler struct {
	client.Client

	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=nvidia.addons.rh-ecosystem-edge.io,namespace=system,resources=monitorings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nvidia.addons.rh-ecosystem-edge.io,namespace=system,resources=monitorings/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nvidia.addons.rh-ecosystem-edge.io,namespace=system,resources=monitorings/finalizers,verbs=update
//+kubebuilder:rbac:groups=monitoring.coreos.com,namespace=system,resources={alertmanagers,prometheuses,alertmanagerconfigs},verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.coreos.com,namespace=system,resources=prometheusrules,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=monitoring.coreos.com,namespace=system,resources=podmonitors,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=monitoring.coreos.com,namespace=system,resources=servicemonitors,verbs=get;list;watch;update;patch;create;delete
//+kubebuilder:rbac:groups="",namespace=system,resources=secrets,verbs=create;get;list;watch;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *MonitoringReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	monitoring := addonv1alpha1.Monitoring{}

	if err := r.Client.Get(ctx, types.NamespacedName{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, &monitoring); err != nil {

		if k8serrors.IsNotFound(err) {
			logger.Info("Monitoring CR not found. Probably deleted.")
			return ctrl.Result{}, nil
		}

		return ctrl.Result{Requeue: true}, fmt.Errorf("could not get Monitoring CR: %v", err)
	}

	if !monitoring.ObjectMeta.DeletionTimestamp.IsZero() {
		if err := r.removeOwnedResources(ctx, &monitoring); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if err := r.reconcilePrometheusKubeRBACProxyConfigMap(ctx, &monitoring); err != nil {
		logger.Error(err, "Reconcilation failed",
			"resource", prometheusKubeRBACProxyConfigMapName,
			"namespace", monitoring.Namespace)
		return ctrl.Result{}, err
	}

	if err := r.reconcilePrometheusService(ctx, &monitoring); err != nil {
		logger.Error(err, "Reconcilation failed",
			"resource", prometheusServiceName,
			"namespace", monitoring.Namespace)
		return ctrl.Result{}, err
	}

	if err := r.reconcilePrometheus(ctx, &monitoring); err != nil {
		logger.Error(err, "Reconcilation failed",
			"resource", prometheusName,
			"namespace", monitoring.Namespace)
		return ctrl.Result{}, err
	}

	if err := r.reconcileAlertManager(ctx, &monitoring); err != nil {
		logger.Error(err, "Reconcilation failed",
			"resource", alertManagerName,
			"namespace", monitoring.Namespace)
		return ctrl.Result{}, err
	}

	if err := r.reconcileAlertManagerConfig(ctx, &monitoring); err != nil {
		logger.Error(err, "Reconcilation failed",
			"resource", alertManagerConfigName,
			"namespace", monitoring.Namespace)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MonitoringReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&addonv1alpha1.Monitoring{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&promv1.Prometheus{}).
		Owns(&promv1.Alertmanager{}).
		Owns(&promv1.PrometheusRule{}).
		Owns(&promv1.ServiceMonitor{}).
		Owns(&promv1alpha1.AlertmanagerConfig{}).
		Complete(r)
}

func (r *MonitoringReconciler) removeOwnedResources(
	ctx context.Context,
	m *addonv1alpha1.Monitoring) error {

	if err := r.deleteAlertManagerConfig(ctx, m); err != nil {
		return err
	}

	if err := r.deleteAlertManager(ctx, m); err != nil {
		return err
	}

	if err := r.deletePrometheus(ctx, m); err != nil {
		return err
	}

	if err := r.deletePrometheusService(ctx, m); err != nil {
		return err
	}

	if err := r.deletePrometheusKubeRBACProxyConfigMap(ctx, m); err != nil {
		return err
	}

	return nil
}
