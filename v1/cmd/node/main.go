package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/gofrs/uuid/v5"
	helpers "github.com/hongkongkiwi/go-currency-nodes/v1/internal/helpers"
	nodeClient "github.com/hongkongkiwi/go-currency-nodes/v1/internal/node_client"
	nodePriceGen "github.com/hongkongkiwi/go-currency-nodes/v1/internal/node_price_gen"
	nodeServer "github.com/hongkongkiwi/go-currency-nodes/v1/internal/node_server"
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
	go nodeServer.StopServer()
	nodeClient.StopPriceUpdates()
	log.Println("Exiting ...")
}

func nodeStart() error {
	var wg sync.WaitGroup
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan)
	atexit.Register(gracefulExitHandler)

	// Create new store
	var storeErr error
	nodeClient.NodePriceStore, storeErr = helpers.NewMemoryCurrencyStore()
	if storeErr != nil {
		// We shouldn't get here! something is very wrong
		panic(storeErr)
	}
	updatesChan := make(chan map[string]*nodePriceGen.PriceCurrency)
	stopChan = make(chan bool)
	priceGen, _ = nodePriceGen.NewPriceGeneratorApi(helpers.NodeCfg.CurrencyPairs, updatesChan)
	// Start generating prices from price API
	wg.Add(1)
	go priceGen.GenRandPricesForever(&wg, helpers.NodeCfg.UpdatesMinFreq, helpers.NodeCfg.UpdatesMaxFreq, stopChan)
	// Start our command server
	wg.Add(1)
	go nodeServer.StartServer(&wg, updatesChan)
	// Subscribe to all currency pairs on server
	go func() {
		var err error
		// Just for fun grab our controller version
		if err = nodeClient.ClientControllerVersion(); err != nil {
			log.Printf("%v", err)
		}
		// Subscribe to updates of our currency pairs
		if err = nodeClient.ClientControllerCurrencyPriceSubscribe(); err != nil {
			log.Printf("%v", err)
		}
		// Pull current prices manually at startup
		// if err = nodeClient.ClientControllerCurrencyPrice(); err != nil {
		// 	log.Printf("%v", err)
		// }
		// Start listening for price updates from our API
		wg.Add(1)
		if err = nodeClient.StartPriceUpdates(&wg, updatesChan, stopChan); err != nil {
			log.Printf("%v", err)
		}
	}()
	go nodeClient.KeepAliveTick(stopChan)
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

