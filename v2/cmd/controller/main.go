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
	controllerCmds "github.com/hongkongkiwi/go-currency-nodes/v2/internal/controller_cmds"
	helpers "github.com/hongkongkiwi/go-currency-nodes/v2/internal/helpers"
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
func storeNodeChannels(uuidStr string, sendChannel chan interface{}, cancelChannel chan bool) (*uuid.UUID, error) {
	uuid, err := uuid.FromString(uuidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid cannot create channel")
	}
	if cancelChannel == nil {
		return nil, fmt.Errorf("invalid cancelChannel passed")
	}
	stores.NodeCancelChannelStore.SetWithUUID(&uuid, cancelChannel)
	if sendChannel == nil {
		return nil, fmt.Errorf("invalid sendChannel passed")
	}
	stores.NodeSendChannelStore.SetWithUUID(&uuid, sendChannel)
	return &uuid, nil
}

func storeCliChannel(uuidStr string, channel chan interface{}) (*uuid.UUID, error) {
	uuid, err := uuid.FromString(uuidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid uuid cannot create channel")
	}
	if channel == nil {
		return nil, fmt.Errorf("invalid channel passed")
	}
	stores.CliChannelStore.SetWithUUID(&uuid, channel)
	return &uuid, nil
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
				stores.CliChannelStore.DeleteUUID(cliUuid)
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
				stores.CliChannelStore.DeleteUUID(cliUuid)
			}
			return nil
		}
		if err != nil {
			log.Printf("receive error %v", err)
			continue
		}
		var setupErr error
		cliUuid, setupErr = storeCliChannel(in.CliUuid, sendChan)
		if setupErr != nil {
			log.Printf("setup error %v", setupErr)
			cancelChan <- true
			if cliUuid != nil {
				stores.CliChannelStore.DeleteUUID(cliUuid)
			}
			// Terminate stream
			return nil
		}
		if in.NodeCommand != nil {
			sendToUUIDs, errs := helpers.UuidsFromStrings(in.NodeUuids)
			if len(errs) > 0 {
				for err := range errs {
					log.Printf("Cli UUID Error: %v", err)
				}
			}
			controllerCmds.HandleCliCommand(in.NodeCommand, cliUuid, sendToUUIDs)
		}
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
				stores.NodeSendChannelStore.DeleteUUID(nodeUuid)
				stores.NodeCancelChannelStore.DeleteUUID(nodeUuid)
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
				stores.NodeSendChannelStore.DeleteUUID(nodeUuid)
				stores.NodeCancelChannelStore.DeleteUUID(nodeUuid)
			}
			return nil
		}
		if err != nil {
			log.Printf("receive error %v", err)
			continue
		}

		var setupErr error
		nodeUuid, setupErr = storeNodeChannels(in.NodeUuid, sendChan, cancelChan)
		if setupErr != nil {
			log.Printf("setup error %v", setupErr)
			cancelChan <- true
			if nodeUuid != nil {
				stores.NodeSendChannelStore.DeleteUUID(nodeUuid)
				stores.NodeCancelChannelStore.DeleteUUID(nodeUuid)
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
			cliSendChannel, _ := stores.CliChannelStore.GetFromUUID(&fromUuid)
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
	stores.NodeSendChannelStore = stores.NewInMemoryUUIDChannelInterfaceStore()
	stores.NodeCancelChannelStore = stores.NewInMemoryUUIDChannelBoolStore()
	stores.CliChannelStore = stores.NewInMemoryUUIDChannelInterfaceStore()
	controllerCmds.CliReplyCount = stores.NewCliReplyCount()

	// create grpc server
	srv := grpc.NewServer()
	pb.RegisterNodeCmdStreamServer(srv, &NodeCmdStreamServer{})
	pb.RegisterCliCmdStreamServer(srv, &CliCmdStreamServer{})

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
