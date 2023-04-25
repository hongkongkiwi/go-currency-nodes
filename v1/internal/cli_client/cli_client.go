/**
 * Cli Client contains the gRPC functions for controlling a Node
 **/

package internal

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strings"
	"time"

	pb "github.com/hongkongkiwi/go-currency-nodes/v1/gen/pb" // imports as package "cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

var clientConn *grpc.ClientConn

// Remove Node Address to connect to
var NodeAddr string
var NodeRequestTimeout time.Duration
var VerboseLog bool = false
var QuietLog bool = false

func funcName() string {
	pc, _, _, _ := runtime.Caller(1)
	names := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	return names[len(names)-1]
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
	defer cancel()
	if connectErr != nil {
		return connectErr
	}
	if VerboseLog {
		log.Printf("gRPC: Request: %s\n", funcName())
	}
	r, err := c.NodeUUID(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC %s: %v", funcName(), err)
	}
	if VerboseLog {
		log.Printf("gRPC: Reply: %s: %s", funcName(), r.NodeUuid)
	} else {
		fmt.Printf("%s\n", r.NodeUuid)
	}
	return nil
}

// rpc NodeVersion (google.protobuf.Empty) returns (NodeVersionReply) {}
func ClientNodeAppVersion() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	defer cancel()
	if connectErr != nil {
		return connectErr
	}
	if VerboseLog {
		log.Printf("gRPC: Request: %s\n", funcName())
	}
	r, err := c.NodeAppVersion(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC %s: %v", funcName(), err)
	}
	if VerboseLog {
		log.Printf("gRPC: Reply: %s: %s\n", funcName(), r.NodeVersion)
	} else {
		fmt.Printf("%s\n", r.NodeVersion)
	}
	return nil
}

func ClientNodeManualPriceUpdate(currencyPair string, price float64) error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	if VerboseLog {
		log.Printf("gRPC: Request: %s\n", funcName())
	}
	_, err := c.NodeManualPriceUpdate(ctx, &pb.NodeManualPriceUpdateReq{
		CurrencyPair: currencyPair,
		Price:        price,
	})
	if err != nil {
		return fmt.Errorf("could not call gRPC %s: %v", funcName(), err)
	}
	if VerboseLog {
		log.Printf("gRPC: Reply: %s\n", funcName())
	} else {
		if !QuietLog {
			fmt.Printf("Successfully Sent\n")
		}
	}
	return nil
}

// rpc NodeStatus (google.protobuf.Empty) returns (NodeStatusReply) {}
func ClientNodeStatus() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	if VerboseLog {
		log.Printf("gRPC: Request: %s\n", funcName())
	}
	r, err := c.NodeStatus(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC %s: %v", funcName(), err)
	}
	if VerboseLog {
		log.Printf("gRPC: Reply: %s: %s\n", funcName(), r)
	} else {
		fmt.Printf("%v\n", protojson.Format(r))
	}
	return nil
}

// rpc NodeUpdatesPause (google.protobuf.Empty) returns (google.protobuf.Empty) {}
func ClientNodeUpdatesPause() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	if VerboseLog {
		log.Printf("gRPC: Request: %s\n", funcName())
	}
	// No interesting reply, ignore any data
	_, err := c.NodeCurrenciesPriceEventsPause(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC %s: %v", funcName(), err)
	}
	if VerboseLog {
		log.Printf("gRPC: Reply: %s\n", funcName())
	} else {
		if !QuietLog {
			fmt.Printf("Successfully Sent\n")
		}
	}
	return nil
}

// rpc NodeUpdatesResume (google.protobuf.Empty) returns (google.protobuf.Empty) {}
func ClientNodeUpdatesResume() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	if VerboseLog {
		log.Printf("gRPC: Request: %s\n", funcName())
	}
	// No interesting reply, ignore any data
	_, err := c.NodeCurrenciesPriceEventsResume(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC %s: %v", funcName(), err)
	}
	if VerboseLog {
		log.Printf("gRPC: Reply: %s\n", funcName())
	} else {
		if !QuietLog {
			fmt.Printf("Successfully Sent\n")
		}
	}
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
	if VerboseLog {
		log.Printf("gRPC: Request: %s\n", funcName())
	}
	r, err := c.NodeCurrencies(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call gRPC %s: %v", funcName(), err)
	}
	if VerboseLog {
		log.Printf("gRPC: Reply: %s: %s\n", funcName(), r)
	} else {
		fmt.Printf("%v\n", protojson.Format(r))
	}
	return nil
}

// rpc NodeAppKill (google.protobuf.Empty) returns (google.protobuf.Empty) {}
func ClientNodeAppKill() error {
	c, ctx, cancel, connectErr := ClientConnect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	defer cancel()
	log.Printf("gRPC: Request: %s\n", funcName())
	// No interesting reply, ignore any data
	_, err := c.NodeAppKill(ctx, &emptypb.Empty{})
	// NOTE: nmeed to handle the fact we never recieve a reply here after asking app to quit
	if err != nil {
		return fmt.Errorf("could not call gRPC %s: %v", funcName(), err)
	}
	if VerboseLog {
		log.Printf("gRPC: Reply: %s\n", funcName())
	} else {
		if !QuietLog {
			fmt.Printf("Successfully Sent\n")
		}
	}
	return nil
}
