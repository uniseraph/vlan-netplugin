SHELL = /bin/bash


TARGET       = omega-vlan-netplugin
PROJECT_NAME = github.com/omega/omega-vlan-netplugin

MAJOR_VERSION = $(shell cat VERSION)
GIT_VERSION   = $(shell git log -1 --pretty=format:%h)
GIT_NOTES     = $(shell git log -1 --oneline)

BUILD_IMAGE     = 1.7.5


IMAGE_NAME = omega/vlan-netplugin
REGISTRY = registry.cn-hangzhou.aliyuncs.com

build:
	docker run --rm -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} make build-local


image:
	cp -r contrib/builder/image IMAGEBUILD && cp bundles/${MAJOR_VERSION}/binary/network-plugins IMAGEBUILD
	docker build --rm -t ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION} IMAGEBUILD
	docker tag ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION} ${REGISTRY}/${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION}
	docker tag ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION} ${REGISTRY}/${IMAGE_NAME}:${MAJOR_VERSION}
	rm -rf IMAGEBUILD

build-local:
	GO15VENDOREXPERIMENT=1 go build -v -ldflags "-B 0x$(shell head -c20 /dev/urandom|od -An -tx1|tr -d ' \n') -X ${PROJECT_NAME}/pkg/logging.ProjectName=${PROJECT_NAME} -X ${PROJECT_NAME}/Version=${MAJOR_VERSION}(${GIT_VERSION})" -o ${TARGET}
	mkdir -p bundles/${MAJOR_VERSION}/binary
	mv ${TARGET} bundles/${MAJOR_VERSION}/binary
	@cd bundles/${MAJOR_VERSION}/binary && $(shell which md5sum) -b ${TARGET} | cut -d' ' -f1  > ${TARGET}.md5



.PHONY: build build-local rpm rpm-build-local image shell
