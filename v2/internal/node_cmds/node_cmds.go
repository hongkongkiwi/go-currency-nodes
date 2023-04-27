/**
* Contains all node commands - modify this file to add more commands
**/
package node_cmds

import (
	"log"
	"math/rand"
	"time"

	"github.com/gofrs/uuid/v5"
	pb "github.com/hongkongkiwi/go-currency-nodes/v2/gen/pb"
	configs "github.com/hongkongkiwi/go-currency-nodes/v2/internal/configs"
	stores "github.com/hongkongkiwi/go-currency-nodes/v2/internal/stores"
	"github.com/tebeka/atexit"
	anypb "google.golang.org/protobuf/types/known/anypb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

var SendChan chan *pb.StreamFromNode
var ConnectedController string
var ConnectedAt time.Time
var SeededRand *rand.Rand

const maxPriceGenInterval = 10 // Maximum seconds between price generation
const minPriceGenInterval = 2  // Minimum seconds between price generation

func getStreamStates() []*pb.CurrencyStreamStateV1 {
	currencyStreamStates := make([]*pb.CurrencyStreamStateV1, 0)
	for currencyPair, streaming := range stores.LatestPricesAPI.GetAllCurrencyPairs() {
		var streamState pb.CurrencyStreamStateV1_StreamState
		if streaming {
			streamState = pb.CurrencyStreamStateV1_StreamStart
		} else {
			streamState = pb.CurrencyStreamStateV1_StreamStop
		}
		knwnPrice := stores.LatestPricesAPI.GetLatestPriceUpdate(currencyPair)
		currencyStreamStates = append(currencyStreamStates, &pb.CurrencyStreamStateV1{
			CurrencyPair:         currencyPair,
			StreamState:          streamState,
			LatestPrice:          &knwnPrice.Price,
			LatestPriceUpdatedAt: timestamppb.New(knwnPrice.UpdatedAt),
		})
	}
	return currencyStreamStates
}

func sendToController(fromUuid *uuid.UUID, response *anypb.Any) {
	if fromUuid == nil {
		SendChan <- &pb.StreamFromNode{
			NodeUuid: configs.NodeCfg.UUID().String(),
			Response: response,
		}
	} else {
		fromUuidStr := fromUuid.String()
		SendChan <- &pb.StreamFromNode{
			FromUuid: &fromUuidStr,
			NodeUuid: configs.NodeCfg.UUID().String(),
			Response: response,
		}
	}
}

func nodeStatusV1(fromUuid *uuid.UUID) {
	genericPb, _ := anypb.New(&pb.NodeStatusV1Reply{
		AppVersion:                configs.NodeAppVersion,
		ConnectedControllerServer: ConnectedController,
		ConnectedAt:               timestamppb.New(ConnectedAt),
		StreamStates:              getStreamStates(),
		Config: &pb.NodeStatusV1Reply_NodeConfig{
			Uuid:              configs.NodeCfg.UUID().String(),
			Name:              configs.NodeCfg.Name,
			VerboseLog:        configs.NodeCfg.VerboseLog,
			ControllerServers: configs.NodeCfg.ControllerServers,
		},
	})
	log.Printf("Response sent: NodeStatusV1Reply")
	sendToController(fromUuid, genericPb)
}

func nodeModifyStreamStateV1(fromUuid *uuid.UUID, newStreamStates []*pb.CurrencyStreamStateV1) {
	// First stop everything streaming
	stores.LatestPricesAPI.StopGeneratingAllPrices()
	// Loop through currency pairs to update (either specifically start or stop)
	for _, currenyStreamState := range newStreamStates {
		streamState := currenyStreamState.StreamState
		currencyPair := currenyStreamState.CurrencyPair
		latestPrice := currenyStreamState.LatestPrice
		latestPriceUpdatedAt := currenyStreamState.LatestPriceUpdatedAt
		if latestPrice != nil && latestPriceUpdatedAt != nil {
			// Update the price store but do not trigger callback since this is a price from server
			stores.LatestPricesAPI.SetLatestPriceUpdateFromValues(currencyPair, *latestPrice, latestPriceUpdatedAt.AsTime(), false)
		}
		if streamState == pb.CurrencyStreamStateV1_StreamStart {
			// Generate prices in a random interval
			generateInterval := time.Duration(SeededRand.Intn(maxPriceGenInterval-minPriceGenInterval)+minPriceGenInterval) * time.Second
			stores.LatestPricesAPI.StartGeneratingPrices(currencyPair, generateInterval, notifyControllerPriceUpdateV1)
		}
	}
	genericPb, _ := anypb.New(&pb.NodeStreamModifyStateV1Reply{})
	log.Printf("Response sent: NodeStreamModifyStateV1Reply")
	sendToController(fromUuid, genericPb)
}

func nodeShutdownV1(fromUuid *uuid.UUID) {
	shutdownAfter := time.Duration(2 * time.Second)
	genericPb, _ := anypb.New(&pb.NodeShutdownV1Reply{
		ShutdownAt: timestamppb.New(time.Now().Add(shutdownAfter)),
	})
	log.Printf("Response sent: NodeShutdownV1Reply")
	sendToController(fromUuid, genericPb)
	<-time.After(shutdownAfter)
	atexit.Exit(0)
}

func notifyUnknownCommandV1(fromUuid *uuid.UUID, unknownCmd *anypb.Any) {
	genericPb, _ := anypb.New(&pb.UnknownCommandV1Notify{
		Command: unknownCmd,
	})
	log.Printf("Notification sent: UnknownCommandV1Notify")
	sendToController(fromUuid, genericPb)
}

func notifyControllerPriceUpdateV1(currencyPair string, price float64, updatedAt time.Time) {
	genericPb, _ := anypb.New(&pb.NodePriceUpdateV1Notify{
		CurrencyPair:   currencyPair,
		Price:          price,
		PriceUpdatedAt: timestamppb.New(updatedAt),
	})
	log.Printf("Notification sent: NodePriceUpdateV1Notify: %s, %0.2f, %d", currencyPair, price, updatedAt)
	sendToController(nil, genericPb)
}

// Send a list of this node known currency pairs to controller
func notifyNodeCurrencyPairs() {
	genericPb, _ := anypb.New(&pb.NodeKnownCurrenciesV1Notify{
		KnownCurrencyPairs: configs.NodeCfg.KnownCurrencyPairs,
	})
	log.Printf("Notification sent: NodeKnownCurrenciesV1Notify: %s", configs.NodeCfg.KnownCurrencyPairs)
	sendToController(nil, genericPb)
}

func HandleControllerConnected(controllerServer string) {
	if stores.LatestPricesAPI != nil {
		go stores.LatestPricesAPI.StopGeneratingAllPrices()
		stores.LatestPricesAPI = nil
	}
	stores.LatestPricesAPI = stores.NewLatestPriceAPI(configs.NodeCfg.KnownCurrencyPairs)
	ConnectedController = controllerServer
	ConnectedAt = time.Now()
}

func HandleControllerReady() {
	// Tell the controller about our known currency pairs
	notifyNodeCurrencyPairs()
}

func HandleControllerDisconnected() {
	if stores.LatestPricesAPI != nil {
		go stores.LatestPricesAPI.StopGeneratingAllPrices()
		stores.LatestPricesAPI = nil
	}
}

// This is the only function exposed
func HandleIncomingCommand(fromUuid *uuid.UUID, inCmd *anypb.Any) {
	cmd, unmarshalErr := inCmd.UnmarshalNew()
	// This error should not happen, maybe invalid PB object, log but ignore
	if unmarshalErr != nil {
		log.Printf("Command received Data Error: %v", unmarshalErr)
		return
	}
	switch cmd := cmd.(type) {
	// Request Node to send it's status
	case *pb.NodeStatusV1Req:
		log.Printf("Command received NodeStatusV1Req")
		nodeStatusV1(fromUuid)
	// Request Node to Modify Stream State
	case *pb.NodeStreamModifyStateV1Req:
		logMap := make(map[string]pb.CurrencyStreamStateV1_StreamState)
		for _, streamState := range cmd.StreamStates {
			logMap[streamState.CurrencyPair] = streamState.StreamState
		}
		log.Printf("Command received NodeStreamModifyStateV1Req: %v", logMap)
		nodeModifyStreamStateV1(fromUuid, cmd.StreamStates)
	// Request this Node to Shutdown
	case *pb.NodeShutdownV1Req:
		log.Printf("Command received NodeShutdownV1Req")
		nodeShutdownV1(fromUuid)
	default:
		log.Printf("Unhandled command received: %v", cmd)
		notifyUnknownCommandV1(fromUuid, inCmd)
	}
}
