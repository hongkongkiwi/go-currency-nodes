package internal

type ControllerConfigStore struct {
	ControllerListenAddr string   // External IP:Port for node connections
	CurrencyPairs        []string // Which currency pairs to allow updates for - anythign else is ignored
}

var ControllerCfg ControllerConfigStore
