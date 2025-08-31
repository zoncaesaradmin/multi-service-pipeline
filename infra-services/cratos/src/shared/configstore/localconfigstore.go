//go:build local
// +build local

package configstore

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
)

// matchesFilter returns true if all key-value pairs in filterMap match those in docMap.
func matchesFilter(docMap, filterMap map[string]interface{}) bool {
	for k, v := range filterMap {
		if docMap[k] != v {
			return false
		}
	}
	return true
}

// applyUpdate applies the updateMap to docMap.
func applyUpdate(docMap, updateMap map[string]interface{}) {
	for k, v := range updateMap {
		docMap[k] = v
	}
}

const (
	errCollectionNotFound = "collection not found"
	errDocumentNotFound   = "document not found"
)

type LocalFileConfigStore struct {
	filePath string
	mu       sync.RWMutex
	data     map[string]map[string]json.RawMessage // collection -> id -> document
}

func NewLocalFileConfigStore(filePath string) (ConfigStore, error) {
	store := &LocalFileConfigStore{
		filePath: filePath,
		data:     make(map[string]map[string]json.RawMessage),
	}
	if err := store.load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return store, nil
}

func (l *LocalFileConfigStore) InsertOne(ctx context.Context, collection string, document interface{}) (interface{}, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.data[collection] == nil {
		l.data[collection] = make(map[string]json.RawMessage)
	}
	b, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	var docMap map[string]interface{}
	if err := json.Unmarshal(b, &docMap); err != nil {
		return nil, err
	}
	id, ok := docMap["_id"].(string)
	if !ok || id == "" {
		return nil, errors.New("document must have a string _id field")
	}
	l.data[collection][id] = b
	if err := l.save(); err != nil {
		return nil, err
	}
	return id, nil
}

func (l *LocalFileConfigStore) FindOne(ctx context.Context, collection string, filter interface{}, result interface{}) error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	coll, ok := l.data[collection]
	if !ok {
		return errors.New(errCollectionNotFound)
	}
	f, err := json.Marshal(filter)
	if err != nil {
		return err
	}
	var filterMap map[string]interface{}
	if err := json.Unmarshal(f, &filterMap); err != nil {
		return err
	}
	for _, doc := range coll {
		var docMap map[string]interface{}
		if err := json.Unmarshal(doc, &docMap); err != nil {
			continue
		}
		if matchesFilter(docMap, filterMap) {
			return json.Unmarshal(doc, result)
		}
	}
	return errors.New(errDocumentNotFound)
}

func (l *LocalFileConfigStore) FindMany(ctx context.Context, collection string, filter interface{}, results interface{}) error {
	l.mu.RLock()
	defer l.mu.RUnlock()
	coll, ok := l.data[collection]
	if !ok {
		return errors.New(errCollectionNotFound)
	}
	f, err := json.Marshal(filter)
	if err != nil {
		return err
	}
	var filterMap map[string]interface{}
	if err := json.Unmarshal(f, &filterMap); err != nil {
		return err
	}
	var matches []json.RawMessage
	for _, doc := range coll {
		var docMap map[string]interface{}
		if err := json.Unmarshal(doc, &docMap); err != nil {
			continue
		}
		if matchesFilter(docMap, filterMap) {
			matches = append(matches, doc)
		}
	}
	b, err := json.Marshal(matches)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, results)
}

func (l *LocalFileConfigStore) UpdateOne(ctx context.Context, collection string, filter interface{}, update interface{}) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	coll, ok := l.data[collection]
	if !ok {
		return errors.New(errCollectionNotFound)
	}
	f, err := json.Marshal(filter)
	if err != nil {
		return err
	}
	var filterMap map[string]interface{}
	if err := json.Unmarshal(f, &filterMap); err != nil {
		return err
	}
	ub, err := json.Marshal(update)
	if err != nil {
		return err
	}
	var updateMap map[string]interface{}
	if err := json.Unmarshal(ub, &updateMap); err != nil {
		return err
	}
	for id, doc := range coll {
		var docMap map[string]interface{}
		if err := json.Unmarshal(doc, &docMap); err != nil {
			continue
		}
		if matchesFilter(docMap, filterMap) {
			applyUpdate(docMap, updateMap)
			newDoc, err := json.Marshal(docMap)
			if err != nil {
				return err
			}
			l.data[collection][id] = newDoc
			return l.save()
		}
	}
	return errors.New(errDocumentNotFound)
}

func (l *LocalFileConfigStore) DeleteOne(ctx context.Context, collection string, filter interface{}) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	coll, ok := l.data[collection]
	if !ok {
		return errors.New(errCollectionNotFound)
	}
	f, err := json.Marshal(filter)
	if err != nil {
		return err
	}
	var filterMap map[string]interface{}
	if err := json.Unmarshal(f, &filterMap); err != nil {
		return err
	}
	for id, doc := range coll {
		var docMap map[string]interface{}
		if err := json.Unmarshal(doc, &docMap); err != nil {
			continue
		}
		if matchesFilter(docMap, filterMap) {
			delete(l.data[collection], id)
			return l.save()
		}
	}
	return errors.New(errDocumentNotFound)
}

func (l *LocalFileConfigStore) CountDocuments(ctx context.Context, collection string, filter interface{}) (int64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	coll, ok := l.data[collection]
	if !ok {
		return 0, nil
	}
	f, err := json.Marshal(filter)
	if err != nil {
		return 0, err
	}
	var filterMap map[string]interface{}
	if err := json.Unmarshal(f, &filterMap); err != nil {
		return 0, err
	}
	var count int64
	for _, doc := range coll {
		var docMap map[string]interface{}
		if err := json.Unmarshal(doc, &docMap); err != nil {
			continue
		}
		if matchesFilter(docMap, filterMap) {
			count++
		}
	}
	return count, nil
}

func (l *LocalFileConfigStore) Close() error {
	return nil
}

func (l *LocalFileConfigStore) save() error {
	f, err := os.Create(l.filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(l.data)
}

func (l *LocalFileConfigStore) load() error {
	f, err := os.Open(l.filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(&l.data)
}
