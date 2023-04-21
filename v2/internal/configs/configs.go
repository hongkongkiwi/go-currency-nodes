package configs

import (
	"github.com/gofrs/uuid/v5"
)

type NodeCfgStruct struct {
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
}

var ControllerCfg ControllerCfgStruct
var NodeCfg NodeCfgStruct
var CliCfg CliCfgStruct
