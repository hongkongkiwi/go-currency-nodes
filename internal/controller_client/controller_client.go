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
	pb "github.com/hongkongkiwi/go-currency-nodes/gen/pb"
	cs "github.com/hongkongkiwi/go-currency-nodes/internal/controller_server"
	helpers "github.com/hongkongkiwi/go-currency-nodes/internal/helpers"
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
func ClientCurrencyPriceUpdatedEvent(updatedCurrencyItems map[string]*helpers.CurrencyStoreItem) error {
	fmt.Printf("ClientCurrencyPriceUpdatedEvent\n")

	// Keep a temporary list of uuids and associated currency items
	sendToUUIDs := make(map[string][]*pb.CurrencyItem)

	// Loop through all currency store item updates
	for currencyPair, storeItem := range updatedCurrencyItems {
		fmt.Printf("ClientCurrencyPriceUpdatedEvent Currencies %s\n", currencyPair)
		// Check all node uuids we should notify for this currency pair
		subs, _ := cs.ControllerSubscriptionsStore.Get(currencyPair)
		if subs == nil {
			fmt.Printf("No subscriptions for currency %s", subs.CurrencyPair)
			continue
		}
		if subs != nil {
			fmt.Printf("ClientCurrencyPriceUpdatedEvent Subscriptions %s\n", subs.UUIDs())
			for _, uuidStr := range subs.UUIDs() {
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
				fmt.Printf("we will send update to this UUID %s\n", uuid)
				sendToUUIDs[uuidStr] = append(sendToUUIDs[uuidStr], &pb.CurrencyItem{
					Price:        storeItem.Price,
					PriceValidAt: timestamppb.New(storeItem.ValidAt),
				})
			}
		}
	}

	// Chceck that we have a connection for each uuid we wamt tp semd tp
	for uuidStr, currencyItems := range sendToUUIDs {
		nodeConnection := cs.NodeConnections[uuidStr]
		if nodeConnection == nil {
			fmt.Printf("-> not connection found for this UUID! %s\n", uuidStr)
			continue
		}
		fmt.Printf("ClientCurrencyPriceUpdatedEvent Node Connection %s %s\n", nodeConnection.NodeUUID, nodeConnection.NodeAddr)
		// Check if our connection is stale
		if time.Now().Before(nodeConnection.StaleAt) {
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
		} else {
			fmt.Printf("ClientCurrencyPriceUpdatedEvent Node Connection Stale %s\n", nodeConnection.NodeUUID)
			// Close connection first
			nodeConnection.Conn.Close()
			// Connection is stale remove it from collection
			delete(cs.NodeConnections, uuidStr)
		}
	}
	return nil
}
