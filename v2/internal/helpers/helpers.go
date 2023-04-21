package helpers

import (
	"fmt"

	"github.com/gofrs/uuid/v5"
)

// Helper method to generate an array of valid uuids
func UuidsFromStrings(uuidStrings []string) ([]*uuid.UUID, []error) {
	// Use a list so we can deduplicate any duplicates
	uuidsMap := make(map[string]*uuid.UUID, len(uuidStrings))
	errors := make([]error, 0)
	for _, uuidString := range uuidStrings {
		uuid, err := uuid.FromString(uuidString)
		if err != nil {
			errors = append(errors, fmt.Errorf("invalid uuid %s", uuidString))
			continue
		}
		uuidsMap[uuidString] = &uuid
	}
	// Build our array for returning
	uuidsArr := make([]*uuid.UUID, len(uuidStrings))
	i := 0
	for _, uuid := range uuidsMap {
		uuidsArr[i] = uuid
		i++
	}
	return uuidsArr, errors
}

func StringsFromUuids(uuids []*uuid.UUID) []string {
	// Use a list so we can deduplicate any duplicates
	uuidsMap := make(map[string]string, len(uuids))
	for _, uuid := range uuids {
		uuidsMap[uuid.String()] = uuid.String()
	}
	// Build our array for returning
	uuidsArr := make([]string, len(uuids))
	i := 0
	for _, uuid := range uuidsMap {
		uuidsArr[i] = uuid
		i++
	}
	return uuidsArr
}
