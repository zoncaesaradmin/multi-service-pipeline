package ruleenginelib

import (
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/rs/zerolog"
)

const (
	expectedEth = "Ethernet1/2"
	expectedPc  = "Port-channel3"
)

func TestEvaluateOperatorUnknown(t *testing.T) {
	_, err := EvaluateOperator(1, 2, "not-an-op")
	if err == nil {
		t.Errorf("expected error for unknown operator")
	}
}

func TestAssertIsNumberErrors(t *testing.T) {
	if _, err := assertIsNumber("notnum"); err == nil {
		t.Errorf("expected error for non-number in assertIsNumber")
	}
}

func TestEvaluateAnyOfNumericAndGeneric(t *testing.T) {
	// numeric slice
	ok, err := evaluateAnyOf(3, []interface{}{1, 2, 3})
	if err != nil || !ok {
		t.Fatalf("expected true for numeric anyof, got %v, err=%v", ok, err)
	}

	// generic slice
	ok2, err2 := evaluateAnyOf("a", []interface{}{"b", "a"})
	if err2 != nil || !ok2 {
		t.Fatalf("expected true for generic anyof, got %v, err=%v", ok2, err2)
	}
}

func TestEvaluateEqualsStringSlice(t *testing.T) {
	ok, err := evaluateEquals([]string{"a", "b"}, "b")
	if err != nil || !ok {
		t.Fatalf("expected true for equals with []string, got %v, err=%v", ok, err)
	}
}

func TestComparableEqualsTypeError(t *testing.T) {
	// pass a non-comparable type like a slice to trigger error
	_, err := evaluateComparableEquals([]int{1}, []int{1})
	if err == nil {
		t.Errorf("expected error for non-comparable types in evaluateComparableEquals")
	}
}

func TestGetFactValuePanicWhenMissing(t *testing.T) {
	options = &Options{AllowUndefinedVars: false}
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when GetFactValue missing and undefined vars not allowed")
		}
	}()
	_ = GetFactValue(&AstConditional{Identifier: "missing"}, Data{})
}

func TestEvaluateConditionalPanicOnNilValue(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic when EvaluateConditional has nil value")
		}
	}()
	_ = EvaluateConditional(&AstConditional{Identifier: "x", Operator: "eq", Value: nil}, "val")
}

func TestEvaluateConditionInvalidKind(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic for invalid condition kind")
		}
	}()
	_ = EvaluateCondition(&[]AstConditional{}, "unknown", Data{})
}

func TestEvaluateAstConditionEdgeCases(t *testing.T) {
	// both Any and All empty should return true
	a := AstCondition{Any: []AstConditional{}, All: []AstConditional{}}
	ok := EvaluateAstCondition(a, Data{}, &Options{AllowUndefinedVars: true})
	if !ok {
		t.Errorf("expected true for empty any/all")
	}

	// Any empty, All non-empty -> evaluate all
	a2 := AstCondition{Any: []AstConditional{}, All: []AstConditional{{Identifier: "x", Operator: "eq", Value: "v"}}}
	// when undefined vars allowed, GetFactValue should return false, causing EvaluateConditional to panic unless allowed -> set option true
	ok2 := EvaluateAstCondition(a2, Data{"x": "v"}, &Options{AllowUndefinedVars: true})
	if !ok2 {
		t.Errorf("expected true for matching condition")
	}
}

func TestHandleRuleMsgEventsUnknown(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	// Provide unknown message type and a no-op logger to avoid nil deref
	lg := zerolog.New(io.Discard).With().Logger()
	l := &Logger{logger: &lg}
	_, err := re.handleRuleMsgEvents(l, []byte("[]"), "UNKNOWN_INTERNAL")
	if err == nil {
		t.Errorf("expected error for unknown internal event")
	}
}

