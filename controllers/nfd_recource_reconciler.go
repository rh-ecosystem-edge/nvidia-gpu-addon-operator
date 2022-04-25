package controllers

import (
	"context"
	"fmt"

	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
)

type NFDResourceReconciler struct{}

var _ ResourceReconciler = &NFDResourceReconciler{}

func (r *NFDResourceReconciler) Reconcile(
	ctx context.Context,
	client client.Client,
	gpuAddon *addonv1alpha1.GPUAddon) ([]metav1.Condition, error) {

	logger := log.FromContext(ctx, "Reconcile Step", "NFD CR")
	conditions := []metav1.Condition{}
	nfd := &nfdv1.NodeFeatureDiscovery{}
	err := client.Get(ctx, types.NamespacedName{
		Namespace: gpuAddon.Namespace,
		Name:      common.GlobalConfig.NfdCrName,
	}, nfd)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			// Need to create
			err = createOwnedCrFromCsvAlmExample(ctx, client, crFromCsvOptions{
				csvPrefix:    common.GlobalConfig.NfdCsvPrefix,
				csvNamespace: common.GlobalConfig.NfdCsvNamespace,
				crName:       common.GlobalConfig.NfdCrName,
				crNamespace:  gpuAddon.Namespace,
				owner:        *gpuAddon,
			})
			logger.Error(err, "Failed to deploy NodeFeatureDiscovery")
			if err != nil {
				conditions = append(conditions, common.NewCondition("NodeFeatureDiscoveryDeployed", "False", "CreateCrFail", "Failed to create NodeFeatureDiscovery CR"))
			} else {
				conditions = append(conditions, common.NewCondition("NodeFeatureDiscoveryDeployed", "True", "CreateCrSuccess", "NodeFeatureDiscovery deployed successfully"))
			}
			return conditions, err
		} else {
			logger.Error(err, "Failed to fetch NodeFeatureDiscovery CR. Client error")
			conditions = append(conditions, common.NewCondition("NodeFeatureDiscoveryDeployed", "False", "FetchCrFail", "Failed to fetch NodeFeatureDiscovery CR"))
			return conditions, err
		}
	}
	// We have an NFD CR in namespace.
	conditions = append(conditions, common.NewCondition("NodeFeatureDiscoveryDeployed", "True", "CreateCrSuccess", "NodeFeatureDiscovery deployed successfully"))
	// TODO update logic if needed.
	// TODO Check state

	logger.Info(fmt.Sprintf("NFD CR already exists in namespace %v. Named %v", gpuAddon.Namespace, nfd.Name))
	return conditions, nil
}
