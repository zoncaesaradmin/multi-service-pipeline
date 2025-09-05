// ...existing code...
package ruleenginelib

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
    		"action": "CREATE_ALERT_RULE"
  		},
		"payload": [
    	{
      		"alertRuleUuid": "03dw-werffd-sdf",
      		"siteId": "swmp3",
      		"ruleName": "rule1",
      		"ruleDescription": "All anomalies",
      		"customMessage": ["Welcome to customer recommendations."],
      		"customAck": false,
      		"actions": [
       	 		{
          			"action": "ACKNOWLEDGE",
          			"applyToActiveAnomlay": false
        		}
      		],
      		"matchCriteriaUuid": "osdf-8230-asdf",
      		"state": true,
      		"lastModifiedTime": 1613722639423,
      		"matchRule": {
        		"severityMatchCriteria": {
          			"valueEquals": [
            			"EVENT_SEVERITY_CRITICAL"
          			]
        		},
        		"affectedObjectMatchCriteria": [
          			{
          				"objectType": "leaf",
          				"valueEquals": [
          					"n1",
          					"n2"
          				]
          			}
          		],
        		"categoryMatchCriteria": {
          			"valueEquals": [
            			"CONNECTIVITY"
          			]
        		},
        		"subCategoryMatchCriteria": null,
        		"eventNameMatchCriteria": {
          			"valueEquals": [
            			"CONNECTIVITY_FLAP"
          			]
        		},
        		"checkCodeMatchCriteria": null
      		}
    	},
    	{
      		"alertRuleUuid": "03dw-werffd-sdf",
      		"siteId": "swmp4",
      		"ruleName": "rule1",
      		"ruleDescription": "All anomalies",
      		"customMessage": ["Welcome to customer recommendations."],
      		"customAck": false,
      		"actions": [
       	 		{
          			"action": "ACKNOWLEDGE",
          			"applyToActiveAnomlay": false
        		}
      		],
      		"matchCriteriaUuid": "osdf-8230-asdf",
      		"state": true,
      		"lastModifiedTime": 1613722639423,
      		"matchRule": {
        		"severityMatchCriteria": {
          			"valueEquals": [
            			"EVENT_SEVERITY_CRITICAL"
          			]
        		},
        		"affectedObjectMatchCriteria": [
          			{
          				"objectType": "leaf",
          				"valueEquals": [
          					"n1",
          					"n2"
          				]
          			}
          		],
        		"categoryMatchCriteria": {
          			"valueEquals": [
            			"CONNECTIVITY"
          			]
        		},
        		"subCategoryMatchCriteria": null,
        		"eventNameMatchCriteria": {
          			"valueEquals": [
            			"CONNECTIVITY_FLAP"
          			]
        		},
        		"checkCodeMatchCriteria": null
      		}
    	}
		],
  		"loggerpayload": {
    		"serviceName": "",
    		"moduleName": "",
    		"logLevel": ""
  		}
	}`

	var rInput RuleMessage
	msgBytes := []byte(j)
	//fmt.Printf("RELIB - received rule msg: %v", string(msgBytes))

	if err := json.Unmarshal(msgBytes, &rInput); err != nil {
		fmt.Printf("Failed to unmarshal rule message: %v\n", err.Error())
		// log and ignore invalid messages
		t.Errorf("Failed to unmarshal rule message: %v", err.Error())
		return
	}

	fmt.Printf("RELIB - received msg unmarshalled: %+v\n", rInput)

	//msgType := rInput.Metadata.Action
	//result := &RuleMsgResult{Action: msgType}

	//fmt.Printf("RELIB - processing valid rule for operation: %s\n", msgType)

	jsonBytes, err := json.Marshal(rInput.Rules)
	if err != nil {
		fmt.Printf("RELIB - failed to marshal rule payload: %v\n", err.Error())

		// log and ignore invalid messages
		t.Errorf("RELIB - failed to marshal rule payload: %v", err.Error())
		return
	}
	//fmt.Printf("RELIB - processed rule payload: %s\n", string(jsonBytes))

	//ruleJsonBytes, err := ConvertToRuleEngineFormat(jsonBytes)
	ruleJsonBytes, err := ConvertToRuleEngineFormat(jsonBytes)
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
