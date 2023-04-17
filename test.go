/**
 * This file provides a nice wrapper to abstract around badgerdb
 * since badgerdb only stores bytes - we will marshel and unmarshell using
 * BSON which is very efficient. We could also use JSON but it's a bit
 * inefficient.
 **/
package main

import (
	"fmt"
	"time"

	"path/filepath"

	badger "github.com/dgraph-io/badger/v3"
	"github.com/gofrs/uuid/v5"
	"go.mongodb.org/mongo-driver/bson"
)

type CurrencyStore struct {
	store *badger.DB
}

// This is what is stored inside our local datastore to represent a currency value
type CurrencyStoreItem struct {
	Price         float64   `bson:"price"`
	ValidAt       time.Time `bson:"valid_at"`
	UpdatedByUUID uuid.UUID `bson:"updated_by_uuid"`
}

type SubscriptionStore struct {
	store *badger.DB
}

type SubscriptionStoreItem struct {
	UUIDs []string `bson:"uuids"`
}

const defaultStoreDir = "/tmp/currency"

func NewDiskCurrencyStore(storeDir, storeName string) (*CurrencyStore, error) {
	cs := &CurrencyStore{}
	cs.openDisk(storeDir, storeName)
	return cs, nil
}

func NewMemoryCurrencyStore() (*CurrencyStore, error) {
	cs := &CurrencyStore{}
	cs.openMem()
	return cs, nil
}

func NewDiskSubscriptionStore(storeDir, storeName string) (*SubscriptionStore, error) {
	cs := &SubscriptionStore{}
	cs.openDisk(storeDir, storeName)
	return cs, nil
}

func NewMemorySubscriptionStore() (*SubscriptionStore, error) {
	cs := &SubscriptionStore{}
	cs.openMem()
	return cs, nil
}

func (cs *CurrencyStore) IsClosed() bool {
	if cs.store == nil || cs.store.IsClosed() {
		return true
	}
	return false
}

func (cs *CurrencyStore) openDisk(storeDir string, storeName string) error {
	if storeDir == "" {
		storeDir = defaultStoreDir
	}
	if storeName == "" {
		return fmt.Errorf("missing store name")
	}
	// Open the Badger database located in the /tmp/badger directory.
	// It will be created if it doesn't exist.
	var currStoreErr error
	opt := badger.DefaultOptions(filepath.Join(storeDir, storeName))
	cs.store, currStoreErr = badger.Open(opt)
	if currStoreErr != nil {
		return currStoreErr
	}
	return nil
}

func (cs *CurrencyStore) openMem() error {
	// Open the Badger database located in the /tmp/badger directory.
	// It will be created if it doesn't exist.
	var currStoreErr error
	opt := badger.DefaultOptions("").WithInMemory(true)
	cs.store, currStoreErr = badger.Open(opt)
	if currStoreErr != nil {
		return currStoreErr
	}
	return nil
}

func (cs *CurrencyStore) Close() {
	if cs.store != nil {
		cs.store.Close()
	}
}

// Sets a key in our store by taking our currency store item and marshelling to BSON
// then we store into KV store
func (cs *CurrencyStore) Set(currencyPair string, currencyStoreItem *CurrencyStoreItem) error {
	if cs.store == nil {
		return fmt.Errorf("currency store is nil")
	} else if cs.store.IsClosed() {
		return fmt.Errorf("currency store closed")
	}
	byteValue, marshelErr := currencyStoreItem.marshelValue()
	if marshelErr != nil {
		return marshelErr
	}
	updateErr := cs.store.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(currencyPair), byteValue)
		return err
	})
	return updateErr
}

// Get a key in our store by taking our key and unmarshelling to BSON
// then we store into KV store
func (cs *CurrencyStore) Get(currencyPair string) (*CurrencyStoreItem, error) {
	if cs.store == nil {
		return nil, fmt.Errorf("currency store is nil")
	} else if cs.store.IsClosed() {
		return nil, fmt.Errorf("currency store closed")
	}
	var valCopy []byte
	viewErr := cs.store.View(func(txn *badger.Txn) error {
		badgerItem, getErr := txn.Get([]byte(currencyPair))
		if getErr != nil {
			return getErr
		}
		var valErr error
		valCopy, valErr = badgerItem.ValueCopy(nil)
		if valErr != nil {
			return valErr
		}
		return nil
	})
	if viewErr != nil {
		return nil, viewErr
	}
	currencyItem, unmarshelErr := marshelCurrencyStoreItemFromValue(valCopy)
	if unmarshelErr != nil {
		return nil, unmarshelErr
	}
	return currencyItem, nil
}

