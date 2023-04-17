/**
 * Node Client Contains the gRPC client for calling functions on Controllers
 **/

package internal

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"
	"time"

	pb "github.com/hongkongkiwi/go-currency-nodes/gen/pb"
	helpers "github.com/hongkongkiwi/go-currency-nodes/internal/helpers"
	nodePriceGen "github.com/hongkongkiwi/go-currency-nodes/internal/node_price_gen"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	PriceUpdatesReady = iota
	PriceUpdatesPause
)

var conn *grpc.ClientConn
var stopPriceUpdatesChan chan bool
var NodePriceStore *helpers.CurrencyStore
var NodePriceUpdatesState int = PriceUpdatesReady

const debugName = "NodeClient"

// Remove Controller Address to connect to
const RequestTimeout uint64 = uint64(1000 * time.Millisecond)

var keepAliveTicker *time.Ticker

// Runs on a schedule and sends a keep alive to server with our address
func KeepAliveTick(keepAliveStopChan chan bool) {
	keepAliveTicker = time.NewTicker(helpers.NodeCfg.KeepAliveInterval)
	defer keepAliveTicker.Stop()
	defer close(keepAliveStopChan)
	for {
		select {
		// Used to close the keep alive
		case <-keepAliveStopChan:
			keepAliveTicker = nil
			return
		// Keep alive tick
		case <-keepAliveTicker.C:
			// Only send a keep alive tick if we are not shutdown
			if connectState, _ := ClientConnectivity(); connectState != connectivity.Shutdown {
				err := ClientControllerKeepAlive()
				if err != nil {
					log.Printf("%v", err)
				}
			}
		}
	}
}

// Helper to reset keep alive
func ResetKeepAlive() error {
	if helpers.NodeCfg.VerboseLog {
		log.Println("Keep alive is reset")
	}
	if keepAliveTicker == nil {
		return fmt.Errorf("keep alive is already stopped")
	}
	keepAliveTicker.Reset(helpers.NodeCfg.KeepAliveInterval)
	return nil
}

// Permanently stop the keep alive or return an error
// func stopKeepAlive() error {
// 	if keepAliveTicker == nil {
//  	return fmt.Errorf("keep alive is already stopped")
// 	}
// 	keepAliveTicker.Stop()
// 	keepAliveStopChan <- true
// 	return nil
// }

func funcName() string {
	pc, _, _, _ := runtime.Caller(1)
	names := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	return names[len(names)-1]
}

// Decided not to use metadata keeping here as it's useful
// func appendAllReqMetadata(ctx context.Context) context.Context {
// 	// append UUID to metadata
// 	ctx = metadata.AppendToOutgoingContext(ctx, "node_uuid", helpers.NodeCfg.UUID.String())
// 	ctx = metadata.AppendToOutgoingContext(ctx, "node_addr", helpers.NodeCfg.ExternalAddr)
// 	return ctx
// }

func ClientConnectivity() (connectivity.State, error) {
	if conn == nil {
		return connectivity.Shutdown, fmt.Errorf("cannot get connectivity.State on nil connection")
	}
	return conn.GetState(), nil
}

func newConnection() error {
	// Set up a connection to the server.
	var err error
	if conn == nil {
		if conn, err = grpc.Dial(helpers.NodeCfg.ControllerAddr, grpc.WithTransportCredentials(insecure.NewCredentials())); err != nil {
			return err
		}
	}
	return nil
}

func StartPriceUpdates(wg *sync.WaitGroup, updatesChan <-chan map[string]*nodePriceGen.PriceCurrency, stopChan chan bool) error {
	defer wg.Done()
	stopPriceUpdatesChan = stopChan
	// Wait for Currency updates
	for {
		select {
		case stop := <-stopChan:
			if stop {
				return nil
			}
		case priceUpdate := <-updatesChan:
			// Keep track of all updated pair codes
			currencyPairs := make([]string, len(priceUpdate))
			i := 0
			// Update the local price store
			for pairKey, pairVal := range priceUpdate {
				currencyPairs[i] = pairKey
				NodePriceStore.Set(pairKey, &helpers.CurrencyStoreItem{
					Price:   pairVal.Price,
					ValidAt: pairVal.GeneratedAt,
				})
				i++
			}
			// Send all these pairs as a price update
			err := ClientControllerCurrencyPriceUpdate(currencyPairs)
			if err != nil {
				log.Printf("%v", err)
			}
			// for pair, priceCurrency := range priceUpdate {
			// 	log.Printf("Price %s Update: %.2f\n", pair, priceCurrency.Price)
			// }
		}
	}
}

func StopPriceUpdates() {
	log.Println("Stopped Receiving Price Updates")
	stopPriceUpdatesChan <- true
}

