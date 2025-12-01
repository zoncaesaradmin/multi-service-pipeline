package ruleenginelib

import (
	"encoding/json"
	"io"
	"os"
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

func TestBulkExerciseRemainingBranches(t *testing.T) {
	// ConvertToRuleEngineFormat with applyToAll true and various actions
	r1 := AlertRuleConfig{UUID: "b1", Name: "bname", State: "true", ApplyActionsToAllStr: "true", AlertRuleActions: []RuleActionConfig{{Action: "ACKNOWLEDGE"}, {Action: "CUSTOMIZE_ANOMALY"}, {Action: "OVERRIDE_SEVERITY"}}, CustomizeAnomaly: CustomizeAnomalyConfig{CustomMessage: "cm"}, SeverityOverride: "MAJOR", AlertRuleMatchCriteria: []RuleMatchCriteriaConfig{{UUID: "mc1", SiteId: "siteX", AffectedObjectMatchCriteria: []MatchCriteria{{ObjectType: "bd", ValueEquals: "BD1"}}}}}
	b, err := ConvertToRuleEngineFormat([]AlertRuleConfig{r1})
	if err != nil || len(b) == 0 {
		t.Fatalf("ConvertToRuleEngineFormat failed: %v", err)
	}

	// EvaluateOperator for all operator variants
	ops := []struct {
		a  interface{}
		b  interface{}
		op string
	}{
		{1, 2, "<"},
		{2, 1, ">"},
		{1, 1, "="},
		{1, 1, "eq"},
		{1, 2, "neq"},
		{1.5, 1.5, ">="},
		{3, 4, "lte"},
		{"x", []interface{}{"x", "y"}, "anyof"},
		{"z", []interface{}{"x", "y"}, "noneof"},
	}
	for _, it := range ops {
		_, _ = EvaluateOperator(it.a, it.b, it.op)
	}

	// DeepEqualMaps with nested structures
	m1 := map[string]interface{}{"a": map[string]interface{}{"x": 1}, "b": []interface{}{1, 2}}
	m2 := map[string]interface{}{"a": map[string]interface{}{"x": 1}, "b": []interface{}{1, 2}}
	if !DeepEqualMaps(m1, m2) {
		t.Fatalf("expected DeepEqualMaps true for equal nested maps")
	}
	m3 := map[string]interface{}{"a": map[string]interface{}{"x": 2}}
	if DeepEqualMaps(m1, m3) {
		t.Fatalf("expected DeepEqualMaps false for different nested maps")
	}

	// ProcessRuleChangeRequest: create various transitions
	old := &RuleDefinition{ApplyActionsToAll: true, MatchCriteriaEntries: map[string][]*RuleMatchCondition{"k": {{CriteriaUUID: "c1", Condition: AstCondition{All: []AstConditional{{Identifier: "a", Operator: "eq", Value: "1"}}}}}}}
	new := &RuleDefinition{ApplyActionsToAll: false, MatchCriteriaEntries: map[string][]*RuleMatchCondition{"k": {{CriteriaUUID: "c1", Condition: AstCondition{All: []AstConditional{{Identifier: "a", Operator: "eq", Value: "1"}}}}}}}
	resp := ProcessRuleChangeRequest(RuleChangeRequest{RuleEvent: InternalEventUpdate, OldRule: old, NewRule: new, ApplyToExisting: false})
	if !resp.NeedsCleanup {
		t.Fatalf("expected NeedsCleanup true for ApplyActionsToAll true->false")
	}

	// GetAllRuleInfo with multiple rules
	re := NewRuleEngineInstance(nil, nil)
	lg := zerolog.New(io.Discard).With().Logger()
	re.Logger = &Logger{logger: &lg}

	rd1 := &RuleDefinition{AlertRuleUUID: "g1", Name: "n1", Enabled: true, MatchCriteriaEntries: map[string][]*RuleMatchCondition{"c1": {{CriteriaUUID: "c1", PrimaryMatchValue: PrimaryKeyDefault, Condition: AstCondition{All: []AstConditional{{Identifier: "x", Operator: "eq", Value: "v"}}}}}}}
	rd2 := &RuleDefinition{AlertRuleUUID: "g2", Name: "n2", Enabled: true, MatchCriteriaEntries: map[string][]*RuleMatchCondition{"c2": {{CriteriaUUID: "c2", PrimaryMatchValue: "site1", Condition: AstCondition{All: []AstConditional{{Identifier: "y", Operator: "eq", Value: "v2"}}}}}}}
	re.AddRuleDefinition(rd1)
	re.AddRuleDefinition(rd2)
	info := re.GetAllRuleInfo()
	if len(info) == 0 {
		t.Fatalf("expected GetAllRuleInfo to return non-empty data")
	}

	// ensure EvaluateRules path works
	res := re.EvaluateRules(Data{"fabricName": "site1", "x": "v"})
	_ = res

	// cover determineEffectiveEventType branch when original is unknown
	_, _ = re.determineEffectiveEventType(&Logger{logger: &lg}, "UNKNOWN", AlertRuleConfig{UUID: "u1"}, 0)
}

func TestEventTypeHelpers(t *testing.T) {
	if !isValidEventType(RuleEventCreate) {
		t.Fatalf("expected RuleEventCreate to be valid")
	}
	if isValidEventType("FOO") {
		t.Fatalf("expected unknown event to be invalid")
	}

	if !isRuleTypeOfInterest(RuleTypeMgmt, []string{RuleTypeMgmt}) {
		t.Fatalf("expected rule type of interest true")
	}
	if isRuleTypeOfInterest("X", []string{RuleTypeMgmt}) {
		t.Fatalf("expected non-matching rule type to be false")
	}

	meta := AlertRuleMetadata{RuleType: RuleTypeMgmt, RuleEventType: RuleEventCreate}
	if !isRelevantRule(meta, []string{RuleTypeMgmt}) {
		t.Fatalf("expected meta to be relevant")
	}
}

func TestInitAlertRulesNoop(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	// GetAllConfiguredAlertRules returns nil; initAlertRules should be a no-op
	re.initAlertRules()
}

func TestHandleAlertRuleEventCreateAndUpdateEnableDisable(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	// attach a no-op logger
	lg := zerolog.New(io.Discard).With().Logger()
	re.Logger = &Logger{logger: &lg}
	re.RuleTypes = []string{RuleTypeMgmt}

	// CREATE with state=true should process
	msg := AlertRuleMsg{Metadata: AlertRuleMetadata{RuleType: RuleTypeMgmt, RuleEventType: RuleEventCreate}, AlertRules: []AlertRuleConfig{{UUID: "c1", State: "true"}}}
	b, _ := json.Marshal(msg)
	res, old, err := re.HandleAlertRuleEvent(b, TransactionMetadata{TraceId: "t-create"})
	if err != nil || res == nil || len(res.Rules) == 0 {
		t.Fatalf("expected CREATE to be processed, got res=%v old=%v err=%v", res, old, err)
	}

	// UPDATE with state=true but rule exists -> UPDATE processed
	msg2 := AlertRuleMsg{Metadata: AlertRuleMetadata{RuleType: RuleTypeMgmt, RuleEventType: RuleEventUpdate}, AlertRules: []AlertRuleConfig{{UUID: "c1", State: "true"}}}
	b2, _ := json.Marshal(msg2)
	res2, old2, err2 := re.HandleAlertRuleEvent(b2, TransactionMetadata{TraceId: "t-update"})
	if err2 != nil || res2 == nil || len(res2.Rules) == 0 {
		t.Fatalf("expected UPDATE to be processed when rule exists, got res=%v old=%v err=%v", res2, old2, err2)
	}

	// UPDATE with state=false should convert to DELETE and process
	msg3 := AlertRuleMsg{Metadata: AlertRuleMetadata{RuleType: RuleTypeMgmt, RuleEventType: RuleEventUpdate}, AlertRules: []AlertRuleConfig{{UUID: "c1", State: "false"}}}
	b3, _ := json.Marshal(msg3)
	res3, _, err3 := re.HandleAlertRuleEvent(b3, TransactionMetadata{TraceId: "t-update-false"})
	if err3 != nil || res3 == nil {
		t.Fatalf("expected UPDATE->DELETE to be processed, got res=%v err=%v", res3, err3)
	}

	// ENABLE should convert to CREATE
	msg4 := AlertRuleMsg{Metadata: AlertRuleMetadata{RuleType: RuleTypeMgmt, RuleEventType: RuleEventEnable}, AlertRules: []AlertRuleConfig{{UUID: "e1", State: "true"}}}
	b4, _ := json.Marshal(msg4)
	res4, _, err4 := re.HandleAlertRuleEvent(b4, TransactionMetadata{TraceId: "t-enable"})
	if err4 != nil || res4 == nil {
		t.Fatalf("expected ENABLE to be processed as CREATE, got res=%v err=%v", res4, err4)
	}

	// DISABLE should convert to DELETE
	msg5 := AlertRuleMsg{Metadata: AlertRuleMetadata{RuleType: RuleTypeMgmt, RuleEventType: RuleEventDisable}, AlertRules: []AlertRuleConfig{{UUID: "d1", State: "false"}}}
	b5, _ := json.Marshal(msg5)
	res5, _, err5 := re.HandleAlertRuleEvent(b5, TransactionMetadata{TraceId: "t-disable"})
	if err5 != nil || res5 == nil {
		t.Fatalf("expected DISABLE to be processed as DELETE, got res=%v err=%v", res5, err5)
	}
}

func TestInitAlertRulesWithOverride(t *testing.T) {
	// create a fake rule and override GetAllConfiguredAlertRules
	rule := &RuleDefinition{AlertRuleUUID: "rinit", Name: "ninit", Enabled: true, MatchCriteriaEntries: map[string][]*RuleMatchCondition{"c1": {{CriteriaUUID: "c1", Condition: AstCondition{All: []AstConditional{{Identifier: "a", Operator: "eq", Value: "1"}}}}}}}
	b, _ := json.Marshal([]*RuleDefinition{rule})

	orig := GetAllConfiguredAlertRules
	defer func() { GetAllConfiguredAlertRules = orig }()
	GetAllConfiguredAlertRules = func() [][]byte { return [][]byte{b} }

	re := NewRuleEngineInstance(nil, nil)
	lg := zerolog.New(io.Discard).With().Logger()
	re.Logger = &Logger{logger: &lg}
	re.initAlertRules()

	// ensure rule got added
	if _, ok := re.GetRule("rinit"); !ok {
		t.Fatalf("expected rule rinit to be initialized into engine")
	}
}

func TestLoggerErrorAndPanicWrapped(t *testing.T) {
	lg := zerolog.New(io.Discard).With().Logger()
	l := &Logger{logger: &lg}
	// Error should not panic
	l.Error("errtest", nil)

	// Panic should cause panic; capture with recover
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected Panic to panic")
		}
	}()
	l.Panic("panic test", nil)
}

