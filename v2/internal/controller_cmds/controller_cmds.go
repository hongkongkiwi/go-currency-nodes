package controller_cmds

import (
	"fmt"
	"log"
	"time"

	"github.com/gofrs/uuid/v5"
	pb "github.com/hongkongkiwi/go-currency-nodes/v2/gen/pb"
	"github.com/hongkongkiwi/go-currency-nodes/v2/internal/configs"
	"github.com/hongkongkiwi/go-currency-nodes/v2/internal/helpers"
	"github.com/hongkongkiwi/go-currency-nodes/v2/internal/stores"
	anypb "google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func controllerListNodesV1Req(cliUuid *uuid.UUID) error {
	genericPb, err := anypb.New(&pb.ControllerListNodesV1Reply{
		NodeUuids: stores.NodeSendChannelStore.KeysAsString(),
	})
	if err != nil {
		return err
	}
	return sendToCli(cliUuid, genericPb)
}

func controllerKickNodesV1Req(cliUuid *uuid.UUID, nodeUuids []*uuid.UUID) error {
	kickedNodes := make([]*uuid.UUID, 0, len(nodeUuids))
	for _, nodeUuid := range nodeUuids {
		cancelChan, _ := stores.NodeCancelChannelStore.GetFromUUID(nodeUuid)
		if cancelChan != nil {
			kickedNodes = append(kickedNodes, nodeUuid)
		}
		cancelChan <- true
	}
	genericPb, _ := anypb.New(&pb.ControllerKickNodesV1Reply{
		KickedNodeUuids: helpers.StringsFromUuids(kickedNodes),
	})
	return sendToCli(cliUuid, genericPb)
}

func controllerAppVersionV1Req(cliUuid *uuid.UUID) error {
	genericPb, _ := anypb.New(&pb.ControllerAppVersionV1Reply{
		ControllerAppVersion: configs.ControllerAppVersion,
	})
	return sendToCli(cliUuid, genericPb)
}

func unknownCliCommandV1(cliUuid *uuid.UUID, unknownCmd *anypb.Any) {
	genericPb, _ := anypb.New(&pb.UnknownCommandV1Notify{
		Command: unknownCmd,
	})
	sendToCli(cliUuid, genericPb)
}

func unknownNodeCommandV1(nodeUuid *uuid.UUID, unknownCmd *anypb.Any) {
	genericPb, _ := anypb.New(&pb.UnknownCommandV1Notify{
		Command: unknownCmd,
	})
	sendToNodesFromController([]*uuid.UUID{nodeUuid}, genericPb)
}

// When node sends it's known currencies to us, we will then tell it to selectively
// stream currencies. This is done on a first come first served basis
func nodeKnownCurrenciesV1Notify(nodeUuid *uuid.UUID, knownCurrencyPairs []string) {
	stores.ControllerNodeCurrencies.SetNodeCurrencies(nodeUuid.String(), knownCurrencyPairs)
	log.Printf("Node (%s) Currency Pairs: %s", nodeUuid.String(), knownCurrencyPairs)
	refreshNodePriceStreams()
}

func nodePriceUpdateV1Notify(nodeUuid *uuid.UUID, currencyPair string, price float64, updatedAt time.Time) {
	log.Printf("Node (%s) Price Update: %s, %0.2f, %d", nodeUuid, currencyPair, price, updatedAt.UnixMilli())
	stores.ControllerPriceStore.SetPrice(currencyPair, price, updatedAt)
}

