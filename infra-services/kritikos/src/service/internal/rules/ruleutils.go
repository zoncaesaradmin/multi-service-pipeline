package rules

import "fmt"

// extractRuleName extracts the rule name from MongoDB document
func extractRuleName(rule map[string]interface{}, fallbackIndex int) string {
	if name, ok := rule["name"].(string); ok && name != "" {
		return name
	}

	// Try alternative field names
	if alertName, ok := rule["alertName"].(string); ok && alertName != "" {
		return alertName
	}

	if ruleIdentifier, ok := rule["ruleIdentifier"].(string); ok && ruleIdentifier != "" {
		return ruleIdentifier
	}

	// Return a fallback name
	return fmt.Sprintf("rule_%d", fallbackIndex)
}

// extractRuleID extracts the rule ID from MongoDB document
func extractRuleID(rule map[string]interface{}, fallbackIndex int) string {
	if id, ok := rule["alertRuleUUID"].(string); ok && id != "" {
		return id
	}

	return fmt.Sprintf("rule_id_%d", fallbackIndex)
}
