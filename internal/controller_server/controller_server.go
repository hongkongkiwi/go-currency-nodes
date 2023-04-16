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

	"github.com/bay0/kvs"
	"github.com/gofrs/uuid/v5"
	helpers "github.com/hongkongkiwi/go-currency-nodes/internal/helpers"
	pb "github.com/hongkongkiwi/go-currency-nodes/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const ControllerVersion = "0.0.3"
const debugName = "ControllerServer"

var priceUpdatesChan chan []*helpers.CurrencyStoreItem
var ControllerPriceStore *kvs.KeyValueStore
var ControllerSubscriptionsStore *kvs.KeyValueStore
var rpcServer *grpc.Server

var NodeSubscriptions map[string][]string // map[currency_pair]uuid
// Keeps track of all node connections
var NodeConnections map[uuid.UUID]*NodeConnection

type NodeConnection struct {
	UUID     uuid.UUID
	NodeAddr string
	Conn     *grpc.ClientConn
	StaleAt  time.Time
}

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
	uuid, uuidErr := uuid.FromString(nodeUuid)
	if uuidErr != nil {
		return fmt.Errorf("invalid node uuid received from client: %s", uuidErr)
	}
	// Grab our existing node connection (if any)
	existingConn := NodeConnections[uuid]
	if existingConn != nil {
		// If our connection is for some reason nil delete from array
		if existingConn.Conn == nil {
			delete(NodeConnections, uuid)
			existingConn = nil
			// Or our Node Address has changed then close and delete old connection
		} else if existingConn.NodeAddr != nodeAddr {
			NodeConnections[uuid].Conn.Close()
			delete(NodeConnections, uuid)
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
		NodeConnections[uuid] = &NodeConnection{
			UUID:     uuid,
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
	log.Printf("%s->gRPC->Incoming: %s", debugName, funcName())
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
	return &pb.ControllerVersionReply{ControllerVersion: ControllerVersion}, nil
}

// Just a keep alive command, no other use
// rpc NodeKeepAlive (NodeKeepAliveReq) returns (google.protobuf.Empty) {}
func (s *grpcControllerServer) ControllerNodeKeepAlive(ctx context.Context, in *pb.NodeKeepAliveReq) (*emptypb.Empty, error) {
	log.Printf("%s->gRPC->Incoming: %s", debugName, funcName())
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
	return &emptypb.Empty{}, nil
}

// Return all currencies that this Node requests from our currency store
// rpc CurrencyPrice (CurrencyPriceReq) returns (CurrencyGetPriceReply) {}
func (s *grpcControllerServer) CurrencyPrice(ctx context.Context, in *pb.CurrencyPriceReq) (*pb.CurrencyPriceReply, error) {
	log.Printf("%s->gRPC->Incoming: %s", debugName, funcName())
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
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
	log.Printf("%s->gRPC->Incoming: %s", debugName, funcName())
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
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
	log.Printf("%s->gRPC->Incoming: %s", debugName, funcName())
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
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

func StartServer(wg *sync.WaitGroup, listenAddr string, updatesChan chan []*helpers.CurrencyStoreItem) error {
	defer wg.Done()
	// Create new store to store prices on controller
	ControllerPriceStore = kvs.NewKeyValueStore(1)
	// Create a new store to store subscriptions
	ControllerSubscriptionsStore = kvs.NewKeyValueStore(1)
	priceUpdatesChan = updatesChan
	lis, listenErr := net.Listen("tcp", listenAddr)
	if listenErr != nil {
		return fmt.Errorf("failed to listen controller server: %v", listenErr)
	}
	rpcServer = grpc.NewServer()
	pb.RegisterControllerCommandsServer(rpcServer, &grpcControllerServer{})
	log.Printf("Controller server listening at %v", lis.Addr())
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
