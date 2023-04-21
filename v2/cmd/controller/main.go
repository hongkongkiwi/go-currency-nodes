package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/gofrs/uuid/v5"
	pb "github.com/hongkongkiwi/go-currency-nodes/v2/gen/pb"
	stores "github.com/hongkongkiwi/go-currency-nodes/v2/internal/stores"
	"github.com/tebeka/atexit"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	// "github.com/tebeka/atexit"
)

const listenAddr = "127.0.0.1"
const listenPort = 5160

var nodeChannelStore *stores.InMemoryUUIDStore
var cliChannelStore *stores.InMemoryUUIDStore

type ChannelCommandWrapper struct {
	Cmd *anypb.Any
}

type NodeCmdStreamServer struct {
	pb.NodeCmdStreamServer
}

type CliCmdStreamServer struct {
	pb.CliCmdStreamServer
}

// Every incoming message has a UUID so store it alongside our send channel
func StoreChannel(uuidStr string, channel chan interface{}) (*uuid.UUID, error) {
	uuid, err := uuid.FromString(uuidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid cannot create channel")
	}
	if channel == nil {
		return nil, fmt.Errorf("invalid channel passed")
	}
	nodeChannelStore.SetWithUUID(&uuid, channel)
	return &uuid, nil
}

// Function to send message over our node streaming channel
func sendToNodes(toUUIDs []*uuid.UUID, fromUUID *uuid.UUID, command *anypb.Any) error {
	if len(toUUIDs) == 0 {
		return fmt.Errorf("must pass atleast one node uuid to send to")
	}
	if fromUUID == nil {
		return fmt.Errorf("must pass from uuid")
	}
	for _, UUID := range toUUIDs {
		if channel, _ := nodeChannelStore.GetFromUUID(UUID); channel != nil {
			// Push our command to the appropriate channel
			channel <- &pb.StreamToNode{
				FromUuid: fromUUID.String(),
				Command:  command,
			}
		} else {
			log.Printf("WARNING: We do not know about UUID: %s ignoring", UUID)
		}
	}
	return nil
}

// Function to send message over our cli streaming channel
func sendToCli(toUUID *uuid.UUID, response *anypb.Any) error {
	if toUUID == nil {
		return fmt.Errorf("must pass cli uuid to send to")
	}
	if channel, _ := cliChannelStore.GetFromUUID(toUUID); channel != nil {
		// Push our command to the appropriate channel
		channel <- &pb.StreamToCli{
			Response: response,
		}
	} else {
		log.Printf("WARNING: We do not know about UUID: %s ignoring", toUUID)
	}
	return nil
}

func NodeAppVersionReq(toUUIDs []*uuid.UUID, fromUUID *uuid.UUID) error {
	log.Printf("NodeAppVersionReq: %v", toUUIDs)
	genericPb, err := anypb.New(&pb.NodeAppVersionReq{})
	if err != nil {
		return fmt.Errorf("unknown Protobuf Struct %v", err)
	}
	sendToNodes(toUUIDs, fromUUID, genericPb)
	return nil
}

func NodeAppVersionReply(appVersionReply *pb.NodeAppVersionReply) error {
	log.Printf("NodeAppVersionReply: %v", appVersionReply)
	return nil
}

func NodeStatusReq(toUUIDs []*uuid.UUID, fromUUID *uuid.UUID) error {
	log.Printf("NodeStatusReq: %v", toUUIDs)
	genericPb, err := anypb.New(&pb.NodeAppVersionReq{})
	if err != nil {
		return fmt.Errorf("unknown Protobuf Struct %v", err)
	}
	sendToNodes(toUUIDs, fromUUID, genericPb)
	return nil
}

func NodeStatusReply(statusReply *pb.NodeStatusReply) error {
	log.Printf("NodeStatusReply: %v", statusReply)
	return nil
}

func NodeUUIDReq(toUUIDs []*uuid.UUID, fromUUID *uuid.UUID) error {
	log.Printf("NodeUUIDReq: %v", toUUIDs)
	genericPb, err := anypb.New(&pb.NodeAppVersionReq{})
	if err != nil {
		return fmt.Errorf("unknown Protobuf Struct %v", err)
	}
	sendToNodes(toUUIDs, fromUUID, genericPb)
	return nil
}

func NodeUUIDReply(uuidReply *pb.NodeUUIDReply) error {
	log.Printf("NodeUUIDReply: %v", uuidReply)
	return nil
}

func ControllerListNodesReq(cliUuid *uuid.UUID) error {
	log.Printf("ControllerListNodes")
	genericPb, err := anypb.New(&pb.ControllerListNodesReply{
		NodeUuids: nodeChannelStore.KeysAsString(),
	})
	if err != nil {
		return err
	}
	return sendToCli(cliUuid, genericPb)
}

