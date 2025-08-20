package processing

import (
	"strings"
	"telemetry/utils/alert"
	relib "telemetry/utils/ruleenginelib"
)

func needsRuleProcessing(aObj *alert.Alert) bool {
	// Placeholder for actual logic to determine if a record needs rule processing
	// This function should return true if the record requires rule processing,
	// and false otherwise.
	return true
}

func recordIdentifier(aObj *alert.Alert) string {
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

	reInput[relib.MatchKeyFabricName] = aObj.GetFabricName()
	reInput[relib.MatchKeyCategory] = aObj.GetCategory()
	reInput[relib.MatchKeyTitle] = aObj.GetTitle()
	reInput[relib.MatchKeySeverity] = aObj.GetSeverity()

	for _, objField := range aObj.EntityNameList {
		switch strings.ToLower(objField.ObjectType) {
		case "leaf":
			reInput[relib.MatchKeyLeaf] = objField.ObjectValue
		case "interface":
			reInput[relib.MatchKeyInterface] = objField.ObjectValue
		case "vni":
			reInput[relib.MatchKeyVni] = objField.ObjectValue
		case "subnet":
			reInput[relib.MatchKeySubnet] = objField.ObjectValue
		case "vrf":
			reInput[relib.MatchKeyVrf] = objField.ObjectValue
		}
	}
	return reInput
}
