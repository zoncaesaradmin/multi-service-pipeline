package extapi

import (
	"encoding/json"
	"sharedgomodule/logging"
	"strconv"
	"strings"
	"telemetry/utils/alert"
	relib "telemetry/utils/ruleenginelib"
)

var severityMap = map[int]string{
	0: "event",
	6: "debug",
	1: "info",
	2: "warning",
	3: "minor",
	4: "major",
	5: "critical",
}

func NeedsRuleProcessing(aObj *alert.Alert) bool {
	// Placeholder for actual logic to determine if a record needs rule processing
	// This function should return true if the record requires rule processing,
	// and false otherwise.

	if strings.EqualFold(aObj.AlertType, "advisory") {
		// skip rule lookup for advisories as there is no such support so far
		return false
	}
	return true
}

func RecordIdentifier(aObj *alert.Alert) string {
	// Placeholder for actual logic to determine the record identifier
	// especially for tracking the path
	// This function should return a string that uniquely identifies the record.
	return aObj.GetAlertId()
}

func ConvertAlertObjectToRuleEngineInput(aObj *alert.Alert) map[string]any {
	reInput := make(map[string]any)
	// NOTE: though it is less code to copy fields by using
	// struct marshal and unmarshal in to a map[string]any,
	// it is faster at runtime to copy fields explicitly
	// so, the value copying logic is explicit and for known/allowed match fields only
	// also, it helps to change the field key mapping if needed

	reInput[relib.MatchKeyFabricName] = aObj.FabricName
	reInput[relib.MatchKeyCategory] = strings.ToUpper(aObj.Category)
	reInput[relib.MatchKeyTitle] = aObj.MnemonicTitle
	reInput[relib.MatchKeySeverity] = strings.ToLower(aObj.Severity)

	for _, objField := range aObj.AffectedObjects {
		switch strings.ToLower(objField.Type) {
		case "switch", "leaf", "node":
			reInput[relib.MatchKeySwitch] = objField.Name
		case "interface":
			reInput[relib.MatchKeyInterface] = relib.NormalizeInterfaceName(objField.Name)
		case "ip", "ipv4", "ipv6", "ipaddress", "address":
			reInput[relib.MatchKeyIp] = objField.Name
		case "vni":
			reInput[relib.MatchKeyVni] = objField.Name
		case "subnet", "network":
			reInput[relib.MatchKeySubnet] = objField.Name
		case "vrf":
			reInput[relib.MatchKeyVrf] = objField.Name
		case "tenant":
			reInput[relib.MatchKeyTenant] = objField.Name
		}
	}
	return reInput
}

func getSeverityInt(doc map[string]interface{}, key string) (int, bool) {
	if val, exists := doc[key]; exists {
		switch v := val.(type) {
		case int:
			return v, true
		case int64:
			return int(v), true // Convert int64 to int
		case float64:
			return int(v), true // Convert float64 to int (truncates decimals)
		case string:
			if i, err := strconv.Atoi(v); err == nil { // strconv.Atoi is for int
				return i, true
			}
		}
	}
	return 0, false
}

func MapJSONToAlert(logger logging.Logger, doc map[string]interface{}, alertObj *alert.Alert) error {
	rawSeveritInt, foundSeverity := getSeverityInt(doc, "severity")
	docForUnmarshal := make(map[string]interface{})
	for k, v := range doc {
		if k != "severity" { // Copy all fields severity since it is int
			docForUnmarshal[k] = v
		}
	}
	jsonBytes, err := json.Marshal(docForUnmarshal)
	if err != nil {
		logger.Errorw("marshal of doc failed", "error", err)
		return err
	}
	err = json.Unmarshal(jsonBytes, alertObj)
	if err != nil {
		logger.Errorw("Unmarshal to alertObj failed", "error", err)
		return err
	}
	if foundSeverity {
		if sevStr, ok := severityMap[rawSeveritInt]; ok {
			alertObj.Severity = sevStr
		}
	}
	return nil
}
