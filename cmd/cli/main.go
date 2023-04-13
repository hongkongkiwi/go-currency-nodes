package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2" // imports as package "cli"
)

const cli_version = "0.0.1"

func nodeVersion() error {
	fmt.Println("nodeVersion: NOT IMPLEMENTED")
	return nil
}

func nodeUuid() error {
	fmt.Println("nodeUuid: NOT IMPLEMENTED")
	return nil
}

func nodeStatus() error {
	fmt.Println("nodeStatus: NOT IMPLEMENTED")
	return nil
}

func nodeSubscriptions() error {
	fmt.Println("nodeSubscriptions: NOT IMPLEMENTED")
	return nil
}

func nodeSubscriptionsSync() error {
	fmt.Println("nodeSubscriptionsSync: NOT IMPLEMENTED")
	return nil
}

func nodePriceRefresh() error {
	fmt.Println("nodePriceRefresh: NOT IMPLEMENTED")
	return nil
}

func nodeLog() error {
	fmt.Println("nodeLog: NOT IMPLEMENTED")
	return nil
}

func nodeUpdatesPause() error {
	fmt.Println("nodeUpdatesPause: NOT IMPLEMENTED")
	return nil
}

func nodeUpdatesResume() error {
	fmt.Println("nodeUpdatesResume: NOT IMPLEMENTED")
	return nil
}

func nodeControllerConnect() error {
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

func cliConnectNode() error {
	fmt.Println("cliConnectNode: NOT IMPLEMENTED")
	return nil
}

func cliDisconnectNode() error {
	fmt.Println("cliDisconnectNode: NOT IMPLEMENTED")
	return nil
}

func main() {
	app := &cli.App{
		Commands: []*cli.Command{
			{
				Name:    "connect",
				Aliases: []string{"c"},
				Usage:   "connect to node",
				Action: func(cCtx *cli.Context) error {
					return cliConnectNode()
				},
			},
			{
				Name:    "disconnect",
				Aliases: []string{"d"},
				Usage:   "disconnect from node",
				Action: func(cCtx *cli.Context) error {
					return cliDisconnectNode()
				},
			},
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
									return nodeVersion()
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
									return nodeControllerConnect()
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
