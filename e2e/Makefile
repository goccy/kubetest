SHELL := /bin/bash

BIN := $(CURDIR)/.bin
PATH := $(abspath $(BIN)):$(PATH)

UNAME_OS := $(shell uname -s)

$(BIN):
	@mkdir -p $(BIN)

KIND := $(BIN)/kind
KIND_VERSION := v0.8.1
$(KIND): | $(BIN)
	@curl -sSLo $(KIND) "https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-$(UNAME_OS)-amd64"
	@chmod +x $(KIND)

TEST_CLUSTER_NAME ?= kubetest-e2e
KUBECONFIG ?= $(CURDIR)/.kube/config
export KUBECONFIG

test-cluster: $(KIND)
	@{ \
	set -e ;\
	if [ "$$(kind get clusters --quiet | grep $(TEST_CLUSTER_NAME))" = "" ]; then \
		$(KIND) create cluster --name $(TEST_CLUSTER_NAME) --config testdata/config/cluster.yaml ;\
	fi ;\
	}

build-image:
	@{ \
		set -e ;\
		if [ "$$(docker image ls -q kubetest:v1)" = "" ]; then \
			docker build -t kubetest:v1 ../ ;\
		fi ;\
	}

delete-image:
	docker image rm kubetest:v1

update-image: delete-image build-image

upload-image: build-image
	@$(KIND) load --name $(TEST_CLUSTER_NAME) docker-image kubetest:v1

deploy: test-cluster upload-image
	kubectl apply -f testdata/config/pod.yaml
	kubectl apply -f https://docs.projectcalico.org/v3.8/manifests/calico.yaml

delete-pod:
	kubectl delete pod job-pod

test:
	kubectl cp ./testjob.yaml job-pod:/go/
	kubectl exec job-pod -- kubetest --in-cluster -f testjob.yaml --template JobName=e2e
