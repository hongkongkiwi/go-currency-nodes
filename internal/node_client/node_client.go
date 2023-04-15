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

	helpers "github.com/hongkongkiwi/go-currency-nodes/internal/helpers"
	nodePriceGen "github.com/hongkongkiwi/go-currency-nodes/internal/node_price_gen"
	pb "github.com/hongkongkiwi/go-currency-nodes/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var AutoReconnect bool = true
var conn *grpc.ClientConn
var client pb.ControllerCommandsClient
var stopPriceUpdatesChan chan bool
var ControllerAddr string

const debugName = "NodeClient"

// Remove Controller Address to connect to
var ControllerRequestTimeout uint64

func funcName() string {
	pc, _, _, _ := runtime.Caller(1)
	names := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	return names[len(names)-1]
}

func ClientConnectivity() connectivity.State {
	return conn.GetState()
}

func newClientConnection() (*pb.ControllerCommandsClient, error) {
	// Set up a connection to the server.
	var err error
	if conn == nil {
		conn, err = grpc.Dial(ControllerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}
	}
	client = pb.NewControllerCommandsClient(conn)
	return &client, nil
}

func StartPriceUpdates(wg *sync.WaitGroup, updatesChan <-chan map[string]*nodePriceGen.PriceCurrency, stopChan chan bool) error {
	defer wg.Done()
	stopPriceUpdatesChan = stopChan
	log.Println("Starting Receiving Price Updates")
	_, err := newClientConnection()
	if err != nil {
		return err
	}
	// Wait for Currency updates
	for {
		select {
		case stop := <-stopChan:
			if stop {
				return nil
			}
		case priceUpdate := <-updatesChan:
			// Grab all keys of our map (e.g. the currencyPairs)
			currencyPairs := make([]string, len(priceUpdate))
			i := 0
			for k := range priceUpdate {
				currencyPairs[i] = k
				i++
			}
			// Send all these pairs as a price update
			ClientControllerCurrencyPriceUpdate(currencyPairs)
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
	_, err := newClientConnection()
	if err != nil {
		return err
	}
	log.Printf("%s->gRPC->Request: %s", debugName, funcName())
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ControllerRequestTimeout))
	defer cancel()
	r, err := client.ControllerVersion(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("%s->gRPC->Error: %s: %v", debugName, funcName(), err)
	}
	log.Printf("%s->gRPC->Reply: %s: %s", debugName, funcName(), r.ControllerVersion)
	return nil
}

// Manually get the price of one or more currencies this is basically polling
// rpc CurrencyPrice (CurrencyPriceReq) returns (CurrencyPriceReply) {}
func ClientControllerCurrencyPrice() error {
	_, err := newClientConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ControllerRequestTimeout))
	defer cancel()
	if len(helpers.NodeCfg.CurrencyPairs) == 0 {
		return nil
	}
	log.Printf("%s->gRPC->Request: %s (%s)", debugName, funcName(), helpers.NodeCfg.CurrencyPairs)
	r, err := client.CurrencyPrice(ctx, &pb.CurrencyPriceReq{CurrencyPairs: helpers.NodeCfg.CurrencyPairs})
	if err != nil {
		return fmt.Errorf("%s->gRPC->Error: %s: %v", debugName, funcName(), err)
	}
	log.Printf("%s->gRPC->Reply: %s: %s", debugName, funcName(), r.CurrencyItems)
	// Update our price store with the returned prices
	for _, currencyItem := range r.CurrencyItems {
		helpers.NodePriceStore.Set(currencyItem.CurrencyPair, &helpers.CurrencyStoreItem{Price: currencyItem.Price, ValidAt: currencyItem.PriceValidAt.AsTime()})
	}
	return nil
}

