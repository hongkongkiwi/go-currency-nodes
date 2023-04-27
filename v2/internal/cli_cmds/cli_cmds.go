/**
* Contains all cli commands - modify this file to add more commands
**/
package cli_cmds

import (
	"fmt"
	"log"

	"github.com/gofrs/uuid/v5"
	pb "github.com/hongkongkiwi/go-currency-nodes/v2/gen/pb"
	configs "github.com/hongkongkiwi/go-currency-nodes/v2/internal/configs"
	"github.com/hongkongkiwi/go-currency-nodes/v2/internal/helpers"
	"google.golang.org/protobuf/encoding/protojson"
	anypb "google.golang.org/protobuf/types/known/anypb"
)

var SendChan chan *pb.StreamFromCli
var CancelChan chan bool

func sendToController(nodeUuids []*uuid.UUID, nodeCommand *anypb.Any) {
	nodeUuidStrings := helpers.StringsFromUuids(nodeUuids)
	SendChan <- &pb.StreamFromCli{
		NodeUuids: nodeUuidStrings,
		CliUuid:   configs.CliCfg.UUID().String(),
		Command:   nodeCommand,
	}
}

func NodeStatusV1Req(nodeUuids []*uuid.UUID) {
	genericPb, _ := anypb.New(&pb.NodeStatusV1Req{})
	sendToController(nodeUuids, genericPb)
}

func NodeStatusV1Reply(nodeUuid *uuid.UUID, reply *pb.NodeStatusV1Reply) {
	fmt.Printf("Node Status (%s): %v", nodeUuid.String(), protojson.Format(reply))
}

func NodeShutdownV1Reply(nodeUuid *uuid.UUID, reply *pb.NodeShutdownV1Reply) {
	fmt.Printf("Node Shutdown Reply (%s): OK", nodeUuid.String())
}

func ControllerKickNodesV1Req(nodeUuids []*uuid.UUID) {
	if len(nodeUuids) == 0 {
		return
	}
	nodeUuidStrings := helpers.StringsFromUuids(nodeUuids)
	genericPb, _ := anypb.New(&pb.ControllerKickNodesV1Req{
		NodeUuids: nodeUuidStrings,
	})
	sendToController(nil, genericPb)
}

func ControllerKickNodesV1Reply(reply *pb.ControllerKickNodesV1Reply) {
	fmt.Printf("Controller Kicked Nodes: %v", reply.KickedNodeUuids)
	CancelChan <- true
}

func ControllerListNodesV1Req() {
	genericPb, _ := anypb.New(&pb.ControllerListNodesV1Req{})
	sendToController(nil, genericPb)
}

func ControllerListNodesV1Reply(reply *pb.ControllerListNodesV1Reply) {
	fmt.Printf("Controller Online Nodes: %v", reply.NodeUuids)
	CancelChan <- true
}

func ControllerAppVersionV1Req() {
	genericPb, _ := anypb.New(&pb.ControllerAppVersionV1Req{})
	sendToController(nil, genericPb)
}

func ControllerAppVersionV1Reply(reply *pb.ControllerAppVersionV1Reply) {
	fmt.Printf("Controller App Version: %s", reply.ControllerAppVersion)
	CancelChan <- true
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
	var nodeUuid uuid.UUID
	if in.NodeUuid != nil {
		nodeUuid, _ = uuid.FromString(*in.NodeUuid)
	}
	switch reply := reply.(type) {
	case *pb.NodeShutdownV1Reply:
		NodeShutdownV1Reply(&nodeUuid, reply)
	case *pb.NodeStatusV1Reply:
		NodeStatusV1Reply(&nodeUuid, reply)
	case *pb.ControllerKickNodesV1Reply:
		ControllerKickNodesV1Reply(reply)
	case *pb.ControllerListNodesV1Reply:
		ControllerListNodesV1Reply(reply)
	case *pb.ControllerAppVersionV1Reply:
		ControllerAppVersionV1Reply(reply)
	default:
		log.Printf("Unhandled Command: %v", reply)
	}
}