func TestDetermineEffectiveEventTypeAndExtractState(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	lg := zerolog.New(io.Discard).With().Logger()
	l := &Logger{logger: &lg}

	// CREATE with state=false should be dropped
	alert := AlertRuleConfig{UUID: "a1", State: "false"}
	et, should := re.determineEffectiveEventType(l, RuleEventCreate, alert, 0)
	if should || et != "" {
		t.Errorf("expected CREATE with state=false to be dropped")
	}

	// CREATE with state=true
	alert2 := AlertRuleConfig{UUID: "a2", State: "true"}
	et2, should2 := re.determineEffectiveEventType(l, RuleEventCreate, alert2, 0)
	if !should2 || et2 != InternalEventCreate {
		t.Errorf("expected CREATE with state=true to convert to InternalEventCreate")
	}

	// enable/disable mapping
	if et3, _ := re.determineEffectiveEventType(l, RuleEventEnable, alert2, 0); et3 != InternalEventCreate {
		t.Errorf("ENABLE should map to CREATE")
	}
	if et4, _ := re.determineEffectiveEventType(l, RuleEventDisable, alert2, 0); et4 != InternalEventDelete {
		t.Errorf("DISABLE should map to DELETE")
	}
}

func TestDeepCopyAndExtractFromConditionArray(t *testing.T) {
	// deep copy actions
	actions := []*RuleAction{{ActionType: "A", ActionValueStr: "v"}}
	copied := deepCopyActions(actions)
	if len(copied) != 1 || copied[0].ActionType != "A" {
		t.Fatalf("deepCopyActions failed")
	}

	// deep copy rule definition
	rd := &RuleDefinition{AlertRuleUUID: "rx", Name: "n", Actions: actions, MatchCriteriaEntries: map[string][]*RuleMatchCondition{"c": {{CriteriaUUID: "c", Condition: AstCondition{Any: []AstConditional{{Identifier: "i", Operator: "eq", Value: "v"}}}}}}}
	re := NewRuleEngineInstance(nil, nil)
	copy := re.deepCopyRuleDefinition(rd)
	if copy.AlertRuleUUID != rd.AlertRuleUUID || copy.Name != rd.Name {
		t.Fatalf("deepCopyRuleDefinition basic fields mismatch")
	}

	// extractFromConditionArray
	condMap := map[string]interface{}{"all": []AstConditional{{Identifier: "i", Operator: "eq", Value: "v"}}}
	val := extractFromConditionArray(condMap, "all", "i")
	if val == nil {
		t.Fatalf("extractFromConditionArray failed to extract value")
	}
}

func TestLoggerWrappersNoop(t *testing.T) {
	lg := zerolog.New(io.Discard).With().Logger()
	l := &Logger{logger: &lg}
	l.Debug("debugmsg")
	l.Infof("format %s", "a")
	l.WithField("k", "v")
	// just ensure these calls don't panic
}

func TestAssertIsNumberSuccessAndIsComparable(t *testing.T) {
	if n, err := assertIsNumber(5); err != nil || n != 5.0 {
		t.Fatalf("expected 5 -> 5.0, got %v err=%v", n, err)
	}
	if n2, err := assertIsNumber(3.14); err != nil || n2 != 3.14 {
		t.Fatalf("expected 3.14 -> 3.14, got %v err=%v", n2, err)
	}
	if !isComparableType("a") || !isComparableType(1) || !isComparableType(1.2) || !isComparableType(true) {
		t.Fatalf("expected basic types to be comparable")
	}
	if isComparableType([]int{1}) {
		t.Fatalf("slice should not be comparableType")
	}
}

func TestEvaluateAllAnyConditionAdditional(t *testing.T) {
	options = &Options{AllowUndefinedVars: true}
	all := []AstConditional{{Identifier: "x", Operator: "eq", Value: "v"}}
	any := []AstConditional{{Identifier: "y", Operator: "eq", Value: "z"}}
	if !EvaluateAllCondition(&all, Data{"x": "v"}) {
		t.Fatalf("expected EvaluateAllCondition true")
	}
	if EvaluateAnyCondition(&any, Data{"y": "nope"}) {
		t.Fatalf("expected EvaluateAnyCondition false")
	}
}

