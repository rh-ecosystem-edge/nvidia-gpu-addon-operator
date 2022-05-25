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

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	configv1 "github.com/openshift/api/config/v1"
	operatorv1 "github.com/openshift/api/operator/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/source"

	gpuv1 "github.com/NVIDIA/gpu-operator/api/v1"
	consolev1alpha1 "github.com/openshift/api/console/v1alpha1"
	nfdv1 "github.com/openshift/cluster-nfd-operator/api/v1"
	operatorsv1 "github.com/operator-framework/api/pkg/operators/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"

	nvidiav1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/controllers/configmap"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/controllers/gpuaddon"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/controllers/monitoring"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/version"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(nvidiav1alpha1.AddToScheme(scheme))
	utilruntime.Must(gpuv1.AddToScheme(scheme))
	utilruntime.Must(nfdv1.AddToScheme(scheme))
	utilruntime.Must(operatorsv1alpha1.AddToScheme(scheme))
	utilruntime.Must(operatorsv1.AddToScheme(scheme))
	utilruntime.Must(consolev1alpha1.AddToScheme(scheme))
	utilruntime.Must(configv1.AddToScheme(scheme))
	utilruntime.Must(operatorv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	common.ProcessConfig()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Namespace:              common.GlobalConfig.AddonNamespace,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "f75da35c.addons.rh-ecosystem-edge.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	gpuAddonController, err := (&gpuaddon.GPUAddonReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "GPUAddon")
		os.Exit(1)
	}

	go func() {
		if err := watchForOwnClusterPoliciesWhenAvailable(gpuAddonController); err != nil {
			setupLog.Error(err, "unable to wait and watch for ClusterPolicy CRD")
			return
		}
	}()

	if err = (&configmap.ConfigMapReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ConfigMap")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	c, err := client.New(ctrlconfig.GetConfigOrDie(), client.Options{Scheme: scheme})
	if err != nil {
		setupLog.Error(err, "failed to create client to jumpstart addon")
		os.Exit(1)
	}
	if err := jumpstartAddon(c); err != nil {
		setupLog.Error(err, "failed to jumpstart addon")
		os.Exit(1)
	}

	setupLog.Info("starting manager", "version", version.Version(), "config", common.GlobalConfig)
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func watchForOwnClusterPoliciesWhenAvailable(c controller.Controller) error {
	if err := wait.PollInfinite(time.Second, isClusterPolicyAvailable()); err != nil {
		return fmt.Errorf("unable to wait for ClusterPolicy CRD: %w", err)
	}

	err := c.Watch(
		&source.Kind{Type: &gpuv1.ClusterPolicy{}},
		&handler.EnqueueRequestForObject{})

	if err != nil {
		return fmt.Errorf("unable to watch for owned ClusterPolicy CRs: %w", err)
	}

	return nil
}

func isClusterPolicyAvailable() wait.ConditionFunc {
	return func() (bool, error) {
		client, err := client.New(ctrlconfig.GetConfigOrDie(), client.Options{Scheme: scheme})
		if err != nil {
			return false, err
		}

		clusterPolicyList := &gpuv1.ClusterPolicyList{}

		err = client.List(context.TODO(), clusterPolicyList)
		unavailable := meta.IsNoMatchError(err)
		if err != nil && !unavailable {
			return false, err
		}

		return !unavailable, nil
	}
}

func jumpstartAddon(client client.Client) error {
	gpuAddon := &nvidiav1alpha1.GPUAddon{}
	err := client.Get(context.TODO(), types.NamespacedName{
		Name:      common.GlobalConfig.AddonID,
		Namespace: common.GlobalConfig.AddonNamespace,
	}, gpuAddon)
	isNotFound := k8serrors.IsNotFound(err)
	if err != nil && !isNotFound {
		return fmt.Errorf("failed to fetch GPUAddon CR %s in %s: %w", common.GlobalConfig.AddonID, common.GlobalConfig.AddonNamespace, err)
	}
	if isNotFound {
		gpuAddon.Spec = nvidiav1alpha1.GPUAddonSpec{
			ConsolePluginEnabled: true,
		}
	}

	gpuAddon.ObjectMeta = metav1.ObjectMeta{
		Name:      common.GlobalConfig.AddonID,
		Namespace: common.GlobalConfig.AddonNamespace,
	}

	result, err := controllerutil.CreateOrPatch(context.TODO(), client, gpuAddon, func() error {
		versionLabel := fmt.Sprintf("%v-version", common.GlobalConfig.AddonLabel)
		gpuAddon.ObjectMeta.Labels = map[string]string{
			versionLabel: version.Version(),
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to reconcile GPUAddon CR %s in %s, result: %v. error: %w", common.GlobalConfig.AddonID, common.GlobalConfig.AddonNamespace, result, err)
	}

	return nil
}
