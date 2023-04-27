/**
 * Stores which nodes are streaming which currency pairs
 */
package stores

import (
	"sync"
	"time"
)

var ControllerPriceStore *inMemoryPriceStore

type priceStore struct {
	price     float64
	updatedAt time.Time
}

type inMemoryPriceStore struct {
	mutex sync.RWMutex
	store map[string]*priceStore
}

func NewPriceStore() *inMemoryPriceStore {
	ps := &inMemoryPriceStore{
		store: make(map[string]*priceStore),
	}
	return ps
}

func (ps *inMemoryPriceStore) SetPrice(currencyPair string, price float64, updatedAt time.Time) error {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()
	ps.store[currencyPair] = &priceStore{
		price:     price,
		updatedAt: updatedAt,
	}
	return nil
}

func (ps *inMemoryPriceStore) GetPrice(currencyPair string) (bool, float64, time.Time) {
	if priceStoreItem, ok := ps.store[currencyPair]; ok && priceStoreItem != nil {
		return true, priceStoreItem.price, priceStoreItem.updatedAt
	}
	return false, 0, time.Now()
}
