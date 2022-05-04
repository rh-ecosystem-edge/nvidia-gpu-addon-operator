package controllers

import (
	"context"
	"errors"
	"fmt"

	gpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
)

const (
	ClusterPolicyDeployedCondition = "ClusterPolicyDeployed"
)

type ClusterPolicyResourceReconciler struct{}

var _ ResourceReconciler = &ClusterPolicyResourceReconciler{}

func (r *ClusterPolicyResourceReconciler) Reconcile(
	ctx context.Context,
	c client.Client,
	gpuAddon *addonv1alpha1.GPUAddon) ([]metav1.Condition, error) {

	logger := log.FromContext(ctx, "Reconcile Step", "ClusterPolicy CR")
	conditions := []metav1.Condition{}
	existingCP := &gpuv1.ClusterPolicy{}

	err := c.Get(ctx, client.ObjectKey{
		Name: common.GlobalConfig.NfdCrName,
	}, existingCP)

	exists := !k8serrors.IsNotFound(err)
	if err != nil && !k8serrors.IsNotFound(err) {
		conditions = append(conditions, r.getDeployedConditionFetchFailed())
		return conditions, err
	}

	cp := &gpuv1.ClusterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.GlobalConfig.ClusterPolicyName,
		},
	}

	if exists {
		cp = existingCP
	}

	res, err := controllerutil.CreateOrPatch(context.TODO(), c, cp, func() error {
		return r.setDesiredClusterPolicy(c, cp, gpuAddon)
	})

	if err != nil {
		conditions = append(conditions, r.getDeployedConditionCreateFailed())
		return conditions, err
	}

	conditions = append(conditions, r.getDeployedConditionCreateSuccess())

	logger.Info("ClusterPolicy reconciled successfully",
		"name", cp.Name,
		"result", res)

	return conditions, nil
}

func (r *ClusterPolicyResourceReconciler) setDesiredClusterPolicy(
	c client.Client,
	cp *gpuv1.ClusterPolicy,
	gpuAddon *addonv1alpha1.GPUAddon) error {

	if cp == nil {
		return errors.New("clusterpolicy cannot be nil")
	}

	enabled := true
	disabled := false

	cp.Spec = gpuv1.ClusterPolicySpec{}

	cp.Spec.Operator = gpuv1.OperatorSpec{
		DefaultRuntime: gpuv1.CRIO,
	}

	cp.Spec.PSP = gpuv1.PSPSpec{
		Enabled: &disabled,
	}

	cp.Spec.Toolkit = gpuv1.ToolkitSpec{
		Enabled: &enabled,
	}

	cp.Spec.DCGM = gpuv1.DCGMSpec{
		Enabled: &enabled,
	}

	cp.Spec.MIGManager = gpuv1.MIGManagerSpec{
		Enabled: &enabled,
	}

	cp.Spec.NodeStatusExporter = gpuv1.NodeStatusExporterSpec{
		Enabled: &enabled,
	}

	cp.Spec.MIG = gpuv1.MIGSpec{
		Strategy: gpuv1.MIGStrategySingle,
	}

	cp.Spec.Validator = gpuv1.ValidatorSpec{
		Env: []corev1.EnvVar{
			{Name: "WITH_WORKLOAD", Value: "true"},
		},
	}

	cp.Spec.Driver = gpuv1.DriverSpec{
		Enabled: &enabled,
	}

	cp.Spec.Driver.UseOpenShiftDriverToolkit = &enabled

	cp.Spec.Driver.GPUDirectRDMA = &gpuv1.GPUDirectRDMASpec{
		Enabled: &disabled,
	}

	cp.Spec.Driver.LicensingConfig = &gpuv1.DriverLicensingConfigSpec{
		NLSEnabled: &disabled,
	}

	// IMPORTANT: cannot set a namespaced owner as a reference on a cluster-scoped resource.
	// "cluster-scoped resource must not have a namespace-scoped owner, owner's namespace x"
	// if err := ctrl.SetControllerReference(gpuAddon, cp, c.Scheme()); err != nil {
	// 	return err
	// }

	return nil
}

func (r *ClusterPolicyResourceReconciler) Delete(ctx context.Context, c client.Client) error {
	cp := &gpuv1.ClusterPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: common.GlobalConfig.ClusterPolicyName,
		},
	}

	err := c.Delete(ctx, cp)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete ClusterPolicy %s: %w", cp.Name, err)
	}

	return nil
}

func (r *ClusterPolicyResourceReconciler) getDeployedConditionFetchFailed() metav1.Condition {
	return common.NewCondition(
		ClusterPolicyDeployedCondition,
		metav1.ConditionFalse,
		"FetchCrFailed",
		"Failed to fetch ClusterPolicy CR")
}

func (r *ClusterPolicyResourceReconciler) getDeployedConditionCreateFailed() metav1.Condition {
	return common.NewCondition(
		ClusterPolicyDeployedCondition,
		metav1.ConditionFalse,
		"CreateCrFailed",
		"Failed to create ClusterPolicy CR")
}

func (r *ClusterPolicyResourceReconciler) getDeployedConditionCreateSuccess() metav1.Condition {
	return common.NewCondition(
		ClusterPolicyDeployedCondition,
		metav1.ConditionTrue,
		"CreateCrSuccess",
		"ClusterPolicy deployed successfully")
}
