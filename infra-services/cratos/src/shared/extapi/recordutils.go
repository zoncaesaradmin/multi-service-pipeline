package extapi

import (
	"strings"
	"telemetry/utils/alert"
	relib "telemetry/utils/ruleenginelib"
)

var severityMap = map[int]string{
	0: "event",
	1: "debug",
	2: "warning",
	3: "minor",
	4: "major",
	5: "critical",
}

func NeedsRuleProcessing(aObj *alert.Alert) bool {
	// Placeholder for actual logic to determine if a record needs rule processing
	// This function should return true if the record requires rule processing,
	// and false otherwise.
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
		case "switch", "leaf":
			reInput[relib.MatchKeySwitch] = objField.Name
		case "interface":
			reInput[relib.MatchKeyInterface] = relib.NormalizeInterfaceName(objField.Name)
		case "ip":
			reInput[relib.MatchKeyIp] = objField.Name
		case "vni":
			reInput[relib.MatchKeyVni] = objField.Name
		case "subnet":
			reInput[relib.MatchKeySubnet] = objField.Name
		case "vrf":
			reInput[relib.MatchKeyVrf] = objField.Name
		}
	}
	return reInput
}
