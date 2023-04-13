all: proto_gen build_all

proto_gen:
	@mkdir -p internal/pb
	protoc --proto_path=./proto proto/*.proto --go_out=:internal/pb --go-grpc_out=:internal/pb

build_all: build_cli build_controller build_node

build_cli:
	@mkdir -p build
	go build github.com/hongkongkiwi/go-currency-nodes/cmd/cli

build_controller:
	@mkdir -p build
	go build github.com/hongkongkiwi/go-currency-nodes/cmd/controller

build_node:
	@mkdir -p build
	go build github.com/hongkongkiwi/go-currency-nodes/cmd/node

# server1:
# 	go run cmd/server/main.go -port 50051

clean:
	 rm -Rf internal/pb/* build/*

.PHONY: all proto_gen build_all build_cli build_controller build_node clean
