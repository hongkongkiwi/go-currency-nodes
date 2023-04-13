package internal

import (
	"github.com/bay0/kvs"
	"github.com/gofrs/uuid/v5"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	PriceUpdatesReady int = iota
	PriceUpdatesPause
)

type CurrencyStoreItem struct {
	Price   float32
	ValidAt *timestamppb.Timestamp
}

func (p *CurrencyStoreItem) Clone() kvs.Value {
	return &CurrencyStoreItem{
		Price:   p.Price,
		ValidAt: p.ValidAt,
	}
}

type NodeConfigStore struct {
	UUID           uuid.UUID
	Name           string
	ListenAddr     string
	ControllerAddr string
	CurrencyPairs  []string
}

var NodeCfg NodeConfigStore
var PriceStore *kvs.KeyValueStore
var PriceUpdatesState int = PriceUpdatesReady
