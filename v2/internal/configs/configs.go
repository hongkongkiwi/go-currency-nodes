package configs

import (
	"github.com/gofrs/uuid/v5"
)

type NodeCfgStruct struct {
	StreamUpdates    bool
	AppVersion       string
	ControllerServer string
	NodeUUID         *uuid.UUID
}

type ControllerCfgStruct struct {
	AppVersion string
}

type CliCfgStruct struct {
	AppVersion       string
	ControllerServer string
	CliUUID          *uuid.UUID
	VerboseLog       bool
}

var ControllerCfg ControllerCfgStruct
var NodeCfg NodeCfgStruct
var CliCfg CliCfgStruct
