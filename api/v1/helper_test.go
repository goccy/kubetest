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
	envtestCfg *rest.Config
)

func getConfig() *rest.Config {
	return envtestCfg
}

func TestMain(m *testing.M) {
	result := func() int {
		os.Setenv(envKubebuilderPath, "../../bin/k8sbin")
		testenv := envtest.Environment{}
		cfg, err := testenv.Start()
		if err != nil {
			panic(err)
		}
		envtestCfg = cfg
		defer testenv.Stop()
		return m.Run()
	}()
	os.Exit(result)
}
