//go:build !local
// +build !local

package datastore

import (
	"context"
	"corekit/configutil"
	"corekit/logging"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	elastic "github.com/disaster37/opensearch/v2"
)

// Config represents the Opensearch configuration
type Config struct {
	URLs          []string `yaml:"urls"`
	SSLCALocation string   `yaml:"ssl.ca.location"`
}

type OpenSearchClient struct {
	client             *elastic.Client
	logger             logging.Logger
	httpClient         *http.Client
	url                string
	urls               []string
	isSecure           bool
	credsRefreshTimer  *time.Timer
	credsRefreshMutex  sync.RWMutex
	credsRefreshNeeded bool
	config             *Config
}

// NewOpenSearchClient creates a new OpenSearch datastore client.
func NewOpenSearchClient(logger logging.Logger) (DatabaseClient, error) {
	client := &OpenSearchClient{
		logger: logger,
	}

	if err := client.loadconfig(); err != nil {
		client.logger.Errorw("Failed to load OpenSearch configuration", "error", err)
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := client.initializeClient(); err != nil {
		return nil, err
	}

	if client.isSecure && os.Getenv("OPENSEARCH_AUTO_REFRESH_CREDS") == "true" {
		client.startCredsRefreshTimer()
	}

	return client, nil
}

func (osc *OpenSearchClient) loadconfig() error {
	config, err := osc.loadYAMLConfig()
	if err != nil {
		return err
	}

	osc.applyEnvOverrides(config)
	if len(config.URLs) == 0 {
		config.URLs = []string{elastic.DefaultURL}
	}
	if err := osc.validateConfig(config); err != nil {
		return err
	}

	osc.config = config
	osc.urls = append([]string(nil), config.URLs...)
	osc.url = osc.urls[0]
	osc.isSecure = strings.HasPrefix(strings.ToLower(osc.url), "https://")
	return nil
}

func (osc *OpenSearchClient) loadYAMLConfig() (*Config, error) {
	config := &Config{}
	configMap := configutil.LoadConfigMap(configutil.ResolveConfFilePath("opensearch.yaml"))
	if configMap == nil {
		return config, nil
	}

	switch urls := configMap["urls"].(type) {
	case []interface{}:
		for _, raw := range urls {
			if url, ok := raw.(string); ok && strings.TrimSpace(url) != "" {
				config.URLs = append(config.URLs, strings.TrimSpace(url))
			}
		}
	case []string:
		for _, url := range urls {
			if strings.TrimSpace(url) != "" {
				config.URLs = append(config.URLs, strings.TrimSpace(url))
			}
		}
	case string:
		if strings.TrimSpace(urls) != "" {
			config.URLs = append(config.URLs, strings.TrimSpace(urls))
		}
	}

	for _, key := range []string{"ssl.ca.location", "ca"} {
		if raw, ok := configMap[key].(string); ok && strings.TrimSpace(raw) != "" {
			config.SSLCALocation = strings.TrimSpace(raw)
			break
		}
	}

	return config, nil
}

func (osc *OpenSearchClient) loadConfigFromEnv() error {
	if osc.config == nil {
		osc.config = &Config{}
	}
	osc.applyEnvOverrides(osc.config)
	return nil
}

func (osc *OpenSearchClient) applyEnvOverrides(config *Config) {
	if config == nil {
		return
	}

	if rawURLs := firstNonEmptyEnv("OPENSEARCH_URLS", "OPENSEARCH_URL", "ES_URL"); rawURLs != "" {
		config.URLs = splitURLs(rawURLs)
	}
	if sslCA := os.Getenv("OPENSEARCH_SSL_CA_LOCATION"); strings.TrimSpace(sslCA) != "" {
		config.SSLCALocation = strings.TrimSpace(sslCA)
	}
}

func (osc *OpenSearchClient) validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("opensearch config is nil")
	}
	if len(config.URLs) == 0 {
		return fmt.Errorf("no opensearch URLs configured")
	}
	for _, url := range config.URLs {
		if strings.TrimSpace(url) == "" {
			return fmt.Errorf("opensearch URL cannot be empty")
		}
	}
	return nil
}

