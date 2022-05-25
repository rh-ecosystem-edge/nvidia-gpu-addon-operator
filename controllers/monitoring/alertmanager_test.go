package monitoring

import (
	"context"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
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

var _ = Describe("AlertManager", Ordered, func() {
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

		var am promv1.Alertmanager

		It("should create the AlertManager CR", func() {
			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects().
				Build()

			r.Client = c

			err := r.reconcileAlertManager(context.TODO(), m)
			Expect(err).ShouldNot(HaveOccurred())

			err = c.Get(context.TODO(), types.NamespacedName{
				Name:      "gpuaddon-alertmanager",
				Namespace: m.Namespace,
			}, &am)
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

		am := &promv1.Alertmanager{
			ObjectMeta: metav1.ObjectMeta{
				Name:      alertManagerName,
				Namespace: m.Namespace,
			},
		}

		It("should delete the AlertManager CR", func() {
			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(am).
				Build()

			r.Client = c

			err := r.deleteAlertManager(context.TODO(), m)
			Expect(err).ShouldNot(HaveOccurred())

			err = c.Get(context.TODO(), types.NamespacedName{
				Name:      alertManagerName,
				Namespace: m.Namespace,
			}, am)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})
	})
})

var _ = Describe("AlertManagerConfig", Ordered, func() {
	Context("Reconcile", func() {
		common.ProcessConfig()

		deadMansSnitchSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.GlobalConfig.DeadMansSnitchSecretName,
				Namespace: "test",
			},
			Data: map[string][]byte{
				"SNITCH_URL": []byte("some-snitch-url"),
			},
		}
		pagerDutySecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      common.GlobalConfig.PagerDutySecretName,
				Namespace: "test",
			},
			Data: map[string][]byte{
				"PAGERDUTY_KEY": []byte("some-service-key"),
			},
		}
		m := &addonv1alpha1.Monitoring{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
		}
		r := &MonitoringReconciler{}

		scheme := scheme.Scheme
		Expect(promv1alpha1.AddToScheme(scheme)).ShouldNot(HaveOccurred())

		var amc promv1alpha1.AlertmanagerConfig

		It("should create the AlertManagerConfig CR", func() {
			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(pagerDutySecret, deadMansSnitchSecret).
				Build()

			r.Client = c

			err := r.reconcileAlertManagerConfig(context.TODO(), m)
			Expect(err).ShouldNot(HaveOccurred())

			err = c.Get(context.TODO(), types.NamespacedName{
				Name:      alertManagerConfigName,
				Namespace: m.Namespace,
			}, &amc)
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
		Expect(promv1alpha1.AddToScheme(scheme)).ShouldNot(HaveOccurred())

		amc := &promv1alpha1.AlertmanagerConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      alertManagerConfigName,
				Namespace: m.Namespace,
			},
		}

		It("should delete the AlertManagerConfig CR", func() {
			c := fake.
				NewClientBuilder().
				WithScheme(scheme).
				WithRuntimeObjects(amc).
				Build()

			r.Client = c

			err := r.deleteAlertManagerConfig(context.TODO(), m)
			Expect(err).ShouldNot(HaveOccurred())

			err = c.Get(context.TODO(), types.NamespacedName{
				Name:      alertManagerConfigName,
				Namespace: m.Namespace,
			}, amc)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})
	})
})
