package ruleenginelib

import (
	"fmt"
)

type Data map[string]interface{}
type Options struct {
	AllowUndefinedVars bool
}

var options *Options

func EvaluateConditional(conditional *AstConditional, dataValue interface{}) bool {
	if conditional.Value == nil {
		panic(fmt.Sprintf("conditional %s has no value", conditional.Fact))
	}
	ok, err := EvaluateOperator(dataValue, conditional.Value, conditional.Operator)
	if err != nil {
		panic(err)
	}
	return ok
}

func GetFactValue(condition *AstConditional, data Data) interface{} {
	value := data[condition.Fact]

	if value == nil {
		if options.AllowUndefinedVars {
			return false
		}
		panic(fmt.Sprintf("value for identifier %s not found", condition.Fact))
	}

	return value
}

func EvaluateAllCondition(conditions *[]AstConditional, dataMap Data) bool {
	isFalse := false

	for _, condition := range *conditions {
		value := GetFactValue(&condition, dataMap)
		if !EvaluateConditional(&condition, value) {
			isFalse = true
		}

		if isFalse {
			return false
		}
	}

	return true
}

func EvaluateAnyCondition(conditions *[]AstConditional, data Data) bool {
	for _, condition := range *conditions {
		value := GetFactValue(&condition, data)
		if EvaluateConditional(&condition, value) {
			return true
		}
	}

	return false
}

func EvaluateCondition(conditions *[]AstConditional, kind string, dataMap Data) bool {
	switch kind {
	case "all":
		return EvaluateAllCondition(conditions, dataMap)
	case "any":
		return EvaluateAnyCondition(conditions, dataMap)
	default:
		panic(fmt.Sprintf("condition type %s is invalid", kind))
	}
}

func EvaluateAstCondition(aCond AstCondition, dataMap Data, opts *Options) bool {
	options = opts
	any, all := false, false

	if len(aCond.Any) == 0 {
		any = true
	} else {
		any = EvaluateCondition(&aCond.Any, "any", dataMap)
	}
	if len(aCond.All) == 0 {
		all = true
	} else {
		all = EvaluateCondition(&aCond.All, "all", dataMap)
	}

	return any && all
}
