# kubetest

CLI and Go library with Custom Resource for ( distributed ) testing on Kubernetes

# Status

WIP

# Installation

## Install as a CLI

```bash
$ go get github.com/goccy/kubetest/cmd/kubetest
```

# How to use CLI

```
Usage:
  kubetest [OPTIONS]

Application Options:
  -n, --namespace=              specify namespace (default: default)
      --in-cluster              specify whether in cluster
  -c, --config=                 specify local kubeconfig path. ( default: $HOME/.kube/config )
  -i, --image=                  specify container image
      --repo=                   specify repository name
  -b, --branch=                 specify branch name
      --rev=                    specify revision ( commit hash )
      --token-from-secret=      specify github auth token from secret resource. specify ( name.key ) style
      --image-pull-secret=      specify image pull secret name
      --max-containers-per-pod= specify max number of container per pod
      --list=                   specify command for listing test
      --list-delimiter=         specify delimiter for list command
      --pattern=                specify test name patter
      --retest                  specify enabled retest if exists failed tests
      --retest-delimiter=       specify delimiter for failed tests at retest command
  -f, --file=                   specify yaml file path
      --template=               specify template parameter for file specified with --file option

Help Options:
  -h, --help                    Show this help message
```

# Custom Resource Definition

## 1. Testing

```yaml
apiVersion: kubetest.io/v1
kind: TestJob
metadata:
  name: testJobName
  namespace: namespaceName
spec:
  git:
    repo: github.com/goccy/kubetest
    branch: master
  template:
    spec:
      containers:
        - name: test
          image: golang:1.14
          command:
            - go
          args:
            - test
            - ./
```

## 2. Distributed Testing

This is Go language example.

```yaml
apiVersion: kubetest.io/v1
kind: TestJob
metadata:
  name: testjob
spec:
  git:
    repo: github.com/goccy/kubetest
    checkout: false
  template:
    spec:
      containers:
        - name: test
          image: kubetest:v1
          command:
            - go
          args:
            - test
            - -v
            - ./
            - -run
            - $(TEST)
          workingDir: /go/src/github.com/goccy/kubetest/_examples
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
```

# How it Works

## Distributed Test

