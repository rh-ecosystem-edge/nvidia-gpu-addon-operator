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

package controllers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/pkg/errors"
	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ConfigMapReconciler reconciles a ConfigMap object
type ConfigMapReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

var AddonDeleteLabel string = fmt.Sprintf("%v-delete", common.GlobalConfig.AddonLabel)

//+kubebuilder:rbac:groups="",namespace=redhat-nvidia-gpu-addon,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",namespace=redhat-nvidia-gpu-addon,resources=configmaps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups="",namespace=redhat-nvidia-gpu-addon,resources=configmaps/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ConfigMap object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *ConfigMapReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	configmap := v1.ConfigMap{}

	if err := r.Client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, &configmap); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("ConfigMap not found. Probably deleted")
			return ctrl.Result{}, nil
		}

		return ctrl.Result{Requeue: true}, fmt.Errorf("Unable to get ConfigMap: %v", err)
	}
	labels := configmap.Labels
	if labels == nil {
		logger.Info("No labels attached to ConfigMap. Ignoring")
		return ctrl.Result{}, nil
	}

	if val, ok := labels[AddonDeleteLabel]; ok {
		toBeDeleted, err := strconv.ParseBool(val)
		if err != nil {
			logger.Error(err, "Invalid value in ConfigMap add-on delete label")
		}
		if toBeDeleted {
			return ctrl.Result{}, errors.Wrap(r.deleteGpuAddonCr(ctx, req.Namespace), "Failed to delete GPUAddon CR")
		}
	}

	logger.Info("Nothing left to do.")
	return ctrl.Result{}, nil
}

func (r *ConfigMapReconciler) deleteGpuAddonCr(ctx context.Context, namespace string) error {
	logger := log.FromContext(ctx).WithValues("Reconcile Step", "DeleteGpuAddonCr")
	logger.Info("Getting GPUAddon CR")
	gpuAddonCrs := addonv1alpha1.GPUAddonList{}
	err := r.Client.List(ctx, &gpuAddonCrs)
	if err != nil {
		return err
	}
	if len(gpuAddonCrs.Items) > 1 {
		logger.Info(fmt.Sprintf("In namespace %v there are multiple (%v) GPUAddon CRs.", namespace, len(gpuAddonCrs.Items)))
	}
	for _, addonCr := range gpuAddonCrs.Items {
		if err := r.Client.Delete(ctx, &addonCr); err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.ConfigMap{}).
		WithEventFilter(configMapFilter()).
		Complete(r)
}

func configMapFilter() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			return isAddonConfigMap(e.ObjectNew)
		},
		CreateFunc: func(e event.CreateEvent) bool {
			return isAddonConfigMap(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return isAddonConfigMap(e.Object)
		},
	}
}

func isAddonConfigMap(object client.Object) bool {
	if object.GetName() == common.GlobalConfig.AddonID && object.GetNamespace() == common.GlobalConfig.AddonNamespace {
		return true
	}
	return false
}