func refreshNodePriceStreams() {
	if configs.ControllerCfg.MaxPriceStreams == 0 {
		return
	}
	nodeCurrencies := make(map[string][]string)
	// Loop through all our currencies and build a list of streaming nodes to send messages to
	for _, currencyPair := range stores.ControllerNodeCurrencies.GetAllCurrencies() {
		count := stores.ControllerNodeCurrencies.CountNodeStreamingCurrency(currencyPair)
		// If we have less nodes than we need then grab some
		if count < configs.ControllerCfg.MaxPriceStreams {
			// Grab a list of nodes to stream randomly
			for _, nodeUuidStr := range stores.ControllerNodeCurrencies.GetRandomNodesForCurrency(currencyPair, true, configs.ControllerCfg.MaxPriceStreams-count) {
				nodeCurrencies[nodeUuidStr] = append(nodeCurrencies[nodeUuidStr], currencyPair)
			}
		}
	}
	for nodeUuidStr, currencies := range nodeCurrencies {
		nodeUuid, _ := uuid.FromString(nodeUuidStr)
		stores.ControllerNodeCurrencies.SetNodeStreamingCurrencies(nodeUuidStr, currencies)
		nodeSendStartStreamingPrices(&nodeUuid, currencies)
	}
}

func nodeSendStartStreamingPrices(nodeUuid *uuid.UUID, currencyPairs []string) {
	newStreamStates := make([]*pb.CurrencyStreamStateV1, 0)
	for _, currencyPair := range currencyPairs {
		if hasPrice, latestPrice, latestPriceUpdatedAt := stores.ControllerPriceStore.GetPrice(currencyPair); hasPrice {
			newStreamStates = append(newStreamStates, &pb.CurrencyStreamStateV1{
				StreamState:          pb.CurrencyStreamStateV1_StreamStart,
				CurrencyPair:         currencyPair,
				LatestPrice:          &latestPrice,
				LatestPriceUpdatedAt: timestamppb.New(latestPriceUpdatedAt),
			})
		} else {
			newStreamStates = append(newStreamStates, &pb.CurrencyStreamStateV1{
				StreamState:          pb.CurrencyStreamStateV1_StreamStart,
				CurrencyPair:         currencyPair,
				LatestPrice:          nil,
				LatestPriceUpdatedAt: nil,
			})
		}
	}
	if genericPb, err := anypb.New(&pb.NodeStreamModifyStateV1Req{
		StreamStates: newStreamStates,
	}); err == nil {
		sendToNodesFromController([]*uuid.UUID{nodeUuid}, genericPb)
	}
}

func HandleNodeReady() {
	// Nothing to do
}

func HandleNodeConnected(nodeUuid *uuid.UUID) {
	if nodeUuid == nil {
		return
	}
	// Clear any stores node stores incase we missed it
	stores.ControllerNodeCurrencies.DeleteNode(nodeUuid.String())
	log.Printf("Node (%s) is connected", nodeUuid)
}

func HandleNodeDisconnected(nodeUuid *uuid.UUID) {
	if nodeUuid == nil {
		return
	}
	// Clear any stores node stores incase we missed it
	stores.ControllerNodeCurrencies.DeleteNode(nodeUuid.String())
	log.Printf("Node (%s) is disconnected", nodeUuid)
	refreshNodePriceStreams()
}

func HandleIncomingNodeCommandToController(genericCmd *anypb.Any, nodeUuid *uuid.UUID) {
	// In this case, incoming messages are actually a reply
	// (excluding initial command) so try to unmarshel it
	cmd, unmrashalErr := genericCmd.UnmarshalNew()
	if unmrashalErr != nil {
		// We couldn't unmarshel a proper PB object
		log.Printf("Node Command Error: %v", unmrashalErr)
		return
	}
	switch cmd := cmd.(type) {
	/** Node Notifications **/
	case *pb.NodeKnownCurrenciesV1Notify:
		log.Printf("Node sent NodeKnownCurrenciesV1Notify")
		nodeKnownCurrenciesV1Notify(nodeUuid, cmd.KnownCurrencyPairs)
	case *pb.NodePriceUpdateV1Notify:
		log.Printf("Node sent NodePriceUpdateV1Notify")
		nodePriceUpdateV1Notify(nodeUuid, cmd.CurrencyPair, cmd.Price, cmd.PriceUpdatedAt.AsTime())
	/** Unknown Command **/
	default:
		log.Printf("Unhandled Node Command: %v", cmd)
		unknownNodeCommandV1(nodeUuid, genericCmd)
	}
}

