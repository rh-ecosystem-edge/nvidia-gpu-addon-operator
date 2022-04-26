package controllers

import (
	"context"
	"fmt"

	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
	NFDDeployedCondition = "NodeFeatureDiscoveryDeployed"

	workerConfig = "core:\n  sleepInterval: 60s\nsources:\n  pci:\n    deviceClassWhitelist:\n    - \"0200\"\n    - \"03\"\n    - \"12\"\n    deviceLabelFields:\n    - \"vendor\"\n"
)

type NFDResourceReconciler struct{}

var _ ResourceReconciler = &NFDResourceReconciler{}

func (r *NFDResourceReconciler) Reconcile(
	ctx context.Context,
	client client.Client,
	gpuAddon *addonv1alpha1.GPUAddon) ([]metav1.Condition, error) {

	logger := log.FromContext(ctx, "Reconcile Step", "NFD CR")
	conditions := []metav1.Condition{}
	existingNFD := &nfdv1.NodeFeatureDiscovery{}

	err := client.Get(ctx, types.NamespacedName{
		Namespace: gpuAddon.Namespace,
		Name:      common.GlobalConfig.NfdCrName,
	}, existingNFD)

	exists := !k8serrors.IsNotFound(err)
	if err != nil && !k8serrors.IsNotFound(err) {
		conditions = append(conditions, r.getDeployedConditionFetchFailed())
		return conditions, err
	}

	nfd := &nfdv1.NodeFeatureDiscovery{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: gpuAddon.Namespace,
			Name:      common.GlobalConfig.NfdCrName,
		},
	}

	if exists {
		nfd = existingNFD
	}

	res, err := controllerutil.CreateOrPatch(context.TODO(), client, nfd, func() error {
		return r.setDesiredNFD(client, nfd, gpuAddon)
	})

	if err != nil {
		conditions = append(conditions, r.getDeployedConditionCreateFailed())
		return conditions, err
	}

	conditions = append(conditions, r.getDeployedConditionCreateSuccess())

	logger.Info("NFD reconciled successfully",
		"name", nfd.Name,
		"namespace", gpuAddon.Namespace,
		"result", res)

	return conditions, nil
}

func (r *NFDResourceReconciler) setDesiredNFD(
	client client.Client,
	nfd *nfdv1.NodeFeatureDiscovery,
	gpuAddon *addonv1alpha1.GPUAddon) error {

	if nfd == nil {
		return errors.New("nfd cannot be nil")
	}

	ocpVersion, err := common.GetOpenShiftVersion(client)
	if err != nil {
		return err
	}

	nfd.Spec = nfdv1.NodeFeatureDiscoverySpec{}

	nfd.Spec.Operand = nfdv1.OperandSpec{
		Image:           fmt.Sprintf("quay.io/openshift/origin-node-feature-discovery:%s", ocpVersion),
		ImagePullPolicy: "Always",
		ServicePort:     12000,
	}
	nfd.Spec.WorkerConfig = &nfdv1.ConfigMap{
		ConfigData: workerConfig,
	}

	if err := ctrl.SetControllerReference(gpuAddon, nfd, client.Scheme()); err != nil {
		return err
	}

	return nil
}

func (r *NFDResourceReconciler) Delete(ctx context.Context, c client.Client) error {
	nfd := &nfdv1.NodeFeatureDiscovery{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.GlobalConfig.AddonNamespace,
			Name:      common.GlobalConfig.NfdCrName,
		},
	}

	err := c.Delete(ctx, nfd)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete NodeFeatureDiscovery %s: %w", nfd.Name, err)
	}

	return nil
}

func (r *NFDResourceReconciler) getDeployedConditionFetchFailed() metav1.Condition {
	return common.NewCondition(
		NFDDeployedCondition,
		metav1.ConditionFalse,
		"FetchCrFailed",
		"Failed to fetch NFD CR")
}

func (r *NFDResourceReconciler) getDeployedConditionCreateFailed() metav1.Condition {
	return common.NewCondition(
		NFDDeployedCondition,
		metav1.ConditionFalse,
		"CreateCrFailed",
		"Failed to create NFD CR")
}

func (r *NFDResourceReconciler) getDeployedConditionCreateSuccess() metav1.Condition {
	return common.NewCondition(
		NFDDeployedCondition,
		metav1.ConditionTrue,
		"CreateCrSuccess",
		"NFD deployed successfully")
}
