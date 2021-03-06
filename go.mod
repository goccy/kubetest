module github.com/goccy/kubetest

go 1.14

require (
	github.com/go-logr/logr v0.4.0
	github.com/goccy/kubejob v0.2.5
	github.com/jessevdk/go-flags v1.5.0
	github.com/lestrrat-go/backoff v1.0.1 // indirect
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.12.0
	github.com/rs/xid v1.3.0
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
	k8s.io/kubectl v0.21.0
	sigs.k8s.io/controller-runtime v0.8.3
)
