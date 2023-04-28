## Connection Flow

Controller awaits connections from two types of clients:
1. Cli client
2. Node Client

### Cli Client

A cli client sends commands to the Controller which then takes action on them, either forwarding them onto specified nodes, or replying directly (in the case of listing which nodes are online etc).

Since nodes reply asyncronously then we do not know how many replies to expect. So the cli client needs to be exited using Control-C when all desired replies come in. NOTE: this can be improved with a reply count of expected replies plus a timeout.

### Node Client

A Node is a very simple client, it only understands a small number of commands.

Known Currency Pairs are configurable at run time for the node. This simulates the Node sitting behind a firewall which can only access some specific upstream currency pairs.

Upon connect a Node client will connect and send a NodeKnownCurrenciesV1Notify message to let hte controller know what currency pairs it can handle.

After this, the Controller will inspect it's store of streaming nodes for each currency pair. If there are no streaming nodes for one or more currency pairs this Node knows about, the controller will ask the node to start streaming prices and optionally will send the latest price it knows about to the Node (that way the Node can decide if it's latest update is newer than the latest one the controller knows about). This happens automatically by the Controller and no Cli client commands are required for this.

There is no state stored on the Node, excepting a NodeLatestPricesApi simulator (this is equivilent to being able to ask the upstream api it's most recent price).

If a Node goes offline, then the controller will see if any other nodes are capable of hnadling the stream and ask them to stream. So, there is no gaurrentee this Node will be asked to stream again when reconnecting.

If the controller goes offline, then the node will continuously attempt to reconnect. When the node reconnects, there is no gaurrentee that the same node will be picked to stream the currency pairs (since it is randomly picked from online nodes for each currency pair).

## Protocol

Due to the nature of gRPC and to keep things more simple we will setup a single stream and push commands back and forth along this.

To handle sending different types of commands we wrap each command up as a generic protbuf Any type, with controllers and nodes responsible for unwrapping and rewrapping them as needed. This allows for expansion in future of commands.

Every command has a version in it's name so that commands which are incompatible in future can be created and the system can be made backwareds compatible.

At a high level here is the gRPC communication stream:

*Node*
```
rpc NodeCommandStream (stream StreamFromNode) returns (stream StreamToNode) {}

// What the node sends to controller
message StreamFromNode {
  string node_uuid = 1; // Our node uuid
  optional string from_uuid = 2; // Cli client uuid
  optional google.protobuf.Any response = 3; // Wrap the response
}

// What the controller sends to node
message StreamToNode {
  optional string from_uuid = 1; // Pass this so the command gets sent back to right cli client
  google.protobuf.Any command = 2; // Generic command wrapper
}
```

*Cli*
```
rpc CliCommandStream (stream StreamFromCli) returns (stream StreamToCli) {}

message StreamFromCli {
  string stream_id = 1; // Unique identifier to track this request, mostly uused for tracking how many replies
  string cli_uuid = 2; // CLI client uuid to return back to
  repeated string node_uuids = 3; // List of node UUIDs to send to
  google.protobuf.Any command = 4; // Command to send to node
}

message StreamToCli {
  string stream_id = 1; // Unique identifier to track this request
  optional string node_uuid = 2; // Our node uuid
  google.protobuf.Any response = 3;
}
```

## Node Commands

*Commands*
Command | Response | Description |
NodeStatusV1Req | NodeStatusV1Reply | Node sends it's current status |
NodeStreamModifyStateV1Req | NodeStreamModifyStateV1Reply | Node will start or stop streaming depending on requests from controller. Any currencies not specifically specified is stopped. |
NodeShutdownV1Req | NodeShutdownV1Reply | Node will shutdown connection gracefully then exit |

*Notifications*
Notification | Direction | Description |
NodeKnownCurrenciesV1Notify | Outgoing | Node sends known currencies (send after connection) |
NodePriceUpdateV1Notify | Outgoing | Node sends a price update (only sent for streaming currencites) |
UnknownCommandV1Notify | Outgoing | Node sends this when it receives an unhandled command |

## Cli Commands

Supports all above Node commands and additionally:

*Commands*
Command | Response | Description |
ControllerListNodesV1Req | ControllerListNodesV1Reply | Requests the controller to list all Nodes (uuids) |
ControllerKickNodesV1Req | ControllerKickNodesV1Reply | Requests the controller to kick a node (nodes will automatically reconnect) |
ControllerAppVersionV1Req | ControllerAppVersionV1Reply | Request the controller to send it's app version |

*Notifications*
Notification | Direction | Description |
CliResponseCountV1Notify | Incoming | Controller sends this so the Cli client knows how many resopnses to wait for |

