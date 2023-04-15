package internal

/**
 * Controller Server Contains the gRPC server for serving functions that the Nodes call
 */

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
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ControllerVersion = "0.0.3"

var priceUpdatesChan chan []*helpers.CurrencyStoreItem
var ControllerPriceStore *kvs.KeyValueStore
var ControllerSubscriptionsStore *kvs.KeyValueStore

type grpcControllerServer struct {
	pb.ControllerCommandsServer
}

type subscriptions struct {
	uuids []string
}

// This is implemented for the KV store
func (p *subscriptions) Clone() kvs.Value {
	return &subscriptions{uuids: p.uuids}
}

// func funcName() string {
// 	pc, _, _, _ := runtime.Caller(1)
// 	names := strings.Split(runtime.FuncForPC(pc).Name(), ".")
// 	return names[len(names)-1]
// }

// rpc ControllerVersion (google.protobuf.Empty) returns (ControllerVersionReply) {}
func (s *grpcControllerServer) ControllerVersion(ctx context.Context, _ *emptypb.Empty) (*pb.ControllerVersionReply, error) {
	return &pb.ControllerVersionReply{ControllerVersion: ControllerVersion}, nil
}

// Return all currencies that this Node requests from our currency store
// rpc CurrencyPrice (CurrencyPriceReq) returns (CurrencyGetPriceReply) {}
func (s *grpcControllerServer) CurrencyPrice(ctx context.Context, in *pb.CurrencyPriceReq) (*pb.CurrencyPriceReply, error) {
	var currencyItems []*pb.CurrencyItem
	for _, currencyPair := range in.CurrencyPairs {
		val, _ := ControllerPriceStore.Get(currencyPair)
		if val != nil {
			currencyItems = append(currencyItems, &pb.CurrencyItem{Price: 123})
		}
	}
	return &pb.CurrencyPriceReply{CurrencyItems: currencyItems}, nil
}

// Update our local currency store with some simple validation
// rpc CurrencyPriceUpdate (CurrencyPriceUpdateReq) returns (CurrencyPriceUpdateReply) {}
func (s *grpcControllerServer) CurrencyPriceUpdate(ctx context.Context, in *pb.CurrencyPriceUpdateReq) (*pb.CurrencyPriceUpdateReply, error) {
	var updatedCurrencyStoreItems []*helpers.CurrencyStoreItem
	// Update local price store
	for _, currencyItem := range in.CurrencyItems {
		// Get our currenct price updated time from price store
		currVal, _ := ControllerPriceStore.Get(currencyItem.CurrencyPair)
		if currVal != nil {
			curr, _ := currVal.(*helpers.CurrencyStoreItem)
			// Do some simple validation to check the new time after after old time and different price
			if curr.ValidAt.Before(currencyItem.PriceValidAt.AsTime()) && curr.Price != currencyItem.Price {
				currencyStoreItem := &helpers.CurrencyStoreItem{Price: currencyItem.Price, ValidAt: currencyItem.PriceValidAt.AsTime()}
				err := ControllerPriceStore.Set(currencyItem.CurrencyPair, currencyStoreItem)
				if err != nil {
					log.Printf("Error updating price in CurrencyPriceUpdate: %s\n", err)
				} else {
					updatedCurrencyStoreItems = append(updatedCurrencyStoreItems, currencyStoreItem)
				}
			}
		}
	}
	// Push all our batched updates at once to the channel for our client
	if len(updatedCurrencyStoreItems) > 0 {
		priceUpdatesChan <- updatedCurrencyStoreItems
	}
	// Build our resopnse to send all current prices that client asked for
	var replyCurrencyItems []*pb.CurrencyItem
	for _, currencyItem := range in.CurrencyItems {
		// Get our currenct price updated time from price store
		currVal, _ := ControllerPriceStore.Get(currencyItem.CurrencyPair)
		if currVal != nil {
			curr, _ := currVal.(*helpers.CurrencyStoreItem)
			replyCurrencyItems = append(replyCurrencyItems, &pb.CurrencyItem{Price: curr.Price, PriceValidAt: timestamppb.New(curr.ValidAt)})
		}
	}
	return &pb.CurrencyPriceUpdateReply{CurrencyItems: replyCurrencyItems}, nil
}

// Subscribe to receive unsolicited updates for some currencies
// rpc CurrencyPriceSubscribe (CurrencyPriceSubscribeReq) returns (CurrencyPriceSubscribeReply) {}
func (s *grpcControllerServer) CurrencyPriceSubscribe(ctx context.Context, in *pb.CurrencyPriceSubscribeReq) (*pb.CurrencyPriceSubscribeReply, error) {
	for _, currencyPair := range in.CurrencyPairs {
		val, _ := ControllerSubscriptionsStore.Get(currencyPair)
		subscriptions, ok := val.(*subscriptions)
		if !ok {
			subscriptions.uuids = append(subscriptions.uuids, in.NodeUuid)
		}
		ControllerSubscriptionsStore.Set(currencyPair, subscriptions)
	}
	return &pb.CurrencyPriceSubscribeReply{}, nil
}

func StartServer(wg *sync.WaitGroup, listenAddr string, updatesChan chan []*helpers.CurrencyStoreItem) {
	defer wg.Done()
	// Create new store to store prices on controller
	ControllerPriceStore = kvs.NewKeyValueStore(1)
	// Create a new store to store subscriptions
	ControllerSubscriptionsStore = kvs.NewKeyValueStore(1)
	priceUpdatesChan = updatesChan
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

func StopServer() {
	log.Println("controller server closed")
}
