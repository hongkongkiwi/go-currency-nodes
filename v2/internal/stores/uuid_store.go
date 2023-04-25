package stores

import (
	"fmt"
	"sync"

	"github.com/gofrs/uuid/v5"
)

var NodeSendChannelStore *InMemoryUUIDChannelInterfaceStore
var NodeCancelChannelStore *InMemoryUUIDChannelBoolStore
var CliChannelStore *InMemoryUUIDChannelInterfaceStore

type InMemoryUUIDChannelInterfaceStore struct {
	mutex sync.RWMutex
	uuid  map[string]chan interface{}
}

type InMemoryUUIDChannelBoolStore struct {
	mutex sync.RWMutex
	uuid  map[string]chan bool
}

func NewInMemoryUUIDChannelInterfaceStore() *InMemoryUUIDChannelInterfaceStore {
	return &InMemoryUUIDChannelInterfaceStore{
		uuid: make(map[string]chan interface{}),
	}
}

func NewInMemoryUUIDChannelBoolStore() *InMemoryUUIDChannelBoolStore {
	return &InMemoryUUIDChannelBoolStore{
		uuid: make(map[string]chan bool),
	}
}

// Sets a uuid from UUID reference
func (store *InMemoryUUIDChannelInterfaceStore) SetWithUUIDString(uuidString string, channel chan interface{}) error {
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

// Sets a uuid from UUID reference
func (store *InMemoryUUIDChannelBoolStore) SetWithUUIDString(uuidString string, channel chan bool) error {
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
func (store *InMemoryUUIDChannelInterfaceStore) SetWithUUID(uuid *uuid.UUID, channel chan interface{}) error {
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

// Sets a uuid from UUID string (just a helper which creates UUID)
func (store *InMemoryUUIDChannelBoolStore) SetWithUUID(uuid *uuid.UUID, channel chan bool) error {
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

func (store *InMemoryUUIDChannelInterfaceStore) GetFromString(uuidString string) (chan interface{}, error) {
	// We just check it's a valid UUID here
	_, err := uuid.FromString(uuidString)
	if err != nil {
		return nil, err
	}
	channel := store.uuid[uuidString]
	return channel, nil
}

func (store *InMemoryUUIDChannelBoolStore) GetFromString(uuidString string) (chan bool, error) {
	// We just check it's a valid UUID here
	_, err := uuid.FromString(uuidString)
	if err != nil {
		return nil, err
	}
	channel := store.uuid[uuidString]
	return channel, nil
}

func (store *InMemoryUUIDChannelInterfaceStore) GetFromUUID(uuid *uuid.UUID) (chan interface{}, error) {
	if uuid == nil {
		return nil, fmt.Errorf("nil uuid passed")
	}
	channel := store.uuid[uuid.String()]
	return channel, nil
}

func (store *InMemoryUUIDChannelBoolStore) GetFromUUID(uuid *uuid.UUID) (chan bool, error) {
	if uuid == nil {
		return nil, fmt.Errorf("nil uuid passed")
	}
	channel := store.uuid[uuid.String()]
	return channel, nil
}

func (store *InMemoryUUIDChannelInterfaceStore) HasUUIDString(uuidString string) bool {
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

func (store *InMemoryUUIDChannelBoolStore) HasUUIDString(uuidString string) bool {
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

func (store *InMemoryUUIDChannelInterfaceStore) HasUUID(uuid *uuid.UUID) (bool, error) {
	if uuid == nil {
		return false, fmt.Errorf("nil uuid passed")
	}
	if existingUuidObj := store.uuid[uuid.String()]; existingUuidObj != nil {
		return true, nil
	}
	return false, nil
}

func (store *InMemoryUUIDChannelBoolStore) HasUUID(uuid *uuid.UUID) (bool, error) {
	if uuid == nil {
		return false, fmt.Errorf("nil uuid passed")
	}
	if existingUuidObj := store.uuid[uuid.String()]; existingUuidObj != nil {
		return true, nil
	}
	return false, nil
}

func (store *InMemoryUUIDChannelInterfaceStore) DeleteString(uuidString string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	// We just check it's a valid UUID here
	if _, err := uuid.FromString(uuidString); err != nil {
		return err
	}
	delete(store.uuid, uuidString)
	return nil
}

func (store *InMemoryUUIDChannelBoolStore) DeleteString(uuidString string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	// We just check it's a valid UUID here
	if _, err := uuid.FromString(uuidString); err != nil {
		return err
	}
	delete(store.uuid, uuidString)
	return nil
}

func (store *InMemoryUUIDChannelInterfaceStore) DeleteUUID(uuid *uuid.UUID) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if uuid == nil {
		return fmt.Errorf("nil uuid passed")
	}
	delete(store.uuid, uuid.String())
	return nil
}

func (store *InMemoryUUIDChannelBoolStore) DeleteUUID(uuid *uuid.UUID) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if uuid == nil {
		return fmt.Errorf("nil uuid passed")
	}
	delete(store.uuid, uuid.String())
	return nil
}

func (store *InMemoryUUIDChannelInterfaceStore) KeysAsUUID() []*uuid.UUID {
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

func (store *InMemoryUUIDChannelBoolStore) KeysAsUUID() []*uuid.UUID {
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

func (store *InMemoryUUIDChannelInterfaceStore) KeysAsString() []string {
	keysStrArr := make([]string, len(store.uuid))
	i := 0
	for keyUUID := range store.uuid {
		keysStrArr[i] = keyUUID
		i++
	}
	return keysStrArr
}

func (store *InMemoryUUIDChannelBoolStore) KeysAsString() []string {
	keysStrArr := make([]string, len(store.uuid))
	i := 0
	for keyUUID := range store.uuid {
		keysStrArr[i] = keyUUID
		i++
	}
	return keysStrArr
}

type CliReplyCount struct {
	mutex  sync.RWMutex
	counts map[string]uint32
}

func NewCliReplyCount() *CliReplyCount {
	return &CliReplyCount{
		counts: make(map[string]uint32),
	}
}

func (c *CliReplyCount) Add(uuidString string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	existing := c.counts[uuidString]
	existing++
	fmt.Printf("added existing %d\n", existing)
	c.counts[uuidString] = existing
}

func (c *CliReplyCount) Clear(uuidString string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.counts[uuidString] = 0
}

func (c *CliReplyCount) Get(uuidString string) uint32 {
	return c.counts[uuidString]
}

func (c *CliReplyCount) Delete(uuidString string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.counts, uuidString)
}
