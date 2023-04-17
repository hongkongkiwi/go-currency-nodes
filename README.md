# Currency Nodes

This project is made up of three different applications:
1. Controller Server (receives connections from nodes, connects to nodes)
2. Node Client (connects to controller, receives connections from controller)
3. Cli Client (connects to node)

## Quickstart

The best way to run a complete setup is to use Docker Compose e.g. `docker compose up -d`

This will spin up 1 controller and 3 nodes, the cli client can then be used locally with the shell script helper.

`./cli node uuid`

## Local Usage

Show all make targets: `make help`

Build Everything Locally: `make`

Run controller: `./build/controller start`

Run node: `./build/node start`

Interact with node via CLI client: `./build/cli node uuid`
