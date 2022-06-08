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
	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	SubscriptionInstalled = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nvidia_gpuaddon_gpu_operator_subscription_installed",
			Help: "Reports whether the NVIDIA GPUAddon GPU Operator OLM Subscription is installed",
		},
		[]string{"current_csv", "installed_csv"},
	)
)

func init() {
	metrics.Registry.MustRegister(
		SubscriptionInstalled,
	)
}
