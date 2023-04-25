package controller_cmds

import (
	"fmt"
	"log"

	"github.com/gofrs/uuid/v5"
	pb "github.com/hongkongkiwi/go-currency-nodes/v2/gen/pb"
	"github.com/hongkongkiwi/go-currency-nodes/v2/internal/configs"
	"github.com/hongkongkiwi/go-currency-nodes/v2/internal/helpers"
	"github.com/hongkongkiwi/go-currency-nodes/v2/internal/stores"
	anypb "google.golang.org/protobuf/types/known/anypb"
)

var CliReplyCount *stores.CliReplyCount

func nodeAppVersionReq(toUUIDs []*uuid.UUID, fromUUID *uuid.UUID) error {
	log.Printf("NodeAppVersionReq: %v", toUUIDs)
	genericPb, err := anypb.New(&pb.NodeAppVersionReq{})
	if err != nil {
		return fmt.Errorf("unknown Protobuf Struct %v", err)
	}
	sendToNodes(toUUIDs, fromUUID, genericPb)
	return nil
}

func nodeStatusReq(toUUIDs []*uuid.UUID, fromUUID *uuid.UUID) error {
	log.Printf("NodeStatusReq: %v", toUUIDs)
	genericPb, err := anypb.New(&pb.NodeStatusReq{})
	if err != nil {
		return fmt.Errorf("unknown Protobuf Struct %v", err)
	}
	sendToNodes(toUUIDs, fromUUID, genericPb)
	return nil
}

func nodeStartStreamReq(toUUIDs []*uuid.UUID, fromUUID *uuid.UUID) error {
	log.Printf("NodeStartStreamReq: %v", toUUIDs)
	genericPb, err := anypb.New(&pb.NodeStartStreamReq{})
	if err != nil {
		return fmt.Errorf("unknown Protobuf Struct %v", err)
	}
	sendToNodes(toUUIDs, fromUUID, genericPb)
	return nil
}

func nodeStopStreamReq(toUUIDs []*uuid.UUID, fromUUID *uuid.UUID) error {
	log.Printf("NodeStopStreamReq: %v", toUUIDs)
	genericPb, err := anypb.New(&pb.NodeStopStreamReq{})
	if err != nil {
		return fmt.Errorf("unknown Protobuf Struct %v", err)
	}
	sendToNodes(toUUIDs, fromUUID, genericPb)
	return nil
}

func controllerListNodesReq(cliUuid *uuid.UUID) error {
	log.Printf("ControllerListNodes")
	genericPb, err := anypb.New(&pb.ControllerListNodesReply{
		NodeUuids: stores.NodeSendChannelStore.KeysAsString(),
	})
	if err != nil {
		return err
	}
	return sendToCli(cliUuid, genericPb)
}

func controllerKickNodesReq(cliUuid *uuid.UUID, nodeUuids []*uuid.UUID) error {
	log.Printf("ControllerKickNodes: %v", nodeUuids)
	kickedNodes := make([]*uuid.UUID, 0, len(nodeUuids))
	for _, nodeUuid := range nodeUuids {
		cancelChan, _ := stores.NodeCancelChannelStore.GetFromUUID(nodeUuid)
		if cancelChan != nil {
			kickedNodes = append(kickedNodes, nodeUuid)
		}
		cancelChan <- true
	}
	genericPb, _ := anypb.New(&pb.ControllerKickNodesReply{
		KickedNodeUuids: helpers.StringsFromUuids(kickedNodes),
	})
	return sendToCli(cliUuid, genericPb)
}

func controllerAppVersionReq(cliUuid *uuid.UUID) error {
	log.Printf("ControllerAppVersion")
	genericPb, _ := anypb.New(&pb.ControllerAppVersionReply{
		ControllerAppVersion: configs.ControllerCfg.AppVersion,
	})
	return sendToCli(cliUuid, genericPb)
}

