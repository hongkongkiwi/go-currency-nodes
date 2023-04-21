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

var ControllerCfg ControllerCfgStruct
var NodeCfg NodeCfgStruct
