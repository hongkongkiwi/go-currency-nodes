/** This package allows reading and writng from config files as well
 * as handling config vars from command line arguments
 */
package configs

import (
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/tebeka/atexit"
	"github.com/urfave/cli/v2"
)

// Updated via external tools
const CliAppVersion = "0.2.3"

// Bring these to the top so we can reference them easily
const ArgCliFlagVerbose = "verbose"
const ArgCliFlagController = "controller"
const ArgCliFlagCliUuid = "uuid"
const ArgCliFlagNodeUuids = "node-uuid"

var ArgEnvCliVerbose = [1]string{"CLI_VERBOSE"}
var ArgEnvCliControllers = [1]string{"CLI_CONTROLLER_ADDRESS"}
var ArgEnvCliUuid = [1]string{"CLI_UUID"}
var ArgEnvCliNodeUuids = [1]string{"CLI_NODE_UUIDS"}

const DefaultCliConnectController = "127.0.0.1:5160"

var CliCfg *CliCfgStruct

// Cli needs a UUID to send messages back to crrect cli client
// Additionally it needs to know which server to connect to
type CliCfgStruct struct {
	ControllerServer string
	Uuid             *uuid.UUID
	VerboseLog       bool
}

/** CliCfg functions **/
func (c *CliCfgStruct) SetUUID(UUID *uuid.UUID) error {
	if UUID == nil {
		return fmt.Errorf("empty UUID passed")
	}
	c.Uuid = UUID
	return nil
}

func (c *CliCfgStruct) SetUUIDFromString(UUIDString string) error {
	newUuid, err := uuid.FromString(UUIDString)
	if err != nil {
		return err
	}
	c.Uuid = &newUuid
	return nil
}

// Get UUID (either from saved or randomly generated)
func (cfg *CliCfgStruct) UUID() *uuid.UUID {
	var err error
	var newUuid uuid.UUID
	if cfg.Uuid == nil {
		newUuid, err = uuid.NewV4()
	}
	if err != nil {
		atexit.Fatal(err)
	} else if cfg.Uuid == nil {
		// This really is a bad thing!
		atexit.Fatal(fmt.Errorf("unable to generate a UUID"))
	}
	cfg.Uuid = &newUuid
	return cfg.Uuid
}

func (cfg *CliCfgStruct) UUIDString() (string, error) {
	return cfg.UUID().String(), nil
}

func NewCliCfgFromArgs(cCtx *cli.Context) (*CliCfgStruct, error) {
	cfg := &CliCfgStruct{
		VerboseLog:       cCtx.Bool(ArgCliFlagVerbose),
		ControllerServer: cCtx.String(ArgCliFlagController),
	}
	if cCtx.String(ArgCliFlagCliUuid) != "" {
		if uuidErr := cfg.SetUUIDFromString(cCtx.String(ArgCliFlagCliUuid)); uuidErr != nil {
			return cfg, uuidErr
		}
	} else {
		newUuid, uuidErr := uuid.NewV4()
		if uuidErr != nil {
			return cfg, uuidErr
		}
		if uuidErr := cfg.SetUUID(&newUuid); uuidErr != nil {
			return cfg, uuidErr
		}
	}
	return cfg, nil
}