func ensureValidUtf8(s string) string {
	if !utf8.ValidString(s) {
		v := make([]rune, 0, len(s))
		for i, r := range s {
			if r == utf8.RuneError {
				_, size := utf8.DecodeRuneInString(s[i:])
				if size == 1 {
					continue
				}
			}
			v = append(v, r)
		}
		s = string(v)
	}
	return s
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
						Name:    "address",
						Aliases: []string{"listen-address"},
						Value:   "127.0.0.1:5051",
						Usage:   "set listen address for this node",
						EnvVars: []string{"NODE_LISTEN_ADDR"},
					},
					&cli.StringFlag{
						Name:    "advertise",
						Aliases: []string{"advertise-address"},
						Value:   "127.0.0.1:5051",
						Usage:   "set advertise address for this node",
						EnvVars: []string{"NODE_ADVERTISE_ADDR"},
					},
					&cli.StringFlag{
						Name:    "controller",
						Value:   "127.0.0.1:5060",
						Usage:   "set controller address",
						EnvVars: []string{"NODE_CONTROLLER_ADDR"},
					},
					&cli.StringFlag{
						Name:    "uuid",
						Aliases: []string{"node-uuid"},
						Value:   "",
						Usage:   "set uuid for this node (will generate random)",
						EnvVars: []string{"NODE_UUID"},
					},
					&cli.StringFlag{
						Name:    "name",
						Aliases: []string{"node-name"},
						Value:   "",
						Usage:   "set name for this node",
						EnvVars: []string{"NODE_NAME"},
					},
					&cli.DurationFlag{
						Name:    "keepalive",
						Aliases: []string{"keepalive-interval"},
						Value:   time.Duration(10 * time.Second),
						Usage:   "set keepalive interval for this node",
						EnvVars: []string{"NODE_KEEPALIVE_INTERVAL"},
					},
					&cli.BoolFlag{
						Name:    "paused",
						Aliases: []string{"start-paused"},
						Value:   false,
						Usage:   "do not generate price updates on this node, just listen",
						EnvVars: []string{"NODE_PAUSE_UPDATES"},
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Value:   false,
						Usage:   "turn on verbose logging for this node",
						EnvVars: []string{"NODE_LOG_VERBOSE"},
					},
					&cli.DurationFlag{
						Name:    "updates-min-freq",
						Value:   time.Duration(1 * time.Second),
						Usage:   "minimum time to random generate an update",
						EnvVars: []string{"NODE_PRICEUPDATES_MIN_FREQ"},
					},
					&cli.DurationFlag{
						Name:    "updates-max-freq",
						Value:   time.Duration(6 * time.Second),
						Usage:   "maximum time to random generate an update",
						EnvVars: []string{"NODE_PRICEUPDATES_MAX_FREQ"},
					},
					&cli.UintFlag{
						Name:    "updates-percent",
						Value:   20,
						Usage:   "percentage to change prices when generating update",
						EnvVars: []string{"NODE_PRICEUPDATES_PERCENT"},
					},
					&cli.StringSliceFlag{
						Name:    "currencies",
						Aliases: []string{"currency-pairs", "pairs"},
						Value: cli.NewStringSlice(
							"USD_HKD",
							"HKD_USD",
							"USD_NZD",
							"NZD_USD",
							"BTC_HKD",
							"HKD_BTC",
							"BTC_USD",
							"USD_BTC",
						),
						Usage:   "set currency pairs for this node",
						EnvVars: []string{"NODE_CURRENCY_PAIRS"},
					},
				},
				Action: func(cCtx *cli.Context) error {
					helpers.NodeCfg.NodeListenAddr = ensureValidUtf8(cCtx.String("address"))
					helpers.NodeCfg.NodeAdvertiseAddr = ensureValidUtf8(cCtx.String("advertise"))
					helpers.NodeCfg.ControllerAddr = ensureValidUtf8(cCtx.String("controller"))
					helpers.NodeCfg.Name = ensureValidUtf8(cCtx.String("name"))
					helpers.NodeCfg.UpdatesMinFreq = cCtx.Duration("updates-min-freq")
					helpers.NodeCfg.UpdatesMaxFreq = cCtx.Duration("updates-max-freq")
					helpers.NodeCfg.UpdatesPercentChange = cCtx.Uint("updates-percent")
					nodeUuid := strings.ToLower(ensureValidUtf8(cCtx.String("uuid")))
					if nodeUuid == "" {
						log.Printf("WARNING: we randomly generated UUID it is better to pass fixed one for this node")
						helpers.NodeCfg.UUID = uuid.Must(uuid.NewV4())
					} else {
						helpers.NodeCfg.UUID = uuid.FromStringOrNil(nodeUuid)
						if helpers.NodeCfg.UUID == uuid.Nil {
							atexit.Fatalf("invalid uuid format %s", helpers.NodeCfg.UUID)
						}
					}
					if helpers.NodeCfg.Name == "" {
						helpers.NodeCfg.Name = fmt.Sprintf("node (%s)", helpers.NodeCfg.UUID.String())
					}
					helpers.NodeCfg.CurrencyPairs = cCtx.StringSlice("currencies")
					helpers.NodeCfg.VerboseLog = cCtx.Bool("verbose")
					nodePriceGen.UpdatesPaused = cCtx.Bool("paused")
					helpers.NodeCfg.KeepAliveInterval = cCtx.Duration("keepalive")

					if helpers.NodeCfg.KeepAliveInterval < 1 {
						return fmt.Errorf("keepalive interval must be greater than 0")
					}
					log.Printf("config node uuid: %s", helpers.NodeCfg.UUID)
					log.Printf("config node name: %s", helpers.NodeCfg.Name)
					log.Printf("config node currency pairs: %s", helpers.NodeCfg.CurrencyPairs)
					log.Printf("config node address: %s", helpers.NodeCfg.NodeListenAddr)
					log.Printf("config keepalive interval: %v", helpers.NodeCfg.KeepAliveInterval)
					log.Printf("config controller address: %s", helpers.NodeCfg.ControllerAddr)
					return nodeStart()

				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		atexit.Fatal(err)
	}
}
