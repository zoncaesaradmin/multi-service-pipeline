package extapi

import "net/http"

const (
	EventSvcBaseURL        = "https://nae-eventservice-svc.cisco-nir.svc:8443"
	AlertMappingsUrlPrefix = "/api/telemetry/v2/events/mappings"
	AllSiteTypesStr        = "CISCO_ACI,CISCO_NX-OS"
)

func FetchSeverityMappings(baseURL, siteType string, severityCache map[string]string, client *http.Client) error {

	// call local or production implementation to get actual data
	err := fetchSeverityMappingInfo(baseURL, siteType, severityCache, client)

	// do the common data handling here

	return err
}
