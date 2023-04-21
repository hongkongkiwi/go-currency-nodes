package internal

type ControllerConfigStore struct {
	ControllerListenAddr string   // External IP:Port for node connections
	CurrencyPairs        []string // Which currency pairs to allow updates for - anythign else is ignored
	VerboseLog           bool     // Whether to output verbose logs or not
	DiskKVDir            string   // Where the kv db is stored
}

var ControllerCfg ControllerConfigStore
