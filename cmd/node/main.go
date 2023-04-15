package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/gofrs/uuid/v5"
	helpers "github.com/hongkongkiwi/go-currency-nodes/internal/helpers"
	nodeClient "github.com/hongkongkiwi/go-currency-nodes/internal/node_client"
	nodePriceGen "github.com/hongkongkiwi/go-currency-nodes/internal/node_price_gen"
	nodeServer "github.com/hongkongkiwi/go-currency-nodes/internal/node_server"
	"github.com/tebeka/atexit"
	"github.com/urfave/cli/v2" // imports as package "cli"
)

var priceGen *nodePriceGen.PriceGeneratorApi
var stopChan chan bool

func nodeVersion() error {
	fmt.Printf("version: %s\n", nodeServer.NodeVersion)
	return nil
}

func multiSignalHandler(signal os.Signal) {
	switch signal {
	// case syscall.SIGHUP:
	// 	// Reload config here
	case syscall.SIGINT:
		log.Println("Signal:", signal.String())
		if stopChan != nil {
			stopChan <- true
		}
		atexit.Exit(0)
	case syscall.SIGTERM:
		log.Println("Signal:", signal.String())
		if stopChan != nil {
			stopChan <- true
		}
		atexit.Exit(0)
	case syscall.SIGQUIT:
		log.Println("Signal:", signal.String())
		if stopChan != nil {
			stopChan <- true
		}
		atexit.Exit(0)
	}
}

func gracefulExitHandler() {
	nodeServer.StopServer()
	nodeClient.StopPriceUpdates()
	log.Println("Exiting ...")
}

func nodeStart() error {
	var wg sync.WaitGroup
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan)
	atexit.Register(gracefulExitHandler)
	wg.Add(3)

	updatesChan := make(chan map[string]*nodePriceGen.PriceCurrency)
	stopChan = make(chan bool)
	priceGen, _ = nodePriceGen.NewPriceGeneratorApi(helpers.NodeCfg.CurrencyPairs, updatesChan)
	// Start generating prices from price API
	go priceGen.GenRandPricesForever(&wg, 1000, 5000, stopChan)
	// Start our command server
	go nodeServer.StartServer(&wg, helpers.NodeCfg.ListenAddr)
	// Subscribe to all currency pairs on server
	go func() {
		// Subscribe to updates of our currency pairs
		nodeClient.ClientControllerCurrencyPriceSubscribe()
		// Pull current prices manually at startup
		nodeClient.ClientControllerCurrencyPrice()
		// Start listening for price updates from our API
		nodeClient.StartPriceUpdates(&wg, updatesChan, stopChan)
	}()
	// Handle exit signals
	go func() {
		for {
			s := <-sigChan
			multiSignalHandler(s)
		}
	}()
	// Wait for our important stuff
	wg.Wait()
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

func fallbackEnvBool(argValue bool, envKey string) bool {
	if !argValue {
		if envValue, envOk := os.LookupEnv(envKey); envOk && envValue != "" {
			boolValue, err := strconv.ParseBool(envValue)
			if err == nil {
				return boolValue
			}
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
				Usage:   "display node app version",
				Action: func(cCtx *cli.Context) error {
					return nodeVersion()
				},
			},
			{
				Name:    "start",
				Aliases: []string{"v"},
				Usage:   "starts the server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "listenAddr",
						Value: "127.0.0.1:5051",
						Usage: "listen address for control commands (NODE_LISTEN_ADDR)",
					},
					&cli.StringFlag{
						Name:  "controllerAddr",
						Value: "127.0.0.1:5052",
						Usage: "controller connect addrss (NODE_CONTROLLER_ADDR)",
					},
					&cli.StringFlag{
						Name:  "uuid",
						Value: "",
						Usage: "uuid for this node (NODE_UUID)",
					},
					&cli.StringFlag{
						Name:  "name",
						Value: "",
						Usage: "name for this node (NODE_NAME)",
					},
					&cli.BoolFlag{
						Name:  "verbose",
						Value: false,
						Usage: "whether to turn on verbose logging for this node (NODE_LOG_VERBOSE)",
					},
					&cli.StringFlag{
						Name:  "currencyPairs",
						Value: "USD_HKD,HKD_USD,USD_NZD,NZD_USD",
						Usage: "currencyPairs for this node (NODE_CURRENCY_PAIRS)",
					},
				},
				Action: func(cCtx *cli.Context) error {
					helpers.NodeCfg.ListenAddr = fallbackEnv(cCtx.String("listenAddr"), "NODE_LISTEN_ADDR")
					nodeClient.ControllerAddr = fallbackEnv(cCtx.String("controllerAddr"), "NODE_CONTROLLER_ADDR")
					helpers.NodeCfg.Name = fallbackEnv(cCtx.String("name"), "NODE_NAME")
					nodeUuid := fallbackEnv(cCtx.String("nodeUuid"), "NODE_UUID")
					helpers.NodeCfg.CurrencyPairs = strings.Split(fallbackEnv(cCtx.String("currencyPairs"), "NODE_CURRENCY_PAIRS"), ",")
					if nodeUuid != "" {
						helpers.NodeCfg.UUID = uuid.Must(uuid.FromString(nodeUuid))
					} else {
						atexit.Fatal("no --nodeUuid or env NODE_UUID!")
					}
					helpers.NodeCfg.VerboseLog = fallbackEnvBool(cCtx.Bool("verbose"), "NODE_LOG_VERBOSE")
					log.Printf("node uuid: %s", helpers.NodeCfg.UUID)
					log.Printf("node currency pairs: %s", helpers.NodeCfg.CurrencyPairs)
					return nodeStart()
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		atexit.Fatal(err)
	}
}
