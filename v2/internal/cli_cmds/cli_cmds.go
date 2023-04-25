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
	streamUuid, _ := uuid.NewV4()
	SendChan <- &pb.StreamFromCli{
		StreamUuid:  streamUuid.String(),
		NodeUuids:   nodeUuidStrings,
		CliUuid:     configs.CliCfg.CliUUID.String(),
		NodeCommand: nodeCommand,
	}
}

func NodeAppVersion(nodeUuids []*uuid.UUID) {
	genericPb, _ := anypb.New(&pb.NodeAppVersionReply{})
	sendToController(nodeUuids, genericPb)
}

func NodeAppVersionReply(nodeUuid *uuid.UUID, reply *pb.NodeAppVersionReply) {
	fmt.Printf("Node App Version (%s): %s", nodeUuid.String(), reply.NodeVersion)
}

func NodeStatus(nodeUuids []*uuid.UUID) {
	genericPb, _ := anypb.New(&pb.NodeStatusReq{})
	sendToController(nodeUuids, genericPb)
}

func NodeStatusReply(nodeUuid *uuid.UUID, reply *pb.NodeStatusReply) {
	fmt.Printf("Node Status (%s): %v", nodeUuid.String(), protojson.Format(reply))
}

func NodeStartStream(nodeUuids []*uuid.UUID) {
	genericPb, _ := anypb.New(&pb.NodeStartStreamReq{})
	sendToController(nodeUuids, genericPb)
}

func NodeStartStreamReply(nodeUuid *uuid.UUID, reply *pb.NodeStartStreamReply) {
	fmt.Printf("Node Start Stream (%s): OK", nodeUuid.String())
}

func NodeStopStream(nodeUuids []*uuid.UUID) {
	genericPb, _ := anypb.New(&pb.NodeStopStreamReq{})
	sendToController(nodeUuids, genericPb)
}

func NodeStopStreamReply(nodeUuid *uuid.UUID, reply *pb.NodeStopStreamReply) {
	fmt.Printf("Node Stop Stream (%s): OK", nodeUuid.String())
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

func ControllerKickNodesReply(reply *pb.ControllerKickNodesReply) {
	fmt.Printf("Controller Kicked Nodes: %v", reply.KickedNodeUuids)
	CancelChan <- true
}

func ControllerListNodes() {
	genericPb, _ := anypb.New(&pb.ControllerListNodesReq{})
	sendToController(nil, genericPb)
}

func ControllerListNodesReply(reply *pb.ControllerListNodesReply) {
	fmt.Printf("Controller Online Nodes: %v", reply.NodeUuids)
	CancelChan <- true
}

func ControllerAppVersion() {
	genericPb, _ := anypb.New(&pb.ControllerAppVersionReq{})
	sendToController(nil, genericPb)
}

func ControllerAppVersionReply(reply *pb.ControllerAppVersionReply) {
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
	log.Printf("replycount: %d", in.ReplyCount)
	switch reply := reply.(type) {
	case *pb.NodeAppVersionReply:
		NodeAppVersionReply(&nodeUuid, reply)
	case *pb.NodeStatusReply:
		NodeStatusReply(&nodeUuid, reply)
	case *pb.NodeStartStreamReply:
		NodeStartStreamReply(&nodeUuid, reply)
	case *pb.NodeStopStreamReply:
		NodeStopStreamReply(&nodeUuid, reply)
	case *pb.ControllerKickNodesReply:
		ControllerKickNodesReply(reply)
	case *pb.ControllerListNodesReply:
		ControllerListNodesReply(reply)
	case *pb.ControllerAppVersionReply:
		ControllerAppVersionReply(reply)
	default:
		log.Printf("Unhandled Command: %v", reply)
	}
}
