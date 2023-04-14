package internal

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/bay0/kvs"
	helpers "github.com/hongkongkiwi/go-currency-nodes/internal/helpers"
	pb "github.com/hongkongkiwi/go-currency-nodes/pb"
	"github.com/tebeka/atexit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

const NodeVersion = "0.0.2"

type grpcControlServer struct {
	pb.UnimplementedNodeControlCommandsServer
}

type grpcPriceEventServer struct {
	pb.UnimplementedNodePriceEventsServer
}

func (s *grpcControlServer) NodeUUID(ctx context.Context, _ *emptypb.Empty) (*pb.NodeUUIDReply, error) {
	log.Printf("gRPC: NodeUUID")
	return &pb.NodeUUIDReply{NodeUuid: helpers.NodeCfg.UUID.String()}, nil
}

func (s *grpcControlServer) NodeVersion(ctx context.Context, _ *emptypb.Empty) (*pb.NodeVersionReply, error) {
	log.Printf("gRPC: NodeVersion")
	return &pb.NodeVersionReply{NodeVersion: NodeVersion}, nil
}

func (s *grpcControlServer) NodeStatus(ctx context.Context, _ *emptypb.Empty) (*pb.NodeStatusReply, error) {
	log.Printf("gRPC: NodeStatus")
	currencyPairs := helpers.NodeCfg.CurrencyPairs
	currencyItems := make([]*pb.CurrencyItem, len(currencyPairs))
	for i, key := range currencyPairs {
		val, _ := helpers.NodePriceStore.Get(key)
		if val != nil {
			priceStore, _ := val.(*helpers.CurrencyStoreItem)
			currencyItems[i] = &pb.CurrencyItem{
				CurrencyPair: key,
				Price:        priceStore.Price,
				PriceValidAt: priceStore.ValidAt,
			}
		} else {
			currencyItems[i] = &pb.CurrencyItem{
				CurrencyPair: key,
			}
		}
	}

	return &pb.NodeStatusReply{
		NodeUuid:               helpers.NodeCfg.UUID.String(),
		NodeName:               helpers.NodeCfg.Name,
		NodeVersion:            NodeVersion,
		NodeUpdatesStreamState: pb.NodeStatusReply_NodeStreamState(helpers.NodePriceUpdatesState),
		ControllerServer:       helpers.NodeCfg.ControllerAddr,
		CurrencyItems:          currencyItems,
		// ConnectionState: ,
		// CurrencyItems: ,
	}, nil
}

// Pause updates of currencies to controller
func (s *grpcControlServer) NodeUpdatesPause(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	log.Printf("gRPC: NodeUpdatesPause")
	helpers.NodePriceUpdatesState = helpers.PriceUpdatesPause
	return &emptypb.Empty{}, nil
}

// Resume updates of currencies to controller
func (s *grpcControlServer) NodeUpdatesResume(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	log.Printf("gRPC: NodeUpdatesResume")
	helpers.NodePriceUpdatesState = helpers.PriceUpdatesReady
	return &emptypb.Empty{}, nil
}

// Forces an update of the local currency value to the server
func (s *grpcControlServer) NodeCurrenciesForceUpdate(ctx context.Context, in *pb.NodeCurrenciesForceUpdateReq) (*emptypb.Empty, error) {
	log.Printf("gRPC: NodeCurrenciesForceUpdate %s", in)
	// Check if connected

	// Update local currency

	// nodeClient.ControllerCurrenciesForceUpdate()
	return &emptypb.Empty{}, nil
}

// Get a list of all currencies defined in our config and their latest knwn prices
func (s *grpcControlServer) NodeCurrencies(ctx context.Context, _ *emptypb.Empty) (*pb.NodeSubscriptionsReply, error) {
	log.Printf("gRPC: NodeCurrencies")
	currency_pairs, _ := helpers.NodePriceStore.Keys()
	return &pb.NodeSubscriptionsReply{CurrencyPairs: currency_pairs}, nil
}

// Force a refresh of our local currency values
func (s *grpcControlServer) NodeCurrenciesRefreshCache(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	log.Printf("gRPC: NodeCurrenciesRefreshCache")
	return &emptypb.Empty{}, nil
}

// Ask the node to connect manually
func (s *grpcControlServer) NodeControllerConnect(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	log.Printf("gRPC: NodeControllerConnect")
	// Check if disconnect, then connect
	// nodeClient.Connect()
	return &emptypb.Empty{}, nil
}

// Ask the node to disconnect manually (will not reconnect until commanded or restarted)
func (s *grpcControlServer) NodeControllerDisconnect(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	log.Printf("gRPC: NodeControllerDisconnect")
	// Check if connected, then disconnect
	// nodeClient.Disconnect()
	return &emptypb.Empty{}, nil
}

// Grab our app log
func (s *grpcControlServer) NodeAppLog(ctx context.Context, _ *emptypb.Empty) (*pb.NodeAppLogReply, error) {
	log.Printf("gRPC: NodeAppLog")
	return nil, nil
}

// Kill the app
func (s *grpcControlServer) NodeAppKill(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	log.Printf("gRPC: NodeAppKill")
	atexit.Exit(0)
	// Sadly we never get here since we quit
	return &emptypb.Empty{}, nil
}

// Controller calls this on node when a new price comes in
func (s *grpcPriceEventServer) CurrencyPriceEvent(ctx context.Context, in *pb.CurrencyPriceEventReq) (*emptypb.Empty, error) {
	// Store the updated prices into our local store
	for _, c := range in.CurrencyItems {
		log.Printf("gRPC: CurrencyPriceEvent: %s", c)
		helpers.NodePriceStore.Set(c.CurrencyPair, &helpers.CurrencyStoreItem{
			Price:   c.Price,
			ValidAt: c.PriceValidAt,
		})
	}
	return &emptypb.Empty{}, nil
}

// Controller calls this when subscriptions change
func (s *grpcPriceEventServer) CurrencySubscriptionsEvent(ctx context.Context, in *pb.CurrencySubscriptionsEventReq) (*emptypb.Empty, error) {
	// Store the updated prices into our local store (sent as part of the subscription update)
	for _, c := range in.CurrencyItems {
		log.Printf("gRPC: CurrencySubscriptionsEvent: %s", c)
		helpers.NodePriceStore.Set(c.CurrencyPair, &helpers.CurrencyStoreItem{
			Price:   c.Price,
			ValidAt: c.PriceValidAt,
		})
	}
	return &emptypb.Empty{}, nil
}

func Start(wg *sync.WaitGroup, listenAddr string) {
	defer wg.Done()
	// Create new store
	helpers.NodePriceStore = kvs.NewKeyValueStore(1)
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Printf("failed to listen: %v", err)
		return
	}
	s := grpc.NewServer()
	pb.RegisterNodeControlCommandsServer(s, &grpcControlServer{})
	pb.RegisterNodePriceEventsServer(s, &grpcPriceEventServer{})
	log.Printf("server listening at %v", lis.Addr())
	// Register reflection to help with debugging
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Printf("failed to serve: %v", err)
		return
	}
}

func Stop() {
	log.Println("server closed")
}
