package main

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/gofrs/uuid/v5"
	pb "github.com/hongkongkiwi/go-currency-nodes/v2/gen/pb"
	"github.com/tebeka/atexit"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	cliCmds "github.com/hongkongkiwi/go-currency-nodes/v2/internal/cli_cmds"
	configs "github.com/hongkongkiwi/go-currency-nodes/v2/internal/configs"
)

const controllerAddr = "127.0.0.1"
const controllerPort = 5160
const appVersion = "0.0.1"

var cliUuid uuid.UUID
var controllerServer string

func connectController(controllerAddr string, controllerPort int) {
	controllerServer = fmt.Sprintf("%s:%d", controllerAddr, controllerPort)
	conn, err := grpc.Dial(
		controllerServer,
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
	doneChan := make(chan bool)

	log.Printf("Connected to Controller %s:%d", controllerAddr, controllerPort)

	// Thread for sending hello command
	go func() {
		for {
			select {
			case <-doneChan:
				return
			case command := <-cliCmds.SendChan:
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
		close(doneChan)
	}()

	<-doneChan
	log.Printf("node exited")
}

func main() {
	var err error
	cliUuid, err = uuid.NewV4()
	if err != nil {
		atexit.Fatalln(err)
	}
	configs.CliCfg.CliUUID = &cliUuid
	configs.CliCfg.ControllerServer = fmt.Sprintf("%s:%d", controllerAddr, controllerPort)
	configs.CliCfg.AppVersion = appVersion
	log.Printf("Cli Client UUID: %s", cliUuid.String())

	cliCmds.SendChan = make(chan *pb.StreamFromCli)
	// Queue up our command
	go cliCmds.ControllerListNodes()
	// Attempt to connect
	connectController(controllerAddr, controllerPort)
}
