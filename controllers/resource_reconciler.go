package controllers

import (
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

type ResourceReconciler interface {
	Reconcile(ctx context.Context, client client.Client, gpuAddon *addonv1alpha1.GPUAddon) ([]metav1.Condition, error)
}
type crFromCsvOptions struct {
	csvPrefix    string
	csvNamespace string
	crName       string
	crNamespace  string
	owner        addonv1alpha1.GPUAddon
}

func createOwnedCrFromCsvAlmExample(ctx context.Context, client client.Client, opt crFromCsvOptions) error {
	logger := log.FromContext(ctx)
	csv, err := common.GetCsvWithPrefix(client, opt.csvNamespace, opt.csvPrefix)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to get CSV name prefix '%v' in namespace '%v'", opt.csvPrefix, opt.csvNamespace))
		return err
	}
	almExample, err := common.GetAlmExamples(csv)
	if err != nil {
		logger.Error(err, "Failed to fetch ALM example from csv")
		return err
	}

	cr, err := common.GetCRasUnstructuredObjectFromAlmExample(almExample, logger)
	if err != nil {
		logger.Error(err, "Failed to generate runtime object from ALM example")
		return err
	}

	cr.SetNamespace(opt.crNamespace)
	cr.SetName(opt.crName)
	cr.SetOwnerReferences(append(cr.GetOwnerReferences(), metav1.OwnerReference{
		APIVersion:         opt.owner.APIVersion,
		Kind:               opt.owner.Kind,
		Name:               opt.owner.Name,
		UID:                opt.owner.UID,
		BlockOwnerDeletion: pointer.Bool(true),
		Controller:         pointer.Bool(true),
	}))

	err = client.Create(ctx, cr)
	if err != nil {
		if k8serrors.IsAlreadyExists(err) {
			logger.Info("CR already exists")
		} else {
			logger.Error(err, "Failed to create CR")
			return err
		}
	}
	return nil
}
