package main

import (
	"errors"
	"sync"
)

// Structures
type KeyValStoreDatabase struct {
	sync.Mutex
	Data         map[string]interface{} `json:"data"`
	Metadata     map[string]int         `json:"metadata"`
	LocalAddress string                 `json:"localAddress"`
}

// Errors
var ErrKeyNotFound = errors.New("key not found")
var ErrInvalidMetadata = errors.New("cannot accept metadata")

// Constructor
func NewKeyValStoreDatabase(localAdd string) *KeyValStoreDatabase {
	return &KeyValStoreDatabase{
		Data:         make(map[string]interface{}),
		Metadata:     make(map[string]int),
		LocalAddress: localAdd,
	}
}

// Gets a key from the kvs
func (kvs *KeyValStoreDatabase) GetData(key string, metadata map[string]int) (value interface{}, currentMetadata map[string]int, err error) {
	// Lock Data
	kvs.Lock()
	defer kvs.Unlock()

	// Check metadata
	metadataValid := kvs.IsMetadataValid(metadata, kvs.LocalAddress)
	if !metadataValid {
		return nil, nil, ErrInvalidMetadata
	}

	// Check if key exists in map
	value, existed := kvs.Data[key]
	if !existed {
		return nil, nil, ErrKeyNotFound
	}

	// Make copy of metadata before unlocking
	currentMetadata = kvs.copyMetadata()

	// return value and metadata
	return value, currentMetadata, nil
}

// Adds Data to the kvs
func (kvs *KeyValStoreDatabase) PutData(key string, value interface{}, metadata map[string]int, sender string) (wasCreated bool, currentMetadata map[string]int, err error) {
	// Lock Database
	kvs.Lock()
	defer kvs.Unlock()

	metadataValid := kvs.IsMetadataValid(metadata, sender)
	if !metadataValid {
		return false, kvs.copyMetadata(), ErrInvalidMetadata
	}

	// Check if key already exists in map
	_, wasCreated = kvs.Data[key]

	// Add data to map
	kvs.Data[key] = value

	// Update metadata
	kvs.incrementMetadata(sender)

	// Make copy of metadata before unlocking
	currentMetadata = kvs.copyMetadata()

	// return success and wasCreated
	return !wasCreated, currentMetadata, nil
}

// exactly the same as PutData, but it doesn't actually stor the data
func (kvs *KeyValStoreDatabase) putJustMetadata(metadata map[string]int, sender string) error {
	// Lock Database
	kvs.Lock()
	defer kvs.Unlock()

	// check if metadata valid
	metadataValid := kvs.IsMetadataValid(metadata, sender)
	if !metadataValid {
		return ErrInvalidMetadata
	}

	kvs.incrementMetadata(sender)
	return nil
}

func (kvs *KeyValStoreDatabase) PutDataNoChecks(key string, value interface{}) {
	kvs.Data[key] = value
}

// Deletes Data from kvs
func (kvs *KeyValStoreDatabase) DeleteData(key string, metadata map[string]int, sender string) (currentMetadata map[string]int, err error) {
	// Lock Data
	kvs.Lock()
	defer kvs.Unlock()

	// Check metadata
	metadataValid := kvs.IsMetadataValid(metadata, sender)
	if !metadataValid {
		return kvs.copyMetadata(), ErrInvalidMetadata
	}

	// Check if key exists in map
	_, existed := kvs.Data[key]
	if !existed {
		return kvs.copyMetadata(), ErrKeyNotFound
	}

	// Delete data from map
	delete(kvs.Data, key)

	// Update metadata in senders position
	kvs.incrementMetadata(sender)

	// Make copy of metadata before unlocking
	currentMetadata = kvs.copyMetadata()

	// return metadata
	return currentMetadata, nil

}

// TODO: refactor this to make non-existant values = 0 when comparing
func (kvs *KeyValStoreDatabase) IsMetadataValid(incomingMetadata map[string]int, sender string) bool {
	if sender == kvs.LocalAddress {
		for replica, time := range incomingMetadata {
			if time > kvs.Metadata[replica] {
				return false
			}
		}
		return true
	} else {
		if incomingMetadata[sender] != (kvs.Metadata[sender] + 1) {
			return false
		}
		for replica, time := range incomingMetadata {
			if replica != sender && time > kvs.Metadata[replica] {
				return false
			}
		}
		return true
	}
}

func (kvs *KeyValStoreDatabase) incrementMetadata(sender string) {
	kvs.Metadata[sender] = kvs.Metadata[sender] + 1
}

func (kvs *KeyValStoreDatabase) copyMetadata() map[string]int {
	copy := make(map[string]int)

	for key, val := range kvs.Metadata {
		copy[key] = val
	}

	return copy
}
