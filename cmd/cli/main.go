package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/urfave/cli/v2" // imports as package "cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const cli_version = "0.0.1"

var conn *grpc.ClientConn

func nodeVersion(nodeAddr string) error {
	connectErr := connect(nodeAddr)
	if connectErr != nil {
		return connectErr
	}
	c := pb.NewGreeterClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Millisecond*80))
	defer cancel()

	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: *name})
	if err != nil {
		return fmt.Errorf("could not greet: %v", err)
	}

	return nil
}

func nodeUuid() error {
	log.Println("nodeUuid: NOT IMPLEMENTED")
	return nil
}

func nodeStatus() error {
	log.Println("nodeStatus: NOT IMPLEMENTED")
	return nil
}

func nodeSubscriptions() error {
	fmt.Println("nodeSubscriptions: NOT IMPLEMENTED")
	return nil
}

func nodeSubscriptionsSync() error {
	log.Println("nodeSubscriptionsSync: NOT IMPLEMENTED")
	return nil
}

func nodePriceRefresh() error {
	log.Println("nodePriceRefresh: NOT IMPLEMENTED")
	return nil
}

func nodeLog() error {
	log.Println("nodeLog: NOT IMPLEMENTED")
	return nil
}

func nodeUpdatesPause() error {
	log.Println("nodeUpdatesPause: NOT IMPLEMENTED")
	return nil
}

func nodeUpdatesResume() error {
	log.Println("nodeUpdatesResume: NOT IMPLEMENTED")
	return nil
}

func nodeControllerConnect(nodeAddress string) error {
	return nil
}

func nodeControllerDisconnect() error {
	return nil
}

func nodeKill() error {
	fmt.Println("nodeKill: NOT IMPLEMENTED")
	return nil
}

func disconnect() error {
	return conn.Close()
}

func connect(nodeAddress string) error {
	// Set up a connection to the server.
	var err error
	conn, err = grpc.Dial(nodeAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	return nil
}

func cliVersion() error {
	fmt.Println("version:", cli_version)
	return nil
}

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "display cli version",
				Action: func(cCtx *cli.Context) error {
					return cliVersion()
				},
			},
			{
				Name:    "node",
				Aliases: []string{"n"},
				Usage:   "options for node control",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "addr",
						Value: "127.0.0.1:5051",
						Usage: "address of the node",
					},
				},
				Subcommands: []*cli.Command{
					{
						Name:    "uuid",
						Aliases: []string{"u"},
						Usage:   "show the node app uuid",
						Action: func(cCtx *cli.Context) error {
							return nodeUuid()
						},
					},
					{
						Name:    "status",
						Aliases: []string{"s"},
						Usage:   "show the node status",
						Action: func(cCtx *cli.Context) error {
							return nodeStatus()
						},
					},
					{
						Name:    "app",
						Aliases: []string{"a"},
						Usage:   "options for node app",
						Subcommands: []*cli.Command{
							{
								Name:    "version",
								Aliases: []string{"v"},
								Usage:   "show the node app version",
								Action: func(cCtx *cli.Context) error {
									nodeAddr := cCtx.String("addr")
									return nodeVersion(nodeAddr)
								},
							},
							{
								Name:    "log",
								Aliases: []string{"l"},
								Usage:   "show the node app log",
								Action: func(cCtx *cli.Context) error {
									return nodeLog()
								},
							},
							{
								Name:    "kill",
								Aliases: []string{"k"},
								Usage:   "kill the node app",
								Action: func(cCtx *cli.Context) error {
									return nodeKill()
								},
							},
						},
					},
					{
						Name:    "updatestream",
						Aliases: []string{"u"},
						Usage:   "updates to controller",
						Subcommands: []*cli.Command{
							{
								Name:    "pause",
								Aliases: []string{"p"},
								Usage:   "pause updating price updates to controller",
								Action: func(cCtx *cli.Context) error {
									return nodeUpdatesPause()
								},
							},
							{
								Name:    "resume",
								Aliases: []string{"r"},
								Usage:   "resume updating price updates to controller",
								Action: func(cCtx *cli.Context) error {
									return nodeUpdatesResume()
								},
							},
						},
					},
					{
						Name:    "subscriptions",
						Aliases: []string{"c"},
						Usage:   "options for node subscriptions",
						Subcommands: []*cli.Command{
							{
								Name:    "list",
								Aliases: []string{"l"},
								Usage:   "list all subscriptions (and known prices)",
								Action: func(cCtx *cli.Context) error {
									return nodeSubscriptions()
								},
							},
							{
								Name:    "sync",
								Aliases: []string{"s"},
								Usage:   "sync the server subscriptions with our config file",
								Action: func(cCtx *cli.Context) error {
									return nodeSubscriptionsSync()
								},
							},
							{
								Name:    "refresh",
								Aliases: []string{"s"},
								Usage:   "get the latest prices from server",
								Action: func(cCtx *cli.Context) error {
									return nodePriceRefresh()
								},
							},
						},
					},
					{
						Name:    "controller",
						Aliases: []string{"c"},
						Usage:   "options for node control",
						Subcommands: []*cli.Command{
							{
								Name:    "connect",
								Aliases: []string{"c"},
								Usage:   "ask node to connect to controller",
								Action: func(cCtx *cli.Context) error {
									return nodeControllerConnect("abc")
								},
							},
							{
								Name:    "disconnect",
								Aliases: []string{"d"},
								Usage:   "ask node to disconnect from controller",
								Action: func(cCtx *cli.Context) error {
									return nodeControllerDisconnect()
								},
							},
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
