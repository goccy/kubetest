package v1_test

import (
	"context"
	"strings"
	"testing"
	"time"

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

func Test_Run(t *testing.T) {
	t.Run("checkout branch", func(t *testing.T) {
		crd := `
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
            - $TEST
          workingDir: /go/src/kubetest/_examples
  distributedTest:
    containerName: test
    maxContainersPerPod: 18
    maxConcurrentNumPerPod: 2
    list:
      command:
        - go
      args:
        - test
        - -list
        - .
      pattern: ^Test
`
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
	})
	t.Run("merge base branch", func(t *testing.T) {
		crd := `
apiVersion: kubetest.io/v1
kind: TestJob
metadata:
  name: testjob
  namespace: default
spec:
  git:
    repo: github.com/goccy/kubetest
    merge:
      base: master
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
            - $TEST
          workingDir: /go/src/kubetest/_examples/success
  distributedTest:
    containerName: test
    maxContainersPerPod: 2
    maxConcurrentNumPerPod: 1
    list:
      command:
        - go
      args:
        - test
        - -list
        - .
      pattern: ^Test
`
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
	})
	t.Run("retest", func(t *testing.T) {
		crd := `
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
            - $TEST
          workingDir: /go/src/kubetest/_examples/failure
  distributedTest:
    containerName: test
    maxContainersPerPod: 2
    retest: true
    list:
      command:
        - go
      args:
        - test
        - -list
        - .
      pattern: ^Test
`
		runner, err := kubetestv1.NewTestJobRunner(cfg)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		var job kubetestv1.TestJob
		if err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(crd), 1024).Decode(&job); err != nil {
			t.Fatalf("%+v", err)
		}
		if err := runner.Run(context.Background(), job); err == nil {
			t.Fatal("expected error")
		}
	})
}

func Test_RunWithDebugLog(t *testing.T) {
	crd := `
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
            - $TEST
          workingDir: /go/src/kubetest/_examples
  distributedTest:
    containerName: test
    maxContainersPerPod: 18
    maxConcurrentNumPerPod: 2
    list:
      command:
        - go
      args:
        - test
        - -list
        - .
      pattern: ^Test
`
	runner, err := kubetestv1.NewTestJobRunner(cfg)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	runner.EnableVerboseLog()
	var job kubetestv1.TestJob
	if err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(crd), 1024).Decode(&job); err != nil {
		t.Fatalf("%+v", err)
	}
	if err := runner.Run(context.Background(), job); err != nil {
		t.Fatalf("%+v", err)
	}
}

func Test_ForceStop(t *testing.T) {
	crd := `
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
            - $TEST
          workingDir: /go/src/kubetest/_examples
  distributedTest:
    containerName: test
    maxContainersPerPod: 18
    maxConcurrentNumPerPod: 2
    list:
      command:
        - go
      args:
        - test
        - -list
        - .
      pattern: ^Test
`
	runner, err := kubetestv1.NewTestJobRunner(cfg)
	if err != nil {
		t.Fatalf("%+v", err)
	}
	runner.EnableVerboseLog()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()

		select {
		case <-time.After(2 * time.Second):
		}
	}()
	var job kubetestv1.TestJob
	if err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(crd), 1024).Decode(&job); err != nil {
		t.Fatalf("%+v", err)
	}
	if err := runner.Run(ctx, job); err != nil {
		t.Fatalf("%+v", err)
	}
}

func Test_RunWithSideCar(t *testing.T) {
	crd := `
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
            - $TEST
          workingDir: /go/src/kubetest/_examples
        - name: sidecar
          image: nginx:latest
          command:
            - nginx
  distributedTest:
    containerName: test
    maxContainersPerPod: 18
    maxConcurrentNumPerPod: 2
    list:
      command:
        - go
      args:
        - test
        - -list
        - .
      pattern: ^Test
`
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
