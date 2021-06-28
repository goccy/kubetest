package v1_test

import (
	"context"
	"os"
	"path/filepath"
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

func Test_RunTest(t *testing.T) {
	t.Parallel()
	t.Run("checkout branch", func(t *testing.T) {
		t.Parallel()
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
          workingDir: /go/src/kubetest/_examples
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
	t.Run("checkout revision", func(t *testing.T) {
		t.Parallel()
		crd := `
apiVersion: kubetest.io/v1
kind: TestJob
metadata:
  name: testjob
  namespace: default
spec:
  git:
    repo: github.com/goccy/kubetest
    rev: HEAD
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
          workingDir: /go/src/kubetest/_examples
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
	t.Run("with prepare", func(t *testing.T) {
		t.Parallel()
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
  prepare:
    image: golang:1.15
    steps:
      - name: "pwd"
        image: golang:1.15
        command: |
          pwd
        workdir: /go/src/kubetest
        env:
          - name: PREPARE
            value: true
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
          workingDir: /go/src/kubetest/_examples
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
}

func Test_RunDistributedTest(t *testing.T) {
	t.Parallel()
	t.Run("checkout branch", func(t *testing.T) {
		t.Parallel()
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
		t.Parallel()
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
		t.Parallel()
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
	t.Run("use shared cache", func(t *testing.T) {
		t.Parallel()
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
    list:
      command:
        - go
      args:
        - test
        - -list
        - .
      pattern: ^Test
    cache:
      - name: cache-test
        command: |
          touch shared-cache.txt
        path: ./cache
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
	t.Run("static list", func(t *testing.T) {
		t.Parallel()
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
          workingDir: /go/src/kubetest/_examples/success
  distributedTest:
    containerName: test
    maxContainersPerPod: 2
    list:
      names:
        - Test_A
        - Test_B
        - Test_C
        - Test_D
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
			t.Fatal(err)
		}
	})
	t.Run("fail list", func(t *testing.T) {
		t.Parallel()
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
        - invalid
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
		runErr := runner.Run(context.Background(), job)
		if runErr == nil {
			t.Fatal("expected error")
		}
		t.Logf("%+v", runErr)
	})
}

func Test_RunWithDebugLog(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func Test_Artifacts(t *testing.T) {
	t.Parallel()
	t.Run("success", func(t *testing.T) {
		t.Parallel()
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
            - -coverprofile
            - cover.out
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
    artifacts:
      paths:
        - cover.out
      output:
        path: artifacts
`
		runner, err := kubetestv1.NewTestJobRunner(cfg)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		var job kubetestv1.TestJob
		if err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(crd), 1024).Decode(&job); err != nil {
			t.Fatalf("%+v", err)
		}
		os.RemoveAll("artifacts")
		if err := runner.Run(context.Background(), job); err != nil {
			t.Fatalf("%+v", err)
		}
		var foundArtifacts bool
		if err := filepath.Walk("artifacts", func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if info.Name() != "cover.out" {
				t.Fatalf("unexpected file name %s", info.Name())
			}
			foundArtifacts = true
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if !foundArtifacts {
			t.Fatal("cannot find artifacts")
		}
	})
	t.Run("failure", func(t *testing.T) {
		t.Parallel()
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
            - -coverprofile
            - cover.out
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
    artifacts:
      paths:
        - invalid.txt
      output:
        path: failure_artifacts
`
		runner, err := kubetestv1.NewTestJobRunner(cfg)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		var job kubetestv1.TestJob
		if err := yaml.NewYAMLOrJSONDecoder(strings.NewReader(crd), 1024).Decode(&job); err != nil {
			t.Fatalf("%+v", err)
		}
		os.RemoveAll("failure_artifacts")
		if err := runner.Run(context.Background(), job); err != nil {
			t.Fatalf("%+v", err)
		}
		var foundInvalidArtifact bool
		if err := filepath.Walk("failure_artifacts", func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			foundInvalidArtifact = true
			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if foundInvalidArtifact {
			t.Fatal("failed to handle invalid artifact")
		}
	})
}
