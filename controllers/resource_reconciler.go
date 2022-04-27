package controllers

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
)

type ResourceReconciler interface {
	Reconcile(ctx context.Context, client client.Client, gpuAddon *addonv1alpha1.GPUAddon) ([]metav1.Condition, error)
	Delete(ctx context.Context, client client.Client) error
}
