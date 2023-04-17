package internal

/**
 * Controller Server Contains the gRPC server for serving functions that the Nodes call
 */

import (
	"context"
	"fmt"
	"log"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gofrs/uuid/v5"
	helpers "github.com/hongkongkiwi/go-currency-nodes/internal/helpers"
	pb "github.com/hongkongkiwi/go-currency-nodes/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/protobuf/encoding/protojson"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ControllerVersion = "0.0.3"
const debugName = "ControllerServer"

var priceUpdatesChan chan map[string]*helpers.CurrencyStoreItem
var ControllerPriceStore *helpers.CurrencyStore
var ControllerSubscriptionsStore *helpers.SubscriptionStore
var rpcServer *grpc.Server

// Keeps track of all node connections
var NodeConnections map[string]*NodeConnection

type NodeConnection struct {
	NodeUUID uuid.UUID
	NodeAddr string
	Conn     *grpc.ClientConn
	StaleAt  time.Time
}

type grpcControllerServer struct {
	pb.ControllerCommandsServer
}

func funcName() string {
	pc, _, _, _ := runtime.Caller(1)
	names := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	return names[len(names)-1]
}

/*
 * This method is key for keeping track of our node connections
 * and is called everytime a node makes an rpc call to us.
 * Our client then uses this list of connections to dial back our
 * nodes with price updates.
 */
func updateNodeConnection(nodeUuid string, nodeAddr string, nodeStaleAt time.Time) error {
	// Grab our existing node connection (if any)
	existingConn := NodeConnections[nodeUuid]
	if existingConn != nil {
		// If our connection is for some reason nil delete from array
		if existingConn.Conn == nil {
			delete(NodeConnections, nodeUuid)
			existingConn = nil
			// Or our Node Address has changed then close and delete old connection
		} else if existingConn.NodeAddr != nodeAddr {
			NodeConnections[nodeUuid].Conn.Close()
			delete(NodeConnections, nodeUuid)
			existingConn = nil
		} else {
			// All in order just update our new stale at value
			existingConn.StaleAt = nodeStaleAt
		}
	}

	// If we don't have a connection entry then create one with our node details
	if existingConn == nil {
		conn, connErr := grpc.Dial(nodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if connErr != nil {
			return fmt.Errorf("cannot establish connection to node: %v", connErr)
		}
		NodeConnections[nodeUuid] = &NodeConnection{
			NodeUUID: uuid.FromStringOrNil(nodeUuid),
			NodeAddr: nodeAddr,
			Conn:     conn,
			StaleAt:  nodeStaleAt,
		}
	}
	return nil
}

// Return our controller version
// rpc ControllerVersion (google.protobuf.Empty) returns (ControllerVersionReply) {}
func (s *grpcControllerServer) ControllerVersion(ctx context.Context, in *pb.ControllerVersionReq) (*pb.ControllerVersionReply, error) {
	log.Printf("%s->gRPC->Incoming: %s\n%s\n", debugName, funcName(), protojson.Format(in))
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
	return &pb.ControllerVersionReply{ControllerVersion: ControllerVersion}, nil
}

// Just a keep alive command, no other use
// rpc NodeKeepAlive (NodeKeepAliveReq) returns (google.protobuf.Empty) {}
func (s *grpcControllerServer) NodeKeepAlive(ctx context.Context, in *pb.NodeKeepAliveReq) (*emptypb.Empty, error) {
	log.Printf("%s->gRPC->Incoming: %s\n%s\n", debugName, funcName(), protojson.Format(in))
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
	return &emptypb.Empty{}, nil
}

// Return all currencies that this Node requests from our currency store
// rpc CurrencyPrice (CurrencyPriceReq) returns (CurrencyGetPriceReply) {}
func (s *grpcControllerServer) CurrencyPrice(ctx context.Context, in *pb.CurrencyPriceReq) (*pb.CurrencyPriceReply, error) {
	log.Printf("%s->gRPC->Incoming: %s\n%s\n", debugName, funcName(), protojson.Format(in))
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
	var currencyItems []*pb.CurrencyItem
	for _, currencyPair := range in.CurrencyPairs {
		if item, _ := ControllerPriceStore.Get(currencyPair); item != nil {
			currencyItems = append(currencyItems, &pb.CurrencyItem{Price: 123})
		}
	}
	return &pb.CurrencyPriceReply{CurrencyItems: currencyItems}, nil
}

// Update our local currency store with some simple validation
// rpc CurrencyPriceUpdate (CurrencyPriceUpdateReq) returns (CurrencyPriceUpdateReply) {}
func (s *grpcControllerServer) CurrencyPriceUpdate(ctx context.Context, in *pb.CurrencyPriceUpdateReq) (*pb.CurrencyPriceUpdateReply, error) {
	log.Printf("%s->gRPC->Incoming: %s\n%s\n", debugName, funcName(), protojson.Format(in))
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
	updatedCurrencyStoreItems := make(map[string]*helpers.CurrencyStoreItem, len(in.CurrencyItems))
	var replyCurrencyItems []*pb.CurrencyItem
	// Update local price store
	for _, currencyItem := range in.CurrencyItems {
		// Get our currenct price updated time from price store
		if currPrice, _ := ControllerPriceStore.Get(currencyItem.CurrencyPair); currPrice != nil {
			// Do some simple validation to check the new time after after old time and different price
			if currencyItem.PriceValidAt.AsTime().After(currPrice.ValidAt) && currPrice.Price != currencyItem.Price {
				currencyStoreItem := &helpers.CurrencyStoreItem{
					Price:         currencyItem.Price,
					ValidAt:       currencyItem.PriceValidAt.AsTime(),
					UpdatedByUUID: uuid.FromStringOrNil(in.NodeUuid),
				}
				ControllerPriceStore.Set(currencyItem.CurrencyPair, currencyStoreItem)
				updatedCurrencyStoreItems[currencyItem.CurrencyPair] = currencyStoreItem
				replyCurrencyItems = append(replyCurrencyItems, &pb.CurrencyItem{
					Price:        currencyStoreItem.Price,
					PriceValidAt: timestamppb.New(currencyStoreItem.ValidAt),
				})
			} else {
				log.Printf("skipped update because it is earlier than our latest update")
			}
			// No existing just create a new one
		} else {
			currencyStoreItem := &helpers.CurrencyStoreItem{
				Price:         currencyItem.Price,
				ValidAt:       currencyItem.PriceValidAt.AsTime(),
				UpdatedByUUID: uuid.FromStringOrNil(in.NodeUuid),
			}
			ControllerPriceStore.Set(currencyItem.CurrencyPair, currencyStoreItem)
			updatedCurrencyStoreItems[currencyItem.CurrencyPair] = currencyStoreItem
			replyCurrencyItems = append(replyCurrencyItems, &pb.CurrencyItem{
				Price:        currencyStoreItem.Price,
				PriceValidAt: timestamppb.New(currencyStoreItem.ValidAt),
			})
		}
	}
	// Push all our batched updates at once to the channel for our client
	if len(updatedCurrencyStoreItems) > 0 {
		// Without this inner loop something something weird happens in golang channel
		if priceUpdatesChan != nil {
			//fmt.Printf("test 123: %v", len(updatedCurrencyStoreItems))
			priceUpdatesChan <- updatedCurrencyStoreItems
		}
	}
	return &pb.CurrencyPriceUpdateReply{CurrencyItems: replyCurrencyItems}, nil
}

// Subscribe to receive unsolicited updates for some currencies
// rpc CurrencyPriceSubscribe (CurrencyPriceSubscribeReq) returns (CurrencyPriceSubscribeReply) {}
func (s *grpcControllerServer) CurrencyPriceSubscribe(ctx context.Context, in *pb.CurrencyPriceSubscribeReq) (*pb.CurrencyPriceSubscribeReply, error) {
	log.Printf("%s->gRPC->Incoming: %s\n%s\n", debugName, funcName(), protojson.Format(in))
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
	var replyCurrencyItems []*pb.CurrencyItem
	// Loop through our currency pairs for subscriptions
	for _, currencyPair := range in.CurrencyPairs {
		// Get all subscriptions for this currency pair
		subItem, _ := ControllerSubscriptionsStore.Get(currencyPair)
		if subItem == nil {
			subItem = helpers.NewSubscriptionStoreItem([]string{in.NodeUuid})
		} else {
			subItem.UUIDs = append(subItem.UUIDs, in.NodeUuid)
		}
		ControllerSubscriptionsStore.Set(currencyPair, subItem)
		if currItem, _ := ControllerPriceStore.Get(currencyPair); currItem != nil {
			replyCurrencyItems = append(replyCurrencyItems, &pb.CurrencyItem{
				Price:        currItem.Price,
				PriceValidAt: timestamppb.New(currItem.ValidAt),
			})
		}
	}
	return &pb.CurrencyPriceSubscribeReply{CurrencyItems: replyCurrencyItems}, nil
}

func StartServer(wg *sync.WaitGroup, priceChan chan map[string]*helpers.CurrencyStoreItem) error {
	defer wg.Done()
	priceUpdatesChan = priceChan
	NodeConnections = make(map[string]*NodeConnection)
	var openErr error
	// Create new store to store prices on controller
	ControllerPriceStore, openErr = helpers.NewDiskCurrencyStore(helpers.ControllerCfg.DiskKVDir, "controller_price_store")
	if openErr != nil {
		return openErr
	}
	// Create a new store to store subscriptions
	ControllerSubscriptionsStore, openErr = helpers.NewDiskSubscriptionStore(helpers.ControllerCfg.DiskKVDir, "controller_subscription_store")
	if openErr != nil {
		return openErr
	}
	lis, listenErr := net.Listen("tcp", helpers.ControllerCfg.ControllerListenAddr)
	if listenErr != nil {
		return fmt.Errorf("failed to listen controller server: %v", listenErr)
	}
	rpcServer = grpc.NewServer()
	pb.RegisterControllerCommandsServer(rpcServer, &grpcControllerServer{})
	log.Printf("Controller server listening at %v\n", lis.Addr())
	// Register reflection to help with debugging
	reflection.Register(rpcServer)
	if serveErr := rpcServer.Serve(lis); serveErr != nil {
		return fmt.Errorf("failed to serve controller server: %v", serveErr)
	}
	// Since serve is blocking we shouldn't get here
	return nil
}

func StopServer() {
	log.Println("controller server closed")
}
