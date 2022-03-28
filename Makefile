## Setup -- cross-OS

ME := $(shell id -un)
$(info ME=$(ME))
INTERACTIVE:=$(shell [ -t 0 ] && echo 1)
$(info INTERACTIVE=$(INTERACTIVE))
ifdef INTERACTIVE
# is a terminal
else
# cron job / other
endif

ifeq ($(OS),Windows_NT)
EXE := .exe
endif

ifeq ($(GOPATH),)
$(error GOPATH undefined)
endif

## Package -- 

ifeq ($(ME),vagrant)
# FIXME: any better way?
ROOTPKG := github.com/terra-farm/go-virtualbox
PKGS := $(filter-out /vendor%,$(shell cd $(GOPATH)/src/$(ROOTPKG) && go list ./...))
else
ROOTPKG := $(shell go list .)
PKGS := $(filter-out /vendor%,$(shell go list ./...))
endif
$(info PKGS=$(PKGS))

default: test lint build-pkgs

## Dependencies --

DEP_NAME := dep
DEP := $(GOPATH)/bin/$(DEP_NAME)$(EXE)

.PHONY: deps
deps: $(DEP)
	go get -t -d -v ./...
ifeq ($(ME),vagrant)
	cd $(GOPATH)/src/$(ROOTPKG) && $(DEP) ensure -v
else
	$(DEP) ensure -v
endif

$(DEP):
	go get -v github.com/golang/dep/cmd/dep

## Build, build tests & run them --

.PHONY: build test
build test:
	go $(@) -v ./...

## build-pkgs -- generate binaries

.PHONY: build-pkgs
build-pkgs: $(foreach pkg,$(PKGS),build-pkg-$(basename $(pkg)))

define build-pkg
build-pkg-$(basename $(1)):
	go build -v $(1)
endef

$(foreach pkg,$(PKGS),$(eval $(call build-pkg,$(pkg))))

# `go get` asks for credentials when needed
ifdef INTERACTIVE
GIT_TERMINAL_PROMPT := 1
export GIT_TERMINAL_PROMPT
endif

## Linting & scanning --

#GOMETALINTER_NAME := gometalinter.v2
GOMETALINTER_VERSION := v1.45.2
GOMETALINTER_NAME := golangci-lint
#GOMETALINTER := $(GOPATH)/bin/$(GOMETALINTER_NAME)$(EXE)

GOMETALINTER:
	build/install-golangci-lint.sh $(GOMETALINTER_VERSION)

.PHONY: lint
lint: GOMETALINTER
	$(GOPATH)/bin/golangci-lint run

## Release -- FIXME not yet ready

BINARY := mytool

VERSION ?= $(shell git describe --tags)

PLATFORMS := windows linux darwin

os = $(word 1, $@)

.PHONY: $(PLATFORMS)
$(PLATFORMS):
	GOOS=$(os) GOARCH=amd64 go build -o release/$(BINARY)-$(VERSION)-$(os)-amd64

.PHONY: release
release: $(PLATFORMS)