func (osc *OpenSearchClient) initializeClient() error {
	if err := osc.setupHTTPClient(); err != nil {
		return err
	}

	client, err := elastic.NewClient(
		elastic.SetURL(osc.urls...),
		elastic.SetHttpClient(osc.httpClient),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
	)
	if err != nil {
		return err
	}

	osc.client = client
	return nil
}

func (osc *OpenSearchClient) setupHTTPClient() error {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if osc.isSecure {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: strings.EqualFold(os.Getenv("OPENSEARCH_SKIP_TLS_VERIFY"), "true"),
		}
	}

	osc.httpClient = &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	return nil
}

func (osc *OpenSearchClient) startCredsRefreshTimer() {
	// No-op: credentials refresh timer not implemented yet.
}

func (osc *OpenSearchClient) checkAndRefreshCreds() error {
	return nil
}

func (osc *OpenSearchClient) reconnect() error {
	return nil
}

func (osc *OpenSearchClient) UpsertIndex(indexName string, mapFilePath string) error {
	if osc.client == nil {
		return fmt.Errorf("opensearch client is not initialized")
	}

	createReq := osc.client.CreateIndex(indexName)
	if data, err := os.ReadFile(mapFilePath); err == nil && len(data) > 0 {
		createReq = createReq.BodyString(string(data))
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	if _, err := createReq.Do(context.Background()); err != nil {
		if strings.Contains(err.Error(), "resource_already_exists_exception") {
			return nil
		}
		return err
	}
	return nil
}

func (osc *OpenSearchClient) applyPlatformSettings(mapping *map[string]interface{}) {
	// No-op: platform-specific settings not implemented yet.
}

func (osc *OpenSearchClient) refreshCredentials() error {
	return nil
}

func (osc *OpenSearchClient) ExecuteQueryWithScrollCallback(index string, query interface{}, scrollSize int, scrollTimeout string, process func(batch []map[string]interface{}) error) error {
	if osc.client == nil {
		return fmt.Errorf("opensearch client is not initialized")
	}
	if process == nil {
		return fmt.Errorf("process callback is required")
	}
	if scrollSize <= 0 {
		scrollSize = 100
	}
	if scrollTimeout == "" {
		scrollTimeout = elastic.DefaultScrollKeepAlive
	}

	ctx := context.Background()
	scrollSvc := osc.client.Scroll(index).KeepAlive(scrollTimeout).Size(scrollSize)
	if query != nil {
		scrollSvc = scrollSvc.Body(query)
	}
	defer func() {
		_ = scrollSvc.Clear(ctx)
	}()

	for {
		result, err := scrollSvc.Do(ctx)
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if result == nil || result.Hits == nil || len(result.Hits.Hits) == 0 {
			return nil
		}

		batch := make([]map[string]interface{}, 0, len(result.Hits.Hits))
		for _, hit := range result.Hits.Hits {
			doc := make(map[string]interface{})
			if len(hit.Source) > 0 {
				if err := json.Unmarshal(hit.Source, &doc); err != nil {
					return err
				}
			}
			if hit.Id != "" {
				doc["_id"] = hit.Id
			}
			batch = append(batch, doc)
		}

		err = process(batch)
		if err == ErrStopScroll {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func splitURLs(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
	})
	urls := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			urls = append(urls, trimmed)
		}
	}
	return urls
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func newDatabaseClient(logger logging.Logger) DatabaseClient {
	logger.Info("Initializing OpenSearch datastore client")
	client, err := NewOpenSearchClient(logger)
	if err != nil {
		logger.Errorw("Failed to create OpenSearch client, this should not happen in production", "error", err)
		panic(fmt.Sprintf("Failed to create Opensearch client: %v", err))
	}
	return client
}
