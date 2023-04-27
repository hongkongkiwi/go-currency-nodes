package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"

	"github.com/gofrs/uuid/v5"
	pb "github.com/hongkongkiwi/go-currency-nodes/v2/gen/pb"
	"github.com/hongkongkiwi/go-currency-nodes/v2/internal/configs"
	controllerCmds "github.com/hongkongkiwi/go-currency-nodes/v2/internal/controller_cmds"
	helpers "github.com/hongkongkiwi/go-currency-nodes/v2/internal/helpers"
	stores "github.com/hongkongkiwi/go-currency-nodes/v2/internal/stores"
	"github.com/tebeka/atexit"
	"github.com/urfave/cli/v2"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	// "github.com/tebeka/atexit"
)

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
		return status.Error(codes.Canceled, "request is canceled")
	case context.DeadlineExceeded:
		return status.Error(codes.DeadlineExceeded, "deadline is exceeded")
	default:
		return nil
	}
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
		if in.Command != nil {
			sendToUUIDs, errs := helpers.UuidsFromStrings(in.NodeUuids)
			if len(errs) > 0 {
				for err := range errs {
					log.Printf("Cli UUID Error: %v", err)
				}
			}
			controllerCmds.HandleIncomingCliCommand(in.Command, cliUuid, sendToUUIDs)
		}
	}
}

func (srv *NodeCmdStreamServer) NodeCommandStream(stream pb.NodeCmdStream_NodeCommandStreamServer) error {
	sendChan := make(chan interface{})
	cancelChan := make(chan bool)

	go sendNodeThread(stream, sendChan, cancelChan)

	var nodeUuid *uuid.UUID
	for {
		// Handle Stream Error
		err := contextError(stream.Context())
		if err != nil {
			cancelChan <- true
			if nodeUuid != nil {
				stores.NodeSendChannelStore.DeleteUUID(nodeUuid)
				stores.NodeCancelChannelStore.DeleteUUID(nodeUuid)
				controllerCmds.HandleNodeDisconnected(nodeUuid)
			}
			// Terminate the stream
			return err
		}

		// receive data from stream
		in, err := stream.Recv()
		if err == io.EOF {
			// return will close stream from server side
			cancelChan <- true
			if nodeUuid != nil {
				stores.NodeSendChannelStore.DeleteUUID(nodeUuid)
				stores.NodeCancelChannelStore.DeleteUUID(nodeUuid)
				controllerCmds.HandleNodeDisconnected(nodeUuid)
			}
			return fmt.Errorf("exit (end of stream)")
		}
		if err != nil {
			continue
		}

		var setupErr error
		nodeUuid, setupErr = storeNodeChannels(in.NodeUuid, sendChan, cancelChan)
		if setupErr != nil {
			cancelChan <- true
			if nodeUuid != nil {
				stores.NodeSendChannelStore.DeleteUUID(nodeUuid)
				stores.NodeCancelChannelStore.DeleteUUID(nodeUuid)
				controllerCmds.HandleNodeDisconnected(nodeUuid)
			}
			// Terminate stream
			return fmt.Errorf("setup error %v", setupErr)
		}

		// Differentiate messages from node depending on if there is a fromUuid
		var fromUuid uuid.UUID
		if in.FromUuid != nil {
			var err error
			fromUuid, err = uuid.FromString(*in.FromUuid)
			if err != nil {
				log.Printf("WARN: node sent invalid uuid: %s", *in.FromUuid)
				continue
			}
		}
		if in.Response == nil {
			log.Printf("Node %s sent empty message", in.NodeUuid)
			continue
		}

		log.Printf("Node sent %s", in)
		if in.FromUuid == nil { // This is a command from Node to Controller
			controllerCmds.HandleIncomingNodeCommandToController(in.Response, nodeUuid)
		} else { // This Command is from Node to Cli so just forward
			cliSendChannel, _ := stores.CliChannelStore.GetFromUUID(&fromUuid)
			// Only send if we have a send channel to Cli
			if cliSendChannel != nil {
				cliSendChannel <- &pb.StreamToCli{
					NodeUuid: &in.NodeUuid,
					Response: in.Response,
				}
			}
		}
	}
}

func controllerVersion() error {
	fmt.Printf("Controller Version: %s", configs.ControllerAppVersion)
	return nil
}

func controllerStart(listenAddress string) error {
	// create listener
	lis, err := net.Listen("tcp", listenAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	// Init our internal channel stores
	stores.NodeSendChannelStore = stores.NewInMemoryUUIDChannelInterfaceStore()
	stores.NodeCancelChannelStore = stores.NewInMemoryUUIDChannelBoolStore()
	stores.CliChannelStore = stores.NewInMemoryUUIDChannelInterfaceStore()
	stores.ControllerNodeCurrencies = stores.NewNodeCurrenciesStore()
	stores.ControllerPriceStore = stores.NewPriceStore()

	// create grpc server
	srv := grpc.NewServer()
	pb.RegisterNodeCmdStreamServer(srv, &NodeCmdStreamServer{})
	pb.RegisterCliCmdStreamServer(srv, &CliCmdStreamServer{})

	log.Printf("Controller listening on %s", listenAddress)
	if err := srv.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve controller: %v", err)
	}
	return nil
}

func handleArgs(cCtx *cli.Context) error {
	var err error
	configs.ControllerCfg, err = configs.NewControllerCfgFromArgs(cCtx)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.UintFlag{
				Name:    configs.ArgControllerCncurrentNodePriceStreams,
				Value:   configs.DefaultControllerCncurrentNodePriceStreams,
				EnvVars: configs.ArgEnvControllerCncurrentNodePriceStreams[:],
				Usage:   "maximum number of automatic currency streams",
			},
			&cli.BoolFlag{
				Name:    configs.ArgControllerFlagVerbose,
				Value:   configs.DefaultControllerVerboseLog,
				EnvVars: configs.ArgEnvControllerVerbose[:],
				Usage:   "turn on verbose logging",
			},
			&cli.StringFlag{
				Name:    configs.ArgControllerFlagListenAddress,
				Value:   configs.DefaultControllerListenAddress,
				EnvVars: configs.ArgEnvControllerListenAddress[:],
				Usage:   "local address to listen on",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "display controller app version",
				Action: func(cCtx *cli.Context) error {
					return controllerVersion()
				},
			},
			{
				Name:    "start",
				Aliases: []string{"v"},
				Usage:   "start this node",
				Action: func(cCtx *cli.Context) error {
					if err := handleArgs(cCtx); err != nil {
						return err
					}
					return controllerStart(configs.ControllerCfg.ListenAddress)
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		atexit.Fatalf("ERROR: %v", err)
	}
}
