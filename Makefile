
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

OS := $(shell go env GOOS)
ARCH ?= $(shell go env GOARCH)
export GOBIN=$(CURDIR)/bin
export PATH=$(GOBIN):$(shell echo $$PATH)

UNAME_OS := $(shell uname -s)

KIND := $(GOBIN)/kind
KIND_VERSION := v0.11.0
$(KIND):
	@curl -sSLo $(KIND) "https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-$(UNAME_OS)-amd64"
	@chmod +x $(KIND)

CLUSTER_NAME ?= kubetest-cluster
KUBECONFIG ?= $(CURDIR)/.kube/config
export KUBECONFIG

.PHONY: tools
tools:
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	make tools/envtest

K8S_VERSION := 1.19.2
tools/envtest: ./bin/k8s/$(K8S_VERSION)-$(OS)-$(ARCH)

./bin/k8s/$(K8S_VERSION)-$(OS)-$(ARCH):
	./bin/setup-envtest use --bin-dir ./bin --os $(OS) --arch $(ARCH) $(K8S_VERSION)
	ln -sf k8s/$(K8S_VERSION)-$(OS)-$(ARCH) bin/k8sbin

all: manager

test-cluster: $(KIND)
	@{ \
	set -e ;\
	if [ "$$(kind get clusters --quiet | grep $(CLUSTER_NAME))" = "" ]; then \
		$(KIND) create cluster --name $(CLUSTER_NAME) --config testdata/config/cluster.yaml ;\
	fi ;\
	}

delete-cluster: $(KIND)
	$(KIND) delete clusters $(CLUSTER_NAME)

deploy: test-cluster
	kubectl apply -f testdata/config/manifest.yaml

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

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager cmd/controller/main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./cmd/controller/main.go

# Install CRDs into a cluster
install: manifests
	kustomize build config/crd | kubectl apply --validate=false -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kustomize build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy-manifests: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kustomize build config/default | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
