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

	consolev1alpha1 "github.com/openshift/api/console/v1alpha1"
	operatorv1 "github.com/openshift/api/operator/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
)

const (
	ConsolePluginDeployedCondition = "ConsolePluginDeployed"

	consolePluginName = "console-plugin-nvidia-gpu"

	ocpVersion4_10 = "4.10"
)

type ConsolePluginResourceReconciler struct{}

var _ ResourceReconciler = &ConsolePluginResourceReconciler{}

func (r *ConsolePluginResourceReconciler) Reconcile(
	ctx context.Context,
	client client.Client,
	gpuAddon *addonv1alpha1.GPUAddon) ([]metav1.Condition, error) {

	logger := log.FromContext(ctx, "Reconcile Step", "ConsolePlugin")
	conditions := []metav1.Condition{}

	supported, err := common.IsOpenShiftVersionAtLeast(client, ocpVersion4_10)
	if err != nil {
		conditions = append(conditions, r.getDeployedConditionFailed(err))
		return conditions, err
	}

	if !supported {
		conditions = append(conditions, r.getDeployedConditionNotSupported())

		logger.Info("ConsolePlugin will not be reconciled as OpenShift version is <4.10",
			"name", consolePluginName,
			"namespace", gpuAddon.Namespace,
			"enabled", gpuAddon.Spec.ConsolePluginEnabled)

		return conditions, nil
	}

	if !gpuAddon.Spec.ConsolePluginEnabled {
		if err := r.Delete(ctx, client); err != nil {
			conditions = append(conditions, r.getDeployedConditionFailed(err))
			return conditions, err
		}

		conditions = append(conditions, r.getDeployedConditionSuccess())

		logger.Info("ConsolePlugin reconciled successfully",
			"name", consolePluginName,
			"namespace", gpuAddon.Namespace,
			"enabled", gpuAddon.Spec.ConsolePluginEnabled)

		return conditions, nil
	}

	if err := r.reconcileConsolePluginDeployment(ctx, client, gpuAddon); err != nil {
		conditions = append(conditions, r.getDeployedConditionFailed(err))
		return conditions, err
	}

	if err := r.reconcileConsolePluginService(ctx, client, gpuAddon); err != nil {
		conditions = append(conditions, r.getDeployedConditionFailed(err))
		return conditions, err
	}

	if err := r.reconcileConsolePluginCR(ctx, client, gpuAddon); err != nil {
		conditions = append(conditions, r.getDeployedConditionFailed(err))
		return conditions, err
	}

	if err := r.patchClusterConsole(ctx, client, gpuAddon); err != nil {
		conditions = append(conditions, r.getDeployedConditionFailed(err))
		return conditions, err
	}

	conditions = append(conditions, r.getDeployedConditionSuccess())

	return conditions, nil
}

func (r *ConsolePluginResourceReconciler) Delete(ctx context.Context, c client.Client) error {
	err := r.deleteConsolePluginCR(ctx, c)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	err = r.deleteConsolePluginService(ctx, c)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	err = r.deleteConsolePluginDeployment(ctx, c)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	return nil
}

func (r *ConsolePluginResourceReconciler) reconcileConsolePluginCR(
	ctx context.Context,
	c client.Client,
	gpuAddon *addonv1alpha1.GPUAddon) error {

	logger := log.FromContext(ctx, "Reconcile Step", "ConsolePlugin CR")
	existingCP := &consolev1alpha1.ConsolePlugin{}
	err := c.Get(ctx, client.ObjectKey{
		Name: consolePluginName,
	}, existingCP)

	exists := !k8serrors.IsNotFound(err)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	cp := &consolev1alpha1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: consolePluginName,
		},
	}

	if exists {
		cp = existingCP
	}

	res, err := controllerutil.CreateOrPatch(context.TODO(), c, cp, func() error {
		return r.setDesiredConsolePlugin(cp, gpuAddon)
	})

	if err != nil {
		return err
	}

	logger.Info("ConsolePlugin CR reconciled successfully",
		"name", cp.Name,
		"result", res)

	return nil
}

