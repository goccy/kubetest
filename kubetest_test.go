package kubetest_test

import (
	"context"
	"strings"
	"testing"

	kubetestv1 "github.com/goccy/kubetest/api/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/rest"
)

var (
	cfg *rest.Config
)

func init() {
	c, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	cfg = c
}

var (
	crd = `
apiVersion: kubetest.io/v1
kind: TestJob
metadata:
  name: testjob
  namespace: default
spec:
  git:
    repo: github.com/goccy/kubetest
    branch: master
    checkoutDir: /go/src/kubetest
  template:
    spec:
      containers:
        - name: test
          image: golang:1.15
          command:
            - go
          args:
            - test
            - -v
            - ./
            - -run
            - $(TEST)
          workingDir: /go/src/kubetest/_examples
  distributedTest:
    containerName: test
    maxContainersPerPod: 6
    list:
      command:
        - go
      args:
        - test
        - -list
        - .
      pattern: ^Test
`
)

func Test_Run(t *testing.T) {
	runner, err := kubetestv1.NewTestJobRunner(cfg)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	var job kubetestv1.TestJob
	if err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(crd), 1024).Decode(&job); err != nil {
		t.Fatalf("%+v", err)
	}
	if err := runner.Run(context.Background(), job); err != nil {
		t.Fatalf("%+v", err)
	}
}