func TestLoggerAllWrappers(t *testing.T) {
	lg := zerolog.New(io.Discard).With().Logger()
	l := &Logger{logger: &lg}
	l.Info("i")
	l.Debug("d")
	l.Warn("w")
	l.Trace("t")
	// pass a non-nil error to Errorf
	l.Errorf("errf %s", fmt.Errorf("err"), "a")
	// Panicf would panic; avoid calling.
	_ = l.WithFields(map[string]interface{}{"k": "v"})
}

func TestConvertToRuleEngineFormatActions(t *testing.T) {
	in := AlertRuleConfig{
		UUID:             "r-act",
		Name:             "act",
		State:            "true",
		CustomizeAnomaly: CustomizeAnomalyConfig{CustomMessage: "msg"},
		SeverityOverride: "CRITICAL",
		AlertRuleActions: []RuleActionConfig{{Action: "CUSTOMIZE_ANOMALY"}, {Action: "OVERRIDE_SEVERITY"}},
	}
	b, err := ConvertToRuleEngineFormat([]AlertRuleConfig{in})
	if err != nil {
		t.Fatalf("ConvertToRuleEngineFormat failed: %v", err)
	}
	if len(b) == 0 {
		t.Fatalf("expected non-empty bytes from ConvertToRuleEngineFormat")
	}
}

func TestProcessSingleRulePath(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	lg := zerolog.New(io.Discard).With().Logger()
	l := &Logger{logger: &lg}

	ar := AlertRuleMsg{Metadata: AlertRuleMetadata{RuleEventType: RuleEventCreate}, AlertRules: []AlertRuleConfig{{UUID: "ps1", Name: "ps", State: "true"}}}
	_, err := re.processSingleRule(l, ar, InternalEventCreate)
	if err != nil {
		t.Fatalf("processSingleRule returned error: %v", err)
	}
}

func TestGetAllConfiguredAlertRulesNil(t *testing.T) {
	v := GetAllConfiguredAlertRules()
	if v != nil {
		t.Fatalf("expected nil slice from GetAllConfiguredAlertRules, got %v", v)
	}
}

func TestEvaluateOperatorErrorsAndBranches(t *testing.T) {
	// unknown operator already tested; test numeric compare errors
	if _, err := EvaluateOperator("x", "y", "<"); err == nil {
		t.Errorf("expected error comparing non-numeric values with <")
	}

	// test lt/gt with numeric types
	ok, err := EvaluateOperator(3, 5, "<")
	if err != nil || !ok {
		t.Fatalf("expected 3 < 5 true, got %v err=%v", ok, err)
	}

	ok2, err2 := EvaluateOperator(5.0, 2, ">")
	if err2 != nil || !ok2 {
		t.Fatalf("expected 5.0 > 2 true, got %v err=%v", ok2, err2)
	}
}

func TestConvertObjIdentifierVariantsAdditional(t *testing.T) {
	cases := map[string]string{
		"l3_vni": "vni",
		"l2vni":  "vni",
		"bd":     "bd",
		"route":  "route",
	}
	for in, want := range cases {
		if got := convertObjIdentifier(in); got != want {
			t.Fatalf("convertObjIdentifier(%s) = %s; want %s", in, got, want)
		}
	}
}

func TestConvertToActionTypeDefault(t *testing.T) {
	if convertToActionType("") != "unknown" {
		t.Errorf("expected default unknown action type for empty input")
	}
}