func (r *ConsolePluginResourceReconciler) patchClusterConsole(
	ctx context.Context,
	c client.Client,
	gpuAddon *addonv1alpha1.GPUAddon) error {

	logger := log.FromContext(ctx, "Reconcile Step", "ConsolePlugin Patch Cluster Console")
	console := &operatorv1.Console{}
	err := c.Get(ctx, client.ObjectKey{
		Name: "cluster",
	}, console)

	if err != nil {
		return err
	}

	patched := console.DeepCopy()

	exists := common.SliceContainsString(patched.Spec.Plugins, consolePluginName)
	if !exists {
		patched.Spec.Plugins = append(patched.Spec.Plugins, consolePluginName)

		if err := c.Patch(ctx, patched, client.MergeFrom(console)); err != nil {
			return err
		}
	}

	logger.Info("ConsolePlugin Cluster Console reconciled successfully",
		"name", patched.Name,
		"plugins", patched.Spec.Plugins)

	return nil
}

func (r *ConsolePluginResourceReconciler) reconcileConsolePluginDeployment(
	ctx context.Context,
	client client.Client,
	gpuAddon *addonv1alpha1.GPUAddon) error {

	logger := log.FromContext(ctx, "Reconcile Step", "ConsolePlugin Deployment")
	existingDP := &appsv1.Deployment{}
	err := client.Get(ctx, types.NamespacedName{
		Name:      consolePluginName,
		Namespace: gpuAddon.Namespace,
	}, existingDP)

	exists := !k8serrors.IsNotFound(err)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	dp := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consolePluginName,
			Namespace: gpuAddon.Namespace,
		},
	}

	if exists {
		dp = existingDP
	}

	res, err := controllerutil.CreateOrPatch(context.TODO(), client, dp, func() error {
		return r.setDesiredConsolePluginDeployment(client, dp, gpuAddon)
	})

	if err != nil {
		return err
	}

	logger.Info("ConsolePlugin Deployment reconciled successfully",
		"name", dp.Name,
		"namespace", dp.Namespace,
		"result", res)

	return nil
}

func (r *ConsolePluginResourceReconciler) reconcileConsolePluginService(
	ctx context.Context,
	client client.Client,
	gpuAddon *addonv1alpha1.GPUAddon) error {

	logger := log.FromContext(ctx, "Reconcile Step", "ConsolePlugin Service")
	existingSVC := &corev1.Service{}
	err := client.Get(ctx, types.NamespacedName{
		Name:      consolePluginName,
		Namespace: gpuAddon.Namespace,
	}, existingSVC)

	exists := !k8serrors.IsNotFound(err)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      consolePluginName,
			Namespace: gpuAddon.Namespace,
		},
	}

	if exists {
		s = existingSVC
	}

	res, err := controllerutil.CreateOrPatch(context.TODO(), client, s, func() error {
		return r.setDesiredConsolePluginService(client, s, gpuAddon)
	})

	if err != nil {
		return err
	}

	logger.Info("ConsolePlugin Service reconciled successfully",
		"name", s.Name,
		"namespace", s.Namespace,
		"result", res)

	return nil
}

func (r *ConsolePluginResourceReconciler) setDesiredConsolePluginDeployment(
	client client.Client,
	dp *appsv1.Deployment,
	gpuAddon *addonv1alpha1.GPUAddon) error {

	if dp == nil {
		return errors.New("deployment cannot be nil")
	}

	labels := map[string]string{
		"app": "console-plugin-nvidia-gpu",
	}

	dp.ObjectMeta.Labels = labels

	twentyFivePercent := intstr.FromString("25%")
	dp.Spec = appsv1.DeploymentSpec{
		Strategy: appsv1.DeploymentStrategy{
			Type: appsv1.RollingUpdateDeploymentStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDeployment{
				MaxUnavailable: &twentyFivePercent,
				MaxSurge:       &twentyFivePercent,
			},
		},
	}

	dp.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: labels,
	}

	dp.Spec.Template = corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
		},
	}

	defaultMode := int32(420)
	volumes := []corev1.Volume{
		{
			Name: "plugin-serving-cert",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  "plugin-serving-cert",
					DefaultMode: &defaultMode,
				},
			},
		},
		{
			Name: "nginx-conf",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "nginx-conf",
					},
					DefaultMode: &defaultMode,
				},
			},
		},
	}

	consolePluginContainer := corev1.Container{
		Name: consolePluginName,
	}
	consolePluginContainer.Image = common.GlobalConfig.ConsolePluginImage
	consolePluginContainer.ImagePullPolicy = corev1.PullAlways
	consolePluginContainer.Ports = []corev1.ContainerPort{
		{
			ContainerPort: 9443,
			Protocol:      corev1.ProtocolTCP,
		},
	}
	consolePluginContainer.Resources = corev1.ResourceRequirements{
		Requests: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse("10m"),
			corev1.ResourceMemory: resource.MustParse("100Mi"),
		},
		Limits: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceCPU:    resource.MustParse("20m"),
			corev1.ResourceMemory: resource.MustParse("200Mi"),
		},
	}

	consolePluginContainer.SecurityContext = &corev1.SecurityContext{
		AllowPrivilegeEscalation: pointer.Bool(true),
		RunAsNonRoot:             pointer.Bool(true),
	}
	consolePluginContainer.VolumeMounts = []corev1.VolumeMount{
		{
			Name:      "plugin-serving-cert",
			ReadOnly:  true,
			MountPath: "/var/serving-cert",
		},
	}
	containers := []corev1.Container{
		consolePluginContainer,
	}

	dp.Spec.Template.Spec = corev1.PodSpec{
		Containers:    containers,
		Volumes:       volumes,
		RestartPolicy: corev1.RestartPolicyAlways,
		DNSPolicy:     corev1.DNSClusterFirst,
	}

	return ctrl.SetControllerReference(gpuAddon, dp, client.Scheme())
}

