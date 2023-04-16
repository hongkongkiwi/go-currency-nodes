package internal

import (
	"time"

	"github.com/bay0/kvs"
)

// This is what is stored inside our local datastore to represent a currency value
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
