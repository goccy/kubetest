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
      --user=              specify user ( organization ) name
      --repo=              specify repository name
      --token=             specify github auth token
      --token-from-secret= specify github auth token from secret resource. specify ( name.key ) style

Help Options:
  -h, --help               Show this help message
```

## Example

Test your private repository ( `github.com/user/repo.git` ) on Kubernetes .

```bash
$ kubetest --image golang:1.14 --repo user/repo --branch feature/branch --token xxxxxxx -- go test -v ./
```

# How to use as a library

```go
  clientset, _ := kubernetes.NewForConfig(config)
  job, _ := kubetest.NewTestJobBuilder(clientset, "default").
    SetUser("user").
    SetRepo("repo").
    SetBranch("feature").
    SetImage("golang:1.14").
    SetToken("xxxxxx").
    SetCommand([]string{"go", "test", "-v", "./"}).
    Build()
  job.Run(context.Background()) // start test and waiting for
```
