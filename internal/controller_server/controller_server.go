package internal

/**
 * Controller Server Contains the gRPC server for serving functions that the Nodes call
 */

import (
	"context"
	"log"
	"net"
	"runtime"
	"strings"
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
	pb.ControllerCommandsServer
}

func funcName() string {
	pc, _, _, _ := runtime.Caller(1)
	names := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	return names[len(names)-1]
}

// rpc ControllerVersion (google.protobuf.Empty) returns (ControllerVersionReply) {}
func (s *grpcControllerServer) ControllerVersion(ctx context.Context, _ *emptypb.Empty) (*pb.ControllerVersionReply, error) {
	return &pb.ControllerVersionReply{ControllerVersion: ControllerVersion}, nil
}

// rpc CurrencyPrice (CurrencyPriceReq) returns (CurrencyGetPriceReply) {}
func (s *grpcControllerServer) CurrencyPrice(ctx context.Context, in *pb.CurrencyPriceReq) (*pb.CurrencyPriceReply, error) {
	return &pb.CurrencyPriceReply{}, nil
}

// rpc CurrencyPriceUpdate (CurrencyPriceUpdateReq) returns (CurrencyPriceUpdateReply) {}
func (s *grpcControllerServer) CurrencyPriceUpdate(ctx context.Context, in *pb.CurrencyPriceUpdateReq) (*pb.CurrencyPriceUpdateReply, error) {
	return &pb.CurrencyPriceUpdateReply{}, nil
}

// rpc CurrencyPriceSubscribe (CurrencyPriceSubscribeReq) returns (CurrencyPriceSubscribeReply) {}
func (s *grpcControllerServer) CurrencyPriceSubscribe(ctx context.Context, in *pb.CurrencyPriceSubscribeReq) (*pb.CurrencyPriceSubscribeReply, error) {
	return &pb.CurrencyPriceSubscribeReply{}, nil
}

func Start(wg *sync.WaitGroup, listenAddr string) {
	defer wg.Done()
	// Create new store
	helpers.ControllerPriceStore = kvs.NewKeyValueStore(1)
	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Printf("Failed to listen controller server: %v", err)
		return
	}
	s := grpc.NewServer()
	pb.RegisterControllerCommandsServer(s, &grpcControllerServer{})
	log.Printf("Controller server listening at %v", lis.Addr())
	// Register reflection to help with debugging
	reflection.Register(s)
	if err := s.Serve(lis); err != nil {
		log.Printf("Failed to serve controller server: %v", err)
		return
	}
}

func Stop() {
	log.Println("controller server closed")
}
