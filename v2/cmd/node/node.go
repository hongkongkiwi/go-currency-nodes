package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gofrs/uuid/v5"
	pb "github.com/hongkongkiwi/go-currency-nodes/v2/gen/pb"
	"github.com/tebeka/atexit"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	configs "github.com/hongkongkiwi/go-currency-nodes/v2/internal/configs"
	nodeCmds "github.com/hongkongkiwi/go-currency-nodes/v2/internal/node_cmds"
)

var ticker *backoff.Ticker

var seededRand *rand.Rand

func nodeLocalVersion() error {
	fmt.Printf("Node Version: %s", configs.NodeAppVersion)
	return nil
}

func handleArgs(cCtx *cli.Context) error {
	var err error
	if configs.NodeCfg, err = configs.NewNodeCfgFromArgs(cCtx); err != nil {
		return err
	}
	return nil
}

func nodeStart() error {
	// var err error
	// nodeUuid, err = uuid.NewV4()
	// if err != nil {
	// 	atexit.Fatalln(err)
	// }
	// configs.NodeCfg.SetUUID(&nodeUuid)
	// configs.NodeCfg.ControllerServers = [1]string{fmt.Sprintf("%s:%d", controllerAddr, controllerPort)}

	// Use backoff so we can have a configurable stratergy later for reconnections
	// For this demo just use a constant backoff
	b := backoff.NewConstantBackOff(4 * time.Second)
	for {
		if ticker == nil {
			ticker = backoff.NewTicker(b)
		}
		<-ticker.C
		// Attempt to connect
		n := seededRand.Int() % len(configs.NodeCfg.ControllerServers)
		connectController(configs.NodeCfg.ControllerServers[n])
	}
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

func connectController(controllerServer string) {
	conn, err := grpc.Dial(
		controllerServer,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// grpc.WithDefaultServiceConfig(retryPolicy),
	)
	if err != nil {
		log.Printf("ERROR can not connect with server %v", err)
		if conn != nil {
			conn.Close()
		}
		return
	}

	// create stream
	client := pb.NewNodeCmdStreamClient(conn)
	stream, err := client.NodeCommandStream(context.Background())
	if err != nil {
		log.Printf("ERROR openn stream error %v", err)
		// Probably unnecessary
		if conn != nil {
			conn.Close()
		}
		return
	}

	// We are connected close the ticker
	if ticker != nil {
		ticker.Stop()
		ticker = nil
	}

	// Do so pre-stream setup
	nodeCmds.HandleControllerConnected(controllerServer)

	ctx := stream.Context()
	sendChanDone := make(chan bool)
	closeChan := make(chan bool)
	exitChan := make(chan bool)
	nodeCmds.SendChan = make(chan *pb.StreamFromNode)

	// Thread for sending commands
	go func() {
		for {
			select {
			case <-sendChanDone:
				if err := stream.CloseSend(); err != nil {
					log.Println(err)
				}
				return
			case command := <-nodeCmds.SendChan:
				if err := stream.Send(command); err != nil {
					log.Printf("can not send %v", err)
					continue
				}
				// log.Printf("command sent: %v", command.Response)
			}
		}
	}()

	// Thread for reciving commands
	go func() {
		for {
			in, err := stream.Recv()
			if err != nil {
				switch err {
				case io.EOF:
					close(closeChan)
				default:
					close(closeChan)
				}
				return
			}
			var fromUuid uuid.UUID
			if in.FromUuid != nil {
				if fromUuid, err = uuid.FromString(*in.FromUuid); err != nil {
					continue
				}
			}
			nodeCmds.HandleIncomingCommand(&fromUuid, in.Command)
		}
	}()

	// Closes exit channel if context is done
	go func() {
		<-ctx.Done()
		if err := contextError(ctx); err != nil {
			log.Printf("Context Error: %v", err)
		}
		close(sendChanDone)
		close(exitChan)
		nodeCmds.HandleControllerDisconnected()
	}()

	nodeCmds.HandleControllerReady()
	<-closeChan
	if conn != nil {
		conn.Close()
	}
	<-exitChan
	log.Printf("node exited\n")
}

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    configs.ArgNodeFlagControllers,
				Value:   cli.NewStringSlice(configs.DefaultNodeConnectController),
				EnvVars: configs.ArgEnvNodeControllers[:],
				Usage:   "one or more controller addresses to connect to",
			},
			&cli.StringSliceFlag{
				Name:    configs.ArgNodeKnownCurrencyPairs,
				Value:   cli.NewStringSlice(configs.DefaultNodeKnownCurrencyPairs...),
				EnvVars: configs.ArgEnvNodeKnownCurrencyPairs[:],
				Usage:   "one or more currency pairs that this node supports",
			},
			&cli.BoolFlag{
				Name:    configs.ArgNodeFlagVerbose,
				Value:   false,
				EnvVars: configs.ArgEnvNodeVerbose[:],
				Usage:   "turn on verbose logging",
			},
			&cli.StringFlag{
				Name:    configs.ArgNodeFlagName,
				EnvVars: configs.ArgEnvNodeName[:],
				Usage:   "name for this node",
			},
			&cli.StringFlag{
				Name:    configs.ArgNodeFlagUuid,
				EnvVars: configs.ArgEnvNodeUuid[:],
				Usage:   "UUID for this node",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "display cli app version",
				Action: func(cCtx *cli.Context) error {
					return nodeLocalVersion()
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
					return nodeStart()
				},
			},
		},
	}

	seededRand = rand.New(rand.NewSource(time.Now().UnixNano()))
	nodeCmds.SeededRand = seededRand
	if err := app.Run(os.Args); err != nil {
		atexit.Fatalf("ERROR: %v", err)
	}
}