// Manually get the price of one or more currencies
// rpc ControllerVersion (google.protobuf.Empty) returns (ControllerVersionReply) {}
func ClientControllerVersion() error {
	connectErr := newConnection()
	if connectErr != nil {
		return connectErr
	}
	log.Printf("%s->gRPC->Request: %s", debugName, funcName())
	ResetKeepAlive()
	c := pb.NewControllerCommandsClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(RequestTimeout))
	defer cancel()
	r, err := c.ControllerVersion(ctx, &pb.ControllerVersionReq{
		NodeUuid:    helpers.NodeCfg.UUID.String(),
		NodeAddr:    helpers.NodeCfg.NodeListenAddr,
		NodeStaleAt: timestamppb.New(time.Now().Add(helpers.NodeCfg.KeepAliveInterval + 1)),
	})
	if err != nil {
		return fmt.Errorf("%s->gRPC->Error: %s: %v", debugName, funcName(), err)
	}
	log.Printf("%s->gRPC->Reply: %s: %s", debugName, funcName(), r.ControllerVersion)
	return nil
}

// Sent by node at a regular interval
// rpc NodeKeepAlive (NodeKeepAliveReq) returns (google.protobuf.Empty) {}
func ClientControllerKeepAlive() error {
	connectErr := newConnection()
	if connectErr != nil {
		return connectErr
	}
	log.Printf("%s->gRPC->Request: %s", debugName, funcName())
	ResetKeepAlive()
	c := pb.NewControllerCommandsClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(RequestTimeout))
	defer cancel()
	_, sendErr := c.NodeKeepAlive(ctx, &pb.NodeKeepAliveReq{
		NodeUuid:    helpers.NodeCfg.UUID.String(),
		NodeAddr:    helpers.NodeCfg.NodeListenAddr,
		NodeStaleAt: timestamppb.New(time.Now().Add(helpers.NodeCfg.KeepAliveInterval + 1)),
	})
	if sendErr != nil {
		return fmt.Errorf("%s->gRPC->Error: %s: %v", debugName, funcName(), sendErr)
	}
	log.Printf("%s->gRPC->Reply: %s", debugName, funcName())
	return nil
}

// Manually get the price of one or more currencies this is basically polling
// rpc CurrencyPrice (CurrencyPriceReq) returns (CurrencyPriceReply) {}
func ClientControllerCurrencyPrice() error {
	connectErr := newConnection()
	if connectErr != nil {
		return connectErr
	}
	if len(helpers.NodeCfg.CurrencyPairs) == 0 {
		return nil
	}
	log.Printf("%s->gRPC->Request: %s (%s)", debugName, funcName(), helpers.NodeCfg.CurrencyPairs)
	ResetKeepAlive()
	c := pb.NewControllerCommandsClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(RequestTimeout))
	defer cancel()
	r, sendErr := c.CurrencyPrice(ctx, &pb.CurrencyPriceReq{
		CurrencyPairs: helpers.NodeCfg.CurrencyPairs,
		NodeUuid:      helpers.NodeCfg.UUID.String(),
		NodeAddr:      helpers.NodeCfg.NodeListenAddr,
		NodeStaleAt:   timestamppb.New(time.Now().Add(helpers.NodeCfg.KeepAliveInterval + 1)),
	})
	if sendErr != nil {
		return fmt.Errorf("%s->gRPC->Error: %s: %v", debugName, funcName(), sendErr)
	}
	log.Printf("%s->gRPC->Reply: %s: %s", debugName, funcName(), r.CurrencyItems)
	// Update our price store with the returned prices
	for _, currencyItem := range r.CurrencyItems {
		NodePriceStore.Set(currencyItem.CurrencyPair, &helpers.CurrencyStoreItem{Price: currencyItem.Price, ValidAt: currencyItem.PriceValidAt.AsTime()})
	}
	return nil
}