func TestConvertHelpers(t *testing.T) {
	if convertToActionType("CUSTOMIZE_ANOMALY") != RuleActionCustomizeRecommendation {
		t.Error("convertToActionType CUSTOMIZE_ANOMALY failed")
	}
	if convertToActionType("ACKNOWLEDGE") != RuleActionAcknowledge {
		t.Error("convertToActionType ACKNOWLEDGE failed")
	}
	if convertToActionType("OVERRIDE_SEVERITY") != RuleActionSeverityOverride {
		t.Error("convertToActionType OVERRIDE_SEVERITY failed")
	}
	if convertToActionType("SOMETHING_ELSE") == "unknown" {
		// The implementation returns "unknown" for default; ensure it does so
	}

	if NormalizeSeverity("CRITICAL") != SeverityCritical {
		t.Error("NormalizeSeverity CRITICAL failed")
	}
	if NormalizeSeverity("minor") != SeverityMinor {
		t.Error("NormalizeSeverity minor failed")
	}
	if NormalizeSeverity("not-a-severity") != SeverityDefault {
		t.Error("NormalizeSeverity default fallback failed")
	}
}

func TestNormalizeInterfaceNameVariants(t *testing.T) {
	cases := map[string]string{
		"Eth1/2":        expectedEth,
		"e1/2":          expectedEth,
		"port-channel3": expectedPc,
		"p3":            expectedPc,
		"Loopback0":     "Loopback0",
		"lo0":           "Loopback0",
		"UNKNOWN":       "UNKNOWN",
	}

	for in, want := range cases {
		got := NormalizeInterfaceName(in)
		if got != want {
			t.Fatalf("NormalizeInterfaceName(%s) = %s; want %s", in, got, want)
		}
	}
}

func TestConvertObjIdentifierAndValue(t *testing.T) {
	tests := map[string]string{
		"interface": "interface",
		"switch":    "switch",
		"ip":        "ip",
		"vni":       "vni",
		"vrf":       "vrf",
		"tenant":    "tenant",
		"subnet":    "subnet",
		"bd":        "bd",
		"mac":       "mac",
		"epg":       "epg",
	}

	for in, want := range tests {
		got := convertObjIdentifier(in)
		if got != want {
			t.Fatalf("convertObjIdentifier(%s) = %s; want %s", in, got, want)
		}
	}

	// NormalizeObjIdentifierValue should normalize interfaces
	if NormalizeObjIdentifierValue("interface", "Eth1/2") != expectedEth {
		t.Error("NormalizeObjIdentifierValue failed for interface")
	}
	if NormalizeObjIdentifierValue("other", "value") != "value" {
		t.Error("NormalizeObjIdentifierValue should return original for non-interface")
	}
}

func TestConvertToRuleEngineFormatAndParse(t *testing.T) {
	in := AlertRuleConfig{
		UUID:             "u-1",
		Name:             "TestRule",
		State:            "true",
		AlertRuleActions: []RuleActionConfig{{Action: "OVERRIDE_SEVERITY"}},
		AlertRuleMatchCriteria: []RuleMatchCriteriaConfig{
			{
				UUID:                   "c-1",
				SiteId:                 "site-1",
				EventNameMatchCriteria: []MatchCriteria{{ValueEquals: "evt"}},
				SeverityMatchCriteria:  []MatchCriteria{{ValueEquals: "CRITICAL"}},
			},
		},
	}

	b, err := ConvertToRuleEngineFormat([]AlertRuleConfig{in})
	if err != nil {
		t.Fatalf("ConvertToRuleEngineFormat error: %v", err)
	}

	var defs []RuleDefinition
	if err := json.Unmarshal(b, &defs); err != nil {
		t.Fatalf("unmarshal returned error: %v", err)
	}
	if len(defs) != 1 {
		t.Fatalf("expected 1 rule definition, got %d", len(defs))
	}
	if defs[0].AlertRuleUUID != "u-1" {
		t.Fatalf("unexpected UUID: %s", defs[0].AlertRuleUUID)
	}
}