func ControllerKickNodesReq(cliUuid *uuid.UUID, nodeUuids []*uuid.UUID) error {
	log.Printf("ControllerKickNodes: %v", nodeUuids)
	genericPb, _ := anypb.New(&pb.ControllerKickNodesReq{
		NodeUuids: stringsFromUuids(nodeUuids),
	})
	return sendToCli(cliUuid, genericPb)
}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return logError(status.Error(codes.Canceled, "request is canceled"))
	case context.DeadlineExceeded:
		return logError(status.Error(codes.DeadlineExceeded, "deadline is exceeded"))
	default:
		return nil
	}
}

func logError(err error) error {
	if err != nil {
		log.Print(err)
	}
	return err
}

// Helper method to generate an array of valid uuids
func uuidsFromStrings(uuidStrings []string) ([]*uuid.UUID, []error) {
	// Use a list so we can deduplicate any duplicates
	uuidsMap := make(map[string]*uuid.UUID, len(uuidStrings))
	errors := make([]error, 0)
	for _, uuidString := range uuidStrings {
		uuid, err := uuid.FromString(uuidString)
		if err != nil {
			errors = append(errors, fmt.Errorf("invalid uuid %s", uuidString))
			continue
		}
		uuidsMap[uuidString] = &uuid
	}
	// Build our array for returning
	uuidsArr := make([]*uuid.UUID, len(uuidStrings))
	i := 0
	for _, uuid := range uuidsMap {
		uuidsArr[i] = uuid
		i++
	}
	return uuidsArr, errors
}

func stringsFromUuids(uuids []*uuid.UUID) []string {
	// Use a list so we can deduplicate any duplicates
	uuidsMap := make(map[string]string, len(uuids))
	for _, uuid := range uuids {
		uuidsMap[uuid.String()] = uuid.String()
	}
	// Build our array for returning
	uuidsArr := make([]string, len(uuids))
	i := 0
	for _, uuid := range uuidsMap {
		uuidsArr[i] = uuid
		i++
	}
	return uuidsArr
}

func sendNodeThread(stream pb.NodeCmdStream_NodeCommandStreamServer, sendChan <-chan interface{}, cancelChan <-chan bool) {
	// We wrap this in a select statement so we don't block reading from channel
	for {
		select {
		case genericCmd := <-sendChan:
			if nodeCmd, ok := genericCmd.(*pb.StreamToNode); ok {
				if err := stream.Send(nodeCmd); err != nil {
					log.Printf("send node command error %v", err)
				}
				log.Printf("sent node command: %v", nodeCmd)
			}
		case <-cancelChan:
			return
		}
	}
}

func sendCliThread(stream pb.CliCmdStream_CliCommandStreamServer, sendChan <-chan interface{}, cancelChan <-chan bool) {
	// We wrap this in a select statement so we don't block reading from channel
	for {
		select {
		case genericCmd := <-sendChan:
			if cliCmd, ok := genericCmd.(*pb.StreamToCli); ok {
				if err := stream.Send(cliCmd); err != nil {
					log.Printf("send cli command error %v", err)
				}
				log.Printf("sent cli command: %v", cliCmd)
			}
		case <-cancelChan:
			return
		}
	}
}

func (srv *CliCmdStreamServer) CliCommandStream(stream pb.CliCmdStream_CliCommandStreamServer) error {
	log.Println("cli stream connection")
	cancelChan := make(chan bool)
	sendChan := make(chan interface{})
	// Process outgoing commands
	go sendCliThread(stream, sendChan, cancelChan)

	var cliUuid *uuid.UUID
	// Process responses
	for {
		err := contextError(stream.Context())
		if err != nil {
			// Terminate the stream
			cancelChan <- true
			if cliUuid != nil {
				cliChannelStore.DeleteUUID(cliUuid)
			}
			return nil
		}
		// receive data from stream
		in, err := stream.Recv()
		if err == io.EOF {
			// return will close stream from server side
			log.Println("exit (end of stream)")
			cancelChan <- true
			if cliUuid != nil {
				cliChannelStore.DeleteUUID(cliUuid)
			}
			return nil
		}
		if err != nil {
			log.Printf("receive error %v", err)
			continue
		}
		var setupErr error
		cliUuid, setupErr = StoreChannel(in.CliUuid, sendChan)
		if setupErr != nil {
			log.Printf("setup error %v", setupErr)
			cancelChan <- true
			if cliUuid != nil {
				cliChannelStore.DeleteUUID(cliUuid)
			}
			// Terminate stream
			return nil
		}
		if in.NodeCommand != nil {
			sendToUUIDs, errs := uuidsFromStrings(in.NodeUuids)
			if len(errs) > 0 {
				for err := range errs {
					log.Printf("Cli UUID Error: %v", err)
				}
			}
			handleCliCommand(in.NodeCommand, cliUuid, sendToUUIDs)
		}
	}
}

