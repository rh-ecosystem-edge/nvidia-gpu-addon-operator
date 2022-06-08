package monitoring

import (
	"context"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
)

var _ = Describe("Prometheus", Ordered, func() {
	Context("Reconcile", func() {
		common.ProcessConfig()

		m := &addonv1alpha1.Monitoring{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
		}
		r := &MonitoringReconciler{}

		scheme := scheme.Scheme
		Expect(promv1.AddToScheme(scheme)).ShouldNot(HaveOccurred())
		Expect(addonv1alpha1.AddToScheme(scheme)).ShouldNot(HaveOccurred())

		var p promv1.Prometheus

		It("should create the Prometheus CR", func() {
			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects().
				Build()

			r.Client = c

			err := r.reconcilePrometheus(context.TODO(), m)
			Expect(err).ShouldNot(HaveOccurred())

			err = c.Get(context.TODO(), types.NamespacedName{
				Name:      "gpuaddon-prometheus",
				Namespace: m.Namespace,
			}, &p)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("Delete", func() {
		m := &addonv1alpha1.Monitoring{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
		}
		r := &MonitoringReconciler{}

		scheme := scheme.Scheme
		Expect(promv1.AddToScheme(scheme)).ShouldNot(HaveOccurred())

		p := &promv1.Prometheus{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prometheusName,
				Namespace: m.Namespace,
			},
		}

		It("should delete the Prometheus CR", func() {
			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(p).
				Build()

			r.Client = c

			err := r.deletePrometheus(context.TODO(), m)
			Expect(err).ShouldNot(HaveOccurred())

			err = c.Get(context.TODO(), types.NamespacedName{
				Name:      prometheusName,
				Namespace: m.Namespace,
			}, p)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})
	})
})

var _ = Describe("Prometheus KubeRBACProxy ConfigMap", Ordered, func() {
	Context("Reconcile", func() {
		common.ProcessConfig()

		m := &addonv1alpha1.Monitoring{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
		}
		r := &MonitoringReconciler{}

		scheme := scheme.Scheme
		Expect(corev1.AddToScheme(scheme)).ShouldNot(HaveOccurred())

		var cm corev1.ConfigMap

		It("should create the Prometheus KubeRBACProxy ConfigMap", func() {
			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects().
				Build()

			r.Client = c

			err := r.reconcilePrometheusKubeRBACProxyConfigMap(context.TODO(), m)
			Expect(err).ShouldNot(HaveOccurred())

			err = c.Get(context.TODO(), types.NamespacedName{
				Name:      "prometheus-kube-rbac-proxy-config",
				Namespace: m.Namespace,
			}, &cm)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("Delete", func() {
		m := &addonv1alpha1.Monitoring{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
		}
		r := &MonitoringReconciler{}

		scheme := scheme.Scheme

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prometheusKubeRBACProxyConfigMapName,
				Namespace: m.Namespace,
			},
		}

		It("should delete the Prometheus KubeRBACProxy ConfigMap", func() {
			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(cm).
				Build()

			r.Client = c

			err := r.deletePrometheusKubeRBACProxyConfigMap(context.TODO(), m)
			Expect(err).ShouldNot(HaveOccurred())

			err = c.Get(context.TODO(), types.NamespacedName{
				Name:      prometheusKubeRBACProxyConfigMapName,
				Namespace: m.Namespace,
			}, cm)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})
	})
})

var _ = Describe("Prometheus Service", Ordered, func() {
	Context("Reconcile", func() {
		common.ProcessConfig()

		m := &addonv1alpha1.Monitoring{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
		}
		r := &MonitoringReconciler{}

		scheme := scheme.Scheme
		Expect(corev1.AddToScheme(scheme)).ShouldNot(HaveOccurred())

		var cm corev1.Service

		It("should create the Prometheus Service", func() {
			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects().
				Build()

			r.Client = c

			err := r.reconcilePrometheusService(context.TODO(), m)
			Expect(err).ShouldNot(HaveOccurred())

			err = c.Get(context.TODO(), types.NamespacedName{
				Name:      "gpuaddon-prometheus-service",
				Namespace: m.Namespace,
			}, &cm)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("Delete", func() {
		m := &addonv1alpha1.Monitoring{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
		}
		r := &MonitoringReconciler{}

		scheme := scheme.Scheme

		s := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      prometheusServiceName,
				Namespace: m.Namespace,
			},
		}

		It("should delete the Prometheus Service", func() {
			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(s).
				Build()

			r.Client = c

			err := r.deletePrometheusService(context.TODO(), m)
			Expect(err).ShouldNot(HaveOccurred())

			err = c.Get(context.TODO(), types.NamespacedName{
				Name:      prometheusServiceName,
				Namespace: m.Namespace,
			}, s)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})
	})
})