// Remove an item from the currency store
func (cs *CurrencyStore) Delete(currencyPair string) error {
	if cs.store == nil {
		return fmt.Errorf("currency store is nil")
	} else if cs.store.IsClosed() {
		return fmt.Errorf("currency store closed")
	}
	updateErr := cs.store.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(currencyPair))
		return err
	})
	return updateErr
}

// Return a count of all keys in our db
func (cs *CurrencyStore) Count() (uint32, error) {
	var keyCount uint32
	err := cs.store.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			keyCount++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return keyCount, nil
}

// Get an array of all keys in the currency store
// This is a bit inefficient as we are basically
// shoving all keys into memory but it's ok in
// our case as our key list is just currency pair names
func (cs *CurrencyStore) Keys() ([]string, error) {
	keyCount, _ := cs.Count()
	keysArr := make([]string, keyCount)
	err := cs.store.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		i := 0
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			keysArr[i] = string(k)
			i++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keysArr, nil
}

// Get an array of all values in the currency store
func (cs *CurrencyStore) Values() ([]*CurrencyStoreItem, error) {
	keyCount, _ := cs.Count()
	valuesArr := make([]*CurrencyStoreItem, keyCount)
	err := cs.store.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		i := 0
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				if storeItem, err := marshelCurrencyStoreItemFromValue(v); err != nil {
					valuesArr[i] = storeItem
				}
				return nil
			})
			if err != nil {
				return err
			}
			i++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return valuesArr, nil
}

func (ss *SubscriptionStore) IsClosed() bool {
	if ss.store == nil || ss.store.IsClosed() {
		return true
	}
	return false
}

func (ss *SubscriptionStore) openDisk(storeDir string, storeName string) error {
	if storeDir == "" {
		storeDir = defaultStoreDir
	}
	if storeName == "" {
		return fmt.Errorf("missing store name")
	}
	// Open the Badger database located in the /tmp/badger directory.
	// It will be created if it doesn't exist.
	var currStoreErr error
	opt := badger.DefaultOptions(filepath.Join(storeDir, storeName))
	ss.store, currStoreErr = badger.Open(opt)
	if currStoreErr != nil {
		return currStoreErr
	}
	return nil
}

func (ss *SubscriptionStore) openMem() error {
	// Open the Badger database located in the /tmp/badger directory.
	// It will be created if it doesn't exist.
	var currStoreErr error
	opt := badger.DefaultOptions("").WithInMemory(true)
	ss.store, currStoreErr = badger.Open(opt)
	if currStoreErr != nil {
		return currStoreErr
	}
	return nil
}

func (ss *SubscriptionStore) Close() {
	if ss.store != nil {
		ss.store.Close()
	}
}

// Sets a key in our store by taking our currency store item and marshelling to BSON
// then we store into KV store
func (ss *SubscriptionStore) Set(currencyPair string, subscriptionStoreItem *SubscriptionStoreItem) error {
	if ss.store == nil {
		return fmt.Errorf("subscription store is nil")
	} else if ss.store.IsClosed() {
		return fmt.Errorf("subscription store closed")
	}
	byteValue, marshelErr := subscriptionStoreItem.marshelValue()
	if marshelErr != nil {
		return marshelErr
	}
	updateErr := ss.store.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(currencyPair), byteValue)
		return err
	})
	return updateErr
}

// Get a key in our store by taking our key and unmarshelling to BSON
// then we store into KV store
func (ss *SubscriptionStore) Get(currencyPair string) (*SubscriptionStoreItem, error) {
	if ss.store == nil {
		return nil, fmt.Errorf("subscription store is nil")
	} else if ss.store.IsClosed() {
		return nil, fmt.Errorf("subscription store closed")
	}
	var valCopy []byte
	viewErr := ss.store.View(func(txn *badger.Txn) error {
		badgerItem, getErr := txn.Get([]byte(currencyPair))
		if getErr != nil {
			return getErr
		}
		var valErr error
		valCopy, valErr = badgerItem.ValueCopy(nil)
		if valErr != nil {
			return valErr
		}
		return nil
	})
	if viewErr != nil {
		return nil, viewErr
	}
	currencyItem, unmarshelErr := marshelSubscriptionStoreItemFromValue(valCopy)
	if unmarshelErr != nil {
		return nil, unmarshelErr
	}
	return currencyItem, nil
}