func HandleIncomingCliCommand(genericCmd *anypb.Any, cliUuid *uuid.UUID, sendToUUIDs []*uuid.UUID) {
	// In this case, incoming messages are actually a reply
	// (excluding initial command) so try to unmarshel it
	cmd, unmrashalErr := genericCmd.UnmarshalNew()
	if unmrashalErr != nil {
		// We couldn't unmarshel a proper PB object
		log.Printf("Cli Node Command Error: %v", unmrashalErr)
		return
	}
	switch cmd := cmd.(type) {

	/** Commands to Controller **/
	case *pb.ControllerAppVersionV1Req:
		log.Printf("Cli Requested ControllerAppVersionV1Req")
		// Handle the command
		err := controllerAppVersionV1Req(cliUuid)
		if err != nil {
			log.Printf("ControllerAppVersionV1 Error: %v", err)
		}
	case *pb.ControllerListNodesV1Req:
		log.Printf("Cli Requested ControllerListNodesV1Req")
		// Handle the command
		err := controllerListNodesV1Req(cliUuid)
		if err != nil {
			log.Printf("ControllerListNodesV1 Error: %v", err)
		}
	case *pb.ControllerKickNodesV1Req:
		log.Printf("Cli Requested ControllerKickNodesV1Req")
		nodeUuids, _ := helpers.UuidsFromStrings(cmd.NodeUuids)
		// Handle the command
		err := controllerKickNodesV1Req(cliUuid, nodeUuids)
		if err != nil {
			log.Printf("ControllerKickNodesV1 Error: %v", err)
		}

	/** Commands to forward to Node(s) **/
	case *pb.NodeStatusV1Req:
		log.Printf("Cli Requested NodeStatusV1Req")
		// Forward the command
		err := sendToNodesFromCli(sendToUUIDs, cliUuid, genericCmd)
		if err != nil {
			log.Printf("NodeStatusV1Req Error: %v", err)
		}
	case *pb.NodeShutdownV1Req:
		log.Printf("Cli Requested NodeShutdownV1Req")
		// Forward the command
		err := sendToNodesFromCli(sendToUUIDs, cliUuid, genericCmd)
		if err != nil {
			log.Printf("NodeShutdownV1Req Error: %v", err)
		}

	/** Unknown Command **/
	default:
		log.Printf("Unhandled Cli Command: %v", cmd)
		unknownCliCommandV1(cliUuid, genericCmd)
	}
}

func sendToNodesFromController(toUUIDs []*uuid.UUID, command *anypb.Any) error {
	if len(toUUIDs) == 0 {
		return fmt.Errorf("must pass atleast one node uuid to send to")
	}
	for _, UUID := range toUUIDs {
		if channel, _ := stores.NodeSendChannelStore.GetFromUUID(UUID); channel != nil {
			// Push our command to the appropriate channel
			channel <- &pb.StreamToNode{
				Command: command,
			}
		}
	}
	return nil
}

// Send message over our node streaming channel
func sendToNodesFromCli(toUUIDs []*uuid.UUID, cliUUID *uuid.UUID, command *anypb.Any) error {
	if len(toUUIDs) == 0 {
		return fmt.Errorf("must pass atleast one node uuid to send to")
	}
	if cliUUID == nil {
		return fmt.Errorf("must pass cli uuid")
	}
	for _, UUID := range toUUIDs {
		if channel, _ := stores.NodeSendChannelStore.GetFromUUID(UUID); channel != nil {
			cliUUIDString := cliUUID.String()
			// Push our command to the appropriate channel
			channel <- &pb.StreamToNode{
				FromUuid: &cliUUIDString,
				Command:  command,
			}
		}
	}
	return nil
}

// Send message over our cli streaming channel
func sendToCli(toUUID *uuid.UUID, response *anypb.Any) error {
	if toUUID == nil {
		return fmt.Errorf("must pass cli uuid to send to")
	}
	if channel, _ := stores.CliChannelStore.GetFromUUID(toUUID); channel != nil {
		// Push our command to the appropriate channel
		channel <- &pb.StreamToCli{
			Response: response,
		}
	} else {
		log.Printf("WARNING: We do not know about UUID: %s ignoring", toUUID)
	}
	return nil
}
