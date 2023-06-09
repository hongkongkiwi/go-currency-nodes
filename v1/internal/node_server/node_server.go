/**
 * Node Server Contains the gRPC server for functions that the Controller calls on the Node
 **/

package internal

import (
	"context"
	"fmt"
	"log"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

	pb "github.com/hongkongkiwi/go-currency-nodes/v1/gen/pb"
	helpers "github.com/hongkongkiwi/go-currency-nodes/v1/internal/helpers"
	nodeClient "github.com/hongkongkiwi/go-currency-nodes/v1/internal/node_client"
	nodePriceGen "github.com/hongkongkiwi/go-currency-nodes/v1/internal/node_price_gen"
	"github.com/tebeka/atexit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const NodeVersion = "0.0.2"

var priceUpdatesChan chan map[string]*nodePriceGen.PriceCurrency

func funcName() string {
	pc, _, _, _ := runtime.Caller(1)
	names := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	return names[len(names)-1]
}

// Events called by the control cli
type grpcNodeCommandsServer struct {
	pb.NodeCommandsServer
}

// Events called by the controller
type grpcNodeEventsServer struct {
	pb.NodeEventsServer
}

// Get uuid of this node
// rpc NodeUUID (google.protobuf.Empty) returns (NodeUUIDReply) {}
func (s *grpcNodeCommandsServer) NodeUUID(ctx context.Context, _ *emptypb.Empty) (*pb.NodeUUIDReply, error) {
	log.Printf("gRPC: %s", funcName())
	return &pb.NodeUUIDReply{NodeUuid: helpers.NodeCfg.UUID.String()}, nil
}

// Get app version from this node
// rpc NodeAppVersion (google.protobuf.Empty) returns (NodeVersionReply) {}
func (s *grpcNodeCommandsServer) NodeAppVersion(ctx context.Context, _ *emptypb.Empty) (*pb.NodeAppVersionReply, error) {
	log.Printf("gRPC: %s", funcName())
	return &pb.NodeAppVersionReply{NodeVersion: NodeVersion}, nil
}

// Send a manual price update to this node
// rpc NodeManualPriceUpdate (NodeManualPriceUpdateReq) returns (google.protobuf.Empty) {}
func (s *grpcNodeCommandsServer) NodeManualPriceUpdate(ctx context.Context, in *pb.NodeManualPriceUpdateReq) (*emptypb.Empty, error) {
	log.Printf("gRPC: %s", funcName())
	updatedPrices := make(map[string]*nodePriceGen.PriceCurrency)
	updatedPrices[in.CurrencyPair] = &nodePriceGen.PriceCurrency{
		Price:       in.Price,
		GeneratedAt: time.Now(),
	}
	priceUpdatesChan <- updatedPrices
	return &emptypb.Empty{}, nil
}

// Get this nodes status
// rpc NodeStatus (google.protobuf.Empty) returns (NodeStatusReply) {}
func (s *grpcNodeCommandsServer) NodeStatus(ctx context.Context, _ *emptypb.Empty) (*pb.NodeStatusReply, error) {
	log.Printf("gRPC: %s", funcName())
	currencyPairs := helpers.NodeCfg.CurrencyPairs
	currencyItems := make([]*pb.CurrencyItem, len(currencyPairs))
	for i, key := range currencyPairs {
		if priceItem, _ := nodeClient.NodePriceStore.Get(key); priceItem != nil {
			currencyItems[i] = &pb.CurrencyItem{
				CurrencyPair: key,
				Price:        priceItem.Price,
				PriceValidAt: timestamppb.New(priceItem.ValidAt),
			}
		} else {
			currencyItems[i] = &pb.CurrencyItem{
				CurrencyPair: key,
			}
		}
	}

	return &pb.NodeStatusReply{
		NodeUuid:          helpers.NodeCfg.UUID.String(),
		NodeName:          helpers.NodeCfg.Name,
		NodeVersion:       NodeVersion,
		NodeUpdatesPaused: nodePriceGen.UpdatesPaused,
		ControllerServer:  helpers.NodeCfg.ControllerAddr,
		ListenAddress:     helpers.NodeCfg.NodeListenAddr,
		AdvertiseAddress:  helpers.NodeCfg.NodeAdvertiseAddr,
		CurrencyItems:     currencyItems,
	}, nil
}

// Stops this node from sending currency price updates (updates are still received and stored locally)
// rpc NodeCurrenciesPriceEventsPause (google.protobuf.Empty) returns (google.protobuf.Empty) {}
func (s *grpcNodeCommandsServer) NodeCurrenciesPriceEventsPause(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	log.Printf("gRPC: %s", funcName())
	nodePriceGen.PauseUpdates()
	return &emptypb.Empty{}, nil
}

// Resume sending currency price updates to controller
// rpc NodeCurrenciesPriceEventsResume (google.protobuf.Empty) returns (google.protobuf.Empty) {}
func (s *grpcNodeCommandsServer) NodeCurrenciesPriceEventsResume(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if helpers.NodeCfg.VerboseLog {
		log.Printf("gRPC: %s", funcName())
	}
	nodePriceGen.ResumeUpdates()
	return &emptypb.Empty{}, nil
}

// Request a list of all local currencies this node knows about
// rpc NodeCurrencies (google.protobuf.Empty) returns (NodeSubscriptionsReply) {}
func (s *grpcNodeCommandsServer) NodeCurrencies(ctx context.Context, _ *emptypb.Empty) (*pb.NodeCurrenciesReply, error) {
	if helpers.NodeCfg.VerboseLog {
		log.Printf("gRPC: %s", funcName())
	}
	currencyPairs, _ := nodeClient.NodePriceStore.Keys()
	fmt.Printf("log: %v", currencyPairs)
	var replyCurrencyItems []*pb.CurrencyItem
	for _, currencyPair := range currencyPairs {
		if localItem, _ := nodeClient.NodePriceStore.Get(currencyPair); localItem != nil {
			replyCurrencyItems = append(replyCurrencyItems, &pb.CurrencyItem{
				CurrencyPair: currencyPair,
				Price:        localItem.Price,
				PriceValidAt: timestamppb.New(localItem.ValidAt),
			})
		}
	}
	return &pb.NodeCurrenciesReply{CurrencyItems: replyCurrencyItems}, nil
}

// Kill the Node
// rpc NodeAppKill (google.protobuf.Empty) returns (google.protobuf.Empty) {}
func (s *grpcNodeCommandsServer) NodeAppKill(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if helpers.NodeCfg.VerboseLog {
		log.Printf("gRPC: %s", funcName())
	}
	atexit.Exit(0)
	// Sadly we never get here since we quit
	return &emptypb.Empty{}, nil
}

// Called when a new price comes in from controller - all prices are cached locally
// rpc CurrencyPriceUpdatedEvent (CurrencyPriceUpdateEventReq) returns (google.protobuf.Empty) {}
func (s *grpcNodeEventsServer) CurrencyPriceUpdatedEvent(ctx context.Context, in *pb.CurrencyPriceUpdateEventReq) (*emptypb.Empty, error) {
	if helpers.NodeCfg.VerboseLog {
		log.Printf("gRPC: %s %v", funcName(), in.CurrencyItems)
	}
	// Store the updated prices into our local store
	for _, c := range in.CurrencyItems {
		if helpers.NodeCfg.VerboseLog {
			log.Printf("gRPC: CurrencyPriceEvent: %s", c)
		}
		nodeClient.NodePriceStore.Set(c.CurrencyPair, &helpers.CurrencyStoreItem{
			Price:   c.Price,
			ValidAt: c.PriceValidAt.AsTime(),
		})
	}

	// Loop so we can format for logging
	logOutputPrices := make([][2]string, len(in.CurrencyItems))
	for i, curr := range in.CurrencyItems {
		logOutputPrices[i] = [2]string{curr.CurrencyPair, fmt.Sprintf("%0.2f", curr.Price)}
	}
	log.Printf("Received price updates from controller %v\n", logOutputPrices)

	return &emptypb.Empty{}, nil
}

// Start our gRPC server
func StartServer(wg *sync.WaitGroup, updatesChan chan map[string]*nodePriceGen.PriceCurrency) {
	defer wg.Done()
	priceUpdatesChan = updatesChan
	lis, err := net.Listen("tcp", helpers.NodeCfg.NodeListenAddr)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return
	}
	s := grpc.NewServer()
	pb.RegisterNodeCommandsServer(s, &grpcNodeCommandsServer{})
	pb.RegisterNodeEventsServer(s, &grpcNodeEventsServer{})
	log.Printf("node server listening at %v", lis.Addr())
	// Register reflection to help with debugging via CLI
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Printf("failed to serve: %v", err)
		return
	}
}

func StopServer() {
	log.Println("node server closed")
}
