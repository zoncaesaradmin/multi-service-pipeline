package ruleenginelib

import (
	"fmt"
)

func EvaluateOperator(dataValue, value interface{}, operator string) (bool, error) {
	switch operator {
	case "anyof":
		switch valSlice := value.(type) {
		case []interface{}:
			factNum, err := assertIsNumber(dataValue)
			if err == nil {
				for _, val := range valSlice {
					valueNum, err := assertIsNumber(val)
					if err == nil && valueNum == factNum {
						return true, nil
					}
				}
			} else {
				for _, val := range valSlice {
					if dataValue == val {
						return true, nil
					}
				}
			}
			return false, nil
		default:
			factNum, err := assertIsNumber(dataValue)
			if err == nil {
				valueNum, err := assertIsNumber(value)
				if err != nil {
					return false, err
				}
				return factNum == valueNum, nil
			}
			return dataValue == value, nil
		}
	case "noneof":
		switch valSlice := value.(type) {
		case []interface{}:
			factNum, err := assertIsNumber(dataValue)
			if err == nil {
				for _, val := range valSlice {
					valueNum, err := assertIsNumber(val)
					if err == nil && valueNum == factNum {
						return false, nil
					}
				}
			} else {
				for _, val := range valSlice {
					if dataValue == val {
						return false, nil
					}
				}
			}
			return true, nil
		default:
			factNum, err := assertIsNumber(dataValue)
			if err == nil {
				valueNum, err := assertIsNumber(value)
				if err != nil {
					return false, err
				}
				return factNum != valueNum, nil
			}
			return dataValue != value, nil
		}
	case "=":
		fallthrough
	case "eq":
		factNum, err := assertIsNumber(dataValue)
		if err == nil {
			valueNum, err := assertIsNumber(value)
			if err != nil {
				return false, err
			}
			return factNum == valueNum, nil
		}
		// If not numbers, check if types are comparable
		switch dataValue.(type) {
		case int, float64, string, bool:
			switch value.(type) {
			case int, float64, string, bool:
				return dataValue == value, nil
			default:
				return false, fmt.Errorf("eq: value type %T not comparable", value)
			}
		default:
			return false, fmt.Errorf("eq: dataValue type %T not comparable", dataValue)
		}
	case "!=":
		fallthrough
	case "neq":
		factNum, err := assertIsNumber(dataValue)
		if err == nil {
			valueNum, err := assertIsNumber(value)
			if err != nil {
				return false, err
			}
			return factNum != valueNum, nil
		}
		// If not numbers, check if types are comparable
		switch dataValue.(type) {
		case int, float64, string, bool:
			switch value.(type) {
			case int, float64, string, bool:
				return dataValue != value, nil
			default:
				return false, fmt.Errorf("neq: value type %T not comparable", value)
			}
		default:
			return false, fmt.Errorf("neq: dataValue type %T not comparable", dataValue)
		}

	case "<":
		fallthrough
	case "lt":
		factNum, err := assertIsNumber(dataValue)
		if err != nil {
			return false, err
		}
		valueNum, err := assertIsNumber(value)
		if err != nil {
			return false, err
		}

		return factNum < valueNum, nil

	case ">":
		fallthrough
	case "gt":
		factNum, err := assertIsNumber(dataValue)
		if err != nil {
			return false, err
		}
		valueNum, err := assertIsNumber(value)
		if err != nil {
			return false, err
		}

		return factNum > valueNum, nil

	case ">=":
		fallthrough
	case "gte":
		factNum, err := assertIsNumber(dataValue)
		if err != nil {
			return false, err
		}
		valueNum, err := assertIsNumber(value)
		if err != nil {
			return false, err
		}

		return factNum >= valueNum, nil

	case "<=":
		fallthrough
	case "lte":
		factNum, err := assertIsNumber(dataValue)
		if err != nil {
			return false, err
		}
		valueNum, err := assertIsNumber(value)
		if err != nil {
			return false, err
		}

		return factNum <= valueNum, nil

	default:
		return false, fmt.Errorf("unrecognised operator %s", operator)
	}
}

func assertIsNumber(v interface{}) (float64, error) {
	isFloat := true
	var d int
	var f float64

	d, ok := v.(int)
	if !ok {
		f, ok = v.(float64)
		if !ok {
			return 0, fmt.Errorf("%s is not a number", v)
		}
	} else {
		isFloat = false
	}

	if isFloat {
		return f, nil
	}
	return float64(d), nil
}
