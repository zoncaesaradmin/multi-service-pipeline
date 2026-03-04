package datastore

import (
	"os"
	"sharedgomodule/utils"
	"strings"
)

func EsIndexPrefix() string {
	// 1) Environment variables
	p := strings.TrimSpace(os.Getenv("ES_INDEX_PREFIX"))
	if p == "" {
		p = strings.TrimSpace(os.Getenv("OPENSEARCH_INDEX_PREFIX"))
	}
	// 2) Config file (conf/opensearch.yaml), key: index_prefix
	if p == "" {
		cfg := utils.LoadConfigMap(utils.ResolveConfFilePath("opensearch.yaml"))
		if cfg != nil {
			if v, ok := cfg["index_prefix"]; ok {
				if s, ok2 := v.(string); ok2 {
					p = strings.TrimSpace(s)
				}
			}
		}
	}
	return p
}