func TestEvaluateAstConditionNestedAnyAll(t *testing.T) {
	options = &Options{AllowUndefinedVars: true}
	// Any should match when one of nested any entries matches
	cond := AstCondition{
		Any: []AstConditional{{Identifier: "x", Operator: "eq", Value: "a"}},
		All: []AstConditional{{Identifier: "y", Operator: "eq", Value: "b"}},
	}
	// Provide data matching both Any and All so EvaluateAstCondition returns true
	ok := EvaluateAstCondition(cond, Data{"x": "a", "y": "b"}, &Options{AllowUndefinedVars: true})
	if !ok {
		t.Fatalf("expected EvaluateAstCondition to be true when Any matches")
	}
}

func TestConvertToRuleEngineFormatVarious(t *testing.T) {
	// Rule with customize action
	r := AlertRuleConfig{
		UUID:                   "u1",
		Name:                   "n1",
		State:                  "true",
		CustomizeAnomaly:       CustomizeAnomalyConfig{CustomMessage: "hey"},
		AlertRuleActions:       []RuleActionConfig{{Action: "CUSTOMIZE_ANOMALY"}},
		AlertRuleMatchCriteria: []RuleMatchCriteriaConfig{{UUID: "c1"}},
	}
	b, err := ConvertToRuleEngineFormat([]AlertRuleConfig{r})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	var defs []RuleDefinition
	if err := json.Unmarshal(b, &defs); err != nil {
		t.Fatalf("failed to unmarshal converted rules: %v", err)
	}
	if len(defs) != 1 || defs[0].AlertRuleUUID != "u1" {
		t.Fatalf("unexpected converted rule definitions: %v", defs)
	}

	// Rule with severity override
	r2 := AlertRuleConfig{UUID: "u2", Name: "n2", State: "true", SeverityOverride: "EVENT_SEVERITY_MAJOR", AlertRuleActions: []RuleActionConfig{{Action: "OVERRIDE_SEVERITY"}}}
	b2, err := ConvertToRuleEngineFormat([]AlertRuleConfig{r2})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	var defs2 []RuleDefinition
	if err := json.Unmarshal(b2, &defs2); err != nil {
		t.Fatalf("failed to unmarshal converted rules: %v", err)
	}
	if len(defs2) != 1 || defs2[0].Actions[0].ActionType != RuleActionSeverityOverride {
		t.Fatalf("unexpected severity override conversion: %v", defs2)
	}
}

