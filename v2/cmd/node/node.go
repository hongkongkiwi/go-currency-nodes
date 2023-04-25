package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gofrs/uuid/v5"
	pb "github.com/hongkongkiwi/go-currency-nodes/v2/gen/pb"
	"github.com/tebeka/atexit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	configs "github.com/hongkongkiwi/go-currency-nodes/v2/internal/configs"
	nodeCmds "github.com/hongkongkiwi/go-currency-nodes/v2/internal/node_cmds"
)

const controllerAddr = "127.0.0.1"
const controllerPort = 5160
const appVersion = "0.0.1"

var nodeUuid uuid.UUID
var controllerServer string
var ticker *backoff.Ticker

func connectController(controllerAddr string, controllerPort int) {

	// var retryPolicy = `{
	// 		"methodConfig": [{
	// 				// config per method or all methods under service
	// 				"name": [{"service": "grpc.examples.echo.Echo"}],
	// 				"waitForReady": true,

	// 				"retryPolicy": {
	// 						"MaxAttempts": 4,
	// 						"InitialBackoff": ".01s",
	// 						"MaxBackoff": ".01s",
	// 						"BackoffMultiplier": 1.0,
	// 						// this value is grpc code
	// 						"RetryableStatusCodes": [ "UNAVAILABLE" ]
	// 				}
	// 		}]
	// }`

	controllerServer = fmt.Sprintf("%s:%d", controllerAddr, controllerPort)
	conn, err := grpc.Dial(
		controllerServer,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		// grpc.WithDefaultServiceConfig(retryPolicy),
	)
	if err != nil {
		log.Printf("ERROR can not connect with server %v", err)
		return
	}

	// create stream
	client := pb.NewNodeCmdStreamClient(conn)
	stream, err := client.NodeCommandStream(context.Background())
	if err != nil {
		log.Printf("ERROR openn stream error %v", err)
		return
	}

	// We are connected close the ticker
	ticker.Stop()
	ticker = nil

	ctx := stream.Context()
	doneChan := make(chan bool)
	nodeCmds.SendChan = make(chan *pb.StreamFromNode)

	// Thread for sending hello command
	go func() {
		for {
			select {
			case <-doneChan:
				return
			case command := <-nodeCmds.SendChan:
				if err := stream.Send(command); err != nil {
					log.Printf("can not send %v", err)
					doneChan <- true
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
				close(doneChan)
				return
			}
			if err != nil {
				log.Printf("can not receive %v", err)
				doneChan <- true
				return
			}
			nodeCmds.HandleIncomingCommand(in.FromUuid, in.Command)
		}
	}()

	// third goroutine closes done channel
	// if context is done
	go func() {
		<-ctx.Done()
		if err := ctx.Err(); err != nil {
			log.Println(err)
		}
		close(doneChan)
	}()

	nodeCmds.ControllerHello()
	<-doneChan
	log.Printf("node exited")
}

func main() {
	var err error
	nodeUuid, err = uuid.NewV4()
	if err != nil {
		atexit.Fatalln(err)
	}
	configs.NodeCfg.NodeUUID = &nodeUuid
	configs.NodeCfg.ControllerServer = fmt.Sprintf("%s:%d", controllerAddr, controllerPort)
	configs.NodeCfg.AppVersion = appVersion
	configs.NodeCfg.StreamUpdates = true

	// Use backoff so we can have a configurable stratergy later for reconnections
	b := backoff.NewConstantBackOff(4 * time.Second)
	for {
		if ticker == nil {
			ticker = backoff.NewTicker(b)
		}
		<-ticker.C
		connectController(controllerAddr, controllerPort)
	}
}
