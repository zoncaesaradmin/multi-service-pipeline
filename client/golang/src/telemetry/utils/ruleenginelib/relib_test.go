// ...existing code...
package ruleenginelib

/*
import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestBaseFunc(t *testing.T) {
	j := `{"payload": [{
		"conditions": [{
			"all": [{
				"identifier": "myVar",
				"operator": "eq",
				"value": "hello world"
			}]
		}],
		"actions": [{
			"type": "result",
			"payload": {
				"data": {
					"say": "Hello World!"
				}
			}
		}]
	}]}`

	rule := ParseJSON(j)
	if fmt.Sprintf("%T", rule) != "*ruleenginelib.RuleBlock" {
		t.Fatalf("expected rule to be *ruleenginelib.RuleBlock, got %T", rule)
	}
	fmt.Printf("Parsed rule: %+v\n", rule.RuleEntries[0])
}

func TestConvertFunc(t *testing.T) {
	j := `{
         "metadata": {
           "ruleType": "ALERT_RULES",
           "alertType": "anomaly",
           "action": "CREATE_ALERT_RULE"
         },
         "alertRulePayload": [
           {
             "uuid": "ABCD-1234-EFGH-5678",
             "name": "test-prio",
             "priority": 1756993805,
             "description": "Alert when CPU usage exceeds threshold",
             "state": "true",
             "customizeAnomaly": {
               "customMessage": "CPU usage is too high, correct it"
             },
             "associatedInsightGroupUuids": ["group-456", "group-789"],
             "alertRuleActions": [
               {
                 "action": "CUSTOMIZE_ANOMALY",
                 "applyToActiveAnomaly": "false"
               },
               {
                 "action": "ACKNOWLEDGE",
                 "applyToActiveAnomaly": "false"
               }
             ],
             "alertRuleMatchCriteria": [
               {
                 "categoryMatchCriteria": [
                   {
                     "valueEquals": "CONNECTIVITY"
                   }
                 ],
                 "eventNameMatchCriteria": [
                   {
                     "valueEquals": "BGP_PEER_CONNECTION_DOWN"
                   }
                 ],
                 "uuid": "ABCD-1234-EFGH-7654",
                 "alertRuleId": "ABCD-1234-EFGH-XTAZ",
                 "siteId": "fabric-1"
               }
             ],
             "lastModifiedTime": 1756993805053,
             "links": [
             ]
    }]}`

	var rInput AlertRuleMsg
	msgBytes := []byte(j)
	//fmt.Printf("RELIB - received rule msg: %v", string(msgBytes))

	if err := json.Unmarshal(msgBytes, &rInput); err != nil {
		fmt.Printf("Failed to unmarshal rule message: %v\n", err.Error())
		// log and ignore invalid messages
		t.Errorf("Failed to unmarshal rule message: %v", err.Error())
		return
	}

	fmt.Printf("RELIB - received msg unmarshalled: %+v\n", rInput)

	ruleJsonBytes, err := ConvertToRuleEngineFormat(rInput.AlertRules)
	if err != nil {
		fmt.Printf("RELIB - failed to convert rule format: %v\n", err.Error())
		return
	}
	if len(ruleJsonBytes) == 0 {
		fmt.Printf("RELIB - invalid empty rule to process, ignored\n")
		t.Errorf("RELIB - invalid empty rule to process, ignored\n")
		return
	}

	fmt.Printf("RELIB - rule converted %s\n", string(ruleJsonBytes))
	//re.handleRuleMsgEvents(ruleJsonBytes, msgType)
	//result.RuleJSON = ruleJsonBytes
}
*/