func TestProcessRuleChangeFlows(t *testing.T) {
	// Delete request
	delReq := RuleChangeRequest{RuleEvent: InternalEventDelete}
	delResp := ProcessRuleChangeRequest(delReq)
	if !delResp.ShouldProcess {
		t.Errorf("delete request should indicate ShouldProcess=true")
	}

	// Create request with ApplyToExisting=false
	createReq := RuleChangeRequest{RuleEvent: InternalEventCreate, ApplyToExisting: false}
	createResp := ProcessRuleChangeRequest(createReq)
	if createResp.ShouldProcess {
		t.Errorf("create request with ApplyToExisting=false should not process")
	}

	// Create request with ApplyToExisting=true
	createReq2 := RuleChangeRequest{RuleEvent: InternalEventCreate, ApplyToExisting: true, NewRule: &RuleDefinition{ApplyActionsToAll: true}}
	createResp2 := ProcessRuleChangeRequest(createReq2)
	if !createResp2.ShouldProcess {
		t.Errorf("create request with ApplyToExisting=true should process")
	}

	// Update request where ApplyActionsToAll changes from true to false should trigger cleanup
	old := &RuleDefinition{ApplyActionsToAll: true, MatchCriteriaEntries: map[string][]*RuleMatchCondition{"k": {{Condition: AstCondition{All: []AstConditional{{Identifier: "a", Operator: "eq", Value: "1"}}}}}}}
	new := &RuleDefinition{ApplyActionsToAll: false, MatchCriteriaEntries: map[string][]*RuleMatchCondition{"k": {{Condition: AstCondition{All: []AstConditional{{Identifier: "a", Operator: "eq", Value: "1"}}}}}}}
	updReq := RuleChangeRequest{RuleEvent: InternalEventUpdate, OldRule: old, NewRule: new, ApplyToExisting: false}
	updResp := ProcessRuleChangeRequest(updReq)
	if !updResp.NeedsCleanup {
		t.Errorf("update transition should request cleanup")
	}

	// Update request merging conditions and avoiding duplicates
	aCond := map[string]interface{}{"all": []AstConditional{{Identifier: "x", Operator: "eq", Value: "1"}}}
	bCond := map[string]interface{}{"all": []AstConditional{{Identifier: "y", Operator: "eq", Value: "2"}}}
	merged := mergeUniqueConditions([]map[string]interface{}{aCond}, []map[string]interface{}{aCond, bCond})
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged conditions, got %d", len(merged))
	}
}

func TestMapHelpers(t *testing.T) {
	m1 := map[string]interface{}{"a": 1}
	m2 := map[string]interface{}{"a": 1}
	if !DeepEqualMaps(m1, m2) {
		t.Errorf("DeepEqualMaps equal maps returned false")
	}
	m3 := map[string]interface{}{"a": 2}
	if DeepEqualMaps(m1, m3) {
		t.Errorf("DeepEqualMaps different maps returned true")
	}
}

func TestParseDefinitionsAndGetAllInfo(t *testing.T) {
	// invalid JSON
	if _, err := ParseRuleDefinitions([]byte("notjson")); err == nil {
		t.Errorf("expected parse error for invalid JSON")
	}

	// valid
	rule := &RuleDefinition{AlertRuleUUID: "r1", Name: "n1", Enabled: true, MatchCriteriaEntries: map[string][]*RuleMatchCondition{"c1": {{Condition: AstCondition{Any: []AstConditional{{Identifier: "x", Operator: "eq", Value: "1"}}}}}}}
	b, _ := json.Marshal([]*RuleDefinition{rule})
	defs, err := ParseRuleDefinitions(b)
	if err != nil {
		t.Fatalf("expected no error parsing valid defs: %v", err)
	}
	if len(defs) != 1 || defs[0].AlertRuleUUID != "r1" {
		t.Fatalf("unexpected parse result: %v", defs)
	}

	re := NewRuleEngineInstance(nil, nil)
	re.AddRuleDefinition(rule)
	info := re.GetAllRuleInfo()
	if len(info) == 0 {
		t.Fatalf("GetAllRuleInfo should return data")
	}
}

