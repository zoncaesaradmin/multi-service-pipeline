// ...existing code...
package ruleenginelib

import (
	"encoding/json"
	"fmt"
	"testing"
)

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
                   },
                   {
                     "valueEquals": "HARDWARE"
                   }
                 ],
                 "affectedObjectMatchCriteria": [
                   {
                     "objectType": "switch",
                     "valueEquals": "leaf-1"
                   },
                   {
                     "objectType": "interface",
                     "valueEquals": "eth1/3"
                   }
                 ],
                 "eventNameMatchCriteria": [
                   {
                     "valueEquals": "BGP_PEER_CONNECTION_DOWN"
                   },
                   {
                     "valueEquals": "AUTOMATIC_PRIVATE_SUBNET_IP"
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

	//fmt.Printf("received msg unmarshalled: %+v\n", rInput)

	ruleJsonBytes, err := ConvertToRuleEngineFormat(rInput.AlertRules)
	if err != nil {
		fmt.Printf("failed to convert rule format: %v\n", err.Error())
		return
	}
	if len(ruleJsonBytes) == 0 {
		fmt.Printf("invalid empty rule to process, ignored\n")
		t.Errorf("invalid empty rule to process, ignored\n")
		return
	}

	//fmt.Printf("converted rules - %s\n", string(ruleJsonBytes))
}
