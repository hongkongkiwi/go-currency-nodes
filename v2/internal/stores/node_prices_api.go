/**
 * Struct to represent an upstream blackbox price API
 */

package stores

import (
	"math/rand"
	"sync"
	"time"
)

var LatestPricesAPI *InMemoryLatestPriceAPI

const priceGenMinPrice = 0
const priceGenMaxPrice = 100

type LatestPriceUpdate struct {
	CurrencyPair string
	Price        float64
	UpdatedAt    time.Time
}

type PriceChangedCallback func(currencyPair string, price float64, updatedAt time.Time)

// This stores currency pairs and PriceUpdate instances
type InMemoryLatestPriceAPI struct {
	mutex             sync.RWMutex
	seededRand        *rand.Rand
	knownPairs        map[string]bool
	priceStore        map[string]*LatestPriceUpdate
	callbackFuncStore map[string]PriceChangedCallback
	tickStopChanStore map[string]chan bool
}

func NewLatestPriceAPI(knownPairs []string) *InMemoryLatestPriceAPI {
	lps := &InMemoryLatestPriceAPI{
		priceStore:        make(map[string]*LatestPriceUpdate),
		callbackFuncStore: make(map[string]PriceChangedCallback),
		tickStopChanStore: make(map[string]chan bool),
		knownPairs:        make(map[string]bool),
		seededRand:        rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	for _, knownPair := range knownPairs {
		lps.knownPairs[knownPair] = false
	}
	return lps
}

func (lps *InMemoryLatestPriceAPI) StartGeneratingPrices(currencyPair string, interval time.Duration, callback PriceChangedCallback) {
	if currencyPair == "" {
		return
	}
	// Stop any existing ticker first
	if _, ok := lps.tickStopChanStore[currencyPair]; ok {
		lps.StopGeneratingPrices(currencyPair)
	}
	lps.setCallback(currencyPair, callback)
	// Make a new channel
	doneChan := make(chan bool)
	lps.mutex.Lock()
	defer lps.mutex.Unlock()
	lps.tickStopChanStore[currencyPair] = doneChan
	lps.knownPairs[currencyPair] = true
	// Start our generator thread
	go func() {
		newPrice := lps.seededRand.Intn(priceGenMaxPrice-priceGenMinPrice) + priceGenMinPrice
		lps.SetLatestPriceUpdateFromValues(currencyPair, float64(newPrice), time.Now(), true)
		ticker := time.NewTicker(interval)
		for {
			select {
			case <-doneChan:
				ticker.Stop()
				return
			case <-ticker.C:
				newPrice := lps.seededRand.Intn(priceGenMaxPrice-priceGenMinPrice) + priceGenMinPrice
				lps.SetLatestPriceUpdateFromValues(currencyPair, float64(newPrice), time.Now(), true)
			}
		}
	}()
}

func (lps *InMemoryLatestPriceAPI) StopGeneratingPrices(currencyPair string) {
	if currencyPair == "" {
		return
	}
	if tickChan, ok := lps.tickStopChanStore[currencyPair]; ok {
		tickChan <- true
		lps.mutex.Lock()
		defer lps.mutex.Unlock()
		lps.knownPairs[currencyPair] = false
		delete(lps.tickStopChanStore, currencyPair)
	}
	lps.clearCallback(currencyPair)
}

func (lps *InMemoryLatestPriceAPI) StopGeneratingAllPrices() {
	for currencyPair := range lps.tickStopChanStore {
		if currencyPair == "" {
			continue
		}
		lps.StopGeneratingPrices(currencyPair)
	}
}

func (lps *InMemoryLatestPriceAPI) setCallback(currencyPair string, callback PriceChangedCallback) {
	if currencyPair == "" {
		return
	}
	if callback == nil {
		return
	}
	lps.mutex.Lock()
	defer lps.mutex.Unlock()
	lps.callbackFuncStore[currencyPair] = callback
}

func (lps *InMemoryLatestPriceAPI) clearCallback(currencyPair string) {
	if currencyPair == "" {
		return
	}
	lps.mutex.Lock()
	defer lps.mutex.Unlock()
	delete(lps.callbackFuncStore, currencyPair)
}

func (lps *InMemoryLatestPriceAPI) SetLatestPriceUpdateFromValues(currencyPair string, price float64, updatedAt time.Time, runCallback bool) {
	if currencyPair == "" {
		return
	}
	lps.SetLatestPriceUpdate(&LatestPriceUpdate{
		CurrencyPair: currencyPair,
		Price:        price,
		UpdatedAt:    updatedAt,
	}, runCallback)
}

func (lps *InMemoryLatestPriceAPI) SetLatestPriceUpdate(priceUpdate *LatestPriceUpdate, runCallback bool) {
	if priceUpdate == nil || priceUpdate.CurrencyPair == "" {
		return
	}
	currentPriceUpdate := lps.priceStore[priceUpdate.CurrencyPair]
	// Check if this price update is earlier than our latest price update
	// If so then ignore it
	if currentPriceUpdate != nil &&
		currentPriceUpdate.UpdatedAt.After(priceUpdate.UpdatedAt) {
		return
	}
	lps.mutex.Lock()
	defer lps.mutex.Unlock()
	lps.priceStore[priceUpdate.CurrencyPair] = priceUpdate
	if callbackFunc := lps.callbackFuncStore[priceUpdate.CurrencyPair]; callbackFunc != nil && runCallback {
		go callbackFunc(priceUpdate.CurrencyPair, priceUpdate.Price, priceUpdate.UpdatedAt)
	}
}

func (lps *InMemoryLatestPriceAPI) GetLatestPriceUpdate(currencyPair string) *LatestPriceUpdate {
	if currencyPair == "" {
		return nil
	}
	return lps.priceStore[currencyPair]
}

func (lps *InMemoryLatestPriceAPI) GetAllCurrencyPairs() map[string]bool {
	return lps.knownPairs
}
