/**
 * Stores which nodes are streaming which currency pairs
 */
package stores

import (
	"math/rand"
	"sync"
	"time"
)

var ControllerNodeCurrencies *inMemoryNodeCurrenciesStore

type inMemoryNodeCurrenciesStore struct {
	mutex      sync.RWMutex
	currencies map[string][]string // map[node_uuid][]currency_pairs
	streaming  map[string][]string // map[curreny_pair][]node_uuid
	seededRand *rand.Rand
}

func NewNodeCurrenciesStore() *inMemoryNodeCurrenciesStore {
	ncs := &inMemoryNodeCurrenciesStore{
		currencies: make(map[string][]string),
		streaming:  make(map[string][]string),
		seededRand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	return ncs
}

// This is only called once at node connect to populate streaming currencies
func (ncs *inMemoryNodeCurrenciesStore) SetNodeCurrencies(nodeUuid string, currencyPairs []string) {
	ncs.mutex.Lock()
	defer ncs.mutex.Unlock()
	ncs.currencies[nodeUuid] = uniqueStrings(currencyPairs)
	// Edge case when we update our node currencies for some reason
	// But a currency which was streaming before has been removed
	for currencyPair, uuids := range ncs.streaming {
		// If this streaming currency pair is not part of our currencyPairs for this node
		if !containsString(currencyPairs, currencyPair) {
			// Remove this node from the streaming list
			ncs.streaming[currencyPair] = removeString(uuids, nodeUuid)
		}
	}
}

// This is called anytime we ask a node to start streaming
func (ncs *inMemoryNodeCurrenciesStore) SetNodeStreamingCurrency(nodeUuid string, currencyPair string) {
	existing := ncs.streaming[currencyPair]
	ncs.mutex.Lock()
	defer ncs.mutex.Unlock()
	ncs.streaming[currencyPair] = append(existing, nodeUuid)
}

func (ncs *inMemoryNodeCurrenciesStore) SetNodeStreamingCurrencies(nodeUuid string, currencyPairs []string) {
	for _, currencyPair := range currencyPairs {
		ncs.SetNodeStreamingCurrency(nodeUuid, currencyPair)
	}
}

func (ncs *inMemoryNodeCurrenciesStore) CountNodeStreamingCurrency(currencyPair string) uint {
	return uint(len(ncs.streaming[currencyPair]))
}

// This is called anytime we ask a node to start streaming
func (ncs *inMemoryNodeCurrenciesStore) DeleteNodeStreaming(nodeUuid string, currencyPair string) {
	ncs.mutex.Lock()
	defer ncs.mutex.Unlock()
	ncs.streaming[currencyPair] = removeString(ncs.streaming[currencyPair], nodeUuid)
}

func (ncs *inMemoryNodeCurrenciesStore) GetNodeCurrencies(nodeUuid string) []string {
	return ncs.currencies[nodeUuid]
}

func (ncs *inMemoryNodeCurrenciesStore) GetAllCurrencies() []string {
	uniqueCurrencies := make(map[string]bool)
	for _, nodeCurrencies := range ncs.currencies {
		for _, nodeCurrency := range nodeCurrencies {
			uniqueCurrencies[nodeCurrency] = true
		}
	}
	returnCurrencies := make([]string, 0)
	for currency := range uniqueCurrencies {
		returnCurrencies = append(returnCurrencies, currency)
	}
	return returnCurrencies
}

// Get a random number of nodes to stream for a currency pair
func (ncs *inMemoryNodeCurrenciesStore) GetRandomNodesForCurrency(currencyPair string, excludeExistingStreamingNodes bool, minItemCount uint) []string {
	returnItems := make([]string, 0)
	if minItemCount == 0 {
		return returnItems
	}
	uuidLottery := make([]string, 0)
	for nodeUuid, currencies := range ncs.currencies {
		if containsString(currencies, currencyPair) &&
			excludeExistingStreamingNodes &&
			!containsString(ncs.streaming[currencyPair], nodeUuid) {
			uuidLottery = append(uuidLottery, nodeUuid)
		} else if containsString(currencies, currencyPair) &&
			!excludeExistingStreamingNodes {
			uuidLottery = append(uuidLottery, nodeUuid)
		}
	}
	for i := 0; i < int(minItemCount) && len(uuidLottery) > 0; i++ {
		n := ncs.seededRand.Int() % len(uuidLottery)
		returnItems = append(returnItems, uuidLottery[n])
		uuidLottery = removeIndex(uuidLottery, n)
	}
	return returnItems
}

func (ncs *inMemoryNodeCurrenciesStore) DeleteNode(nodeUuid string) {
	ncs.mutex.Lock()
	defer ncs.mutex.Unlock()
	delete(ncs.currencies, nodeUuid)
	for currencyPair, uuids := range ncs.streaming {
		ncs.streaming[currencyPair] = removeString(uuids, nodeUuid)
	}
}

func containsString(s []string, wantedItem string) bool {
	for _, item := range s {
		if item == wantedItem {
			return true
		}
	}
	return false
}

func removeIndex(s []string, index int) []string {
	return append(s[:index], s[index+1:]...)
}

func removeString(s []string, removeItem string) []string {
	for i, item := range s {
		if item == removeItem {
			s = removeIndex(s, i)
		}
	}
	return s
}

func uniqueStrings(s []string) []string {
	// First turn the knownCurrencyPairs into a map, this removes duplicates
	unique := make(map[string]bool)
	for _, item := range s {
		unique[item] = true
	}
	output := make([]string, 0)
	for key := range unique {
		output = append(output, key)
	}
	return output
}
