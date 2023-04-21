/**
* Contains all cli commands - modify this file to add more commands
**/
package cli_cmds

import (
	"log"

	"github.com/gofrs/uuid/v5"
	pb "github.com/hongkongkiwi/go-currency-nodes/v2/gen/pb"
	configs "github.com/hongkongkiwi/go-currency-nodes/v2/internal/configs"
	"github.com/hongkongkiwi/go-currency-nodes/v2/internal/helpers"
	anypb "google.golang.org/protobuf/types/known/anypb"
)

var SendChan chan *pb.StreamFromCli

func sendToController(nodeUuids []*uuid.UUID, nodeCommand *anypb.Any) {
	nodeUuidStrings := helpers.StringsFromUuids(nodeUuids)
	SendChan <- &pb.StreamFromCli{
		NodeUuids:   nodeUuidStrings,
		CliUuid:     configs.CliCfg.CliUUID.String(),
		NodeCommand: nodeCommand,
	}
}

func NodeAppVersion(nodeUuids []*uuid.UUID) {
	genericPb, _ := anypb.New(&pb.NodeAppVersionReply{})
	sendToController(nodeUuids, genericPb)
}

func NodeStatus(nodeUuids []*uuid.UUID) {
	genericPb, _ := anypb.New(&pb.NodeStatusReq{})
	sendToController(nodeUuids, genericPb)
}

func NodeUUID(nodeUuids []*uuid.UUID) {
	genericPb, _ := anypb.New(&pb.NodeUUIDReply{})
	sendToController(nodeUuids, genericPb)
}

func ControllerKickNodes(nodeUuids []*uuid.UUID) {
	if len(nodeUuids) == 0 {
		return
	}
	nodeUuidStrings := helpers.StringsFromUuids(nodeUuids)
	genericPb, _ := anypb.New(&pb.ControllerKickNodesReq{
		NodeUuids: nodeUuidStrings,
	})
	sendToController(nil, genericPb)
}

func ControllerListNodes() {
	genericPb, _ := anypb.New(&pb.ControllerListNodesReq{})
	sendToController(nil, genericPb)
}

func HandleIncomingReply(in *pb.StreamToCli) {
	if in.Response == nil {
		return
	}
	reply, unmrashalErr := in.Response.UnmarshalNew()
	if unmrashalErr != nil {
		// We couldn't unmarshel a proper PB object
		log.Printf("Command received Data Error: %v", unmrashalErr)
		return
	}
	switch reply := reply.(type) {
	case *pb.NodeAppVersionReply:
		log.Printf("Command response NodeAppVersionReply: %v", reply)
	case *pb.NodeStatusReply:
		log.Printf("Command received NodeStatusReply: %v", reply)
	case *pb.NodeUUIDReply:
		log.Printf("Command received NodeUUIDReply: %v", reply)
	case *pb.ControllerKickNodesReply:
		log.Printf("Command received ControllerKickNodesReply: %v", reply)
	case *pb.ControllerListNodesReply:
		log.Printf("Command received ControllerListNodesReply: %v", reply)
	default:
		log.Printf("Unhandled Command: %v", reply)
	}
}
