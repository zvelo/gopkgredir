ALL_DIRS=$(shell find . \( -path ./Godeps -o -path ./vendor -o -path ./.git \) -prune -o -type d -print)
GO_PKGS=$(foreach pkg, $(shell go list ./...), $(if $(findstring /vendor/, $(pkg)), , $(pkg)))
EXECUTABLE=gopkgredir
DOCKER_DIR=Docker
DOCKER_FILE=$(DOCKER_DIR)/Dockerfile
GO_FILES=$(foreach dir, $(ALL_DIRS), $(wildcard $(dir)/*.go))
SRC_DIR = $(shell cd ../../.. && pwd)
METALINT = "gometalinter --cyclo-over=10 -D gotype -t"
DOCKER_RELEASE_TAG=$(shell date +%Y%m%d-%H%M%S)
GOLANG_VERSION ?= 1.6
GO_LINTERS ?= golint "go vet"

GIT_COMMIT=unknown

ifeq ("$(CIRCLECI)", "true")
	CI_SERVICE = circle-ci
	export GIT_BRANCH = $(CIRCLE_BRANCH)
	GIT_COMMIT = $(CIRCLE_SHA1)
endif

export GO15VENDOREXPERIMENT=1

all: build

lint:
	@for pkg in $(GO_PKGS); do \
		for cmd in $(GO_LINTERS); do \
			eval "$$cmd $$pkg" || true; \
		done; \
	done

metalint:
	@for pkg in $(GO_PKGS); do \
		eval "$(METALINT) $(SRC_DIR)/$$pkg" | grep -v '\/vendor\/' || true; \
	done

build: $(EXECUTABLE)

$(EXECUTABLE): $(GO_FILES)
	go build -v -o $(EXECUTABLE) $(PKG)

clean:
	@rm -f $(EXECUTABLE) $(DOCKER_DIR)/$(EXECUTABLE)

save: .save-stamp

.save-stamp: $(GO_FILES)
	@rm -rf ./Godeps ./vendor
	GOOS=linux GOARCH=amd64 godep save ./...
	@touch .save-stamp

$(DOCKER_DIR)/$(EXECUTABLE): $(GO_FILES)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -v -tags netgo -installsuffix netgo -o $(DOCKER_DIR)/$(EXECUTABLE) $(PKG)

image: .image-stamp

.image-stamp: $(DOCKER_DIR)/$(EXECUTABLE) $(DOCKER_FILE)
	docker build -t zvelo/$(EXECUTABLE) $(DOCKER_DIR)
	@touch .image-stamp

push:
	docker tag -f $(DOCKER_IMAGE):latest $(DOCKER_IMAGE):$(DOCKER_RELEASE_TAG)
	docker push $(DOCKER_IMAGE):latest
	docker push $(DOCKER_IMAGE):$(DOCKER_RELEASE_TAG)

$(HOME)/go/go$(GOLANG_VERSION).linux-amd64.tar.gz:
	@mkdir -p $(HOME)/go
	wget https://storage.googleapis.com/golang/go$(GOLANG_VERSION).linux-amd64.tar.gz -O $(HOME)/go/go$(GOLANG_VERSION).linux-amd64.tar.gz

$(HOME)/go/go$(GOLANG_VERSION)/bin/go: $(HOME)/go/go$(GOLANG_VERSION).linux-amd64.tar.gz
	@tar -C $(HOME)/go -zxf $(HOME)/go/go$(GOLANG_VERSION).linux-amd64.tar.gz
	@mv $(HOME)/go/go $(HOME)/go/go$(GOLANG_VERSION)
	@touch $(HOME)/go/go$(GOLANG_VERSION)/bin/go

install_go: $(HOME)/go/go$(GOLANG_VERSION)/bin/go

.PHONY: all lint build clean save image push install_go
