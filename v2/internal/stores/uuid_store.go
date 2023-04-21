package stores

import (
	"fmt"
	"sync"

	"github.com/gofrs/uuid/v5"
)

type InMemoryUUIDStore struct {
	mutex sync.RWMutex
	uuid  map[string]chan interface{}
}

func NewInMemoryUUIDStore() *InMemoryUUIDStore {
	return &InMemoryUUIDStore{
		uuid: make(map[string]chan interface{}),
	}
}

// Sets a uuid from UUID reference
func (store *InMemoryUUIDStore) SetWithUUIDString(uuidString string, channel chan interface{}) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	// Check for valid UUID
	_, err := uuid.FromString(uuidString)
	if err != nil {
		return err
	}
	if channel == nil {
		return fmt.Errorf("cannot store empty channel")
	}
	existingChan := store.uuid[uuidString]
	if existingChan == nil {
		store.uuid[uuidString] = channel
	} else if existingChan != channel {
		close(channel)
		store.uuid[uuidString] = channel
	}
	return nil
}

// Sets a uuid from UUID string (just a helper which creates UUID)
func (store *InMemoryUUIDStore) SetWithUUID(uuid *uuid.UUID, channel chan interface{}) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()

	if channel == nil {
		return fmt.Errorf("cannot store empty channel")
	}
	existingChan := store.uuid[uuid.String()]
	if existingChan == nil {
		store.uuid[uuid.String()] = channel
	} else if existingChan != channel {
		close(channel)
		store.uuid[uuid.String()] = channel
	}
	return nil
}

func (store *InMemoryUUIDStore) GetFromString(uuidString string) (chan interface{}, error) {
	// We just check it's a valid UUID here
	_, err := uuid.FromString(uuidString)
	if err != nil {
		return nil, err
	}
	channel := store.uuid[uuidString]
	return channel, nil
}

func (store *InMemoryUUIDStore) GetFromUUID(uuid *uuid.UUID) (chan interface{}, error) {
	if uuid == nil {
		return nil, fmt.Errorf("nil uuid passed")
	}
	channel := store.uuid[uuid.String()]
	return channel, nil
}

func (store *InMemoryUUIDStore) HasUUIDString(uuidString string) bool {
	// We just check it's a valid UUID here
	_, err := uuid.FromString(uuidString)
	if err != nil {
		return false
	}
	if existingUuidObj := store.uuid[uuidString]; existingUuidObj != nil {
		return true
	}
	return false
}

func (store *InMemoryUUIDStore) HasUUID(uuid *uuid.UUID) (bool, error) {
	if uuid == nil {
		return false, fmt.Errorf("nil uuid passed")
	}
	if existingUuidObj := store.uuid[uuid.String()]; existingUuidObj != nil {
		return true, nil
	}
	return false, nil
}

func (store *InMemoryUUIDStore) DeleteString(uuidString string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	// We just check it's a valid UUID here
	if _, err := uuid.FromString(uuidString); err != nil {
		return err
	}
	delete(store.uuid, uuidString)
	return nil
}

func (store *InMemoryUUIDStore) DeleteUUID(uuid *uuid.UUID) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if uuid == nil {
		return fmt.Errorf("nil uuid passed")
	}
	delete(store.uuid, uuid.String())
	return nil
}

func (store *InMemoryUUIDStore) KeysAsUUID() []*uuid.UUID {
	keysUUIDArr := make([]*uuid.UUID, len(store.uuid))
	i := 0
	for keyStr := range store.uuid {
		if keyUUID, err := uuid.FromString(keyStr); err != nil {
			keysUUIDArr[i] = &keyUUID
			i++
		}
	}
	return keysUUIDArr
}

func (store *InMemoryUUIDStore) KeysAsString() []string {
	keysStrArr := make([]string, len(store.uuid))
	i := 0
	for keyUUID := range store.uuid {
		keysStrArr[i] = keyUUID
		i++
	}
	return keysStrArr
}
