package internal

import (
	"time"

	"github.com/bay0/kvs"
	"github.com/gofrs/uuid/v5"
)

type CurrencyStoreItem struct {
	Price   float64
	ValidAt time.Time
}

// This is implemented for the KV store
func (p *CurrencyStoreItem) Clone() kvs.Value {
	return &CurrencyStoreItem{
		Price:   p.Price,
		ValidAt: p.ValidAt,
	}
}

type NodeConfigStore struct {
	UUID          uuid.UUID
	Name          string
	ListenAddr    string
	CurrencyPairs []string
	VerboseLog    bool
}

type ControllerConfigStore struct {
	ListenAddr string
}

var NodeCfg NodeConfigStore
var ControllerCfg ControllerConfigStore
