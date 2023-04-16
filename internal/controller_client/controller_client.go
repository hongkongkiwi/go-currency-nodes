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
	"time"

	"github.com/gofrs/uuid/v5"
	cs "github.com/hongkongkiwi/go-currency-nodes/internal/controller_server"
	helpers "github.com/hongkongkiwi/go-currency-nodes/internal/helpers"
	pb "github.com/hongkongkiwi/go-currency-nodes/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const debugName = "ControllerClient"

var RequestTimeout uint64

func funcName() string {
	pc, _, _, _ := runtime.Caller(1)
	names := strings.Split(runtime.FuncForPC(pc).Name(), ".")
	return names[len(names)-1]
}

// Controller calls this on node when a new price comes in
// rpc CurrencyPriceUpdatedEvent (CurrencyPriceUpdateEventReq) returns (google.protobuf.Empty) {}
func ClientCurrencyPriceUpdatedEvent(updatedCurrencyStoreItems map[string]*helpers.CurrencyStoreItem) error {
	// Keep a temporary list of currencies & connections to send out
	currenciesToSend := make(map[*cs.NodeConnection][]*pb.CurrencyItem)
	// Loop through all currency store item updates
	for currencyPair, storeItem := range updatedCurrencyStoreItems {
		// Check all node uuids we should notify for this currency pair
		scriptionUuids := cs.NodeSubscriptions[currencyPair]
		for i := 0; i < len(scriptionUuids); i++ {
			uuid, uuidErr := uuid.FromString(scriptionUuids[i])
			if uuidErr != nil {
				log.Printf("invalid node uuid sent: %s", uuidErr)
				continue
			}
			nodeConnection := cs.NodeConnections[uuid]
			if nodeConnection != nil {
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
					delete(cs.NodeConnections, uuid)
				}
			}
		}
	}
	// Loop through each connection and send out relevant currency items
	for nodeConnection, currencyItems := range currenciesToSend {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(RequestTimeout))
		defer cancel()
		log.Printf("%s->gRPC->Outgoing: %s (%s)", debugName, funcName(), helpers.NodeCfg.CurrencyPairs)
		c := pb.NewNodeEventsClient(nodeConnection.Conn)
		_, sendErr := c.CurrencyPriceUpdatedEvent(ctx, &pb.CurrencyPriceUpdateEventReq{CurrencyItems: currencyItems})
		if sendErr != nil {
			log.Printf("%s->gRPC->ERROR Outgoing: %s (%v)", debugName, funcName(), sendErr)
		} else {
			// Send currency updates to this UUID
			fmt.Printf("Sent to [%s]: %s", &nodeConnection.UUID, currencyItems)
		}
	}
	return nil
}
