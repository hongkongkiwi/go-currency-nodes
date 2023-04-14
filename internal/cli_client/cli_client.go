package internal

/** This file contains helper functions for the CLI Clients **/

import (
	"context"
	"fmt"
	"log"
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

func ClientConnect(nodeAddress string) (pb.NodeControlCommandsClient, context.Context, context.CancelFunc, error) {
	// Set up a connection to the server.
	var err error
	clientConn, err = grpc.Dial(nodeAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, nil, err
	}
	c := pb.NewNodeControlCommandsClient(clientConn)
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
	log.Printf("gRPC: Request: NodeUUID")
	r, err := c.NodeUUID(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeUUID: %v", err)
	}
	log.Printf("gRPC: Reply: NodeUUID: %s", r.NodeUuid)
	return nil
}

// rpc NodeVersion (google.protobuf.Empty) returns (NodeVersionReply) {}
func ClientNodeAppVersion() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: NodeVersion")
	r, err := c.NodeVersion(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeVersion: %v", err)
	}
	log.Printf("gRPC: Reply: NodeVersion: %s", r.NodeVersion)
	return nil
}

// rpc NodeStatus (google.protobuf.Empty) returns (NodeStatusReply) {}
func ClientNodeStatus() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: NodeStatus")
	r, err := c.NodeStatus(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeStatus: %v", err)
	}
	log.Printf("gRPC: Reply: NodeStatus: %s", r)
	return nil
}

// rpc NodeUpdatesPause (google.protobuf.Empty) returns (google.protobuf.Empty) {}
func ClientNodeUpdatesPause() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: NodeUpdatesPause")
	// No interesting reply, ignore any data
	_, err := c.NodeUpdatesPause(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeUpdatesPause: %v", err)
	}
	log.Printf("gRPC: Reply: NodeUpdatesPause")
	return nil
}

// rpc NodeUpdatesResume (google.protobuf.Empty) returns (google.protobuf.Empty) {}
func ClientNodeUpdatesResume() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: NodeUpdatesResume")
	// No interesting reply, ignore any data
	_, err := c.NodeUpdatesResume(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeUpdatesResume: %v", err)
	}
	log.Printf("gRPC: Reply: NodeUpdatesResume")
	return nil
}

// rpc NodeCurrenciesForceUpdate (NodeCurrenciesForceUpdateReq) returns (google.protobuf.Empty) {}
func ClientNodeCurrenciesForceUpdate(currencyPairs []string) error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: NodeCurrenciesForceUpdate")
	// No interesting reply, ignore any data
	_, err := c.NodeCurrenciesForceUpdate(ctx, &pb.NodeCurrenciesForceUpdateReq{CurrencyPairs: currencyPairs})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeCurrenciesForceUpdate: %v", err)
	}
	log.Printf("gRPC: Reply: NodeCurrenciesForceUpdate")
	return nil
}

// rpc NodeCurrencies (google.protobuf.Empty) returns (NodeSubscriptionsReply) {}
func ClientNodeCurrencies() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: NodeCurrencies")
	r, err := c.NodeCurrencies(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeCurrencies: %v", err)
	}
	log.Printf("gRPC: Reply: NodeCurrencies %s", r)
	return nil
}

// rpc NodeCurrenciesRefreshCache (google.protobuf.Empty) returns (google.protobuf.Empty) {}
func ClientNodeCurrenciesRefreshCache() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: NodeCurrenciesRefreshCache")
	// No interesting reply, ignore any data
	_, err := c.NodeCurrenciesRefreshCache(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeCurrenciesRefreshCache: %v", err)
	}
	log.Printf("gRPC: Reply: NodeCurrenciesRefreshCache")
	return nil
}

// rpc NodeControllerConnect(google.protobuf.Empty) returns (google.protobuf.Empty) {}
func ClientNodeControllerConnect() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: NodeControllerConnect")
	// No interesting reply, ignore any data
	_, err := c.NodeControllerConnect(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeControllerConnect: %v", err)
	}
	log.Printf("gRPC: Reply: NodeControllerConnect")
	return nil
}

// rpc NodeControllerDisconnect(google.protobuf.Empty) returns (google.protobuf.Empty) {}
func ClientNodeControllerDisconnect() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: NodeControllerDisconnect")
	// No interesting reply, ignore any data
	_, err := c.NodeControllerDisconnect(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeControllerDisconnect: %v", err)
	}
	log.Printf("gRPC: Reply: NodeControllerDisconnect")
	return nil
}

// rpc NodeAppLog (google.protobuf.Empty) returns (NodeAppLogReply) {}
func ClientNodeAppLog() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: NodeAppLog")
	r, err := c.NodeAppLog(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeAppLog: %v", err)
	}
	log.Printf("gRPC: Reply: NodeAppLog\n%s", r.Log)
	return nil
}

// rpc NodeAppKill (google.protobuf.Empty) returns (google.protobuf.Empty) {}
func ClientNodeAppKill() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: NodeAppKill")
	// No interesting reply, ignore any data
	_, err := c.NodeAppKill(ctx, &emptypb.Empty{})
	// NOTE: nmeed to handle the fact we never recieve a reply here after asking app to quit
	if err != nil {
		return fmt.Errorf("could not call gRPC NodeAppKill: %v", err)
	}
	log.Printf("gRPC: Reply: NodeAppKill")
	return nil
}
