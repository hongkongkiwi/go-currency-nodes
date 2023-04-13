package main

import (
	"fmt"
	"log"
	"os"

	cliClient "github.com/hongkongkiwi/go-currency-nodes/internal/cli_client"
	"github.com/urfave/cli/v2" // imports as package "cli"
)

const cli_version = "0.0.1"

func nodeVersion(nodeAddr string) error {
	return cliClient.NodeUUID()
	return nil
}

func nodeUuid() error {
	return cliClient.NodeUUID()
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

func nodeControllerConnect(controllerAddress string) error {
	fmt.Println("nodeControllerConnect: NOT IMPLEMENTED")
	return nil
}

func nodeControllerDisconnect() error {
	fmt.Println("nodeControllerDisconnect: NOT IMPLEMENTED")
	return nil
}

func nodeKill() error {
	fmt.Println("nodeKill: NOT IMPLEMENTED")
	return nil
}

func cliVersion() error {
	fmt.Println("version:", cli_version)
	return nil
}

func fallbackEnv(argValue, envKey string) string {
	if argValue == "" {
		if envValue, envOk := os.LookupEnv(envKey); envOk && envValue != "" {
			return envValue
		}
	}
	return argValue
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
						Usage: "address of the node to connect to [NODE_REMOTE_ADDR]",
					},
				},
				Subcommands: []*cli.Command{
					{
						Name:    "uuid",
						Aliases: []string{"u"},
						Usage:   "show the node app uuid",
						Action: func(cCtx *cli.Context) error {
							cliClient.NodeAddr = fallbackEnv(cCtx.String("addr"), "NODE_REMOTE_ADDR")
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
