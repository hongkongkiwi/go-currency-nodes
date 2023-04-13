all: proto_gen build

proto_gen:
	@mkdir -p internal/pb
	protoc --proto_path=./proto proto/*.proto --go_out=:internal/pb --go-grpc_out=:internal/pb

cli:
	go run github.com/hongkongkiwi/go-currency-nodes/cmd/cli

controller:
	go run github.com/hongkongkiwi/go-currency-nodes/cmd/controller

node:
	go run github.com/hongkongkiwi/go-currency-nodes/cmd/node

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
	 rm -Rf internal/pb/* build/*

.PHONY: all proto_gen build build_cli build_controller build_node clean
