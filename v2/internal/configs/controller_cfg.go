/** This package allows reading and writng from config files as well
 * as handling config vars from command line arguments
 */
package configs

import "github.com/urfave/cli/v2"

// Updated via external tools
const ControllerAppVersion = "0.2.1"

const ArgControllerFlagVerbose = "verbose"
const ArgControllerFlagListenAddress = "listen-address"
const ArgControllerCncurrentNodePriceStreams = "pricestream-count"

var ArgEnvControllerVerbose = [2]string{"CONTROLLER_VERBOSE", "CONTROLLER_VERBOSE_LOG"}
var ArgEnvControllerListenAddress = [3]string{"CONTROLLER_ADDRESS", "CONTROLLER_LISTEN_ADDRESS", "CONTROLLER_LISTEN_ADDR"}
var ArgEnvControllerCncurrentNodePriceStreams = [2]string{"CONTROLLER_PRICESTREAM_COUNT"}

// At most this number of nodes to stream the same currency pair
// Setting this to 0 will disable automatic node price streaming
const DefaultControllerCncurrentNodePriceStreams = 1
const DefaultControllerListenAddress = "127.0.0.1:5160"
const DefaultControllerVerboseLog = true

var ControllerCfg *ControllerCfgStruct

// Nothing special to configure for controller except logging
type ControllerCfgStruct struct {
	ListenAddress   string
	VerboseLog      bool
	MaxPriceStreams uint
}

func NewControllerCfgFromArgs(cCtx *cli.Context) (*ControllerCfgStruct, error) {
	cfg := &ControllerCfgStruct{
		VerboseLog:      cCtx.Bool(ArgControllerFlagVerbose),
		ListenAddress:   cCtx.String(ArgControllerFlagListenAddress),
		MaxPriceStreams: cCtx.Uint(ArgControllerCncurrentNodePriceStreams),
	}
	return cfg, nil
}
