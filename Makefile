#####################
# PROJECT VARIABLES #
#####################

# Adjust these as necessary for your project

# The name of the repository
REPO_NAME = databalancer-logan

# The name and path to the main of the binaries to build, separated by '::'
# You could have multiple binaries that you create here, for example:
# BINARIES = \
	client::cmd/client/main.go \
	server::cmd/server/main.go
BINARIES = databalancer::cmd/databalancer/main.go

# Where to place the output of the 'build' target to execute while developing
BUILD_DIR = build

# Where to install the binaries, so that you can easily run them outside of
# the project directory. NOTE: this directory must be in your $PATH to work.
INSTALL_DIR = ~/.local/bin

##############
# VERSIONING #
##############

# These vars are for setting versions in the build flags
# You probably don't need to touch these.
ifeq ($(shell uname), Darwin)
	# If on macOS, set the shell to bash explicitly
	SHELL := /bin/bash
	CURRENT_PLATFORM = darwin
else
	CURRENT_PLATFORM = linux
endif
GOPATH ?= $(HOME)/go
PATH := $(GOPATH)/bin:$(PATH)
VERSION = $(shell git describe --tags --always --dirty)
BRANCH = $(shell git rev-parse --abbrev-ref HEAD)
REVISION = $(shell git rev-parse HEAD)
REVSHORT = $(shell git rev-parse --short HEAD)
USER = $(shell whoami)
# To populate version metadata, we use unix tools to get certain data
GOVERSION = $(shell go version | awk '{print $$3}')
NOW	= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS = "\
	-X github.com/kolide/${REPO_NAME}/vendor/github.com/kolide/kit/version.appName=${APP_NAME} \
	-X github.com/kolide/${REPO_NAME}/vendor/github.com/kolide/kit/version.version=${VERSION} \
	-X github.com/kolide/${REPO_NAME}/vendor/github.com/kolide/kit/version.branch=${BRANCH} \
	-X github.com/kolide/${REPO_NAME}/vendor/github.com/kolide/kit/version.revision=${REVISION} \
	-X github.com/kolide/${REPO_NAME}/vendor/github.com/kolide/kit/version.buildDate=${NOW} \
	-X github.com/kolide/${REPO_NAME}/vendor/github.com/kolide/kit/version.buildUser=${USER} \
	-X github.com/kolide/${REPO_NAME}/vendor/github.com/kolide/kit/version.goVersion=${GOVERSION}"

###########
# TARGETS #
###########

.PHONY: build clean deps install run test xp

all: build

# build each binary in list of binaries set in the project variables
build: clean deps
	@for BINARY in $(BINARIES) ; do \
		name=$${BINARY%%::*}; \
		path=$${BINARY#*::}; \
		go build -i -o $(BUILD_DIR)/$$name -ldflags $(LDFLAGS) $$path && \
		echo "Built $$name binary at path: $(BUILD_DIR)/$$name"; \
	done

# clean up build artifacts
clean:
	@echo "Cleaning up old builds..."
	@rm -r $(BUILD_DIR) >/dev/null 2>&1 ||:

# install dep tool (if necessary) and dependencies
deps:
	@command -v dep >/dev/null 2>&1 || echo "Installing dep tool..." \
		&& go get -u github.com/golang/dep/cmd/dep
	@echo "Installing dependencies..."
	@dep ensure -vendor-only && echo "Dependencies installed successfully."

# install the binaries to make it easy to run them outside the project
install:
	@echo "Copying binaries to $(INSTALL_DIR)..."
	@for BINARY in $(BINARIES) ; do \
		name=$${BINARY%%::*}; \
		cp $(BUILD_DIR)/$$name $(INSTALL_DIR); \
	done

# run the tests with coverage and race detector
test:
	go test -cover -race -v ./...

# cross-compile each binary in the list of binaries set in the project variables
xp: clean deps
	@for BINARY in $(BINARIES) ; do \
		name=$${BINARY%%::*}; \
		path=$${BINARY#*::}; \
		echo "Building MacOS binary for $$name..."; \
		GOOS=darwin CGO_ENABLED=0 go build -i -o $(BUILD_DIR)/darwin/$$name -ldflags $(LDFLAGS) $$path; \
		echo "Building Linux binary for $$name..."; \
		GOOS=linux CGO_ENABLED=0 go build -i -o $(BUILD_DIR)/linux/$$name -ldflags $(LDFLAGS) $$path; \
	done