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
	pb "github.com/hongkongkiwi/go-currency-nodes/v1/gen/pb"
	helpers "github.com/hongkongkiwi/go-currency-nodes/v1/internal/helpers"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
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

const nodeConnCleanupInterval = 10 * time.Second

var nodeConnCleanupTicker *time.Ticker

type NodeConnection struct {
	NodeUUID uuid.UUID
	NodeAddr string
	Conn     *grpc.ClientConn
	StaleAt  time.Time
}

type grpcControllerServer struct {
	pb.ControllerCommandsServer
}

// Runs on a schedule and sends a keep alive to server with our address
func CleanupNodeConnectionTick() {
	nodeConnCleanupTicker = time.NewTicker(nodeConnCleanupInterval)
	defer nodeConnCleanupTicker.Stop()
	for {
		// Cleanup tick
		<-nodeConnCleanupTicker.C
		for uuidStr, nodeConnection := range NodeConnections {
			// Check if our connection is stale
			if time.Now().After(nodeConnection.StaleAt) {
				// if helpers.ControllerCfg.VerboseLog {
				log.Printf("Node %s has become stale removing it", uuidStr)
				if nodeConnection.Conn != nil {
					if nodeConnection.Conn.GetState() != connectivity.Shutdown {
						// Close connection first
						nodeConnection.Conn.Close()
					}
					// Remove from NodeConnections list
					delete(NodeConnections, uuidStr)
				}
			}
		}
	}
}

func funcName() string {
	pc, _, _, _ := runtime.Caller(1)
	names := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	return names[len(names)-1]
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
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
			// Or our Node Address has changed then close and delete old connection for this UUID
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
	if helpers.ControllerCfg.VerboseLog {
		log.Printf("%s->gRPC->Incoming: %s\n%s\n", debugName, funcName(), protojson.Format(in))
	}
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
	return &pb.ControllerVersionReply{ControllerVersion: ControllerVersion}, nil
}

// Just a keep alive command, no other use
// rpc NodeKeepAlive (NodeKeepAliveReq) returns (google.protobuf.Empty) {}
func (s *grpcControllerServer) NodeKeepAlive(ctx context.Context, in *pb.NodeKeepAliveReq) (*emptypb.Empty, error) {
	if helpers.ControllerCfg.VerboseLog {
		log.Printf("%s->gRPC->Incoming: %s\n%s\n", debugName, funcName(), protojson.Format(in))
	}
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
	log.Printf("Node %s is still alive", in.NodeUuid)
	return &emptypb.Empty{}, nil
}

// Return all currencies that this Node requests from our currency store
// rpc CurrencyPrice (CurrencyPriceReq) returns (CurrencyGetPriceReply) {}
func (s *grpcControllerServer) CurrencyPrice(ctx context.Context, in *pb.CurrencyPriceReq) (*pb.CurrencyPriceReply, error) {
	if helpers.ControllerCfg.VerboseLog {
		log.Printf("%s->gRPC->Incoming: %s\n%s\n", debugName, funcName(), protojson.Format(in))
	}
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
	var currencyItems []*pb.CurrencyItem
	for _, currencyPair := range in.CurrencyPairs {
		if item, _ := ControllerPriceStore.Get(currencyPair); item != nil {
			currencyItems = append(currencyItems, &pb.CurrencyItem{
				CurrencyPair: currencyPair,
				Price:        item.Price,
				PriceValidAt: timestamppb.New(item.ValidAt),
			})
		}
	}
	return &pb.CurrencyPriceReply{CurrencyItems: currencyItems}, nil
}

