package activeflowhandler

import (
	"testing"

	"monorepo/bin-flow-manager/models/action"

	"go.uber.org/mock/gomock"
)

func Test_compareCondition_string(t *testing.T) {

	tests := []struct {
		name string

		condition action.OptionConditionCommonCondition
		a         string
		b         string

		expectedRes bool
	}{
		{
			name: "string equal match",

			condition: action.OptionConditionCommonConditionEqual,

			a: "hello",
			b: "hello",

			expectedRes: true,
		},
		{
			name: "string not equal match",

			condition: action.OptionConditionCommonConditionNotEqual,

			a: "hello",
			b: "world",

			expectedRes: true,
		},
		{
			name: "string greater match",

			condition: action.OptionConditionCommonConditionGreater,

			a: "zello",
			b: "world",

			expectedRes: true,
		},
		{
			name: "string greater equal match",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: "hello",
			b: "hello",

			expectedRes: true,
		},
		{
			name: "string less match",

			condition: action.OptionConditionCommonConditionLess,

			a: "hello",
			b: "world",

			expectedRes: true,
		},
		{
			name: "string less equal match",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: "hello",
			b: "hello",

			expectedRes: true,
		},

		{
			name: "string equal unmatch",

			condition: action.OptionConditionCommonConditionEqual,

			a: "hello",
			b: "world",

			expectedRes: false,
		},
		{
			name: "string not equal unmatch",

			condition: action.OptionConditionCommonConditionNotEqual,

			a: "hello",
			b: "hello",

			expectedRes: false,
		},
		{
			name: "string greater unmatch",

			condition: action.OptionConditionCommonConditionGreater,

			a: "hello",
			b: "hello",

			expectedRes: false,
		},
		{
			name: "string greater equal unmatch",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: "hello",
			b: "world",

			expectedRes: false,
		},
		{
			name: "string less unmatch",

			condition: action.OptionConditionCommonConditionLess,

			a: "hello",
			b: "hello",

			expectedRes: false,
		},
		{
			name: "string less equal unmatch",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: "world",
			b: "hello",

			expectedRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res := compareCondition(tt.condition, tt.a, tt.b)
			if res != tt.expectedRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_compareCondition_float(t *testing.T) {

	tests := []struct {
		name string

		condition action.OptionConditionCommonCondition
		a         float32
		b         float32

		expectedRes bool
	}{
		{
			name: "equal match",

			condition: action.OptionConditionCommonConditionEqual,

			a: 123.1,
			b: 123.1,

			expectedRes: true,
		},
		{
			name: "not equal match",

			condition: action.OptionConditionCommonConditionNotEqual,

			a: 123.1,
			b: 456.1,

			expectedRes: true,
		},
		{
			name: "greater match",

			condition: action.OptionConditionCommonConditionGreater,

			a: 456.1,
			b: 123.1,

			expectedRes: true,
		},
		{
			name: "greater equal(greater) match",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 457.1,
			b: 456.1,

			expectedRes: true,
		},
		{
			name: "greater equal(equal) match",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 456.1,
			b: 456.1,

			expectedRes: true,
		},
		{
			name: "less match",

			condition: action.OptionConditionCommonConditionLess,

			a: 123.1,
			b: 456.1,

			expectedRes: true,
		},
		{
			name: "less equal(less) match",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: 123.1,
			b: 456.1,

			expectedRes: true,
		},
		{
			name: "less equal(equal) match",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: 123.1,
			b: 123.1,

			expectedRes: true,
		},
		{
			name: "equal unmatch",

			condition: action.OptionConditionCommonConditionEqual,

			a: 123.1,
			b: 456.1,

			expectedRes: false,
		},
		{
			name: "not equal unmatch",

			condition: action.OptionConditionCommonConditionNotEqual,

			a: 123.1,
			b: 123.1,

			expectedRes: false,
		},
		{
			name: "greater unmatch",

			condition: action.OptionConditionCommonConditionGreater,

			a: 123.1,
			b: 456.1,

			expectedRes: false,
		},
		{
			name: "greater unmatch",

			condition: action.OptionConditionCommonConditionGreater,

			a: 123.1,
			b: 123.1,

			expectedRes: false,
		},
		{
			name: "greater equal(greater) unmatch",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 123.1,
			b: 456.1,

			expectedRes: false,
		},
		{
			name: "greater equal unmatch",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 123.1,
			b: 456.1,

			expectedRes: false,
		},
		{
			name: "less unmatch",

			condition: action.OptionConditionCommonConditionLess,

			a: 123.1,
			b: 123.1,

			expectedRes: false,
		},
		{
			name: "less unmatch",

			condition: action.OptionConditionCommonConditionLess,

			a: 456.1,
			b: 123.1,

			expectedRes: false,
		},
		{
			name: "less equal unmatch",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: 456.1,
			b: 123.1,

			expectedRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res := compareCondition(tt.condition, tt.a, tt.b)
			if res != tt.expectedRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedRes, res)
			}
		})
	}
}

func Test_compareCondition_int(t *testing.T) {

	tests := []struct {
		name string

		condition action.OptionConditionCommonCondition
		a         int
		b         int

		expectedRes bool
	}{
		{
			name: "equal match",

			condition: action.OptionConditionCommonConditionEqual,

			a: 123,
			b: 123,

			expectedRes: true,
		},
		{
			name: "not equal match",

			condition: action.OptionConditionCommonConditionNotEqual,

			a: 123,
			b: 456,

			expectedRes: true,
		},
		{
			name: "greater match",

			condition: action.OptionConditionCommonConditionGreater,

			a: 456,
			b: 123,

			expectedRes: true,
		},
		{
			name: "greater equal(greater) match",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 457,
			b: 456,

			expectedRes: true,
		},
		{
			name: "greater equal(equal) match",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 456,
			b: 456,

			expectedRes: true,
		},
		{
			name: "less match",

			condition: action.OptionConditionCommonConditionLess,

			a: 123,
			b: 456,

			expectedRes: true,
		},
		{
			name: "less equal(less) match",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: 123,
			b: 456,

			expectedRes: true,
		},
		{
			name: "less equal(equal) match",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: 123,
			b: 123,

			expectedRes: true,
		},
		{
			name: "equal unmatch",

			condition: action.OptionConditionCommonConditionEqual,

			a: 123,
			b: 456,

			expectedRes: false,
		},
		{
			name: "not equal unmatch",

			condition: action.OptionConditionCommonConditionNotEqual,

			a: 123,
			b: 123,

			expectedRes: false,
		},
		{
			name: "greater unmatch",

			condition: action.OptionConditionCommonConditionGreater,

			a: 123,
			b: 456,

			expectedRes: false,
		},
		{
			name: "greater unmatch",

			condition: action.OptionConditionCommonConditionGreater,

			a: 123,
			b: 123,

			expectedRes: false,
		},
		{
			name: "greater equal(greater) unmatch",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 123,
			b: 456,

			expectedRes: false,
		},
		{
			name: "greater equal unmatch",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 123,
			b: 456,

			expectedRes: false,
		},
		{
			name: "less unmatch",

			condition: action.OptionConditionCommonConditionLess,

			a: 123,
			b: 123,

			expectedRes: false,
		},
		{
			name: "less unmatch",

			condition: action.OptionConditionCommonConditionLess,

			a: 456,
			b: 123,

			expectedRes: false,
		},
		{
			name: "less equal unmatch",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: 456,
			b: 123,

			expectedRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res := compareCondition(tt.condition, tt.a, tt.b)
			if res != tt.expectedRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedRes, res)
			}
		})
	}
}
