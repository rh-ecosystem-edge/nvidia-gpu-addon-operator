package controllers

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	gpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type ClusterPolicyResourceReconciler struct{}

var _ ResourceReconciler = &ClusterPolicyResourceReconciler{}

func (r *ClusterPolicyResourceReconciler) Reconcile(ctx context.Context, client client.Client, gpuAddon *addonv1alpha1.GPUAddon) ([]metav1.Condition, error) {
	logger := log.FromContext(ctx, "Reconcile Step", "ClusterPolicy CR")
	conditions := []metav1.Condition{}
	clusterpolicy := &gpuv1.ClusterPolicy{}
	err := client.Get(ctx, types.NamespacedName{
		Namespace: gpuAddon.Namespace,
		Name:      common.GlobalConfig.NfdCrName,
	}, clusterpolicy)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Need to create
			err = createOwnedCrFromCsvAlmExample(ctx, client, crFromCsvOptions{
				csvPrefix:    common.GlobalConfig.GpuCsvPrefix,
				csvNamespace: common.GlobalConfig.GpuCsvNamespace,
				crName:       common.GlobalConfig.ClusterPolicyName,
				crNamespace:  gpuAddon.Namespace,
				owner:        *gpuAddon,
			})
			logger.Error(err, "Failed to deploy ClusterPolicy")
			if err != nil {
				conditions = append(conditions, common.NewCondition("ClusterPolicyDeployed", "False", "CreateCrFail", "Failed to create ClusterPolicy CR"))
			} else {
				conditions = append(conditions, common.NewCondition("ClusterPolicyDeployed", "True", "CreateCrSuccess", "ClusterPolicy deployed successfully"))
			}
			return conditions, err
		} else {
			logger.Error(err, "Failed to fetch ClusterPolicy CR. Client error")
			conditions = append(conditions, common.NewCondition("ClusterPolicyDeployed", "False", "FetchCrFail", "Failed to fetch ClusterPolicy CR"))
			return conditions, err
		}
	}
	// We have an NFD CR in namespace.
	conditions = append(conditions, common.NewCondition("ClusterPolicyDeployed", "True", "CreateCrSuccess", "ClusterPolicy deployed successfully"))
	// TODO update logic if needed.
	// TODO Check state

	logger.Info(fmt.Sprintf("ClusterPolicy CR already exists in namespace %v. Named %v", gpuAddon.Namespace, clusterpolicy.Name))
	return conditions, nil
}
