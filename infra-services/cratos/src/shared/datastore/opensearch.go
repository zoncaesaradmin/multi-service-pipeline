package datastore

import (
	context "context"
	"encoding/json"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/disaster37/opensearch/v2"
)

type OpenSearchClientImpl struct {
	client *opensearch.Client
}

// NewOpenSearchClient creates a new OpenSearch client.
// Example usage:
//
//	client, err := NewOpenSearchClient(opensearch.SetURL("http://localhost:9200"))
func NewOpenSearchClient(options ...opensearch.ClientOptionFunc) (OpenSearchClient, error) {
	client, err := opensearch.NewClient(options...)
	if err != nil {
		return nil, err
	}
	return &OpenSearchClientImpl{client: client}, nil
}

func (c *OpenSearchClientImpl) Index(ctx context.Context, index, id string, body interface{}) error {
	resp, err := c.client.Index().Index(index).Id(id).BodyJson(body).Do(ctx)
	if err != nil {
		return err
	}
	if resp.Result != "created" && resp.Result != "updated" {
		return fmt.Errorf("index error: %v", resp.Result)
	}
	return nil
}

func (c *OpenSearchClientImpl) BulkIndex(ctx context.Context, index string, docs []BulkDoc) error {
	bulk := c.client.Bulk()
	for _, doc := range docs {
		switch doc.Action {
		case "index", "create":
			req := opensearch.NewBulkIndexRequest().Index(index).Doc(doc.Body)
			if doc.ID != "" {
				req.Id(doc.ID)
			}
			bulk.Add(req)
		case "delete":
			req := opensearch.NewBulkDeleteRequest().Index(index)
			if doc.ID != "" {
				req.Id(doc.ID)
			}
			bulk.Add(req)
		}
	}
	resp, err := bulk.Do(ctx)
	if err != nil {
		return err
	}
	if resp.Errors {
		return fmt.Errorf("bulk error: %v", resp)
	}
	return nil
}

func (c *OpenSearchClientImpl) Get(ctx context.Context, index, id string, out interface{}) error {
	resp, err := c.client.Get().Index(index).Id(id).Do(ctx)
	if err != nil {
		return err
	}
	if !resp.Found {
		return fmt.Errorf("document not found")
	}
	return sonic.Unmarshal(resp.Source, out)
}

func (c *OpenSearchClientImpl) Search(ctx context.Context, index string, query interface{}, page, pageSize int, out interface{}) (total int, err error) {
	from := (page - 1) * pageSize
	resp, err := c.client.Search(index).
		From(from).
		Size(pageSize).
		Source(query).
		Do(ctx)
	if err != nil {
		return 0, err
	}
	total = int(resp.Hits.TotalHits.Value)
	var items []json.RawMessage
	for _, hit := range resp.Hits.Hits {
		items = append(items, hit.Source)
	}
	data, err := sonic.Marshal(items)
	if err != nil {
		return total, err
	}
	return total, sonic.Unmarshal(data, out)
}

func (c *OpenSearchClientImpl) Delete(ctx context.Context, index, id string) error {
	resp, err := c.client.Delete().Index(index).Id(id).Do(ctx)
	if err != nil {
		return err
	}
	if resp.Result != "deleted" {
		return fmt.Errorf("delete error: %v", resp.Result)
	}
	return nil
}

func (c *OpenSearchClientImpl) Close() error {
	// opensearch.Client does not require explicit close, but implement if needed
	return nil
}

// ScrollQuery executes a query with scroll and invokes callback for each batch of results
func (c *OpenSearchClientImpl) ScrollQuery(ctx context.Context, index string, query interface{}, pageSize int, callback func([]json.RawMessage) bool) error {
	// Marshal query to JSON and set as request body
	body, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("failed to marshal query: %w", err)
	}
	scroll := c.client.Scroll(index).Size(pageSize).Body(string(body))
	for {
		resp, err := scroll.Do(ctx)
		if err != nil {
			if err == opensearch.ErrNoClient {
				return fmt.Errorf("no opensearch client available")
			}
			if resp != nil && resp.ScrollId != "" {
				c.client.ClearScroll().ScrollId(resp.ScrollId).Do(ctx)
			}
			// Instead of returning error, break on EOF for mock compatibility
			if err.Error() == "EOF" {
				break
			}
			return err
		}
		if len(resp.Hits.Hits) == 0 {
			break
		}
		var items []json.RawMessage
		for _, hit := range resp.Hits.Hits {
			items = append(items, hit.Source)
		}
		if !callback(items) {
			break
		}
		if resp.ScrollId == "" {
			break
		}
		scroll = c.client.Scroll().ScrollId(resp.ScrollId)
	}
	return nil
}
