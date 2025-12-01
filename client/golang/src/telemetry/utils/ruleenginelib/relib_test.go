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
                     "resourceType": "switch",
                     "valueEquals": "leaf-1"
                   },
                   {
                     "resourceType": "interface",
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

func TestConvertObjIdentifier(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"interface", MatchKeyInterface},
		{"switch", MatchKeySwitch},
		{"node", MatchKeySwitch},
		{"leaf", MatchKeySwitch},
		{"vni", MatchKeyVni},
		{"ip", MatchKeyIp},
		{"ipv4", MatchKeyIp},
		{"ipv6", MatchKeyIp},
		{"address", MatchKeyIp},
		{"vni", MatchKeyVni},
		{"l2vni", MatchKeyVni},
		{"l3vni", MatchKeyVni},
		{"l2_vni", MatchKeyVni},
		{"l3_vni", MatchKeyVni},
		{"vrf", MatchKeyVrf},
		{"tenant", MatchKeyTenant},
		{"subnet", MatchKeySubnet},
		{"network", MatchKeySubnet},
		{"bd", MatchKeyBd},
		{"bridge_domain", MatchKeyBd},
		{"route", MatchKeyRoute},
		{"configuration_compliance", MatchKeyComplianceRule},
		{"communication_compliance", MatchKeyComplianceRule},
		{"CONFIGURATION_COMPLIANCE", MatchKeyComplianceRule},
		{"mac", MatchKeyMac},
		{"MAC", MatchKeyMac},
		{"epg", MatchKeyEpg},
		{"EPG", MatchKeyEpg},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		result := convertObjIdentifier(tt.input)
		if result != tt.expected {
			t.Errorf("convertObjIdentifier(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
