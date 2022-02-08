module github.com/goccy/kubetest

go 1.16

require (
	github.com/bradleyfalzon/ghinstallation/v2 v2.0.3
	github.com/go-git/go-git/v5 v5.4.2
	github.com/go-logr/logr v0.4.0
	github.com/goccy/kubejob v0.2.13
	github.com/google/go-github/v29 v29.0.2
	github.com/jessevdk/go-flags v1.5.0
	github.com/lestrrat-go/backoff v1.0.1
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.12.0
	golang.org/x/mod v0.3.1-0.20200828183125-ce943fd02449 // indirect
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/tools v0.1.0 // indirect
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
	k8s.io/component-base v0.21.0 // indirect
	sigs.k8s.io/controller-runtime v0.8.3
)
