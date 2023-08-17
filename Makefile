SHELL := /bin/bash

export GOBIN := $(CURDIR)/bin
export PATH := $(GOBIN):$(PATH)

CLUSTER_NAME ?= kubetest-cluster
KUBECONFIG ?= $(CURDIR)/.kube/config
export KUBECONFIG

.PHONY: tools
tools:
	cd tools && GOFLAGS='-mod=readonly' go install \
		sigs.k8s.io/controller-runtime/tools/setup-envtest \
		sigs.k8s.io/kind

cluster/create: tools
	@{ \
	set -e ;\
	if [ "$$(kind get clusters --quiet | grep $(CLUSTER_NAME))" = "" ]; then \
		$(GOBIN)/kind create cluster --name $(CLUSTER_NAME) --config testdata/config/cluster.yaml ;\
	fi ;\
	}

cluster/delete:
	$(GOBIN)/kind delete clusters $(CLUSTER_NAME)

.PHONY: deploy
deploy: cluster/create deploy/image
	kubectl apply -f testdata/config/manifest.yaml

deploy/image:
	docker build --progress plain -f Dockerfile --target agent . -t 'kubetest:latest'
	kind load docker-image --name $(CLUSTER_NAME) 'kubetest:latest'

.PHONY: wait
wait:
	{ \
	set -e ;\
	while true; do \
		POD_NAME=$$(KUBECONFIG=$(KUBECONFIG) kubectl get pod | grep Running | grep kubetest-deployment | awk '{print $$1}'); \
		if [ "$$POD_NAME" != "" ]; then \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	}

.PHONY: test
test:
	{ \
	set -e ;\
	while true; do \
		POD_NAME=$$(KUBECONFIG=$(KUBECONFIG) kubectl get pod | grep Running | grep kubetest-deployment | awk '{print $$1}'); \
		if [ "$$POD_NAME" != "" ]; then \
			go test -race -v ./cmd/kubetest; \
			kubectl exec -it $$POD_NAME -- go test -race -v -coverprofile=coverage.out -covermode=atomic ./api/v1 -count=1; \
			exit $$?; \
		fi; \
		sleep 1; \
	done; \
	}

.PHONY: test-run
test-run:
	{ \
	set -e ;\
	while true; do \
		POD_NAME=$$(KUBECONFIG=$(KUBECONFIG) kubectl get pod | grep Running | grep kubetest-deployment | awk '{print $$1}'); \
		if [ "$$POD_NAME" != "" ]; then \
			kubectl exec -it $$POD_NAME -- go test -race -v -coverprofile=coverage.out -covermode=atomic ./api/v1 -count=1 -run $(TEST); \
			exit $$?; \
		fi; \
		sleep 1; \
	done; \
	}
