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
  -n, --namespace=         specify namespace (default: default)
      --in-cluster         specify whether in cluster
  -c, --config=            specify local kubeconfig path. ( default: $HOME/.kube/config )
  -i, --image=             specify container image
  -b, --branch=            specify branch name
      --rev=               specify revision ( commit hash )
      --repo=              specify repository name
      --token-from-secret= specify github auth token from secret resource. specify ( name.key ) style

Help Options:
  -h, --help               Show this help message
```

## Example

Test your private repository ( `github.com/user/repo.git` ) on Kubernetes .

```bash
$ kubetest --image golang:1.14 --repo github.com/user/repo --branch feature/branch --token-from-secret name.key -- go test -v ./
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
            - -run
            - $TEST
  distributedTest:
    containerName: test
    maxContainersPerPod: 16
    list:
      command:
        - go
      args:
        - test
        - -list
        - ./
      pattern: '^"Test'
```