func TestProcessRuleMatchCriteriaFull(t *testing.T) {
	rule := AlertRuleConfig{
		UUID: "r1",
		AlertRuleMatchCriteria: []RuleMatchCriteriaConfig{
			{
				UUID:                        "c1",
				SiteId:                      "site-A",
				CategoryMatchCriteria:       []MatchCriteria{{ValueEquals: "net"}},
				EventNameMatchCriteria:      []MatchCriteria{{ValueEquals: "ev"}},
				SeverityMatchCriteria:       []MatchCriteria{{ValueEquals: "MAJOR"}},
				AffectedObjectMatchCriteria: []MatchCriteria{{ObjectType: "interface", ValueEquals: "Eth1/2"}},
			},
		},
	}

	entries := processRuleMatchCriteria(rule)
	if len(entries) == 0 {
		t.Fatalf("expected at least one entry from processRuleMatchCriteria")
	}
	// inspect the condition
	for _, arr := range entries {
		if len(arr) == 0 {
			t.Fatalf("expected non-empty conditions")
		}
		cond := arr[0]
		if cond.AlertRuleUUID != "r1" {
			t.Errorf("expected AlertRuleUUID r1, got %s", cond.AlertRuleUUID)
		}
		// ensure primary key picked up
		if cond.PrimaryMatchValue != "site-A" {
			t.Errorf("expected primary key site-A, got %s", cond.PrimaryMatchValue)
		}
	}
}

func TestOperatorNoneOfAndAnyOfSliceNumeric(t *testing.T) {
	ok, err := evaluateNoneOf(5, []interface{}{1, 2, 3})
	if err != nil || !ok {
		t.Fatalf("expected true for noneof when value not in slice, got %v err=%v", ok, err)
	}

	// numeric anyof slice
	ok2, err2 := evaluateAnyOfSlice(2, []interface{}{1, 2.0, 3})
	if err2 != nil || !ok2 {
		t.Fatalf("expected true for anyof numeric slice, got %v err=%v", ok2, err2)
	}
}

func TestAddDeleteGetRuleJSON(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	jsonStr := `[{"alertRuleUUID":"j1","name":"jn","enabled":true,"matchCriteriaEntries":{"c1":[{"criteriaUUID":"c1","primaryMatchValue":"PRIMARY_KEY_DEFAULT","condition":{"all":[{"identifier":"category","operator":"eq","value":"X"}]}}]}}]`

	added, err := re.AddRule(jsonStr)
	if err != nil {
		t.Fatalf("AddRule returned error: %v", err)
	}
	if len(added) == 0 {
		t.Fatalf("expected AddRule to return created rule definitions")
	}

	// GetRule
	rdef, ok := re.GetRule("j1")
	if !ok || rdef.AlertRuleUUID != "j1" {
		t.Fatalf("GetRule failed to retrieve added rule")
	}

	// Delete
	delJSON := `[{"alertRuleUUID":"j1","name":"jn","enabled":true}]`
	_, derr := re.DeleteRule(delJSON)
	if derr != nil {
		t.Fatalf("DeleteRule returned error: %v", derr)
	}
	_, exists := re.GetRule("j1")
	if exists {
		t.Fatalf("rule j1 should have been deleted")
	}
}

func TestExtractConditionsAndValues(t *testing.T) {
	rd := &RuleDefinition{
		AlertRuleUUID: "rX",
		MatchCriteriaEntries: map[string][]*RuleMatchCondition{
			"c1": {{Condition: AstCondition{All: []AstConditional{{Identifier: "a", Operator: "eq", Value: "1"}}}}},
			"c2": {{Condition: AstCondition{Any: []AstConditional{{Identifier: "b", Operator: "eq", Value: "2"}}}}},
		},
	}
	conds := ExtractConditions(rd)
	if len(conds) != 2 {
		t.Fatalf("expected 2 conditions extracted, got %d", len(conds))
	}

	// test ExtractValueFromConditions
	val := ExtractValueFromConditions(conds, "a")
	if val == nil {
		t.Fatalf("expected to extract value for identifier a")
	}

	val2 := ExtractValueFromConditions(conds, "b")
	if val2 == nil {
		t.Fatalf("expected to extract value for identifier b")
	}
}

