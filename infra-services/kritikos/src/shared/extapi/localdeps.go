//go:build local
// +build local

package extapi

import "net/http"

func fetchSeverityMappingInfo(baseURL, siteType string, severityCache map[string]string, client *http.Client) error {
	// have local implementation here
	return nil
}
