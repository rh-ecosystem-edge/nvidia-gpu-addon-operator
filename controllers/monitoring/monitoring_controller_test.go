package monitoring

import (
	"context"
	"reflect"
	"time"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	promv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	addonv1alpha1 "github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/api/v1alpha1"
	"github.com/rh-ecosystem-edge/nvidia-gpu-addon-operator/internal/common"
)

var _ = Describe("Monitoring Reconcile", Ordered, func() {
	Context("Create Reconcile", func() {
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
		monitoring := &addonv1alpha1.Monitoring{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test",
			},
		}
		r := newTestMonitoringReconciler(monitoring, pagerDutySecret, deadMansSnitchSecret)

		It("should not throw an error", func() {
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: monitoring.Namespace,
					Name:      monitoring.Name,
				},
			}
			_, err := r.Reconcile(context.TODO(), req)

			Expect(err).ShouldNot(HaveOccurred())
		})

		cm := &corev1.ConfigMap{}
		It("should reconcile the Prometheus KubeRBACProxy ConfigMap", func() {
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Namespace: monitoring.Namespace,
				Name:      prometheusKubeRBACProxyConfigMapName,
			}, cm)
			Expect(err).ShouldNot(HaveOccurred())
		})

		s := &corev1.Service{}
		It("should reconcile the Prometheus Service successfully", func() {
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Namespace: monitoring.Namespace,
				Name:      prometheusServiceName,
			}, s)
			Expect(err).ShouldNot(HaveOccurred())
		})

		p := &promv1.Prometheus{}
		It("should reconcile the Prometheus CR successfully", func() {
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Namespace: monitoring.Namespace,
				Name:      prometheusName,
			}, p)
			Expect(err).ShouldNot(HaveOccurred())
		})

		am := &promv1.Alertmanager{}
		It("should reconcile the AlertManager CR successfully", func() {
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Namespace: monitoring.Namespace,
				Name:      alertManagerName,
			}, am)
			Expect(err).ShouldNot(HaveOccurred())
		})

		amc := &promv1alpha1.AlertmanagerConfig{}
		It("should reconcile the AlertManagerConfig CR successfully", func() {
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Namespace: monitoring.Namespace,
				Name:      alertManagerConfigName,
			}, amc)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})

	Context("Delete Reconcile", func() {
		now := metav1.NewTime(time.Now())
		monitoring := &addonv1alpha1.Monitoring{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test",
				Namespace:         "test",
				DeletionTimestamp: &now,
				UID:               types.UID("uid-uid"),
			},
		}

		kind := reflect.TypeOf(addonv1alpha1.Monitoring{}).Name()
		gvk := addonv1alpha1.GroupVersion.WithKind(kind)
		ctrlRef := metav1.OwnerReference{
			APIVersion:         gvk.GroupVersion().String(),
			Kind:               gvk.Kind,
			Name:               monitoring.Name,
			UID:                monitoring.UID,
			BlockOwnerDeletion: pointer.BoolPtr(true),
			Controller:         pointer.BoolPtr(true),
		}

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:            prometheusKubeRBACProxyConfigMapName,
				Namespace:       monitoring.Namespace,
				OwnerReferences: []metav1.OwnerReference{ctrlRef},
			},
		}
		s := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:            prometheusServiceName,
				Namespace:       monitoring.Namespace,
				OwnerReferences: []metav1.OwnerReference{ctrlRef},
			},
		}
		p := &promv1.Prometheus{
			ObjectMeta: metav1.ObjectMeta{
				Name:            prometheusName,
				Namespace:       monitoring.Namespace,
				OwnerReferences: []metav1.OwnerReference{ctrlRef},
			},
		}
		am := &promv1.Alertmanager{
			ObjectMeta: metav1.ObjectMeta{
				Name:            alertManagerName,
				Namespace:       monitoring.Namespace,
				OwnerReferences: []metav1.OwnerReference{ctrlRef},
			},
		}
		amc := &promv1alpha1.AlertmanagerConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:            alertManagerConfigName,
				Namespace:       monitoring.Namespace,
				OwnerReferences: []metav1.OwnerReference{ctrlRef},
			},
		}
		r := newTestMonitoringReconciler(monitoring, cm, s, p, am, amc)

		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: monitoring.Namespace,
				Name:      monitoring.Name,
			},
		}
		_, err := r.Reconcile(context.TODO(), req)
		Expect(err).ShouldNot(HaveOccurred())

		It("should delete the Prometheus KuberRBACProxy ConfigMap", func() {
			c := &corev1.ConfigMap{}
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Name:      prometheusKubeRBACProxyConfigMapName,
				Namespace: monitoring.Namespace,
			}, c)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should delete the Prometheus Service", func() {
			s := &corev1.Service{}
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Name:      prometheusServiceName,
				Namespace: monitoring.Namespace,
			}, s)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should delete the Prometheus CR", func() {
			p := &promv1.Prometheus{}
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Name:      prometheusName,
				Namespace: monitoring.Namespace,
			}, p)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should delete the AlertManager CR", func() {
			am := &promv1.Alertmanager{}
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Name:      prometheusName,
				Namespace: monitoring.Namespace,
			}, am)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})

		It("should delete the AlertManagerConfig CR", func() {
			amc := &promv1alpha1.AlertmanagerConfig{}
			err := r.Client.Get(context.TODO(), types.NamespacedName{
				Name:      prometheusName,
				Namespace: monitoring.Namespace,
			}, amc)
			Expect(err).Should(HaveOccurred())
			Expect(k8serrors.IsNotFound(err)).To(BeTrue())
		})
	})
})

func newTestMonitoringReconciler(objs ...runtime.Object) *MonitoringReconciler {
	s := scheme.Scheme

	Expect(addonv1alpha1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(promv1.AddToScheme(s)).ShouldNot(HaveOccurred())
	Expect(promv1alpha1.AddToScheme(s)).ShouldNot(HaveOccurred())

	c := fake.NewClientBuilder().WithScheme(s).WithRuntimeObjects(objs...).Build()

	return &MonitoringReconciler{
		Client: c,
		Scheme: s,
	}
}
