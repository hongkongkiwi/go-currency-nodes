package internal

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

type NodeConfigStore struct {
	UUID                 uuid.UUID // UUID of this node (must be unique)
	Name                 string    // Name of this node
	NodeListenAddr       string    // IP:Port to listen to incoming price events from controller
	NodeAdvertiseAddr    string    // External IP:Port to advertise to controllers
	ControllerAddr       string    // Controller server we should connect to
	CurrencyPairs        []string  // Which currency pairs to subscribe to e.g. ["USD_HKD", "HKD_USD"]
	VerboseLog           bool      // Whether to output verbose logs or not
	KeepAliveInterval    time.Duration
	UpdatesMinFreq       time.Duration
	UpdatesMaxFreq       time.Duration
	UpdatesPercentChange uint
}

var NodeCfg NodeConfigStore
