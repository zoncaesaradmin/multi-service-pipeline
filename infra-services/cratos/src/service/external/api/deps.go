package api

func FetchAlertMappings() map[string]string {

	// call local or production implementation to get actual data
	_ = fetchAlertMappingInfo()

	// do the common data handling here

	return map[string]string{}
}
