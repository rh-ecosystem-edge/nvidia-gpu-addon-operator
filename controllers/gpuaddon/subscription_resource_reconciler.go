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
	"errors"
	"fmt"

	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
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
	SubscriptionDeployedCondition = "SubscriptionDeployed"

	packageName      = "gpu-operator-certified"
	subscriptionName = "gpu-operator-certified"
)

type SubscriptionResourceReconciler struct{}

var _ ResourceReconciler = &SubscriptionResourceReconciler{}

func (r *SubscriptionResourceReconciler) Reconcile(
	ctx context.Context,
	client client.Client,
	gpuAddon *addonv1alpha1.GPUAddon) ([]metav1.Condition, error) {

	logger := log.FromContext(ctx, "Reconcile Step", "Subscription CR")
	conditions := []metav1.Condition{}
	existingSubscription := &operatorsv1alpha1.Subscription{}

	err := client.Get(ctx, types.NamespacedName{
		Namespace: gpuAddon.Namespace,
		Name:      subscriptionName,
	}, existingSubscription)

	exists := !k8serrors.IsNotFound(err)
	if err != nil && !k8serrors.IsNotFound(err) {
		conditions = append(conditions, r.getDeployedConditionFetchFailed())
		return conditions, err
	}

	s := &operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: gpuAddon.Namespace,
			Name:      subscriptionName,
		},
	}

	if exists {
		s = existingSubscription

		if s.Status.InstalledCSV != "" {
			SubscriptionInstalled.WithLabelValues().Set(1)
		} else {
			SubscriptionInstalled.WithLabelValues().Set(0)
		}
	}

	res, err := controllerutil.CreateOrPatch(context.TODO(), client, s, func() error {
		return r.setDesiredSubscription(client, s, gpuAddon)
	})

	if err != nil {
		conditions = append(conditions, r.getDeployedConditionCreateFailed())
		return conditions, err
	}

	conditions = append(conditions, r.getDeployedConditionCreateSuccess())

	logger.Info("Subscription reconciled successfully",
		"name", s.Name,
		"namespace", s.Namespace,
		"result", res)

	return conditions, nil
}

func (r *SubscriptionResourceReconciler) setDesiredSubscription(
	client client.Client,
	s *operatorsv1alpha1.Subscription,
	gpuAddon *addonv1alpha1.GPUAddon) error {

	if s == nil {
		return errors.New("subscription cannot be nil")
	}

	ocpVersion, err := common.GetOpenShiftVersion(client)
	if err != nil {
		return err
	}

	lastIndex := len(OpenShiftGPUOperatorCompatibilityMatrix[ocpVersion]) - 1

	s.Spec = &operatorsv1alpha1.SubscriptionSpec{
		CatalogSource:          "addon-nvidia-gpu-addon-catalog",
		CatalogSourceNamespace: common.GlobalConfig.AddonNamespace,
		Channel:                OpenShiftGPUOperatorCompatibilityMatrix[ocpVersion][lastIndex],
		Package:                packageName,
		InstallPlanApproval:    operatorsv1alpha1.ApprovalAutomatic,
	}

	if err := ctrl.SetControllerReference(gpuAddon, s, client.Scheme()); err != nil {
		return err
	}

	return nil
}

func (r *SubscriptionResourceReconciler) Delete(ctx context.Context, c client.Client) error {
	s := &operatorsv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.GlobalConfig.AddonNamespace,
			Name:      subscriptionName,
		},
	}

	err := c.Delete(ctx, s)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete Subscription %s: %w", s.Name, err)
	}

	csv, err := common.GetCsvWithPrefix(c, common.GlobalConfig.AddonNamespace, packageName)
	if err != nil {
		return err
	}

	err = c.Delete(ctx, csv)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete GPU Operator CSV %s: %w", csv.Name, err)
	}

	return nil
}

func (r *SubscriptionResourceReconciler) getDeployedConditionFetchFailed() metav1.Condition {
	return common.NewCondition(
		SubscriptionDeployedCondition,
		metav1.ConditionFalse,
		"FetchCrFailed",
		"Failed to fetch Subscription CR")
}

func (r *SubscriptionResourceReconciler) getDeployedConditionCreateFailed() metav1.Condition {
	return common.NewCondition(
		SubscriptionDeployedCondition,
		metav1.ConditionFalse,
		"CreateCrFailed",
		"Failed to create Subscription CR")
}

func (r *SubscriptionResourceReconciler) getDeployedConditionCreateSuccess() metav1.Condition {
	return common.NewCondition(
		SubscriptionDeployedCondition,
		metav1.ConditionTrue,
		"CreateCrSuccess",
		"Subscription deployed successfully")
}