// Update our local currency store with some simple validation
// rpc CurrencyPriceUpdate (CurrencyPriceUpdateReq) returns (CurrencyPriceUpdateReply) {}
func (s *grpcControllerServer) CurrencyPriceUpdate(ctx context.Context, in *pb.CurrencyPriceUpdateReq) (*pb.CurrencyPriceUpdateReply, error) {
	if helpers.ControllerCfg.VerboseLog {
		log.Printf("%s->gRPC->Incoming: %s\n%s\n", debugName, funcName(), protojson.Format(in))
	}
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
	updatedCurrencyStoreItems := make(map[string]*helpers.CurrencyStoreItem, len(in.CurrencyItems))
	var replyCurrencyItems []*pb.CurrencyItem
	// Update local price store
	for _, currencyItem := range in.CurrencyItems {
		log.Printf("Node %s update for %s (%0.2f)", in.NodeUuid, currencyItem.CurrencyPair, currencyItem.Price)
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
					CurrencyPair: currencyItem.CurrencyPair,
					Price:        currencyStoreItem.Price,
					PriceValidAt: timestamppb.New(currencyStoreItem.ValidAt),
				})
			} else {
				log.Printf("Node %s update REJECTED_OLD_PRICE for %s (%0.2f)", in.NodeUuid, currencyItem.CurrencyPair, currencyItem.Price)
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
				CurrencyPair: currencyItem.CurrencyPair,
				Price:        currencyStoreItem.Price,
				PriceValidAt: timestamppb.New(currencyStoreItem.ValidAt),
			})
		}
	}
	// Push all our batched updates at once to the channel for our client
	if len(updatedCurrencyStoreItems) > 0 {
		// Without this inner loop something something weird happens in golang channel
		if priceUpdatesChan != nil {
			priceUpdatesChan <- updatedCurrencyStoreItems
		}
	}
	return &pb.CurrencyPriceUpdateReply{CurrencyItems: replyCurrencyItems}, nil
}

// Subscribe to receive unsolicited updates for some currencies
// rpc CurrencyPriceSubscribe (CurrencyPriceSubscribeReq) returns (CurrencyPriceSubscribeReply) {}
func (s *grpcControllerServer) CurrencyPriceSubscribe(ctx context.Context, in *pb.CurrencyPriceSubscribeReq) (*pb.CurrencyPriceSubscribeReply, error) {
	if helpers.ControllerCfg.VerboseLog {
		log.Printf("%s->gRPC->Incoming: %s\n%s\n", debugName, funcName(), protojson.Format(in))
	}
	// Update connection details based on this request
	updateNodeConnection(in.NodeUuid, in.NodeAddr, in.NodeStaleAt.AsTime())
	var replyCurrencyItems []*pb.CurrencyItem
	// Loop through ALL local currency pairs so we can see whether to subscribe or not
	for _, currencyPair := range helpers.ControllerCfg.CurrencyPairs {
		// Check if we want to subscribe to this currency pair
		if contains(in.CurrencyPairs, currencyPair) {
			subItem, _ := ControllerSubscriptionsStore.Get(currencyPair)
			if subItem != nil {
				subItem.SetUUID(in.NodeUuid)
			} else {
				subItem = helpers.NewSubscriptionStoreItemFromUUID(currencyPair, in.NodeUuid)
			}
			// Save our updated item back the store
			setErr := ControllerSubscriptionsStore.Set(currencyPair, subItem)
			if setErr != nil {
				log.Printf("Error setting in currency subscription store %v", setErr)
			}
			if currItem, _ := ControllerPriceStore.Get(currencyPair); currItem != nil {
				replyCurrencyItems = append(replyCurrencyItems, &pb.CurrencyItem{
					CurrencyPair: currencyPair,
					Price:        currItem.Price,
					PriceValidAt: timestamppb.New(currItem.ValidAt),
				})
			}
			// Unsubscribe from this currency pair since it's not in our list
		} else {
			if subItem, _ := ControllerSubscriptionsStore.Get(currencyPair); subItem != nil {
				subItem.DeleteUUID(in.NodeUuid)
				ControllerSubscriptionsStore.Set(currencyPair, subItem)
			}
		}
	}
	log.Printf("Node %s subscribed to %v\n", in.NodeUuid, in.CurrencyPairs)
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
	go CleanupNodeConnectionTick()
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