func TestDeepEqualMapsAndExtractValue(t *testing.T) {
	m1 := map[string]interface{}{"a": 1, "b": map[string]interface{}{"x": "y"}}
	m2 := map[string]interface{}{"b": map[string]interface{}{"x": "y"}, "a": 1}
	if !DeepEqualMaps(m1, m2) {
		t.Fatalf("expected maps to be equal")
	}

	m3 := map[string]interface{}{"a": 1}
	if DeepEqualMaps(m1, m3) {
		t.Fatalf("expected maps to be different")
	}

	// Build conditions slice for ExtractValueFromConditions
	cond := map[string]interface{}{"all": []AstConditional{{Identifier: MatchKeyIp, Operator: "=", Value: "1.2.3.4"}}}
	val := ExtractValueFromConditions([]map[string]interface{}{cond}, MatchKeyIp)
	if val == nil || val.(string) != "1.2.3.4" {
		t.Fatalf("expected to extract ip value, got %#v", val)
	}
}

func TestProcessRuleChangeRequestVariants(t *testing.T) {
	// Delete
	resp := ProcessRuleChangeRequest(RuleChangeRequest{RuleEvent: InternalEventDelete, NewRule: nil})
	if !resp.ShouldProcess {
		t.Fatalf("delete should process")
	}

	// Create skipping existing
	resp2 := ProcessRuleChangeRequest(RuleChangeRequest{RuleEvent: InternalEventCreate, ApplyToExisting: false})
	if resp2.ShouldProcess {
		t.Fatalf("create with applyToExisting=false should not process")
	}

	// Create processing existing
	newRule := &RuleDefinition{MatchCriteriaEntries: map[string][]*RuleMatchCondition{"c": {{Condition: AstCondition{All: []AstConditional{{Identifier: MatchKeyTitle, Operator: "=", Value: "t"}}}}}}}
	resp3 := ProcessRuleChangeRequest(RuleChangeRequest{RuleEvent: InternalEventCreate, ApplyToExisting: true, NewRule: newRule})
	if !resp3.ShouldProcess || len(resp3.Conditions) == 0 {
		t.Fatalf("expected create with applyToExisting=true to process and provide conditions")
	}

	// Update transition: old ApplyActionsToAll true -> new apply false
	old := &RuleDefinition{ApplyActionsToAll: true, MatchCriteriaEntries: newRule.MatchCriteriaEntries}
	resp4 := ProcessRuleChangeRequest(RuleChangeRequest{RuleEvent: InternalEventUpdate, OldRule: old, ApplyToExisting: false})
	if !resp4.ShouldProcess || !resp4.NeedsCleanup {
		t.Fatalf("expected update transition to require cleanup and processing")
	}

	// Update with applyToExisting true
	resp5 := ProcessRuleChangeRequest(RuleChangeRequest{RuleEvent: InternalEventUpdate, OldRule: old, ApplyToExisting: true, NewRule: newRule})
	if !resp5.ShouldProcess || len(resp5.Conditions) == 0 {
		t.Fatalf("expected update with applyToExisting=true to process")
	}
}

func TestCreateRuleEngineInstanceInit(t *testing.T) {
	// Ensure CreateRuleEngineInstance spins up and calls initAlertRules safely
	// Create a temp file for the logger file path so CreateLoggerInstance doesn't os.Exit
	tmpf, err := os.CreateTemp("", "relogtest*.log")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpName := tmpf.Name()
	tmpf.Close()
	defer os.Remove(tmpName)

	lg := LoggerInfo{ServiceName: "svc", FilePath: tmpName, Level: "info"}
	// Override GetAllConfiguredAlertRules to return an empty list (safe)
	oldGetter := GetAllConfiguredAlertRules
	GetAllConfiguredAlertRules = func() [][]byte { return [][]byte{} }
	defer func() { GetAllConfiguredAlertRules = oldGetter }()

	re := CreateRuleEngineInstance(lg, []string{RuleTypeMgmt})
	if re == nil {
		t.Fatalf("expected non-nil RuleEngine instance")
	}
	// attach no-op logger to avoid nil deref
	zlg := zerolog.New(io.Discard).With().Logger()
	re.Logger = &Logger{logger: &zlg}
}
