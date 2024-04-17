package activeflowhandler

import (
	"time"

	"monorepo/bin-flow-manager/models/action"
)

type types interface {
	int | int8 | int16 | int32 | int64 | float32 | float64 | string | time.Month
}

func compareCondition[T types](condition action.OptionConditionCommonCondition, a T, b T) bool {

	switch condition {
	case action.OptionConditionCommonConditionEqual:
		return compareEqual(a, b)
	case action.OptionConditionCommonConditionNotEqual:
		return compareNotEqual(a, b)
	case action.OptionConditionCommonConditionGreater:
		return compareGreater(a, b)
	case action.OptionConditionCommonConditionGreaterEqual:
		return compareGreaterEqual(a, b)
	case action.OptionConditionCommonConditionLess:
		return compareLess(a, b)
	case action.OptionConditionCommonConditionLessEqual:
		return compareLessEqual(a, b)

	default:
		// no handler found
		return false
	}
}

func compareEqual[T types](a T, b T) bool {
	return a == b
}

func compareNotEqual[T types](a T, b T) bool {
	return !compareEqual(a, b)
}

func compareGreater[T types](a T, b T) bool {
	return a > b
}

func compareGreaterEqual[T types](a T, b T) bool {
	return a >= b
}

func compareLess[T types](a T, b T) bool {
	return a < b
}

func compareLessEqual[T types](a T, b T) bool {
	return a <= b
}