func (r *ConsolePluginResourceReconciler) setDesiredConsolePlugin(
	cp *consolev1alpha1.ConsolePlugin,
	gpuAddon *addonv1alpha1.GPUAddon) error {

	if cp == nil {
		return errors.New("consoleplugin cannot be nil")
	}

	cp.Spec = consolev1alpha1.ConsolePluginSpec{
		DisplayName: "Console Plugin NVIDIA GPU Template",
		Service: consolev1alpha1.ConsolePluginService{
			Name:      consolePluginName,
			Namespace: gpuAddon.Namespace,
			Port:      9443,
			BasePath:  "/",
		},
	}

	return nil
}

func (r *ConsolePluginResourceReconciler) setDesiredConsolePluginService(
	client client.Client,
	s *corev1.Service,
	gpuAddon *addonv1alpha1.GPUAddon) error {

	if s == nil {
		return errors.New("service cannot be nil")
	}

	s.ObjectMeta.Annotations = map[string]string{
		"service.alpha.openshift.io/serving-cert-secret-name": "plugin-serving-cert",
	}

	s.Spec = corev1.ServiceSpec{
		Ports: []corev1.ServicePort{
			corev1.ServicePort{
				Name:       "9443-tcp",
				Protocol:   corev1.ProtocolTCP,
				Port:       9443,
				TargetPort: intstr.FromInt(9443),
			},
		},
		Selector: map[string]string{
			"app": "console-plugin-nvidia-gpu",
		},
		Type: corev1.ServiceTypeClusterIP,
	}

	return ctrl.SetControllerReference(gpuAddon, s, client.Scheme())
}

func (r *ConsolePluginResourceReconciler) deleteConsolePluginCR(ctx context.Context, c client.Client) error {
	cp := &consolev1alpha1.ConsolePlugin{
		ObjectMeta: metav1.ObjectMeta{
			Name: consolePluginName,
		},
	}

	err := c.Delete(ctx, cp)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete ConsolePlugin CR %s: %w", cp.Name, err)
	}

	return nil
}

func (r *ConsolePluginResourceReconciler) deleteConsolePluginDeployment(ctx context.Context, c client.Client) error {
	dp := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.GlobalConfig.AddonNamespace,
			Name:      consolePluginName,
		},
	}

	err := c.Delete(ctx, dp)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete ConsolePlugin Deployment %s: %w", dp.Name, err)
	}

	return nil
}

func (r *ConsolePluginResourceReconciler) deleteConsolePluginService(ctx context.Context, c client.Client) error {
	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: common.GlobalConfig.AddonNamespace,
			Name:      consolePluginName,
		},
	}

	err := c.Delete(ctx, s)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete ConsolePlugin Service %s: %w", s.Name, err)
	}

	return nil
}

func (r *ConsolePluginResourceReconciler) getDeployedConditionFailed(err error) metav1.Condition {
	return common.NewCondition(
		ConsolePluginDeployedCondition,
		metav1.ConditionTrue,
		"Failed",
		err.Error())
}

func (r *ConsolePluginResourceReconciler) getDeployedConditionSuccess() metav1.Condition {
	return common.NewCondition(
		ConsolePluginDeployedCondition,
		metav1.ConditionTrue,
		"Success",
		"ConsolePlugin deployed successfully")
}

func (r *ConsolePluginResourceReconciler) getDeployedConditionNotSupported() metav1.Condition {
	return common.NewCondition(
		ConsolePluginDeployedCondition,
		metav1.ConditionTrue,
		"NotSupported",
		"ConsolePlugin is not supported when OpenShift version <4.10")
}
