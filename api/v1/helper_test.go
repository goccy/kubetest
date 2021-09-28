package v1

import (
	"os"
	"testing"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

const (
	envKubebuilderPath = "KUBEBUILDER_ASSETS"
)

var (
	kubecfg   *rest.Config
	runModes  []RunMode
	inCluster bool
)

func init() {
	c, err := rest.InClusterConfig()
	if err == nil {
		inCluster = true
		kubecfg = c
		runModes = []RunMode{
			RunModeLocal,
			RunModeKubernetes,
		}
	}
}

func getConfig() *rest.Config {
	return kubecfg
}

func getRunModes() []RunMode {
	return runModes
}

func TestMain(m *testing.M) {
	result := func() int {
		if kubecfg == nil {
			os.Setenv(envKubebuilderPath, "../../bin/k8sbin")
			testenv := envtest.Environment{}
			cfg, err := testenv.Start()
			if err != nil {
				panic(err)
			}
			kubecfg = cfg
			runModes = []RunMode{
				RunModeLocal,
			}
			defer testenv.Stop()
		}
		return m.Run()
	}()
	os.Exit(result)
}
