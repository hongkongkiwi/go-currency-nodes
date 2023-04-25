package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gofrs/uuid/v5"
	pb "github.com/hongkongkiwi/go-currency-nodes/v2/gen/pb"
	"github.com/tebeka/atexit"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	cliCmds "github.com/hongkongkiwi/go-currency-nodes/v2/internal/cli_cmds"
	"github.com/hongkongkiwi/go-currency-nodes/v2/internal/configs"
	"github.com/hongkongkiwi/go-currency-nodes/v2/internal/helpers"
)

const defaultControllerAddr = "127.0.0.1"
const defaultControllerPort = 5160
const appVersion = "0.0.1"

func connectController() {
	conn, err := grpc.Dial(
		configs.CliCfg.ControllerServer,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Printf("ERROR can not connect with server %v", err)
		return
	}

	// create stream
	client := pb.NewCliCmdStreamClient(conn)
	stream, err := client.CliCommandStream(context.Background())
	if err != nil {
		log.Printf("ERROR openn stream error %v", err)
		return
	}

	ctx := stream.Context()

	log.Printf("Connected to Controller %s", configs.CliCfg.ControllerServer)

	// Thread for sending hello command
	go func() {
		for {
			select {
			case <-cliCmds.CancelChan:
				return
			case command := <-cliCmds.SendChan:
				if err := stream.Send(command); err != nil {
					log.Printf("can not send %v", err)
					cliCmds.CancelChan <- true
				}
				log.Printf("command sent: %v", command)
			}
		}
		// if err := stream.CloseSend(); err != nil {
		// 	log.Println(err)
		// }
	}()

	// Thread for reciving commands
	// if stream is finished it closes done channel
	go func() {
		for {
			in, err := stream.Recv()
			if err == io.EOF {
				if cliCmds.CancelChan != nil {
					cliCmds.CancelChan <- true
					close(cliCmds.CancelChan)
				}
				return
			}
			if err != nil {
				log.Printf("can not receive %v", err)
				cliCmds.CancelChan <- true
				return
			}
			cliCmds.HandleIncomingReply(in)
		}
	}()

	// third goroutine closes done channel
	// if context is done
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Println(err)
		}
		if cliCmds.CancelChan != nil {
			cliCmds.CancelChan <- true
			close(cliCmds.CancelChan)
		}
	}()

	<-cliCmds.CancelChan
}

func cliVersion() error {
	fmt.Println("cli version:", appVersion)
	return nil
}

func clientArgs(cCtx *cli.Context) error {
	configs.CliCfg.VerboseLog = cCtx.Bool("verbose")
	uuidStr := cCtx.String("uuid")
	cliUuid, err := uuid.FromString(uuidStr)
	if err != nil {
		if configs.CliCfg.VerboseLog {
			log.Printf("WARNING: Missing or invalid Cli UUID. We will generate a random one")
		}
		cliUuid, _ = uuid.NewV4()
	}
	configs.CliCfg.AppVersion = appVersion
	configs.CliCfg.CliUUID = &cliUuid
	configs.CliCfg.ControllerServer = cCtx.String("controller")
	if configs.CliCfg.VerboseLog {
		log.Printf("config cli uuid: %s", configs.CliCfg.CliUUID)
		log.Printf("config controller addr: %s", configs.CliCfg.ControllerServer)
	}
	return nil
}

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "controller",
				Value:   fmt.Sprintf("%s:%d", defaultControllerAddr, defaultControllerPort),
				EnvVars: []string{"CLI_CONTROLLER_ADDRESS"},
				Usage:   "controller address to connect to",
			},
			&cli.BoolFlag{
				Name:    "verbose",
				Value:   false,
				EnvVars: []string{"CLI_VERBOSE"},
				Usage:   "turn on verbose logging",
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "display cli app version",
				Action: func(cCtx *cli.Context) error {
					return cliVersion()
				},
			},
			{
				Name:    "controller",
				Aliases: []string{"n"},
				Usage:   "options for controller control",
				Subcommands: []*cli.Command{
					{
						Name:    "nodes",
						Aliases: []string{"s"},
						Usage:   "show all nodes connected",
						Action: func(cCtx *cli.Context) error {
							if err := clientArgs(cCtx); err != nil {
								return err
							}
							cliCmds.SendChan = make(chan *pb.StreamFromCli)
							cliCmds.CancelChan = make(chan bool)
							// Queue up our command
							go cliCmds.ControllerListNodes()
							// Attempt to connect
							connectController()
							return nil
						},
					},
				},
			},
			{
				Name:    "node",
				Aliases: []string{"n"},
				Usage:   "options for node control",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:     "node-uuid",
						Required: true,
						EnvVars:  []string{"CLI_NODE_UUIDS"},
						Usage:    "node uuid to send command to",
					},
				},
				Subcommands: []*cli.Command{
					{
						Name:    "status",
						Aliases: []string{"s"},
						Usage:   "show the node status",
						Action: func(cCtx *cli.Context) error {
							if err := clientArgs(cCtx); err != nil {
								return err
							}
							nodeUuidsArg := cCtx.StringSlice("node-uuid")
							ndoeUuids, errors := helpers.UuidsFromStrings(nodeUuidsArg)
							if len(errors) > 0 {
								return fmt.Errorf("errors with the passed node UUIDs %v", errors)
							}
							cliCmds.SendChan = make(chan *pb.StreamFromCli)
							// Queue up our command
							go cliCmds.NodeStatus(ndoeUuids)
							// Attempt to connect
							connectController()
							return nil
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		atexit.Fatalf("ERROR: %v", err)
	}
}
