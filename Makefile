# Copyright 2017 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Build the node-problem-detector image.

.PHONY: all build-container build clean vet fmt version Dockerfile

all: build-container

# VERSION is the version of the binary.
VERSION:=$(shell git describe --tags --dirty)

# TAG is the tag of the container image, default to binary version.
TAG?=$(VERSION)

# PKG is the package name of node problem detector repo.
PKG:=k8s.io/coredump-detector

# PKG_SOURCES are all the go source code.
PKG_SOURCES:=$(shell find pkg cmd -name '*.go')

ifneq ($(KUBERNETES_HTTP_PROXY),)
	BUILD_ARG:=--build-arg https_proxy=$(KUBERNETES_HTTPS_PROXY) --build-arg http_proxy=$(KUBERNETES_HTTP_PROXY) --build-arg no_proxy=$(KUBERNETES_NO_PROXY)
endif

vet:
	go list ./... | grep -v "./vendor/*" | xargs go vet

fmt:
	find . -type f -name "*.go" | grep -v "./vendor/*" | xargs gofmt -s -w -l

version:
	@echo $(VERSION)

./bin/coredump-detector: $(PKG_SOURCES)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux go build -o bin/coredump-detector \
	     -ldflags '-X $(PKG)/pkg/version.version=$(VERSION)' \
	     cmd/coredump_detector.go

./bin/coredump-controller: $(PKG_SOURCES)
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux go build -o bin/coredump-controller \
	     -ldflags '-X $(PKG)/pkg/version.version=$(VERSION)' \
	     cmd/coredump_controller.go

build-detector-container: ./bin/coredump-detector Dockerfile-detector
	stat ./bin/kubectl >/dev/null 2>&1 || (echo "We need a kubectl binary inserting to image, please copy a kubectl file into dir bin/"; exit 1)
	docker build $(BUILD_ARG) -t coredump-detector:$(TAG) . -f  Dockerfile-detector

build-controller-container: ./bin/coredump-controller Dockerfile-controller
	docker build $(BUILD_ARG) -t coredump-controller:$(TAG) . -f  Dockerfile-controller


test: vet fmt
	go test -timeout=1m -v -race ./pkg/...

build: ./bin/coredump-detector ./bin/coredump-controller

build-container: build-detector-container build-controller-container

clean:
	rm -f bin/coredump-detector
	rm -f bin/coredump-controller
