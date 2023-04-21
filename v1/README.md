# Currency Nodes

This project is made up of three different applications:
1. Controller Server (receives connections from nodes, connects to nodes)
2. Node Client (connects to controller, receives connections from controller)
3. Cli Client (connects to node)

## Quickstart

You can start each service one by one via docker:

`./controller_start`

`./node1_start`

`./node2_start`

`./node3_start`

You can then view the logs of all nodes at once with `docker compose logs -f` or each via `docker compose logs -f controller1` etc.

By default the cli client will connect to node1, you can change it by passing the --addr flag to the cli client to pass the node server address (check docker-comopse.yml for appropriate address).

The cli client has extensive help and documents it's commands.

```
./cli node
NAME:
   cli node - options for node control

USAGE:
   cli node command [command options] [arguments...]

COMMANDS:
   uuid, u           show the node app uuid
   currencies, c     show the currencies this node knows about
   status, s         show the node status
   app, a            options for node app
   priceupdates, u   price
   subscriptions, c  list all subscriptions (and known prices)
   help, h           Shows a list of commands or help for one command

OPTIONS:
   --addr value, --address value, --remote-address value  address of the node to connect to (default: "127.0.0.1:5051") [$CLI_NODE_REMOTE_ADDR]
   --timeout value, --remote-timeout value                timeout for calls to node (default: 150ms) [$CLI_NODE_REMOTE_TIMEOUT]
   --verbose                                              turn on verbose logging (default: false) [$CLI_VERBOSE]
   --quiet                                                turn off all logging but reply data (default: false) [$CLI_QUIET]
   --help, -h                                             show help
```

Show UUID of this node

```
./cli node uuid
b501cec4-36dd-4d7c-a1b9-5abc1d9290da
```

Show all currencies that this node has prices for

```
./cli node currencies
{
  "currencyItems":  [
    {
      "currencyPair":  "HKD_USD",
      "price":  0.022937600000000002,
      "priceValidAt":  "2023-04-17T17:13:57.009Z"
    },
    {
      "currencyPair":  "NZD_USD",
      "price":  19.238400000000002,
      "priceValidAt":  "2023-04-17T17:08:41.020Z"
    },
    {
      "currencyPair":  "USD_HKD",
      "price":  23.14993664000001,
      "priceValidAt":  "2023-04-17T17:13:57.009Z"
    },
    {
      "currencyPair":  "USD_NZD",
      "price":  6.323200000000001,
      "priceValidAt":  "2023-04-17T17:13:57.009Z"
    }
  ]
}
```

Send a manual price update (for testing)

```
./cli node priceupdates new --currency-pair HKD_USD -
-price 50
Successfully Sent
```

Stop automatic updates from backend price generator API

```
./cli node priceupdates pause
Successfully Sent
```

Resume automatic updates from backend price generator API

```
./cli node priceupdates resume
Successfully Sent
```

## Local Development

Show all make targets: `make help`

Build Everything Locally: `make`

Run controller: `./build/controller start`

Run node: `./build/node start`

Interact with node via CLI client: `./build/cli node uuid`
