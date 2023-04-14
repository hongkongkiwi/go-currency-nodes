# Can set these as ENV variables
PROTOC_BIN ?= protoc
GO_BIN ?= go
BUILD_DIR ?= build
# Hardcoded for now
GO_PKG := github.com/hongkongkiwi/go-currency-nodes
PROTO_INPUT_DIR := proto
PROTO_OUTPUT_DIR := pb

all: gen build

check_protoc:
ifeq (, $(shell which $(PROTOC_BIN)))
 	$(error "No protoc binary found in PATH")
endif

check_go:
ifeq (, $(shell which $(GO_BIN)))
 	$(error "No go binary found in PATH")
endif

gen: proto_gen

proto_gen: check_protoc
	@mkdir -p $(PROTO_OUTPUT_DIR)
	@rm -Rf $(PROTO_OUTPUT_DIR)/*
	protoc --proto_path=./$(PROTO_INPUT_DIR) $(PROTO_INPUT_DIR)/*.proto --go_out=:$(PROTO_OUTPUT_DIR) --go-grpc_out=:$(PROTO_OUTPUT_DIR)

cli: build_cli

controller: build_controller

node: build_node

build: build_cli build_controller build_node

build_cli: check_go
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/cli $(GO_PKG)/cmd/cli

build_controller: check_go
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/controller $(GO_PKG)/cmd/controller

build_node: check_go
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/node $(GO_PKG)/cmd/node

clean:
	 rm -Rf $(PROTO_OUTPUT_DIR)/* $(BUILD_DIR)/*

.PHONY: all check_protoc check_go gen proto_gen build build_cli build_controller build_node clean cli node controller
