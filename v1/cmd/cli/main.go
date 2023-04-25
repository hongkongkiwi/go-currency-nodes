package main

import (
	"fmt"
	"log"
	"os"
	"time"

	cliClient "github.com/hongkongkiwi/go-currency-nodes/v1/internal/cli_client"
	"github.com/urfave/cli/v2" // imports as package "cli"
)

const cli_version = "0.0.1"
const defaultNodeAddr = "127.0.0.1:5051"
const defaultNodeRequestTimeout = time.Duration(time.Millisecond * 150)

func cliVersion() error {
	fmt.Println("cli version:", cli_version)
	return nil
}

func clientArgs(cCtx *cli.Context) {
	cliClient.NodeAddr = cCtx.String("addr")
	cliClient.NodeRequestTimeout = cCtx.Duration("timeout")
	cliClient.VerboseLog = cCtx.Bool("verbose")
	cliClient.QuietLog = cCtx.Bool("quiet")
	if cliClient.VerboseLog {
		log.Printf("config node remote addr: %s", cliClient.NodeAddr)
		log.Printf("config node remote timeout: %s", cliClient.NodeRequestTimeout)
	}
}

func main() {
	app := &cli.App{
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
				Name:    "node",
				Aliases: []string{"n"},
				Usage:   "options for node control",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "addr",
						Aliases: []string{"address", "remote-address"},
						Value:   defaultNodeAddr,
						EnvVars: []string{"CLI_NODE_REMOTE_ADDR"},
						Usage:   "address of the node to connect to",
					},
					&cli.DurationFlag{
						Name:    "timeout",
						Aliases: []string{"remote-timeout"},
						Value:   defaultNodeRequestTimeout,
						EnvVars: []string{"CLI_NODE_REMOTE_TIMEOUT"},
						Usage:   "timeout for calls to node",
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Value:   false,
						EnvVars: []string{"CLI_VERBOSE"},
						Usage:   "turn on verbose logging",
					},
					&cli.BoolFlag{
						Name:    "quiet",
						Value:   false,
						EnvVars: []string{"CLI_QUIET"},
						Usage:   "turn off all logging but reply data",
					},
				},
				Subcommands: []*cli.Command{
					{
						Name:    "uuid",
						Aliases: []string{"u"},
						Usage:   "show the node app uuid",
						Action: func(cCtx *cli.Context) error {
							clientArgs(cCtx)
							return cliClient.ClientNodeUUID()
						},
					},
					{
						Name:    "currencies",
						Aliases: []string{"c"},
						Usage:   "show the currencies this node knows about",
						Action: func(cCtx *cli.Context) error {
							clientArgs(cCtx)
							return cliClient.ClientNodeCurrencies()
						},
					},
					{
						Name:    "status",
						Aliases: []string{"s"},
						Usage:   "show the node status",
						Action: func(cCtx *cli.Context) error {
							clientArgs(cCtx)
							return cliClient.ClientNodeStatus()
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
									clientArgs(cCtx)
									return cliClient.ClientNodeAppVersion()
								},
							},
							{
								Name:    "kill",
								Aliases: []string{"k"},
								Usage:   "kill the node app",
								Action: func(cCtx *cli.Context) error {
									clientArgs(cCtx)
									return cliClient.ClientNodeAppKill()
								},
							},
						},
					},
					{
						Name:    "priceupdates",
						Aliases: []string{"u"},
						Usage:   "price",
						Subcommands: []*cli.Command{
							{
								Name:    "new",
								Aliases: []string{"n"},
								Flags: []cli.Flag{
									&cli.StringFlag{
										Name:     "currency-pair",
										Aliases:  []string{"pair"},
										Required: true,
										Usage:    "currency pair to set (must be in nodes configured pairs)",
									},
									&cli.Float64Flag{
										Name:     "price",
										Required: true,
										Usage:    "new price",
									},
								},
								Usage: "manually send a price",
								Action: func(cCtx *cli.Context) error {
									clientArgs(cCtx)
									return cliClient.ClientNodeManualPriceUpdate(cCtx.String("currency-pair"), cCtx.Float64("price"))
								},
							},
							{
								Name:    "pause",
								Aliases: []string{"p"},
								Usage:   "pause updating price updates to controller",
								Action: func(cCtx *cli.Context) error {
									clientArgs(cCtx)
									return cliClient.ClientNodeUpdatesPause()
								},
							},
							{
								Name:    "resume",
								Aliases: []string{"r"},
								Usage:   "resume updating price updates to controller",
								Action: func(cCtx *cli.Context) error {
									clientArgs(cCtx)
									return cliClient.ClientNodeUpdatesResume()
								},
							},
						},
					},
					{
						Name:    "subscriptions",
						Aliases: []string{"c"},
						Usage:   "list all subscriptions (and known prices)",
						Action: func(cCtx *cli.Context) error {
							clientArgs(cCtx)
							return cliClient.ClientNodeCurrencies()
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
