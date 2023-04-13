all: proto_gen build

gen: proto_gen

proto_gen:
	@mkdir -p pb
	protoc --proto_path=./proto proto/*.proto --go_out=:pb --go-grpc_out=:pb

cli: build_cli

controller: build_controller

node: build_node

build: build_cli build_controller build_node

build_cli:
	@mkdir -p build
	go build -o build/cli github.com/hongkongkiwi/go-currency-nodes/cmd/cli

build_controller:
	@mkdir -p build
	go build -o build/controller github.com/hongkongkiwi/go-currency-nodes/cmd/controller

build_node:
	@mkdir -p build
	go build -o build/node github.com/hongkongkiwi/go-currency-nodes/cmd/node

# server1:
# 	go run cmd/server/main.go -port 50051

clean:
	 rm -Rf pb/* build/*

.PHONY: all gen proto_gen build build_cli build_controller build_node clean cli node controller
