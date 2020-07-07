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
  name: testjob-sample
spec:
  image: golang:1.14
  repo: github.com/user/repo
  branch: master
  command:
    - go
    - test
    - -v
    - ./
  token: # if test for private repository, add oauth token to secret before testing and use it.
    secretKeyRef:
      name: oauth-token
      key: oauth
```

## 2. Distributed Testing

This is Go language example.

```yaml
apiVersion: kubetest.io/v1
kind: TestJob
metadata:
  name: testjob-sample
spec:
  image: golang:1.14
  repo: github.com/user/repo
  branch: master
  command:
    - go
    - test
    - -v
    - ./
    - -run
    - '{{.Test}}' # '{{.Test}}' is special keyword, this section is replaced to each test name on runtime.
  token:
    secretKeyRef:
      name: oauth-token
      key: oauth

  # for distributed testing parameters.
  distributedTest:
    # concurrent number for testing.
    concurrent: 2

    # output testing list to stdout
    listCommand:
      - go
      - test
      - -list
      - Test

    # filter testing list by this regular expression.
    pattern: ^Test

    # restart testing for failed tests
    retest: true

    # delimiter for testing list of retest ( default: white space )
    retestDelimiter: '|'
```
