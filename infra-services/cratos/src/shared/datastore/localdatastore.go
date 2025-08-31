//go:build local
// +build local

package datastore

import (
	context "context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// LocalDatastoreClient is a file-based implementation of OpenSearchClient for local/testing use
// Data is stored in a file: index -> id -> document
// Persistence and concurrency safety provided
type LocalDatastoreClient struct {
	filePath string
	mu       sync.RWMutex
	data     map[string]map[string]json.RawMessage
}

func NewLocalDatastoreClient(filePath string) *LocalDatastoreClient {
	c := &LocalDatastoreClient{
		filePath: filePath,
		data:     make(map[string]map[string]json.RawMessage),
	}
	_ = c.load()
	return c
}

func (c *LocalDatastoreClient) Index(ctx context.Context, index, id string, body interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data[index] == nil {
		c.data[index] = make(map[string]json.RawMessage)
	}
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	c.data[index][id] = b
	return c.save()
}

func (c *LocalDatastoreClient) BulkIndex(ctx context.Context, index string, docs []BulkDoc) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data[index] == nil {
		c.data[index] = make(map[string]json.RawMessage)
	}
	for _, doc := range docs {
		if doc.Action == "delete" {
			delete(c.data[index], doc.ID)
			continue
		}
		b, err := json.Marshal(doc.Body)
		if err != nil {
			return err
		}
		c.data[index][doc.ID] = b
	}
	return c.save()
}

func (c *LocalDatastoreClient) Get(ctx context.Context, index, id string, out interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if err := c.load(); err != nil {
		return err
	}
	if c.data[index] == nil {
		return fmt.Errorf("index not found")
	}
	b, ok := c.data[index][id]
	if !ok {
		return fmt.Errorf("document not found")
	}
	return json.Unmarshal(b, out)
}

func (c *LocalDatastoreClient) Search(ctx context.Context, index string, query interface{}, page, pageSize int, out interface{}) (total int, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if err := c.load(); err != nil {
		return 0, err
	}
	if c.data[index] == nil {
		return 0, nil
	}
	var items []json.RawMessage
	for _, b := range c.data[index] {
		items = append(items, b)
	}
	total = len(items)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	paged := items[start:end]
	pagedBytes, err := json.Marshal(paged)
	if err != nil {
		return total, err
	}
	return total, json.Unmarshal(pagedBytes, out)
}

func (c *LocalDatastoreClient) Delete(ctx context.Context, index, id string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.data[index] == nil {
		return fmt.Errorf("index not found")
	}
	delete(c.data[index], id)
	return c.save()
}

func (c *LocalDatastoreClient) ScrollQuery(ctx context.Context, index string, query interface{}, pageSize int, callback func([]json.RawMessage) bool) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if err := c.load(); err != nil {
		return err
	}
	if c.data[index] == nil {
		return nil
	}
	var items []json.RawMessage
	for _, b := range c.data[index] {
		items = append(items, b)
	}
	for i := 0; i < len(items); i += pageSize {
		end := i + pageSize
		if end > len(items) {
			end = len(items)
		}
		batch := items[i:end]
		if !callback(batch) {
			break
		}
	}
	return nil
}

func (c *LocalDatastoreClient) Close() error {
	return nil
}

func (c *LocalDatastoreClient) save() error {
	f, err := os.Create(c.filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(c.data)
}

func (c *LocalDatastoreClient) load() error {
	f, err := os.Open(c.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()
	return json.NewDecoder(f).Decode(&c.data)
}
