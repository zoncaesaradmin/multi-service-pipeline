// ...existing code...
package ruleenginelib

import (
	"fmt"
	"testing"
)

func TestParseJSON(t *testing.T) {
	j := `{
		"condition": {
			"any": [{
				"identifier": "myVar",
				"operator": "eq",
				"value": "hello world"
			}]
		},
		"event": {
			"type": "result",
			"payload": {
				"data": {
					"say": "Hello World!"
				}   
			}
		}
	}`

	rule := ParseJSON(j)
	if fmt.Sprintf("%T", rule) != "*ruleenginelib.RuleBlock" {
		t.Fatalf("expected rule to be *ruleenginelib.RuleBlock, got %T", rule)
	}
}
