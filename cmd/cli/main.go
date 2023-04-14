package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	cliClient "github.com/hongkongkiwi/go-currency-nodes/internal/cli_client"
	"github.com/urfave/cli/v2" // imports as package "cli"
)

const cli_version = "0.0.1"
const defaultNodeAddr = "127.0.0.1:5051"
const defaultNodeRequestTimeout = uint64(time.Millisecond * 80)

func cliVersion() error {
	fmt.Println("version:", cli_version)
	return nil
}

// Passed arguments override environment variables
func fallbackStrEnv(argValue, envKey string) string {
	if argValue == "" {
		if envValue, envOk := os.LookupEnv(envKey); envOk && envValue != "" {
			return envValue
		}
	}
	return argValue
}

func fallbackUint64Env(argValue uint64, envKey string) uint64 {
	if argValue == 0 {
		if envValue, envOk := os.LookupEnv(envKey); envOk && envValue != "" {
			i, err := strconv.ParseUint(envValue, 10, 64)
			if err == nil {
				return i
			}
		}
	}
	return argValue
}

func handleArgs(cCtx *cli.Context) {
	cliClient.NodeAddr = fallbackStrEnv(cCtx.String("addr"), "NODE_REMOTE_ADDR")
	cliClient.NodeRequestTimeout = fallbackUint64Env(cCtx.Uint64("timeout"), "NODE_REMOTE_TIMEOUT")
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
						Name:  "addr",
						Value: defaultNodeAddr,
						Usage: "address of the node to connect to [NODE_REMOTE_ADDR]",
					},
					&cli.Uint64Flag{
						Name:  "timeout",
						Value: defaultNodeRequestTimeout,
						Usage: "timeout for calls to node [NODE_REMOTE_TIMEOUT]",
					},
				},
				Subcommands: []*cli.Command{
					{
						Name:    "uuid",
						Aliases: []string{"u"},
						Usage:   "show the node app uuid",
						Action: func(cCtx *cli.Context) error {
							handleArgs(cCtx)
							return cliClient.ClientNodeUUID()
						},
					},
					{
						Name:    "status",
						Aliases: []string{"s"},
						Usage:   "show the node status",
						Action: func(cCtx *cli.Context) error {
							handleArgs(cCtx)
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
									handleArgs(cCtx)
									return cliClient.ClientNodeAppVersion()
								},
							},
							{
								Name:    "log",
								Aliases: []string{"l"},
								Usage:   "show the node app log",
								Action: func(cCtx *cli.Context) error {
									handleArgs(cCtx)
									return cliClient.ClientNodeAppLog()
								},
							},
							{
								Name:    "kill",
								Aliases: []string{"k"},
								Usage:   "kill the node app",
								Action: func(cCtx *cli.Context) error {
									handleArgs(cCtx)
									return cliClient.ClientNodeAppKill()
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
									handleArgs(cCtx)
									return cliClient.ClientNodeUpdatesPause()
								},
							},
							{
								Name:    "resume",
								Aliases: []string{"r"},
								Usage:   "resume updating price updates to controller",
								Action: func(cCtx *cli.Context) error {
									handleArgs(cCtx)
									return cliClient.ClientNodeUpdatesResume()
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
									handleArgs(cCtx)
									return cliClient.ClientNodeCurrencies()
								},
							},
							{
								Name:    "sync",
								Aliases: []string{"s"},
								Usage:   "sync the server subscriptions with our config file",
								Action: func(cCtx *cli.Context) error {
									handleArgs(cCtx)
									return cliClient.ClientNodeCurrenciesRefreshCache()
								},
							},
							{
								Name:    "refresh",
								Aliases: []string{"s"},
								Usage:   "get the latest prices from server",
								Action: func(cCtx *cli.Context) error {
									handleArgs(cCtx)
									return cliClient.ClientNodeCurrenciesRefreshCache()
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
									handleArgs(cCtx)
									return cliClient.ClientNodeControllerConnect()
								},
							},
							{
								Name:    "disconnect",
								Aliases: []string{"d"},
								Usage:   "ask node to disconnect from controller",
								Action: func(cCtx *cli.Context) error {
									handleArgs(cCtx)
									return cliClient.ClientNodeControllerDisconnect()
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
