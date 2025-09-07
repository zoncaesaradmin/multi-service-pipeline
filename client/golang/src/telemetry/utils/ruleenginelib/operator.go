package ruleenginelib

import (
	"fmt"
)

// EvaluateOperator evaluates a comparison between dataValue and value using the specified operator
func EvaluateOperator(dataValue, value interface{}, operator string) (bool, error) {
	switch operator {
	case "anyof":
		return evaluateAnyOf(dataValue, value)
	case "noneof":
		return evaluateNoneOf(dataValue, value)
	case "=", "eq":
		return evaluateEquals(dataValue, value)
	case "!=", "neq":
		return evaluateNotEquals(dataValue, value)
	case "<", "lt":
		return evaluateLessThan(dataValue, value)
	case ">", "gt":
		return evaluateGreaterThan(dataValue, value)
	case ">=", "gte":
		return evaluateGreaterThanOrEqual(dataValue, value)
	case "<=", "lte":
		return evaluateLessThanOrEqual(dataValue, value)
	default:
		return false, fmt.Errorf("unrecognised operator %s", operator)
	}
}

// evaluateAnyOf checks if dataValue matches any value in the provided slice or single value
func evaluateAnyOf(dataValue, value interface{}) (bool, error) {
	switch valSlice := value.(type) {
	case []interface{}:
		return evaluateAnyOfSlice(dataValue, valSlice)
	default:
		return evaluateEquals(dataValue, value)
	}
}

// evaluateAnyOfSlice checks if dataValue matches any value in the slice
func evaluateAnyOfSlice(dataValue interface{}, valSlice []interface{}) (bool, error) {
	factNum, err := assertIsNumber(dataValue)
	if err == nil {
		return evaluateAnyOfNumeric(factNum, valSlice), nil
	}
	return evaluateAnyOfGeneric(dataValue, valSlice), nil
}

// evaluateAnyOfNumeric checks numeric values against slice
func evaluateAnyOfNumeric(factNum float64, valSlice []interface{}) bool {
	for _, val := range valSlice {
		if valueNum, err := assertIsNumber(val); err == nil && valueNum == factNum {
			return true
		}
	}
	return false
}

// evaluateAnyOfGeneric checks generic values against slice
func evaluateAnyOfGeneric(dataValue interface{}, valSlice []interface{}) bool {
	for _, val := range valSlice {
		if dataValue == val {
			return true
		}
	}
	return false
}

// evaluateNoneOf checks if dataValue matches none of the values in the provided slice or single value
func evaluateNoneOf(dataValue, value interface{}) (bool, error) {
	result, err := evaluateAnyOf(dataValue, value)
	return !result, err
}

// evaluateEquals checks if two values are equal
func evaluateEquals(dataValue, value interface{}) (bool, error) {
	factNum, err := assertIsNumber(dataValue)
	if err == nil {
		valueNum, err := assertIsNumber(value)
		if err != nil {
			return false, err
		}
		return factNum == valueNum, nil
	}
	return evaluateComparableEquals(dataValue, value)
}

// evaluateComparableEquals checks equality for comparable types
func evaluateComparableEquals(dataValue, value interface{}) (bool, error) {
	if !isComparableType(dataValue) {
		return false, fmt.Errorf("eq: dataValue type %T not comparable", dataValue)
	}
	if !isComparableType(value) {
		return false, fmt.Errorf("eq: value type %T not comparable", value)
	}
	return dataValue == value, nil
}

// evaluateNotEquals checks if two values are not equal
func evaluateNotEquals(dataValue, value interface{}) (bool, error) {
	result, err := evaluateEquals(dataValue, value)
	return !result, err
}

// evaluateLessThan checks if dataValue < value (numeric comparison)
func evaluateLessThan(dataValue, value interface{}) (bool, error) {
	factNum, err := assertIsNumber(dataValue)
	if err != nil {
		return false, err
	}
	valueNum, err := assertIsNumber(value)
	if err != nil {
		return false, err
	}
	return factNum < valueNum, nil
}

// evaluateGreaterThan checks if dataValue > value (numeric comparison)
func evaluateGreaterThan(dataValue, value interface{}) (bool, error) {
	factNum, err := assertIsNumber(dataValue)
	if err != nil {
		return false, err
	}
	valueNum, err := assertIsNumber(value)
	if err != nil {
		return false, err
	}
	return factNum > valueNum, nil
}

// evaluateGreaterThanOrEqual checks if dataValue >= value (numeric comparison)
func evaluateGreaterThanOrEqual(dataValue, value interface{}) (bool, error) {
	factNum, err := assertIsNumber(dataValue)
	if err != nil {
		return false, err
	}
	valueNum, err := assertIsNumber(value)
	if err != nil {
		return false, err
	}
	return factNum >= valueNum, nil
}

// evaluateLessThanOrEqual checks if dataValue <= value (numeric comparison)
func evaluateLessThanOrEqual(dataValue, value interface{}) (bool, error) {
	factNum, err := assertIsNumber(dataValue)
	if err != nil {
		return false, err
	}
	valueNum, err := assertIsNumber(value)
	if err != nil {
		return false, err
	}
	return factNum <= valueNum, nil
}

// isComparableType checks if a value is of a comparable type
func isComparableType(v interface{}) bool {
	switch v.(type) {
	case int, float64, string, bool:
		return true
	default:
		return false
	}
}

// assertIsNumber converts a value to float64 if it's numeric
func assertIsNumber(v interface{}) (float64, error) {
	switch val := v.(type) {
	case int:
		return float64(val), nil
	case float64:
		return val, nil
	default:
		return 0, fmt.Errorf("%v is not a number", v)
	}
}
