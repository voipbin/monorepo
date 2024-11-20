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

		expectRes bool
	}{
		{
			name: "string equal match",

			condition: action.OptionConditionCommonConditionEqual,

			a: "hello",
			b: "hello",

			expectRes: true,
		},
		{
			name: "string not equal match",

			condition: action.OptionConditionCommonConditionNotEqual,

			a: "hello",
			b: "world",

			expectRes: true,
		},
		{
			name: "string greater match",

			condition: action.OptionConditionCommonConditionGreater,

			a: "zello",
			b: "world",

			expectRes: true,
		},
		{
			name: "string greater equal match",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: "hello",
			b: "hello",

			expectRes: true,
		},
		{
			name: "string less match",

			condition: action.OptionConditionCommonConditionLess,

			a: "hello",
			b: "world",

			expectRes: true,
		},
		{
			name: "string less equal match",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: "hello",
			b: "hello",

			expectRes: true,
		},

		{
			name: "string equal unmatch",

			condition: action.OptionConditionCommonConditionEqual,

			a: "hello",
			b: "world",

			expectRes: false,
		},
		{
			name: "string not equal unmatch",

			condition: action.OptionConditionCommonConditionNotEqual,

			a: "hello",
			b: "hello",

			expectRes: false,
		},
		{
			name: "string greater unmatch",

			condition: action.OptionConditionCommonConditionGreater,

			a: "hello",
			b: "hello",

			expectRes: false,
		},
		{
			name: "string greater equal unmatch",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: "hello",
			b: "world",

			expectRes: false,
		},
		{
			name: "string less unmatch",

			condition: action.OptionConditionCommonConditionLess,

			a: "hello",
			b: "hello",

			expectRes: false,
		},
		{
			name: "string less equal unmatch",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: "world",
			b: "hello",

			expectRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res := compareCondition(tt.condition, tt.a, tt.b)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
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

		expectRes bool
	}{
		{
			name: "equal match",

			condition: action.OptionConditionCommonConditionEqual,

			a: 123.1,
			b: 123.1,

			expectRes: true,
		},
		{
			name: "not equal match",

			condition: action.OptionConditionCommonConditionNotEqual,

			a: 123.1,
			b: 456.1,

			expectRes: true,
		},
		{
			name: "greater match",

			condition: action.OptionConditionCommonConditionGreater,

			a: 456.1,
			b: 123.1,

			expectRes: true,
		},
		{
			name: "greater equal(greater) match",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 457.1,
			b: 456.1,

			expectRes: true,
		},
		{
			name: "greater equal(equal) match",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 456.1,
			b: 456.1,

			expectRes: true,
		},
		{
			name: "less match",

			condition: action.OptionConditionCommonConditionLess,

			a: 123.1,
			b: 456.1,

			expectRes: true,
		},
		{
			name: "less equal(less) match",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: 123.1,
			b: 456.1,

			expectRes: true,
		},
		{
			name: "less equal(equal) match",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: 123.1,
			b: 123.1,

			expectRes: true,
		},
		{
			name: "equal unmatch",

			condition: action.OptionConditionCommonConditionEqual,

			a: 123.1,
			b: 456.1,

			expectRes: false,
		},
		{
			name: "not equal unmatch",

			condition: action.OptionConditionCommonConditionNotEqual,

			a: 123.1,
			b: 123.1,

			expectRes: false,
		},
		{
			name: "greater unmatch",

			condition: action.OptionConditionCommonConditionGreater,

			a: 123.1,
			b: 456.1,

			expectRes: false,
		},
		{
			name: "greater unmatch",

			condition: action.OptionConditionCommonConditionGreater,

			a: 123.1,
			b: 123.1,

			expectRes: false,
		},
		{
			name: "greater equal(greater) unmatch",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 123.1,
			b: 456.1,

			expectRes: false,
		},
		{
			name: "greater equal unmatch",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 123.1,
			b: 456.1,

			expectRes: false,
		},
		{
			name: "less unmatch",

			condition: action.OptionConditionCommonConditionLess,

			a: 123.1,
			b: 123.1,

			expectRes: false,
		},
		{
			name: "less unmatch",

			condition: action.OptionConditionCommonConditionLess,

			a: 456.1,
			b: 123.1,

			expectRes: false,
		},
		{
			name: "less equal unmatch",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: 456.1,
			b: 123.1,

			expectRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res := compareCondition(tt.condition, tt.a, tt.b)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
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

		expectRes bool
	}{
		{
			name: "equal match",

			condition: action.OptionConditionCommonConditionEqual,

			a: 123,
			b: 123,

			expectRes: true,
		},
		{
			name: "not equal match",

			condition: action.OptionConditionCommonConditionNotEqual,

			a: 123,
			b: 456,

			expectRes: true,
		},
		{
			name: "greater match",

			condition: action.OptionConditionCommonConditionGreater,

			a: 456,
			b: 123,

			expectRes: true,
		},
		{
			name: "greater equal(greater) match",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 457,
			b: 456,

			expectRes: true,
		},
		{
			name: "greater equal(equal) match",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 456,
			b: 456,

			expectRes: true,
		},
		{
			name: "less match",

			condition: action.OptionConditionCommonConditionLess,

			a: 123,
			b: 456,

			expectRes: true,
		},
		{
			name: "less equal(less) match",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: 123,
			b: 456,

			expectRes: true,
		},
		{
			name: "less equal(equal) match",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: 123,
			b: 123,

			expectRes: true,
		},
		{
			name: "equal unmatch",

			condition: action.OptionConditionCommonConditionEqual,

			a: 123,
			b: 456,

			expectRes: false,
		},
		{
			name: "not equal unmatch",

			condition: action.OptionConditionCommonConditionNotEqual,

			a: 123,
			b: 123,

			expectRes: false,
		},
		{
			name: "greater unmatch",

			condition: action.OptionConditionCommonConditionGreater,

			a: 123,
			b: 456,

			expectRes: false,
		},
		{
			name: "greater unmatch",

			condition: action.OptionConditionCommonConditionGreater,

			a: 123,
			b: 123,

			expectRes: false,
		},
		{
			name: "greater equal(greater) unmatch",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 123,
			b: 456,

			expectRes: false,
		},
		{
			name: "greater equal unmatch",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 123,
			b: 456,

			expectRes: false,
		},
		{
			name: "less unmatch",

			condition: action.OptionConditionCommonConditionLess,

			a: 123,
			b: 123,

			expectRes: false,
		},
		{
			name: "less unmatch",

			condition: action.OptionConditionCommonConditionLess,

			a: 456,
			b: 123,

			expectRes: false,
		},
		{
			name: "less equal unmatch",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: 456,
			b: 123,

			expectRes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			res := compareCondition(tt.condition, tt.a, tt.b)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
