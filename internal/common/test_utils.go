package common

import (
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ContainCondition(conditions []metav1.Condition, cond_type string, cond_status metav1.ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Type == cond_type {
			return condition.Status == cond_status
		}
	}
	return false
}

func NewCsv(namespace string, name string, almExample string) *operatorsv1alpha1.ClusterServiceVersion {
	csv := &operatorsv1alpha1.ClusterServiceVersion{}
	csv.Name = name
	csv.Namespace = namespace
	csv.ObjectMeta.Annotations = map[string]string{
		"alm-examples": almExample,
	}
	return csv
}

var NfdAlmExample string = `
      [
        {
          "apiVersion": "nfd.openshift.io/v1",
          "kind": "NodeFeatureDiscovery",
          "metadata": {
            "name": "nfd-instance",
            "namespace": "openshift-nfd"
          },
          "spec": {
            "customConfig": {
              "configData": "#    - name: \"more.kernel.features\"\n#      matchOn:\n#      - loadedKMod: [\"example_kmod3\"]\n#    - name: \"more.features.by.nodename\"\n#      value: customValue\n#      matchOn:\n#      - nodename: [\"special-.*-node-.*\"]\n"
            },
            "instance": "",
            "operand": {
              "image": "quay.io/openshift/origin-node-feature-discovery:4.8",
              "imagePullPolicy": "Always",
              "namespace": "openshift-nfd"
            },
            "workerConfig": {
              "configData": "core:\n#  labelWhiteList:\n#  noPublish: false\n  sleepInterval: 60s\n#  sources: [all]\n#  klog:\n#    addDirHeader: false\n#    alsologtostderr: false\n#    logBacktraceAt:\n#    logtostderr: true\n#    skipHeaders: false\n#    stderrthreshold: 2\n#    v: 0\n#    vmodule:\n##   NOTE: the following options are not dynamically run-time configurable\n##         and require a nfd-worker restart to take effect after being changed\n#    logDir:\n#    logFile:\n#    logFileMaxSize: 1800\n#    skipLogHeaders: false\nsources:\n#  cpu:\n#    cpuid:\n##     NOTE: whitelist has priority over blacklist\n#      attributeBlacklist:\n#        - \"BMI1\"\n#        - \"BMI2\"\n#        - \"CLMUL\"\n#        - \"CMOV\"\n#        - \"CX16\"\n#        - \"ERMS\"\n#        - \"F16C\"\n#        - \"HTT\"\n#        - \"LZCNT\"\n#        - \"MMX\"\n#        - \"MMXEXT\"\n#        - \"NX\"\n#        - \"POPCNT\"\n#        - \"RDRAND\"\n#        - \"RDSEED\"\n#        - \"RDTSCP\"\n#        - \"SGX\"\n#        - \"SSE\"\n#        - \"SSE2\"\n#        - \"SSE3\"\n#        - \"SSE4.1\"\n#        - \"SSE4.2\"\n#        - \"SSSE3\"\n#      attributeWhitelist:\n#  kernel:\n#    kconfigFile: \"/path/to/kconfig\"\n#    configOpts:\n#      - \"NO_HZ\"\n#      - \"X86\"\n#      - \"DMI\"\n  pci:\n    deviceClassWhitelist:\n      - \"0200\"\n      - \"03\"\n      - \"12\"\n    deviceLabelFields:\n#      - \"class\"\n      - \"vendor\"\n#      - \"device\"\n#      - \"subsystem_vendor\"\n#      - \"subsystem_device\"\n#  usb:\n#    deviceClassWhitelist:\n#      - \"0e\"\n#      - \"ef\"\n#      - \"fe\"\n#      - \"ff\"\n#    deviceLabelFields:\n#      - \"class\"\n#      - \"vendor\"\n#      - \"device\"\n#  custom:\n#    - name: \"my.kernel.feature\"\n#      matchOn:\n#        - loadedKMod: [\"example_kmod1\", \"example_kmod2\"]\n#    - name: \"my.pci.feature\"\n#      matchOn:\n#        - pciId:\n#            class: [\"0200\"]\n#            vendor: [\"15b3\"]\n#            device: [\"1014\", \"1017\"]\n#        - pciId :\n#            vendor: [\"8086\"]\n#            device: [\"1000\", \"1100\"]\n#    - name: \"my.usb.feature\"\n#      matchOn:\n#        - usbId:\n#          class: [\"ff\"]\n#          vendor: [\"03e7\"]\n#          device: [\"2485\"]\n#        - usbId:\n#          class: [\"fe\"]\n#          vendor: [\"1a6e\"]\n#          device: [\"089a\"]\n#    - name: \"my.combined.feature\"\n#      matchOn:\n#        - pciId:\n#            vendor: [\"15b3\"]\n#            device: [\"1014\", \"1017\"]\n#          loadedKMod : [\"vendor_kmod1\", \"vendor_kmod2\"]\n"
            }
          }
        }
      ]
`

var ClusterPolicyAlmExample string = `
      [
        {
          "apiVersion": "nvidia.com/v1",
          "kind": "ClusterPolicy",
          "metadata": {
            "name": "gpu-cluster-policy"
          },
          "spec": {
            "dcgmExporter": {
              "config": {
                "name": ""
              }
            },
            "dcgm": {
              "enabled": true
            },
            "daemonsets": {
            },
            "devicePlugin": {
            },
            "driver": {
              "enabled": true,
              "use_ocp_driver_toolkit": true,
              "repoConfig": {
                "configMapName": ""
              },
              "certConfig": {
                "name": ""
              },
              "licensingConfig": {
                "nlsEnabled": false,
                "configMapName": ""
              },
              "virtualTopology": {
                "config": ""
              },
              "kernelModuleConfig": {
                "name": ""
              }
            },
            "gfd": {
            },
            "migManager": {
              "enabled": true
            },
            "nodeStatusExporter": {
              "enabled": true
            },
            "operator": {
              "defaultRuntime": "crio",
              "deployGFD": true,
              "initContainer": {
              }
            },
            "mig": {
              "strategy": "single"
            },
            "toolkit": {
              "enabled": true
            },
            "validator": {
              "plugin": {
                "env": [
                  {
                    "name": "WITH_WORKLOAD",
                    "value": "true"
                  }
                ]
              }
            }
          }
        }
      ]
`
