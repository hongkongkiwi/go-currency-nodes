/**
 * Cli Client contains the gRPC functions for controlling a Node
 **/

package internal

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	pb "github.com/hongkongkiwi/go-currency-nodes/pb" // imports as package "cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

var clientConn *grpc.ClientConn

// Remove Node Address to connect to
var NodeAddr string
var NodeRequestTimeout uint64

func funcName() string {
	pc, _, _, _ := runtime.Caller(1)
	return runtime.FuncForPC(pc).Name()
}

func ClientConnect(nodeAddress string) (pb.NodeCommandsClient, context.Context, context.CancelFunc, error) {
	// Set up a connection to the server.
	var err error
	clientConn, err = grpc.Dial(nodeAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, nil, err
	}
	c := pb.NewNodeCommandsClient(clientConn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(NodeRequestTimeout))
	return c, ctx, cancel, err
}

func ClientDisconnect() {
	clientConn.Close()
}

// rpc NodeUUID (google.protobuf.Empty) returns (NodeUUIDReply) {}
func ClientNodeUUID() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: %s", funcName())
	r, err := c.NodeUUID(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeUUID: %v", err)
	}
	log.Printf("gRPC: Reply: %s: %s", funcName(), r.NodeUuid)
	return nil
}

// rpc NodeVersion (google.protobuf.Empty) returns (NodeVersionReply) {}
func ClientNodeAppVersion() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: %s", funcName())
	r, err := c.NodeAppVersion(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeVersion: %v", err)
	}
	log.Printf("gRPC: Reply: %s: %s", funcName(), r.NodeVersion)
	return nil
}

// rpc NodeStatus (google.protobuf.Empty) returns (NodeStatusReply) {}
func ClientNodeStatus() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: %s", funcName())
	r, err := c.NodeStatus(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeStatus: %v", err)
	}
	log.Printf("gRPC: Reply: %s: %s", funcName(), r)
	return nil
}

// rpc NodeUpdatesPause (google.protobuf.Empty) returns (google.protobuf.Empty) {}
func ClientNodeUpdatesPause() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: %s", funcName())
	// No interesting reply, ignore any data
	_, err := c.NodeCurrenciesPriceEventsPause(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeUpdatesPause: %v", err)
	}
	log.Printf("gRPC: Reply: %s", funcName())
	return nil
}

// rpc NodeUpdatesResume (google.protobuf.Empty) returns (google.protobuf.Empty) {}
func ClientNodeUpdatesResume() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: %s", funcName())
	// No interesting reply, ignore any data
	_, err := c.NodeCurrenciesPriceEventsResume(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeUpdatesResume: %v", err)
	}
	log.Printf("gRPC: Reply: %s", funcName())
	return nil
}

// Get details of all currencies this Node knows about
// rpc NodeCurrencies (google.protobuf.Empty) returns (NodeCurrenciesReply) {}
func ClientNodeCurrencies() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: %s", funcName())
	r, err := c.NodeCurrencies(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeCurrencies: %v", err)
	}
	log.Printf("gRPC: Reply: %s: %s", funcName(), r)
	return nil
}

// rpc NodeControllerConnect(google.protobuf.Empty) returns (google.protobuf.Empty) {}
func ClientNodeControllerConnect() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: %s", funcName())
	// No interesting reply, ignore any data
	_, err := c.NodeControllerConnect(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeControllerConnect: %v", err)
	}
	log.Printf("gRPC: Reply: %s", funcName())
	return nil
}

// rpc NodeControllerDisconnect(google.protobuf.Empty) returns (google.protobuf.Empty) {}
func ClientNodeControllerDisconnect() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: %s", funcName())
	// No interesting reply, ignore any data
	_, err := c.NodeControllerDisconnect(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeControllerDisconnect: %v", err)
	}
	log.Printf("gRPC: Reply: %s", funcName())
	return nil
}

// rpc NodeAppKill (google.protobuf.Empty) returns (google.protobuf.Empty) {}
func ClientNodeAppKill() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: %s", funcName())
	// No interesting reply, ignore any data
	_, err := c.NodeAppKill(ctx, &emptypb.Empty{})
	// NOTE: nmeed to handle the fact we never recieve a reply here after asking app to quit
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeAppKill: %v", err)
	}
	log.Printf("gRPC: Reply: %s", funcName())
	return nil
}