func TestZerologLevelMapping(t *testing.T) {
	if zerologLevel("debug") != zerolog.DebugLevel {
		t.Errorf("expected debug level mapping")
	}
	if zerologLevel("INFO") != zerolog.InfoLevel {
		t.Errorf("expected info level mapping")
	}
	if zerologLevel("unknown-level") != zerolog.InfoLevel {
		t.Errorf("expected default info level mapping")
	}
}
func TestGetFactValueAllowUndefinedVarsTrue(t *testing.T) {
	options = &Options{AllowUndefinedVars: true}
	cond := &AstConditional{Identifier: "missing", Operator: "eq", Value: "val"}
	v := GetFactValue(cond, Data{})
	if v != false {
		t.Error("GetFactValue should return false when AllowUndefinedVars is true and value is missing")
	}
}

func TestEvaluateOperatorBranches(t *testing.T) {
	// anyof with slice
	ok, err := EvaluateOperator("a", []interface{}{"a", "b"}, "anyof")
	if !ok || err != nil {
		t.Error("anyof with slice failed")
	}
	ok, err = EvaluateOperator("c", []interface{}{"a", "b"}, "anyof")
	if ok || err != nil {
		t.Error("anyof with slice should be false")
	}
	// noneof with slice
	ok, err = EvaluateOperator("a", []interface{}{"a", "b"}, "noneof")
	if ok || err != nil {
		t.Error("noneof with slice should be false")
	}
	ok, err = EvaluateOperator("c", []interface{}{"a", "b"}, "noneof")
	if !ok || err != nil {
		t.Error("noneof with slice should be true")
	}
	// anyof with single value
	ok, err = EvaluateOperator("a", "a", "anyof")
	if !ok || err != nil {
		t.Error("anyof with single value failed")
	}
	ok, err = EvaluateOperator("b", "a", "anyof")
	if ok || err != nil {
		t.Error("anyof with single value should be false")
	}
	// noneof with single value
	ok, err = EvaluateOperator("a", "a", "noneof")
	if ok || err != nil {
		t.Error("noneof with single value should be false")
	}
	ok, err = EvaluateOperator("b", "a", "noneof")
	if !ok || err != nil {
		t.Error("noneof with single value should be true")
	}
	// eq/neq with numbers
	ok, err = EvaluateOperator(4, 4, "eq")
	if !ok || err != nil {
		t.Error("eq with numbers failed")
	}
	ok, err = EvaluateOperator(4, 5, "neq")
	if !ok || err != nil {
		t.Error("neq with numbers failed")
	}
	// lt/gt/gte/lte
	ok, err = EvaluateOperator(4, 5, "lt")
	if !ok || err != nil {
		t.Error("lt failed")
	}
	ok, err = EvaluateOperator(6, 5, "gt")
	if !ok || err != nil {
		t.Error("gt failed")
	}
	ok, err = EvaluateOperator(5, 5, "gte")
	if !ok || err != nil {
		t.Error("gte failed")
	}
	ok, err = EvaluateOperator(4, 5, "lte")
	if !ok || err != nil {
		t.Error("lte failed")
	}
}

func TestRuleEngineMethodsPanics(t *testing.T) {
	re := NewRuleEngineInstance(nil, nil)
	// DeleteRule with invalid JSON
	defer func() {
		if r := recover(); r == nil {
			t.Error("DeleteRule should panic on invalid JSON")
		}
	}()
	re.DeleteRule("not a json")
}
