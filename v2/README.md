# Currency Nodes

This project is made up of three different applications:
1. Controller Server (awaits connections from nodes or cli)
2. Node Client (connects to controller) - Connects sends known currencies and waits for commands
3. Cli Client (connects to controller) - Sends commands to controller and nodes

## Protocol

In this version, the only thing that needs to be configured is what currencies each node knows about. In the sample this is already configured in the [docker-compose.yml](https://github.com/hongkongkiwi/go-currency-nodes/blob/main/v2/docker-compose.yml) file.

The controller will accept any currencies that the node knows about (in a future version this might be hard coded so the controller doesn't accept rubbish currency pairs).

The goal of the controller to always ensure that exactly 1 node is transmitting data for each currency pair. Upon connect the nodes will specify which currency pairs they support upon connect but will not automatically transmit anything else. They will maintain a consistent connection to the server.

When the controller receives currency pairs OR when a node disconnects, the controller will automatically ensure that 1 node is transmitting currency data for each pair (if possible) by requesting nodes to start streaming. If there are more than 1 node which knows about a specific currency pair, then the controller will randomly pick one. The minimum number of nodes can be configured, so you can ensure for example atleast 2 nodes are transmitting the same currency pairs.

If a node goes offline, then the controller will pick another node which knows about the currency pair to transmit.

Prices are received and stored, but nothing is done with the prices and no checks are done to ensure the prices are valid. This is just for illustration.

More details on specific commands is [briefly documented here](https://github.com/hongkongkiwi/go-currency-nodes/blob/main/v2/docs/Commands.md).

The cli client can send commands to every node (via the controller), a subset of nodes or only the controller. Only a few commands are supported, but more can be easily added.

## Quickstart

To spin up nodes and server in background and inspect logs:

```
docker compose up -d
docker compose logs -f
```

## Local Development

Show all make targets: `make help`

Build Everything Locally: `make`

Run controller: `./build/controller start`

Run node: `./build/node start` (multiple nodes can be run)

Interact with controller via CLI client: `./build/cli controller nodes`

Interact with node via CLI client: `./build/cli node --node-uuid "<UUID>" status`

