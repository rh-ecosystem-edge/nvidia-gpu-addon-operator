package controllers

import (
	"context"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
)

type ConsolePluginResourceReconciler struct{}

var _ ResourceReconciler = &ConsolePluginResourceReconciler{}

func (r *ConsolePluginResourceReconciler) Reconcile(
	ctx context.Context,
	client client.Client,
	gpuAddon *addonv1alpha1.GPUAddon) ([]metav1.Condition, error) {

	logger := log.FromContext(ctx, "Reconcile Step", "Console Plugin Reconcile")
	if gpuAddon.Spec.ConsolePluginEnabled {
		return enableConsolePlugin(ctx, client, logger)
	} else {
		return disableConsolePlugin(ctx, client, logger)
	}
}

func (r *ConsolePluginResourceReconciler) Delete(ctx context.Context, c client.Client) error {
	return nil
}

func enableConsolePlugin(ctx context.Context, client client.Client, logger logr.Logger) ([]metav1.Condition, error) {
	return nil, nil
}

func disableConsolePlugin(ctx context.Context, client client.Client, logger logr.Logger) ([]metav1.Condition, error) {
	return nil, nil
}
