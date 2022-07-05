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

package gpuaddon

import (
	"context"
	"fmt"
	"strings"

	consolev1alpha1 "github.com/openshift/api/console/v1alpha1"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
)

// GPUAddonReconciler reconciles a GPUAddon object
type GPUAddonReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// List of other resources managed by this operator.
var resourceOrderedReconcilers = []ResourceReconciler{
	&NFDResourceReconciler{},
	&SubscriptionResourceReconciler{},
	&ClusterPolicyResourceReconciler{},
	&ConsolePluginResourceReconciler{},
}

//+kubebuilder:rbac:groups=nvidia.addons.rh-ecosystem-edge.io,namespace=system,resources=gpuaddons,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nvidia.addons.rh-ecosystem-edge.io,namespace=system,resources=gpuaddons/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nvidia.addons.rh-ecosystem-edge.io,namespace=system,resources=gpuaddons/finalizers,verbs=update
//+kubebuilder:rbac:groups=nvidia.com,resources=clusterpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nfd.openshift.io,namespace=system,resources=nodefeaturediscoveries,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operators.coreos.com,namespace=system,resources=clusterserviceversions,verbs=get;list;watch;delete
//+kubebuilder:rbac:groups=config.openshift.io,resources=clusterversions,verbs=get;list;watch
//+kubebuilder:rbac:groups=operators.coreos.com,namespace=system,resources=subscriptions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=console.openshift.io,resources=consoleplugins,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.openshift.io,resources=consoles,verbs=get;list;watch;patch
//+kubebuilder:rbac:groups=apps,namespace=system,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",namespace=system,resources=services,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the GPUAddon object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *GPUAddonReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("GPUAddon Reconcile start")
	gpuAddon := addonv1alpha1.GPUAddon{}

	if err := r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, &gpuAddon); err != nil {
		if k8serrors.IsNotFound(err) {
			logger.Info("GPUAddon config not found. Probably deleted")
			return ctrl.Result{}, nil
		}
	}

	if !gpuAddon.ObjectMeta.DeletionTimestamp.IsZero() {
		logger.Info(fmt.Sprintf("GPUAddon CR %v/%v marked for deletion", req.Namespace, req.Name))
		if controllerutil.ContainsFinalizer(&gpuAddon, common.GlobalConfig.AddonID) {

			err := r.removeOwnedResources(ctx)
			if err != nil {
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(&gpuAddon, common.GlobalConfig.AddonID)

			if err := r.Update(ctx, &gpuAddon); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
			}
		}
		return ctrl.Result{}, nil
	}
	addonConditions := []metav1.Condition{}

	if err := r.registerFinilizerIfNeeded(ctx, &gpuAddon); err != nil {
		return ctrl.Result{}, r.patchStatus(ctx, gpuAddon, addonConditions, err)
	}

	for _, rr := range resourceOrderedReconcilers {
		conditions, err := rr.Reconcile(ctx, r.Client, &gpuAddon)
		addonConditions = append(addonConditions, conditions...)
		if err != nil {
			logger.Error(err, "Reconcilation failed", "resource", gpuAddon.Name, "namespace", gpuAddon.Namespace)
			return ctrl.Result{}, r.patchStatus(ctx, gpuAddon, addonConditions, err)
		}
	}

	return ctrl.Result{}, r.patchStatus(ctx, gpuAddon, addonConditions, nil)
}

// SetupWithManager sets up the controller with the Manager.
func (r *GPUAddonReconciler) SetupWithManager(mgr ctrl.Manager) (controller.Controller, error) {
	return ctrl.NewControllerManagedBy(mgr).
		For(&addonv1alpha1.GPUAddon{}).
		Owns(&operatorsv1alpha1.Subscription{}).
		Owns(&nfdv1.NodeFeatureDiscovery{}).
		Owns(&consolev1alpha1.ConsolePlugin{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Build(r)
}

func (r *GPUAddonReconciler) patchStatus(ctx context.Context, gpuAddon addonv1alpha1.GPUAddon, conditions []metav1.Condition, err error) error {
	patch := client.MergeFrom(gpuAddon.DeepCopy())
	gpuAddon.Status.Conditions = conditions
	if err != nil {
		gpuAddon.Status.Phase = addonv1alpha1.GPUAddonPhaseFailed
	} else {
		for _, condition := range conditions {
			gpuAddon.Status.Phase = addonv1alpha1.GPUAddonPhaseReady
			if condition.Status == metav1.ConditionFalse {
				gpuAddon.Status.Phase = addonv1alpha1.GPUAddonPhaseInstalling
				break
			}
		}
		// Not relevant now - But later with updates
		for _, condition := range conditions {
			if strings.HasSuffix("Update", condition.Type) {
				gpuAddon.Status.Phase = addonv1alpha1.GPUAddonPhaseUpdating
				break
			}
		}
	}
	patchErr := r.Status().Patch(ctx, &gpuAddon, patch)
	if patchErr != nil {
		return fmt.Errorf("failed to patch status: %w", patchErr)
	}
	return err
}

func (r *GPUAddonReconciler) registerFinilizerIfNeeded(ctx context.Context, gpuAddon *addonv1alpha1.GPUAddon) error {
	if controllerutil.ContainsFinalizer(gpuAddon, common.GlobalConfig.AddonID) {
		return nil
	}

	controllerutil.AddFinalizer(gpuAddon, common.GlobalConfig.AddonID)

	if err := r.Update(ctx, gpuAddon); err != nil {
		return fmt.Errorf("failed to add finalizer: %w", err)
	}

	return nil
}

func (r *GPUAddonReconciler) removeOwnedResources(ctx context.Context) error {
	deleted := make([]bool, len(resourceOrderedReconcilers))

	for i := len(resourceOrderedReconcilers) - 1; i >= 0; i-- {
		removed, err := resourceOrderedReconcilers[i].Delete(ctx, r.Client)
		if err != nil {
			return err
		}
		deleted[i] = removed
	}

	for i := range deleted {
		if !deleted[i] {
			return fmt.Errorf("not all resources have been deleted yet, won't remove add-on CSV")
		}
	}

	if err := r.removeSelfCsv(ctx); err != nil {
		return err
	}

	return nil
}

func (r *GPUAddonReconciler) removeSelfCsv(ctx context.Context) error {
	logger := log.FromContext(ctx).WithValues("Reconcile Step", "Addon CSV Deletion")
	logger.Info("Cleanup Reconcile | Delete own CSV")

	addonCsv, err := common.GetCsvWithPrefix(r.Client, common.GlobalConfig.AddonNamespace, common.GlobalConfig.AddonID)
	if err != nil {
		return err
	}

	err = r.Delete(ctx, addonCsv)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete GPUAddon Operator CSV %s: %w", addonCsv.Name, err)
	}

	return nil
}