`kubetest` has a mechanism to efficiently use Kubernetes resources to perform distributed testing in order to run fast tests.
Here is an example of how kubetest does distributed testing, using an example in [e2e/testjob.yaml](https://github.com/goccy/kubetest/blob/master/e2e/testjob.yaml).

The `testjob.yaml` is written as follows:

```yaml
apiVersion: kubetest.io/v1
kind: TestJob
metadata:
  name: testjob
spec:
  git:
    repo: github.com/goccy/kubetest
    checkout: false
  template:
    spec:
      containers:
        - name: test
          image: kubetest:v1
          command:
            - go
          args:
            - test
            - -v
            - ./
            - -run
            - $(TEST)
          workingDir: /go/src/github.com/goccy/kubetest/_examples
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
```

Running `make deploy` under the `e2e` directory and then `make test` will run the tests using the `testjob.yaml` on your local Kubernetes cluster.

In `make test`, after attaching to the test container, the following command is executed.

```bash
$ kubetest --in-cluster -f testjob.yaml
```

The `kubetest` CLI reads the definition of `TestJob`, which is a CRD of Kubernetes, and executes the test for the specified Kubernetes cluster.

When the `--in-cluster` option is specified, it means the CLI is executed in the target Kubernetes cluster and works according to the privileges of ServiceAccount specified in the pod running the `kubetest` CLI.
(See [here](https://github.com/goccy/kubetest/blob/master/README.md#serviceaccount) for permissions required to run `kubetest`)

The `testjob.yaml` passed to the argument of the `-f` option is the same as the one shown above.

### Execution flow of kubetest

The `kubetest` is divided into the following two phases.

1. **List Phase**: get a list of testing.
2. **Test Phase**: Divide the test list into multiple pods according to the specified parameters and execute them in multiple pods.

In the `testjob.yaml` example, we move into the `_examples` directory to run the tests. In the `_examples` directory, there is test code like this: 

- `_examples/go_test.go`

```go
package test

import (
	"testing"
	"time"
)

func Test_A(t *testing.T) {
	t.Log("Test A")
	time.Sleep(time.Second)
}

func Test_B(t *testing.T) {
	t.Log("Test B")
	time.Sleep(time.Second)
}

func Test_C(t *testing.T) {
	t.Log("Test C")
	time.Sleep(time.Second)
}

func Test_D(t *testing.T) {
	t.Log("Test D")
	time.Sleep(time.Second)
}

func Test_E(t *testing.T) {
	t.Log("Test E")
	time.Sleep(time.Second)
}

func Test_F(t *testing.T) {
	t.Log("Test F")
	time.Sleep(time.Second)
}

func Test_G(t *testing.T) {
	t.Log("Test G")
	time.Sleep(time.Second)
}

func Test_H(t *testing.T) {
	t.Log("Test H")
	time.Sleep(time.Second)
}

func Test_I(t *testing.T) {
	t.Log("Test I")
	time.Sleep(time.Second)
}

func Test_J(t *testing.T) {
	t.Log("Test J")
	time.Sleep(time.Second)
}

func Test_K(t *testing.T) {
	t.Log("Test K")
	time.Sleep(time.Second)
}

func Test_L(t *testing.T) {
	t.Log("Test L")
	time.Sleep(time.Second)
}

func Test_M(t *testing.T) {
	t.Log("Test M")
	time.Sleep(time.Second)
}

func Test_N(t *testing.T) {
	t.Log("Test N")
	time.Sleep(time.Second)
}
```

There are tests from `Test_A` to `Test_N`, and each test case is designed to sleep for 1 second, so it would take almost 14 seconds to run a normal test.

If we run a distributed test against this test code, we get the following figure.

<img width="968" alt="kubetest workflow" src="https://user-images.githubusercontent.com/209884/97277516-ecee0d00-187b-11eb-9bcd-76b898d0e230.png">

#### List Phase

First, we use the parameters specified in the `distributedTest.list`, we run the `go test -list .` under the `_examples` directory to get a list of tests. Then we filter the results of the `go test -list .` according to the rules described in `distributedTest.list.pattern`. By default, it splits the output into a single line of output, starting with `Test`, since the output is split by `\n` by default.
This gives you a list of names from `Test_A` to `Test_N`.

#### Test Phase

In the Test Phase, we first distribute the list of test obtained in the **List Phase** to `distributedTest.maxContainersPerPod` with the specified number. In this example, `6` is specified, so there will be `6` test containers running for each pod.
In this example, the total number of tests is `14` from `A` to `N`, so there are three pods with `6` / `6` / `2` containers running.

Environment variables for each test container are set in the `template.spec.containers`.
The container specified here will be the one with the name specified by the `distributedTest.containerName`.

The `kubetest` sets the environment variable **`TEST`** to the environment variables of each container. This variable is the name of the test obtained from the **List Phase**, and unit tests are run with this value. 
Since `Go` can specify the test target with `-run`, you can run the tests individually by writing the following command.

```bash
$ go test -v ./ -run $(TEST)
```

You will get the following results when executing `kubetest`.

```
kubetest --in-cluster -f testjob.yaml
get listing of tests...
list: elapsed time 3.635022 sec
[POD 0] TEST=Test_N go test -v ./ -run $(TEST)
[POD 0] === RUN   Test_N
[POD 0]     go_test.go:74: Test N
[POD 0] --- PASS: Test_N (1.00s)
[POD 0] PASS
[POD 0] ok      github.com/goccy/kubetest/_examples     1.005s

[POD 1] TEST=Test_G go test -v ./ -run $(TEST)
[POD 1] === RUN   Test_G
[POD 1]     go_test.go:39: Test G
[POD 1] --- PASS: Test_G (1.00s)
[POD 1] PASS
[POD 1] ok      github.com/goccy/kubetest/_examples     1.018s

[POD 2] TEST=Test_B go test -v ./ -run $(TEST)
[POD 2] === RUN   Test_B
[POD 2]     go_test.go:14: Test B
[POD 2] --- PASS: Test_B (1.00s)
[POD 2] PASS
[POD 2] ok      github.com/goccy/kubetest/_examples     1.010s

[POD 2] TEST=Test_C go test -v ./ -run $(TEST)
[POD 2] === RUN   Test_C
[POD 2]     go_test.go:19: Test C
[POD 2] --- PASS: Test_C (1.00s)
[POD 2] PASS
[POD 2] ok      github.com/goccy/kubetest/_examples     1.009s

[POD 0] TEST=Test_M go test -v ./ -run $(TEST)
[POD 0] === RUN   Test_M
[POD 0]     go_test.go:69: Test M
[POD 0] --- PASS: Test_M (1.00s)
[POD 0] PASS
[POD 0] ok      github.com/goccy/kubetest/_examples     1.006s

[POD 1] TEST=Test_H go test -v ./ -run $(TEST)
[POD 1] === RUN   Test_H
[POD 1]     go_test.go:44: Test H
[POD 1] --- PASS: Test_H (1.00s)
[POD 1] PASS
[POD 1] ok      github.com/goccy/kubetest/_examples     1.011s

[POD 2] TEST=Test_E go test -v ./ -run $(TEST)
[POD 2] === RUN   Test_E
[POD 2]     go_test.go:29: Test E
[POD 2] --- PASS: Test_E (1.00s)
[POD 2] PASS
[POD 2] ok      github.com/goccy/kubetest/_examples     1.005s

[POD 2] TEST=Test_A go test -v ./ -run $(TEST)
[POD 2] === RUN   Test_A
[POD 2]     go_test.go:9: Test A
[POD 2] --- PASS: Test_A (1.00s)
[POD 2] PASS
[POD 2] ok      github.com/goccy/kubetest/_examples     1.009s

[POD 1] TEST=Test_I go test -v ./ -run $(TEST)
[POD 1] === RUN   Test_I
[POD 1]     go_test.go:49: Test I
[POD 1] --- PASS: Test_I (1.00s)
[POD 1] PASS
[POD 1] ok      github.com/goccy/kubetest/_examples     1.004s

[POD 1] TEST=Test_L go test -v ./ -run $(TEST)
[POD 1] === RUN   Test_L
[POD 1]     go_test.go:64: Test L
[POD 1] --- PASS: Test_L (1.00s)
[POD 1] PASS
[POD 1] ok      github.com/goccy/kubetest/_examples     1.005s

[POD 2] TEST=Test_F go test -v ./ -run $(TEST)
[POD 2] === RUN   Test_F
[POD 2]     go_test.go:34: Test F
[POD 2] --- PASS: Test_F (1.00s)
[POD 2] PASS
[POD 2] ok      github.com/goccy/kubetest/_examples     1.004s

[POD 1] TEST=Test_J go test -v ./ -run $(TEST)
[POD 1] === RUN   Test_J
[POD 1]     go_test.go:54: Test J
[POD 1] --- PASS: Test_J (1.00s)
[POD 1] PASS
[POD 1] ok      github.com/goccy/kubetest/_examples     1.005s

[POD 1] TEST=Test_K go test -v ./ -run $(TEST)
[POD 1] === RUN   Test_K
[POD 1]     go_test.go:59: Test K
[POD 1] --- PASS: Test_K (1.00s)
[POD 1] PASS
[POD 1] ok      github.com/goccy/kubetest/_examples     1.003s

[POD 2] TEST=Test_D go test -v ./ -run $(TEST)
[POD 2] === RUN   Test_D
[POD 2]     go_test.go:24: Test D
[POD 2] --- PASS: Test_D (1.00s)
[POD 2] PASS
[POD 2] ok      github.com/goccy/kubetest/_examples     1.003s

test: elapsed time 12.042257 sec
{"details":{"tests":[{"elapsedTimeSec":8,"name":"Test_N","testResult":"success"},{"elapsedTimeSec":9,"name":"Test_G","testResult":"success"},{"elapsedTimeSec":8,"name":"Test_C","testResult":"success"},{"elapsedTimeSec":9,"name":"Test_K","testResult":"success"},{"elapsedTimeSec":10,"name":"Test_D","testResult":"success"},{"elapsedTimeSec":9,"name":"Test_B","testResult":"success"},{"elapsedTimeSec":9,"name":"Test_E","testResult":"success"},{"elapsedTimeSec":9,"name":"Test_I","testResult":"success"},{"elapsedTimeSec":10,"name":"Test_J","testResult":"success"},{"elapsedTimeSec":10,"name":"Test_M","testResult":"success"},{"elapsedTimeSec":9,"name":"Test_H","testResult":"success"},{"elapsedTimeSec":10,"name":"Test_A","testResult":"success"},{"elapsedTimeSec":9,"name":"Test_L","testResult":"success"},{"elapsedTimeSec":9,"name":"Test_F","testResult":"success"}]},"elapsedTimeSec":15,"job":"testjob","startedAt":"2020-10-27T09:28:10.3624005Z","testResult":"success"}
```

The `[POD 0]` in the output shows the executed `Pod` with the index. Since three `Pods` are used in the example, the output has three indices: `0`, `1`, and `2`.

At the end of the execution result, the test result is output as a JSON log and you can see the execution result and elapsed time of each test case.

### Pod Content

This section describes what goes on in the Pod for the test execution during a distributed test run.

<img width="388" alt="pod" src="https://user-images.githubusercontent.com/209884/97283569-59203f00-1883-11eb-99fc-f1ad3374b25d.png">

As shown in the above figure, when a pod is launched, `Init Containers` , `Containers` are launched in that order. The `Init Containers` is run to prep the pod for testing, and is used for `git clone` `git switch` to checkout to a particular revision of the repository under test and to build a cache to be shared by the containers, as shown in the figure.
The `checkoutDir` specified here is `volumeMount` as an `emptyDir` and the same directory is mounted as `volumes` to make it reusable by the containers.

```yaml
spec:
  git:
    repo: github.com/goccy/kubetest
    revision: abcdefg
    checkoutDir: /home/workspace
```

As shown above, the `.spec.git.repo` and `.spec.git.revision` will work as shown in the figure. You can also specify the `branch` name instead of the `revision`.

If the code to be tested is already pre-installed in image, as in the example, you can run the test without using `Init Containers` by specifying the following.

```yaml
spec:
  git:
    checkout: false
```

When you run a test, you may want to run a different container as a sidecar. In this case, you can write the following to start a `mysql` container as a sidecar in each pod.

```yaml
spec:
  template:
    spec:
      containers:
        - name: sidecar
          image: mysql
        - name: test
distributedTest:
  containerName: test
```

## ServiceAccount

```yaml
kind: ServiceAccount
apiVersion: v1
metadata:
  name: kubetest
---
kind: Role
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: kubetest
rules:
  - apiGroups:
      - batch
    resources:
      - jobs
    verbs:
      - create
      - delete
  - apiGroups:
      - ""
    resources:
      - pods
      - pods/log
    verbs:
      - get
      - watch
  - apiGroups:
      - ""
    resources:
      - secrets
    verbs:
      - get
---
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1beta1
metadata:
  name: kubetest
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: kubetest
subjects:
- kind: ServiceAccount
  name: kubetest
---
```