// Update the price of one or more currenices(normally sent 1 at a time but can be batched)
// rpc CurrencyPriceUpdate (CurrencyPriceUpdateReq) returns (CurrencyPriceUpdateReply) {}
func ClientControllerCurrencyPriceUpdateAll() error {
	_, err := newClientConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ControllerRequestTimeout))
	defer cancel()

	currencyPairs, _ := helpers.NodePriceStore.Keys()
	if len(currencyPairs) == 0 {
		return nil
	}
	log.Printf("%s->gRPC->Request: %s (%s)", debugName, funcName(), currencyPairs)
	var currencyItems []*pb.CurrencyItem
	// Send all our local prices to controller
	for _, currencyPair := range currencyPairs {
		val, _ := helpers.NodePriceStore.Get(currencyPair)
		currencyStoreItem, _ := val.(*helpers.CurrencyStoreItem)
		if currencyStoreItem != nil {
			protoCurrencyItem := &pb.CurrencyItem{CurrencyPair: currencyPair, Price: currencyStoreItem.Price, PriceValidAt: timestamppb.New(currencyStoreItem.ValidAt)}
			currencyItems = append(currencyItems, protoCurrencyItem)
		}
	}
	if len(currencyItems) > 0 {
		r, err := client.CurrencyPriceUpdate(ctx, &pb.CurrencyPriceUpdateReq{CurrencyItems: currencyItems})
		if err != nil {
			return fmt.Errorf("%s->gRPC->Error: %s: %v", debugName, funcName(), err)
		}
		log.Printf("%s->gRPC->Reply: %s: %s", debugName, funcName(), r.CurrencyItems)
	}
	return nil
}

// Update the price of one or more currenices(normally sent 1 at a time but can be batched)
// rpc CurrencyPriceUpdate (CurrencyPriceUpdateReq) returns (CurrencyPriceUpdateReply) {}
func ClientControllerCurrencyPriceUpdate(currencyPairs []string) error {
	_, err := newClientConnection()
	if err != nil {
		return err
	}
	log.Printf("%s->gRPC->Request: %s (%s)", debugName, funcName(), currencyPairs)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ControllerRequestTimeout))
	defer cancel()

	var currencyItems []*pb.CurrencyItem
	// Send all specified currency pair prices to controller
	for _, currencyPair := range currencyPairs {
		val, _ := helpers.NodePriceStore.Get(currencyPair)
		currencyStoreItem, _ := val.(*helpers.CurrencyStoreItem)
		if currencyStoreItem != nil {
			protoCurrencyItem := &pb.CurrencyItem{CurrencyPair: currencyPair, Price: currencyStoreItem.Price, PriceValidAt: timestamppb.New(currencyStoreItem.ValidAt)}
			currencyItems = append(currencyItems, protoCurrencyItem)
		}
	}
	if len(currencyItems) > 0 {
		r, err := client.CurrencyPriceUpdate(ctx, &pb.CurrencyPriceUpdateReq{CurrencyItems: currencyItems})
		if err != nil {
			return fmt.Errorf("%s->gRPC->Error: %s: %v", debugName, funcName(), err)
		}
		log.Printf("%s->gRPC->Reply: %s: %s", debugName, funcName(), r.CurrencyItems)
	}
	return nil
}

// Subscribe to currency updates
// rpc CurrencyPriceSubscribe (CurrencyPriceSubscribeReq) returns (CurrencySubscribeReply) {}
func ClientControllerCurrencyPriceSubscribe() error {
	_, err := newClientConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(ControllerRequestTimeout))
	defer cancel()

	if len(helpers.NodeCfg.CurrencyPairs) == 0 {
		return nil
	}
	log.Printf("%s->gRPC->Request: %s (%s)", debugName, funcName(), helpers.NodeCfg.CurrencyPairs)
	r, err := client.CurrencyPriceSubscribe(ctx, &pb.CurrencyPriceSubscribeReq{CurrencyPairs: helpers.NodeCfg.CurrencyPairs})
	if err != nil {
		return fmt.Errorf("%s->gRPC->Error: %s: %v", debugName, funcName(), err)
	}
	log.Printf("%s->gRPC->Reply: %s: %s", debugName, funcName(), r.CurrencyPairs)
	return nil
}
