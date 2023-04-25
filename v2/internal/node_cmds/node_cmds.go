/**
* Contains all node commands - modify this file to add more commands
**/
package node_cmds

import (
	"log"

	pb "github.com/hongkongkiwi/go-currency-nodes/v2/gen/pb"
	configs "github.com/hongkongkiwi/go-currency-nodes/v2/internal/configs"
	anypb "google.golang.org/protobuf/types/known/anypb"
)

var SendChan chan *pb.StreamFromNode

func sendToController(fromUuid string, response *anypb.Any) {
	if fromUuid == "" {
		SendChan <- &pb.StreamFromNode{
			NodeUuid: configs.NodeCfg.NodeUUID.String(),
			Response: response,
		}
	} else {
		SendChan <- &pb.StreamFromNode{
			FromUuid: &fromUuid,
			NodeUuid: configs.NodeCfg.NodeUUID.String(),
			Response: response,
		}
	}
}

func nodeAppVersion(fromUuid string) {
	genericPb, _ := anypb.New(&pb.NodeAppVersionReply{
		NodeVersion: configs.NodeCfg.AppVersion,
	})
	sendToController(fromUuid, genericPb)
}

func nodeStatus(fromUuid string) {
	genericPb, _ := anypb.New(&pb.NodeStatusReply{
		NodeUuid:         configs.NodeCfg.NodeUUID.String(),
		NodeVersion:      configs.NodeCfg.AppVersion,
		ControllerServer: configs.NodeCfg.ControllerServer,
		StreamUpdates:    configs.NodeCfg.StreamUpdates,
	})
	sendToController(fromUuid, genericPb)
}

func nodeStartStream(fromUuid string) {
	configs.NodeCfg.StreamUpdates = true
	genericPb, _ := anypb.New(&pb.NodeStartStreamReply{})
	sendToController(fromUuid, genericPb)
}

func nodeStopStream(fromUuid string) {
	configs.NodeCfg.StreamUpdates = false
	genericPb, _ := anypb.New(&pb.NodeStopStreamReply{})
	sendToController(fromUuid, genericPb)
}

// Just an empty command to initiate the stream
func ControllerHello() {
	sendToController("", nil)
}

func HandleIncomingCommand(fromUuid string, inCmd *anypb.Any) {
	cmd, unmrashalErr := inCmd.UnmarshalNew()
	if unmrashalErr != nil {
		// We couldn't unmarshel a proper PB object
		log.Printf("Command received Data Error: %v", unmrashalErr)
		return
	}
	switch cmd := cmd.(type) {
	case *pb.NodeAppVersionReq:
		log.Printf("Command received NodeAppVersionReq")
		nodeAppVersion(fromUuid)
	case *pb.NodeStatusReq:
		log.Printf("Command received NodeStatusReq")
		nodeStatus(fromUuid)
	case *pb.NodeStartStreamReq:
		log.Printf("Command received NodeStartStreamReq")
		nodeStartStream(fromUuid)
	case *pb.NodeStopStreamReq:
		log.Printf("Command received NodeStopStreamReq")
		nodeStopStream(fromUuid)
	default:
		log.Printf("Unhandled Command: %v", cmd)
	}
}