func HandleCliCommand(genericCmd *anypb.Any, cliUuid *uuid.UUID, sendToUUIDs []*uuid.UUID) {
	// In this case, incoming messages are actually a reply
	// (excluding initial command) so try to unmarshel it
	cmd, unmrashalErr := genericCmd.UnmarshalNew()
	if unmrashalErr != nil {
		// We couldn't unmarshel a proper PB object
		log.Printf("Cli Node Command Error: %v", unmrashalErr)
		return
	}
	switch cmd := cmd.(type) {
	case *pb.ControllerAppVersionReq:
		log.Printf("Cli Requested ControllerAppVersion")
		err := controllerAppVersionReq(cliUuid)
		if err != nil {
			log.Printf("ControllerListNodes Error: %v", err)
		}
	case *pb.ControllerListNodesReq:
		log.Printf("Cli Requested ControllerListNodes")
		err := controllerListNodesReq(cliUuid)
		if err != nil {
			log.Printf("ControllerListNodes Error: %v", err)
		}
	case *pb.ControllerKickNodesReq:
		log.Printf("Cli Requested ControllerKickNode")
		nodeUuids, _ := helpers.UuidsFromStrings(cmd.NodeUuids)
		err := controllerKickNodesReq(cliUuid, nodeUuids)
		if err != nil {
			log.Printf("ControllerKickNode Error: %v", err)
		}
	case *pb.NodeAppVersionReq:
		log.Printf("Cli Requested NodeAppVersion")
		err := nodeAppVersionReq(sendToUUIDs, cliUuid)
		if err != nil {
			log.Printf("NodeAppVersionReq Error: %v", err)
		}
	case *pb.NodeStatusReq:
		log.Printf("Cli Requested NodeStatusReq")
		err := nodeStatusReq(sendToUUIDs, cliUuid)
		if err != nil {
			log.Printf("NodeStatusReq Error: %v", err)
		}
	case *pb.NodeStartStreamReq:
		log.Printf("Cli Requested NodeStartStreamReq")
		err := nodeStartStreamReq(sendToUUIDs, cliUuid)
		if err != nil {
			log.Printf("NodeStartStreamReq Error: %v", err)
		}
	case *pb.NodeStopStreamReq:
		log.Printf("Cli Requested NodeStopStreamReq")
		err := nodeStopStreamReq(sendToUUIDs, cliUuid)
		if err != nil {
			log.Printf("NodeStopStreamReq Error: %v", err)
		}
	default:
		log.Printf("Unhandled Command: %v", cmd)
	}
}

// Function to send message over our node streaming channel
func sendToNodes(toUUIDs []*uuid.UUID, fromUUID *uuid.UUID, command *anypb.Any) error {
	if len(toUUIDs) == 0 {
		return fmt.Errorf("must pass atleast one node uuid to send to")
	}
	if fromUUID == nil {
		return fmt.Errorf("must pass from uuid")
	}
	// Loop twice here so we don't race in getting reply count
	for _, UUID := range toUUIDs {
		if channel, _ := stores.NodeSendChannelStore.GetFromUUID(UUID); channel != nil {
			CliReplyCount.Add(fromUUID.String())
		} else {
			log.Printf("WARNING: We do not know about UUID: %s ignoring", UUID)
		}
	}
	for _, UUID := range toUUIDs {
		if channel, _ := stores.NodeSendChannelStore.GetFromUUID(UUID); channel != nil {
			// Push our command to the appropriate channel
			channel <- &pb.StreamToNode{
				FromUuid: fromUUID.String(),
				Command:  command,
			}
		}
	}
	return nil
}

// Function to send message over our cli streaming channel
func sendToCli(toUUID *uuid.UUID, response *anypb.Any) error {
	if toUUID == nil {
		return fmt.Errorf("must pass cli uuid to send to")
	}
	if channel, _ := stores.CliChannelStore.GetFromUUID(toUUID); channel != nil {
		// Push our command to the appropriate channel
		channel <- &pb.StreamToCli{
			ReplyCount: CliReplyCount.Get(toUUID.String()),
			Response:   response,
		}
	} else {
		log.Printf("WARNING: We do not know about UUID: %s ignoring", toUUID)
	}
	return nil
}