// Return a count of all keys in our db
func (ss *SubscriptionStore) Count() (uint32, error) {
	var keyCount uint32
	err := ss.store.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			keyCount++
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return keyCount, nil
}

// Remove an item from the subscription store
func (ss *SubscriptionStore) Delete(currencyPair string) error {
	if ss.store == nil {
		return fmt.Errorf("subscription store is nil")
	} else if ss.store.IsClosed() {
		return fmt.Errorf("subscription store closed")
	}
	updateErr := ss.store.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(currencyPair))
		return err
	})
	return updateErr
}

// Get an array of all keys in the subscription store
// This is a bit inefficient as we are basically
// shoving all keys into memory but it's ok in
// our case as our key list is just currency pair names
func (ss *SubscriptionStore) Keys() ([]string, error) {
	keyCount, _ := ss.Count()
	keysArr := make([]string, keyCount)
	err := ss.store.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		i := 0
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			keysArr[i] = string(k)
			i++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return keysArr, nil
}

// Get an array of all values in the subscription store
func (ss *SubscriptionStore) Values() ([]*SubscriptionStoreItem, error) {
	keyCount, _ := ss.Count()
	valuesArr := make([]*SubscriptionStoreItem, keyCount)
	err := ss.store.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		i := 0
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				if storeItem, err := marshelSubscriptionStoreItemFromValue(v); err != nil {
					valuesArr[i] = storeItem
				}
				return nil
			})
			if err != nil {
				return err
			}
			i++
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return valuesArr, nil
}

// Marshel a currency store item from BSON
func (csi *CurrencyStoreItem) marshelValue() ([]byte, error) {
	_, data, err := bson.MarshalValue(csi)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Marshel a currency store item from BSON
func marshelCurrencyStoreItemFromValue(data []byte) (*CurrencyStoreItem, error) {
	newCsi := &CurrencyStoreItem{}
	err := bson.Unmarshal(data, &newCsi)
	if err != nil {
		return nil, err
	}
	return newCsi, nil
}

// Marshel a subscription store item to BSON
func (ssi *SubscriptionStoreItem) marshelValue() ([]byte, error) {
	_, data, err := bson.MarshalValue(ssi)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Unmarshel a subscription store item from BSON
func marshelSubscriptionStoreItemFromValue(data []byte) (*SubscriptionStoreItem, error) {
	newSsi := &SubscriptionStoreItem{}
	err := bson.Unmarshal(data, &newSsi)
	if err != nil {
		return nil, err
	}
	return newSsi, nil
}

func main() {
	cs, csErr := NewDiskCurrencyStore("/tmp/currency", "cs")
	if csErr != nil {
		fmt.Printf("%v", csErr)
	}
	csi, getErr := cs.Get("ABC")
	if getErr != nil {
		fmt.Printf("error getting %v\n", getErr)
	}
	fmt.Printf("Got Object: %v\n", csi)

	setErr := cs.Set("ABC2", &CurrencyStoreItem{
		Price:   15.00,
		ValidAt: time.Now(),
	})
	if setErr != nil {
		fmt.Printf("error setting %s\n", setErr)
	}
	setErr = cs.Set("ABC3", &CurrencyStoreItem{
		Price:   42,
		ValidAt: time.Now(),
	})
	if setErr != nil {
		fmt.Printf("error setting %s\n", setErr)
	}

	csi, getErr = cs.Get("ABC2")
	if getErr != nil {
		fmt.Printf("error getting %v\n", getErr)
	}
	fmt.Printf("Got Object: %v\n", csi)

	delErr := cs.Delete("ABC3")
	if delErr != nil {
		fmt.Printf("error deleting %v\n", delErr)
	}
	delErr = cs.Delete("ABC4")
	if delErr != nil {
		fmt.Printf("error deleting %v\n", delErr)
	}

	count, countErr := cs.Count()
	if countErr != nil {
		fmt.Printf("error counting %v\n", countErr)
	}
	fmt.Printf("Object Count: %v\n", count)
	keys, keysErr := cs.Keys()
	if keysErr != nil {
		fmt.Printf("error getting keys %v\n", keysErr)
	}
	fmt.Printf("Store Keys: %v\n", keys)
	// ss, ssErr := NewAndOpenSubscriptionStore("/tmp/currency", "ss")
	// if ssErr != nil {
	// 	fmt.Printf("%v", ssErr)
	// }
}
