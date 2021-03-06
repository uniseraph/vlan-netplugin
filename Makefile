SHELL = /bin/bash

TARGET       = vlan-netplugin
PROJECT_NAME = github.com/omega/vlan-netplugin

MAJOR_VERSION = $(shell cat VERSION)
GIT_VERSION   = $(shell git log -1 --pretty=format:%h)
GIT_NOTES     = $(shell git log -1 --oneline)

BUILD_IMAGE     = golang:1.7.5


IMAGE_NAME = omega/vlan-netplugin
REGISTRY = registry.cn-hangzhou.aliyuncs.com

CLUSTER_STORE = zk://localhost:2181
TRUNK_DEV     = eth0

build:
	docker run --rm -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} make local

image:
	cp -r contrib/builder/image IMAGEBUILD && cp bundles/${MAJOR_VERSION}/binary/${TARGET} IMAGEBUILD
	docker build --rm -t ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION} IMAGEBUILD
	docker tag ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION} ${REGISTRY}/${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION}
	docker tag ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION} ${REGISTRY}/${IMAGE_NAME}:${MAJOR_VERSION}
	docker tag ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION} ${IMAGE_NAME}:${MAJOR_VERSION}
	rm -rf IMAGEBUILD

local:
	CGO_ENABLED=0 go build -v -ldflags "-B 0x$(shell head -c20 /dev/urandom|od -An -tx1|tr -d ' \n') -X ${PROJECT_NAME}/pkg/logging.ProjectName=${PROJECT_NAME} -X ${PROJECT_NAME}/Version=${MAJOR_VERSION}(${GIT_VERSION})" -o ${TARGET}
	mkdir -p bundles/${MAJOR_VERSION}/binary
	mv ${TARGET} bundles/${MAJOR_VERSION}/binary
	@cd bundles/${MAJOR_VERSION}/binary && $(shell which md5sum) -b ${TARGET} | cut -d' ' -f1  > ${TARGET}.md5

push:
	docker push ${REGISTRY}/${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION}
	docker push ${REGISTRY}/${IMAGE_NAME}:${MAJOR_VERSION}

run: build image
	docker run -ti --rm -v $(shell pwd):$(shell pwd) -v /var/run/docker.sock:/var/run/docker.sock -w $(shell pwd) -e DOCKER_HOST=unix:///var/run/docker.sock -e NP_CLUSTER_STORE=$(CLUSTER_STORE) -e NP_ETH=${TRUNK_DEV} -e LOG_LEVEL=debug docker/compose:1.9.0 up -d

default: build

all: build image run

.PHONY: build local image push
