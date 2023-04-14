package internal

import (
	"context"
	"log"
	"net"
	"sync"

	"github.com/bay0/kvs"
	helpers "github.com/hongkongkiwi/go-currency-nodes/internal/helpers"
	pb "github.com/hongkongkiwi/go-currency-nodes/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/types/known/emptypb"
)

const ControllerVersion = "0.0.3"

type grpcControllerServer struct {
	pb.UnimplementedControllerPriceCommandsServer
}

// rpc ControllerVersion (google.protobuf.Empty) returns (ControllerVersionReply) {}
func (s *grpcControllerServer) ControllerVersion(ctx context.Context, _ *emptypb.Empty) (*pb.ControllerVersionReply, error) {
	return &pb.ControllerVersionReply{ControllerVersion: ControllerVersion}, nil
}

// rpc CurrencyPrice (CurrencyPriceReq) returns (CurrencyGetPriceReply) {}
func (s *grpcControllerServer) CurrencyPrice(ctx context.Context, in *pb.CurrencyPriceReq) (*pb.CurrencyGetPriceReply, error) {
	return &pb.CurrencyGetPriceReply{}, nil
}

// rpc CurrencyUpdatePrice (CurrencyUpdatePriceReq) returns (CurrencyUpdatePriceReply) {}
func (s *grpcControllerServer) CurrencyUpdatePrice(ctx context.Context, in *pb.CurrencyUpdatePriceReq) (*pb.CurrencyUpdatePriceReply, error) {
	return &pb.CurrencyUpdatePriceReply{}, nil
}

// rpc CurrencySubscribe (CurrencySubscribeReq) returns (CurrencySubscribeReply) {}
func (s *grpcControllerServer) CurrencySubscribe(ctx context.Context, in *pb.CurrencySubscribeReq) (*pb.CurrencySubscribeReply, error) {
	return &pb.CurrencySubscribeReply{}, nil
}

func Start(wg *sync.WaitGroup, listenAddr string) {
	defer wg.Done()
	// Create new store
	helpers.ControllerPriceStore = kvs.NewKeyValueStore(1)
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Printf("failed to listen controller server: %v", err)
		return
	}
	s := grpc.NewServer()
	pb.RegisterControllerPriceCommandsServer(s, &grpcControllerServer{})
	log.Printf("controller server listening at %v", lis.Addr())
	// Register reflection to help with debugging
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Printf("failed to serve controller server: %v", err)
		return
	}
}

func Stop() {
	log.Println("controller server closed")
}