// Update the price of one or more currenices(normally sent 1 at a time but can be batched)
// rpc CurrencyPriceUpdate (CurrencyPriceUpdateReq) returns (CurrencyPriceUpdateReply) {}
func ClientControllerCurrencyPriceUpdateAll() error {
	connectErr := newConnection()
	if connectErr != nil {
		return connectErr
	}
	currencyPairs, _ := NodePriceStore.Keys()
	if len(currencyPairs) == 0 {
		return nil
	}
	var currencyItems []*pb.CurrencyItem
	// Send all our local prices to controller
	for _, currencyPair := range currencyPairs {
		if currencyStoreItem, _ := NodePriceStore.Get(currencyPair); currencyStoreItem != nil {
			protoCurrencyItem := &pb.CurrencyItem{
				CurrencyPair: currencyPair,
				Price:        currencyStoreItem.Price,
				PriceValidAt: timestamppb.New(currencyStoreItem.ValidAt),
			}
			currencyItems = append(currencyItems, protoCurrencyItem)
		}
	}
	if len(currencyItems) > 0 {
		log.Printf("%s->gRPC->Request: %s (%s)", debugName, funcName(), currencyPairs)
		ResetKeepAlive()
		c := pb.NewControllerCommandsClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(RequestTimeout))
		defer cancel()
		r, sendErr := c.CurrencyPriceUpdate(ctx, &pb.CurrencyPriceUpdateReq{
			CurrencyItems: currencyItems,
			NodeUuid:      helpers.NodeCfg.UUID.String(),
			NodeAddr:      helpers.NodeCfg.NodeListenAddr,
			NodeStaleAt:   timestamppb.New(time.Now().Add(helpers.NodeCfg.KeepAliveInterval + 1)),
		})
		if sendErr != nil {
			return fmt.Errorf("%s->gRPC->Error: %s: %v", debugName, funcName(), sendErr)
		}
		log.Printf("%s->gRPC->Reply: %s: %s", debugName, funcName(), r.CurrencyItems)
	}
	return nil
}

// Update the price of one or more currenices(normally sent 1 at a time but can be batched)
// rpc CurrencyPriceUpdate (CurrencyPriceUpdateReq) returns (CurrencyPriceUpdateReply) {}
func ClientControllerCurrencyPriceUpdate(currencyPairs []string) error {
	connectErr := newConnection()
	if connectErr != nil {
		return connectErr
	}
	var currencyItems []*pb.CurrencyItem
	// Send all specified currency pair prices to controller
	for _, currencyPair := range currencyPairs {
		if currencyStoreItem, _ := NodePriceStore.Get(currencyPair); currencyStoreItem != nil {
			protoCurrencyItem := &pb.CurrencyItem{
				CurrencyPair: currencyPair,
				Price:        currencyStoreItem.Price,
				PriceValidAt: timestamppb.New(currencyStoreItem.ValidAt),
			}
			currencyItems = append(currencyItems, protoCurrencyItem)
		}
	}
	if len(currencyItems) > 0 {
		log.Printf("%s->gRPC->Request: %s (%s)", debugName, funcName(), currencyPairs)
		ResetKeepAlive()
		c := pb.NewControllerCommandsClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(RequestTimeout))
		defer cancel()
		r, sendErr := c.CurrencyPriceUpdate(ctx, &pb.CurrencyPriceUpdateReq{
			CurrencyItems: currencyItems,
			NodeUuid:      helpers.NodeCfg.UUID.String(),
			NodeAddr:      helpers.NodeCfg.NodeListenAddr,
			NodeStaleAt:   timestamppb.New(time.Now().Add(helpers.NodeCfg.KeepAliveInterval + 1)),
		})
		if sendErr != nil {
			return fmt.Errorf("%s->gRPC->Error: %s: %v", debugName, funcName(), sendErr)
		}
		log.Printf("%s->gRPC->Reply: %s: %s", debugName, funcName(), r.CurrencyItems)
	}
	return nil
}

// Subscribe to currency updates
// rpc CurrencyPriceSubscribe (CurrencyPriceSubscribeReq) returns (CurrencySubscribeReply) {}
func ClientControllerCurrencyPriceSubscribe() error {
	connectErr := newConnection()
	if connectErr != nil {
		return connectErr
	}

	if len(helpers.NodeCfg.CurrencyPairs) == 0 {
		return nil
	}
	log.Printf("%s->gRPC->Request: %s (%s)", debugName, funcName(), helpers.NodeCfg.CurrencyPairs)
	ResetKeepAlive()

	c := pb.NewControllerCommandsClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(RequestTimeout))
	defer cancel()
	r, sendErr := c.CurrencyPriceSubscribe(ctx, &pb.CurrencyPriceSubscribeReq{
		CurrencyPairs: helpers.NodeCfg.CurrencyPairs,
		NodeUuid:      helpers.NodeCfg.UUID.String(),
		NodeAddr:      helpers.NodeCfg.NodeListenAddr,
		NodeStaleAt:   timestamppb.New(time.Now().Add(helpers.NodeCfg.KeepAliveInterval + 1)),
	})
	if sendErr != nil {
		return fmt.Errorf("%s->gRPC->Error: %s: %v", debugName, funcName(), sendErr)
	}
	log.Printf("%s->gRPC->Reply: %s\n%s\n", debugName, funcName(), protojson.Format(r))
	return nil
}
