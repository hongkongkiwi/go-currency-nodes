/** This package allows reading and writng from config files as well
 * as handling config vars from command line arguments
 */
package configs

import (
	"fmt"
	"sync"

	"github.com/gofrs/uuid/v5"
	"github.com/tebeka/atexit"
	"github.com/urfave/cli/v2"
)

// Updated via external tools
const NodeAppVersion = "0.2.0"

const ArgNodeFlagVerbose = "verbose"
const ArgNodeFlagControllers = "controller"
const ArgNodeFlagName = "name"
const ArgNodeFlagUuid = "uuid"
const ArgNodeKnownCurrencyPairs = "currency-pair"

var ArgEnvNodeVerbose = [2]string{"NODE_VERBOSE", "NODE_VERBOSE_LOG"}
var ArgEnvNodeControllers = [3]string{"NODE_CONTROLLER", "NODE_CONTROLLER_ADDRESS", "NODE_CONTROLLER_ADDR"}
var ArgEnvNodeUuid = [1]string{"NODE_UUID"}
var ArgEnvNodeName = [1]string{"NODE_NAME"}
var ArgEnvNodeKnownCurrencyPairs = [1]string{"NODE_CURRENCY_PAIRS"}

const DefaultNodeConnectController = "127.0.0.1:5160"
const DefaultNodeVerboseLog = true

var DefaultNodeKnownCurrencyPairs = []string{
	"USD_HKD",
	"HKD_USD",
}

var NodeCfg *NodeCfgStruct

// Node has a list of servers it will randomly try
// Also node MUST have a uuid so we can keep track
type NodeCfgStruct struct {
	mutex              sync.RWMutex
	KnownCurrencyPairs []string // Node can get data for these currency pairs
	ControllerServers  []string
	Name               string
	IsUuidGenerated    bool
	VerboseLog         bool
	uuid               *uuid.UUID
}

func NewNodeCfgFromArgs(cCtx *cli.Context) (*NodeCfgStruct, error) {
	cfg := &NodeCfgStruct{
		VerboseLog:         cCtx.Bool(ArgNodeFlagVerbose),
		ControllerServers:  cCtx.StringSlice(ArgNodeFlagControllers),
		KnownCurrencyPairs: cCtx.StringSlice(ArgNodeKnownCurrencyPairs),
		IsUuidGenerated:    (cCtx.String(ArgNodeFlagUuid) == ""),
		Name:               cCtx.String(ArgNodeFlagName),
	}
	newUuid, err := uuid.FromString(cCtx.String(ArgNodeFlagUuid))
	if err == nil && newUuid.String() != "00000000-0000-0000-0000-000000000000" {
		cfg.SetUUID(&newUuid)
	} else {
		cfg.UUID()
	}
	if len(cfg.ControllerServers) == 0 {
		return cfg, fmt.Errorf("no valid controllers passed")
	}
	return cfg, nil
}

/** NodeCfg functions **/
func (cfg *NodeCfgStruct) SetUUID(UUID *uuid.UUID) error {
	if UUID == nil {
		return fmt.Errorf("empty UUID passed")
	}
	cfg.mutex.Lock()
	defer cfg.mutex.Unlock()
	cfg.uuid = UUID
	cfg.IsUuidGenerated = false
	return nil
}

func (cfg *NodeCfgStruct) SetUUIDFromString(UUIDString string) error {
	newUuid, err := uuid.FromString(UUIDString)
	if err != nil {
		return err
	}
	return cfg.SetUUID(&newUuid)
}

func (cfg *NodeCfgStruct) UUID() *uuid.UUID {
	var err error
	var newUuid uuid.UUID
	if cfg.uuid == nil {
		newUuid, err = uuid.NewV4()
		if err != nil {
			atexit.Fatal(err)
		}
		cfg.SetUUID(&newUuid)
	}
	return cfg.uuid
}
