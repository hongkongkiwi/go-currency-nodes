package internal

import (
	"context"
	"fmt"
	"time"

	pb "github.com/hongkongkiwi/go-currency-nodes/pb" // imports as package "cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

var conn *grpc.ClientConn
var NodeAddr string

func Disconnect() {
	conn.Close()
}

func Connect(nodeAddress string) error {
	// Set up a connection to the server.
	var err error
	conn, err = grpc.Dial(nodeAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	return nil
}

func NodeUUID() error {
	connectErr := Connect(NodeAddr)
	if connectErr != nil {
		return connectErr
	}
	c := pb.NewNodeControlCommandsClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(time.Millisecond*80))
	defer cancel()
	r, err := c.NodeUUID(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("could not call NodeUUID: %v", err)
	}
	fmt.Printf("NodeUUID: %s", r.NodeUuid)
	return nil
}
