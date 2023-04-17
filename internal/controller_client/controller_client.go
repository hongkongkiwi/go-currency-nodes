/**
 * Controller Client calls rpc CurrencyPriceUpdatedEvent on subscribed nodes
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

	"github.com/gofrs/uuid/v5"
	cs "github.com/hongkongkiwi/go-currency-nodes/internal/controller_server"
	helpers "github.com/hongkongkiwi/go-currency-nodes/internal/helpers"
	pb "github.com/hongkongkiwi/go-currency-nodes/pb"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const debugName = "ControllerClient"

var RequestTimeout uint64 = uint64(250 * time.Millisecond)

func funcName() string {
	pc, _, _, _ := runtime.Caller(1)
	names := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	return names[len(names)-1]
}

func StartMonitoringForPriceUpdates(wg *sync.WaitGroup, priceChan chan map[string]*helpers.CurrencyStoreItem, stopChan chan bool) {
	for {
		select {
		case priceUpdates := <-priceChan:
			ClientCurrencyPriceUpdatedEvent(priceUpdates)
		case <-stopChan:
			wg.Done()
		}
	}
}

// Controller calls this on node when a new price comes in
// rpc CurrencyPriceUpdatedEvent (CurrencyPriceUpdateEventReq) returns (google.protobuf.Empty) {}
func ClientCurrencyPriceUpdatedEvent(updatedCurrencyStoreItems map[string]*helpers.CurrencyStoreItem) error {
	fmt.Printf("ClientCurrencyPriceUpdatedEvent\n")
	// Keep a temporary list of currencies & connections to send out
	currenciesToSend := make(map[*cs.NodeConnection][]*pb.CurrencyItem)
	// Loop through all currency store item updates
	for currencyPair, storeItem := range updatedCurrencyStoreItems {
		fmt.Printf("ClientCurrencyPriceUpdatedEvent Currencies %s\n", currencyPair)
		// Check all node uuids we should notify for this currency pair
		subsForCurrency, _ := cs.ControllerSubscriptionsStore.Get(currencyPair)
		if subsForCurrency == nil {
			fmt.Printf("No subscriptions for currency %s", subsForCurrency)
		}
		if subsForCurrency != nil {
			fmt.Printf("ClientCurrencyPriceUpdatedEvent Subscriptions %s\n", subsForCurrency.UUIDs)
			for i := 0; i < len(subsForCurrency.UUIDs); i++ {
				uuidStr := subsForCurrency.UUIDs[i]
				uuid, uuidErr := uuid.FromString(uuidStr)
				if uuidErr != nil {
					log.Printf("invalid node uuid sent: %s", uuidErr)
					continue
				}
				// Don't resend this update to the same uuid
				if storeItem.UpdatedByUUID == uuid {
					fmt.Printf("skip UUID because it sent this update %s\n", uuid)
					continue
				}
				nodeConnection := cs.NodeConnections[uuidStr]
				if nodeConnection != nil {
					fmt.Printf("ClientCurrencyPriceUpdatedEvent Node Connection %s %s\n", nodeConnection.NodeUUID, nodeConnection.NodeAddr)
					// Check if our connection is stale
					if nodeConnection.StaleAt.Before(time.Now()) {
						currenciesToSend[nodeConnection] = append(currenciesToSend[nodeConnection], &pb.CurrencyItem{
							Price:        storeItem.Price,
							PriceValidAt: timestamppb.New(storeItem.ValidAt),
						})
					} else {
						// Close connection first
						nodeConnection.Conn.Close()
						// Connection is stale remove it from collection
						delete(cs.NodeConnections, uuidStr)
					}
				}
			}
		}
	}
	// Loop through each connection and send out relevant currency items
	for nodeConnection, currencyItems := range currenciesToSend {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(RequestTimeout))
		defer cancel()
		reply := &pb.CurrencyPriceUpdateEventReq{CurrencyItems: currencyItems}
		log.Printf("%s->gRPC->Outgoing: %s\n%s\n", debugName, funcName(), protojson.Format(reply))
		c := pb.NewNodeEventsClient(nodeConnection.Conn)
		_, sendErr := c.CurrencyPriceUpdatedEvent(ctx, reply)
		if sendErr != nil {
			log.Printf("%s->gRPC->ERROR Outgoing: %s\n%v\n", debugName, funcName(), sendErr)
		} else {
			// Send currency updates to this UUID
			fmt.Printf("Sent update to to [%s]: %s\n", &nodeConnection.NodeUUID, currencyItems)
		}
	}
	return nil
}
