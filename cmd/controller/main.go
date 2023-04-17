package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	controllerClient "github.com/hongkongkiwi/go-currency-nodes/internal/controller_client"
	controllerServer "github.com/hongkongkiwi/go-currency-nodes/internal/controller_server"
	helpers "github.com/hongkongkiwi/go-currency-nodes/internal/helpers"
	"github.com/tebeka/atexit"
	"github.com/urfave/cli/v2" // imports as package "cli"
)

var stopChan chan bool

func controllerVersion() error {
	fmt.Printf("version: %s\n", controllerServer.ControllerVersion)
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
	stopChan <- true
	log.Println("Exiting ...")
}

func controllerStart() error {
	var wg sync.WaitGroup
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan)
	atexit.Register(gracefulExitHandler)

	stopChan = make(chan bool)
	// This is going to be a bottleneck - I need to learn more about channels
	// Buffer of 100 should be ok though
	priceChan := make(chan map[string]*helpers.CurrencyStoreItem, 100) // Price updates

	// Start our command server
	wg.Add(1)
	go controllerServer.StartServer(&wg, priceChan)
	wg.Add(1)
	go controllerClient.StartMonitoringForPriceUpdates(&wg, priceChan, stopChan)
	go func() {
		stop := <-stopChan
		if stop {
			controllerServer.StopServer()
			wg.Done()
		}
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

func main() {
	app := &cli.App{
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
				Usage:   "starts the server",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "address",
						Aliases: []string{"listen_address"},
						Value:   "127.0.0.1:5052",
						Usage:   "set listen address for this controller",
						EnvVars: []string{"CONTROLLER_LISTEN_ADDR"},
					},
					&cli.StringFlag{
						Name:    "db_dir",
						Aliases: []string{"database_dir"},
						Value:   "/tmp/currency",
						Usage:   "directory for this controllers persistant db",
						EnvVars: []string{"CONTROLLER_DB_DIR"},
					},
					&cli.BoolFlag{
						Name:    "verbose",
						Value:   false,
						Usage:   "turn on verbose logging for this node",
						EnvVars: []string{"CONTROLLER_LOG_VERBOSE"},
					},
					&cli.StringSliceFlag{
						Name:    "currencies",
						Aliases: []string{"currency_pairs"},
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
						Usage:   "set currency pairs for this controller",
						EnvVars: []string{"CONTROLLER_CURRENCY_PAIRS"},
					},
				},
				Action: func(cCtx *cli.Context) error {
					helpers.ControllerCfg.ControllerListenAddr = cCtx.String("address")
					helpers.ControllerCfg.CurrencyPairs = cCtx.StringSlice("currencies")
					helpers.ControllerCfg.VerboseLog = cCtx.Bool("verbose")
					helpers.ControllerCfg.DiskKVDir = cCtx.String("db_dir")
					log.Printf("config controller currency pairs: %s", helpers.ControllerCfg.CurrencyPairs)
					log.Printf("config controller listen address: %s", helpers.ControllerCfg.ControllerListenAddr)
					log.Printf("config controller disk database dir: %s", helpers.ControllerCfg.DiskKVDir)
					return controllerStart()
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		atexit.Fatal(err)
	}
}
