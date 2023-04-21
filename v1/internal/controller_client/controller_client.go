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
	// Keep a temporary list of uuids and associated currency items
	sendToUUIDs := make(map[string][]*pb.CurrencyItem)

	// Loop through all currency store item updates
	for currencyPair, storeItem := range updatedCurrencyItems {
		// Check all node uuids we should notify for this currency pair
		subs, _ := cs.ControllerSubscriptionsStore.Get(currencyPair)
		if subs == nil {
			continue
		}
		if subs != nil {
			for _, uuidStr := range subs.UUIDs() {
				uuid, uuidErr := uuid.FromString(uuidStr)
				if uuidErr != nil {
					continue
				}
				// Don't resend this update to the same uuid
				if storeItem.UpdatedByUUID == uuid {
					continue
				}
				sendToUUIDs[uuidStr] = append(sendToUUIDs[uuidStr], &pb.CurrencyItem{
					CurrencyPair: currencyPair,
					Price:        storeItem.Price,
					PriceValidAt: timestamppb.New(storeItem.ValidAt),
				})
			}
		}
	}

	// Chceck that we have a connection for each uuid we wamt tp semd tp
	for uuidStr, currencyItems := range sendToUUIDs {
		nodeConnection := cs.NodeConnections[uuidStr]
		// No node connection for this uuid so just skip (must not be connected)
		if nodeConnection == nil {
			continue
		}
		// Check if our connection is stale
		if time.Now().Before(nodeConnection.StaleAt) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(RequestTimeout))
			defer cancel()
			reply := &pb.CurrencyPriceUpdateEventReq{CurrencyItems: currencyItems}
			if helpers.ControllerCfg.VerboseLog {
				log.Printf("%s->gRPC->Outgoing: %s\n%s\n", debugName, funcName(), protojson.Format(reply))
			}
			c := pb.NewNodeEventsClient(nodeConnection.Conn)
			_, sendErr := c.CurrencyPriceUpdatedEvent(ctx, reply)
			if sendErr != nil {
				log.Printf("%s->gRPC->ERROR Outgoing: %s\n%v\n", debugName, funcName(), sendErr)
			} else {
				logOutputPrices := make([][2]string, len(currencyItems))
				// Loop so we can format for logging
				for i, curr := range currencyItems {
					logOutputPrices[i] = [2]string{curr.CurrencyPair, fmt.Sprintf("%0.2f", curr.Price)}
				}
				log.Printf("Pushed new prices to %s for %v\n", nodeConnection.NodeUUID, logOutputPrices)
			}
		} else {
			if helpers.ControllerCfg.VerboseLog {
				log.Printf("%s Discovered Node Connection Stale %s\n", funcName(), nodeConnection.NodeUUID)
			}
			// Close connection first
			nodeConnection.Conn.Close()
			// Connection is stale remove it from collection
			delete(cs.NodeConnections, uuidStr)
		}
	}
	return nil
}
