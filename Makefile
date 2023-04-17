# Can set these as ENV variables
PROTOC_BIN ?= protoc
GO_BIN ?= go
DOCKER_BIN := docker2
BUILD_DIR ?= build
# Hardcoded for now
DOCKER_TAG := github.com/hongkongkiwi/go-currency-nodes
GO_PKG := github.com/hongkongkiwi/go-currency-nodes
PROTO_INPUT_DIR := proto
PROTO_OUTPUT_DIR := gen/pb

all: gen build

.PHONE: check_deps
check_deps: check_go check_docker check_protoc
	@printf 'All dependencies checked successfully\n'

.PHONY: check_protoc
check_protoc:
	@command -v protoc >/dev/null 2>&1 || (echo >&2 "ERROR: protoc is required."; exit 1)

.PHONY: check_go
check_go:
	@command -v go >/dev/null 2>&1 || (echo >&2 "ERROR: go is required."; exit 1)

.PHONY: check_docker
check_docker:
	@command -v docker >/dev/null 2>&1 || (echo >&2 "ERROR: docker is required."; exit 1)

.PHONY: gen
gen: proto_gen

.PHONY: proto_gen # Generate all protobuf files
proto_gen: check_protoc
	@mkdir -p $(PROTO_OUTPUT_DIR)
	@rm -Rf $(PROTO_OUTPUT_DIR)/*
	protoc --proto_path=./$(PROTO_INPUT_DIR) $(PROTO_INPUT_DIR)/*.proto --go_out=:$(PROTO_OUTPUT_DIR) --go-grpc_out=:$(PROTO_OUTPUT_DIR)

.PHONY: cli
cli: build_cli

.PHONY: controller
controller: build_controller

.PHONY: node
node: build_node

.PHONY: build_docker # Build all docker images
build_docker: build_docker_cli build_docker_node build_docker_controller

.PHONY: run_docker_cli # Run docker iamge for cli app
run_docker_cli: check_docker
	docker run --rm -it github.com/hongkongkiwi/go-currency-nodes/cli

.PHONY: build_docker_cli # Build docker iamge for cli app
build_docker_cli: check_docker
	docker build -f docker/Dockerfile --build-arg APP_NAME=cli -t $(DOCKER_TAG)/cli .

.PHONY: run_docker_node # Run docker iamge for node app
run_docker_node: check_docker
	docker run --rm -it github.com/hongkongkiwi/go-currency-nodes/node

.PHONY: build_docker_node # Build docker iamge for node app
build_docker_node: check_docker
	docker build -f docker/Dockerfile --build-arg APP_NAME=node -t $(DOCKER_TAG)/node .

.PHONY: run_docker_controller # Run docker iamge for controller app
run_docker_controller: check_docker
	docker run --rm -it github.com/hongkongkiwi/go-currency-nodes/controller

.PHONY: build_docker_controller # Build docker iamge for controller app
build_docker_controller: check_docker
	docker build -f docker/Dockerfile --build-arg APP_NAME=controller -t $(DOCKER_TAG)/controller .

.PHONY: build # Build all apps
build: build_cli build_controller build_node

.PHONY: build_cli # Build cli app
build_cli: check_go proto_gen
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/cli $(GO_PKG)/cmd/cli

.PHONY: build_controller # Build controller app
build_controller: check_go proto_gen
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/controller $(GO_PKG)/cmd/controller

.PHONY: build_node # Build node app
build_node: check_go proto_gen
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/node $(GO_PKG)/cmd/node

.PHONY: clean # Clean all build files
clean:
	 rm -Rf $(PROTO_OUTPUT_DIR)/* $(BUILD_DIR)/*

.PHONY: list # List all targets
list:
	@sed -n 's/^\.PHONY: \([^\s]*\)\s*#\s*\(.*\)$$/\1/p' Makefile

.PHONY: help # Generate list of targets with descriptions                                                                
help:
	@printf 'Common make targets:\n'
	@sed -n 's/^\.PHONY: \([^\s]*\)\s*#\s*\(.*\)$$/ \1-\2/p' Makefile