func handleCliCommand(genericCmd *anypb.Any, cliUuid *uuid.UUID, sendToUUIDs []*uuid.UUID) {
	// In this case, incoming messages are actually a reply
	// (excluding initial command) so try to unmarshel it
	cmd, unmrashalErr := genericCmd.UnmarshalNew()
	if unmrashalErr != nil {
		// We couldn't unmarshel a proper PB object
		log.Printf("Cli Node Command Error: %v", unmrashalErr)
		return
	}
	switch cmd := cmd.(type) {
	case *pb.ControllerListNodesReq:
		log.Printf("Cli Requested ControllerListNodes")
		err := ControllerListNodesReq(cliUuid)
		if err != nil {
			log.Printf("ControllerListNodes Error: %v", err)
		}
	case *pb.ControllerKickNodesReq:
		log.Printf("Cli Requested ControllerKickNode")
		nodeUuids, _ := uuidsFromStrings(cmd.NodeUuids)
		err := ControllerKickNodesReq(cliUuid, nodeUuids)
		if err != nil {
			log.Printf("ControllerKickNode Error: %v", err)
		}
	case *pb.NodeAppVersionReq:
		log.Printf("Cli Requested NodeAppVersion")
		err := NodeAppVersionReq(sendToUUIDs, cliUuid)
		if err != nil {
			log.Printf("NodeAppVersionReq Error: %v", err)
		}
	case *pb.NodeStatusReq:
		log.Printf("Cli Requested NodeStatusReq")
		err := NodeStatusReq(sendToUUIDs, cliUuid)
		if err != nil {
			log.Printf("NodeStatusReq Error: %v", err)
		}
	case *pb.NodeUUIDReq:
		log.Printf("Cli Requested NodeUUIDReq")
		err := NodeStatusReq(sendToUUIDs, cliUuid)
		if err != nil {
			log.Printf("NodeUUIDReq Error: %v", err)
		}
	default:
		log.Printf("Unhandled Command: %v", cmd)
	}
}

func (srv *NodeCmdStreamServer) NodeCommandStream(stream pb.NodeCmdStream_NodeCommandStreamServer) error {
	sendChan := make(chan interface{})
	cancelChan := make(chan bool)
	go sendNodeThread(stream, sendChan, cancelChan)

	timerChan := time.After(5 * time.Second)

	go func() {
		for {
			<-timerChan
			genericCmd, _ := anypb.New(&pb.NodeAppVersionReq{})
			from, _ := uuid.NewV4()
			sendChan <- &pb.StreamToNode{
				FromUuid: from.String(),
				Command:  genericCmd,
			}
			timerChan = time.After(5 * time.Second)
		}
	}()

	var nodeUuid *uuid.UUID
	for {
		err := contextError(stream.Context())
		if err != nil {
			cancelChan <- true
			if nodeUuid != nil {
				nodeChannelStore.DeleteUUID(nodeUuid)
			}
			// Terminate the stream
			return nil
		}
		// receive data from stream
		in, err := stream.Recv()
		if err == io.EOF {
			// return will close stream from server side
			log.Println("exit (end of stream)")
			cancelChan <- true
			if nodeUuid != nil {
				nodeChannelStore.DeleteUUID(nodeUuid)
			}
			return nil
		}
		if err != nil {
			log.Printf("receive error %v", err)
			continue
		}

		var setupErr error
		nodeUuid, setupErr = StoreChannel(in.NodeUuid, sendChan)
		if setupErr != nil {
			log.Printf("setup error %v", setupErr)
			cancelChan <- true
			if nodeUuid != nil {
				nodeChannelStore.DeleteUUID(nodeUuid)
			}
			// Terminate stream
			return nil
		}

		var fromUuid uuid.UUID
		if in.FromUuid != nil {
			var err error
			fromUuid, err = uuid.FromString(*in.FromUuid)
			if err != nil {
				log.Printf("node sent invalid uuid: %s", *in.FromUuid)
				continue
			}
		}
		if in.Response == nil {
			log.Printf("Node %s opened stream", in.NodeUuid)
			continue
		}
		log.Printf("Node sent %s", in)
		if in.FromUuid != nil {
			cliSendChannel, _ := cliChannelStore.GetFromUUID(&fromUuid)
			if cliSendChannel != nil {
				cliSendChannel <- &pb.StreamToCli{
					NodeUuid: &in.NodeUuid,
					Response: in.Response,
				}
			}
		}
	}
}

func listen(addr string, port int) error {
	// create listener
	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", listenAddr, listenPort))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	// Init our internal channel stores
	nodeChannelStore = stores.NewInMemoryUUIDStore()
	cliChannelStore = stores.NewInMemoryUUIDStore()

	// create grpc server
	srv := grpc.NewServer()
	pb.RegisterNodeCmdStreamServer(srv, &NodeCmdStreamServer{})
	//pb.RegisterCliCmdStreamServer(srv, &CliCmdStreamServer{})

	log.Printf("Controller listening on %s:%d", listenAddr, listenPort)
	if err := srv.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve controller: %v", err)
	}
	return nil
}

func main() {
	if err := listen(listenAddr, listenPort); err != nil {
		atexit.Fatal(err)
	}
}
