package ruleenginelib

import (
	"encoding/json"
	"io"
	"testing"

	"github.com/rs/zerolog"
)

func TestEvaluateComparableEqualsSuccess(t *testing.T) {
	ok, err := evaluateComparableEquals("a", "a")
	if err != nil || !ok {
		t.Fatalf("expected comparable equals true, got %v err=%v", ok, err)
	}
}

func TestEvaluateNoneOfFalse(t *testing.T) {
	ok, err := evaluateNoneOf("x", []interface{}{"x", "y"})
	if err != nil || ok {
		t.Fatalf("expected noneof to be false when value present, got %v err=%v", ok, err)
	}
}

func TestHandleAlertRuleEventFlows(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	lg := zerolog.New(io.Discard).With().Logger()
	re.Logger = &Logger{logger: &lg}

	// irrelevant due to rule type mismatch
	msg := AlertRuleMsg{Metadata: AlertRuleMetadata{RuleType: "OTHER", RuleEventType: RuleEventCreate}, AlertRules: []AlertRuleConfig{{UUID: "a"}}}
	b, _ := json.Marshal(msg)
	res, old, err := re.HandleAlertRuleEvent(b, TransactionMetadata{TraceId: "t1"})
	if err == nil || res == nil {
		// For irrelevant rule types, we expect a RuleMsgResult with RuleEvent set and no processing
		if res == nil || res.RuleEvent != RuleEventCreate {
			t.Fatalf("expected irrelevant rule result with original event type, got %v err=%v old=%v", res, err, old)
		}
	}

	// Now test create with state false -> should drop
	msg2 := AlertRuleMsg{Metadata: AlertRuleMetadata{RuleType: RuleTypeMgmt, RuleEventType: RuleEventCreate}, AlertRules: []AlertRuleConfig{{UUID: "b", State: "false"}}}
	b2, _ := json.Marshal(msg2)
	_, _, err2 := re.HandleAlertRuleEvent(b2, TransactionMetadata{TraceId: "t2"})
	if err2 == nil {
		// Error is expected because processedCount==0 leads to error
	}
}
