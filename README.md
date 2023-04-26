Currency Nodes
================

A command and control interface for nodes sending updates streaming data updates.

This example system might be useful in the case of calling external APIs and having the data streamed back upon request.

There are two major versions:

## [v1](https://github.com/hongkongkiwi/go-currency-nodes/v1)

This version uses gRPC Unary Protocol to send messages back and forth between controller and nodes. 

The nodes start streaming immediately and there is not retrying on any sent messages.

This version requires Firewall ports to be open on each node, so the connection is Dual Unary for some commands.

## [v2](https://github.com/hongkongkiwi/go-currency-nodes/v2)

This version implements gRPC Bidirectional streaming with a generic message wrapper.

This is an improved version and allows for a permanent tcp connection to be maintained for messages.

Note: Nodes will not start performing any functions until instructed by the server.

Additionally a Cli client can send messages through the central server
