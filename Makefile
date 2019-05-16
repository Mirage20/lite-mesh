


PROJECT_ROOT := $(realpath $(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
BUILD_DIRECTORY := out
BUILD_ROOT := $(PROJECT_ROOT)/$(BUILD_DIRECTORY)
GIT_REVISION := $(shell git rev-parse --verify HEAD)

MAIN_PACKAGES := controller discovery envoy-bootstrap
BUILD_TARGETS := $(addprefix build., $(MAIN_PACKAGES))


VERSION ?= $(GIT_REVISION)

DOCKER_TARGETS := $(addprefix docker., $(MAIN_PACKAGES))
DOCKER_PUSH_TARGETS := $(addprefix docker-push., $(MAIN_PACKAGES))
DOCKER_REPO ?= mirage20
DOCKER_IMAGE_PREFIX := lite-mesh
DOCKER_IMAGE_TAG ?= $(VERSION)

all: build artifacts

.PHONY: $(BUILD_TARGETS)
$(BUILD_TARGETS):
	$(eval TARGET=$(patsubst build.%,%,$@))
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o $(BUILD_ROOT)/$(TARGET) $(PROJECT_ROOT)/cmd/$(TARGET)

.PHONY: build
build: $(BUILD_TARGETS)


.PHONY: $(DOCKER_TARGETS)
$(DOCKER_TARGETS): docker.% : build.% prepare-docker-build
	$(eval TARGET=$(patsubst docker.%,%,$@))
	docker build -f $(PROJECT_ROOT)/docker/$(TARGET)/Dockerfile $(BUILD_ROOT) -t $(DOCKER_REPO)/$(DOCKER_IMAGE_PREFIX)-$(TARGET):$(DOCKER_IMAGE_TAG)
	docker tag $(DOCKER_REPO)/$(DOCKER_IMAGE_PREFIX)-$(TARGET):$(DOCKER_IMAGE_TAG) $(DOCKER_REPO)/$(DOCKER_IMAGE_PREFIX)-$(TARGET):latest

.PHONY: docker
docker: $(DOCKER_TARGETS)

.PHONY: $(DOCKER_PUSH_TARGETS)
$(DOCKER_PUSH_TARGETS): docker-push.% : docker.%
	$(eval TARGET=$(patsubst docker-push.%,%,$@))
	docker push $(DOCKER_REPO)/$(DOCKER_IMAGE_PREFIX)-$(TARGET):$(DOCKER_IMAGE_TAG)
	docker push $(DOCKER_REPO)/$(DOCKER_IMAGE_PREFIX)-$(TARGET):latest

.PHONY: docker-push
docker-push: $(DOCKER_PUSH_TARGETS)


.PHONY: prepare-docker-build
prepare-docker-build:
	cp $(PROJECT_ROOT)/envoy-bootstrap-template.yaml $(BUILD_ROOT)
