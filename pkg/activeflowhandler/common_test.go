package activeflowhandler

import (
	"testing"

	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
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
			name: "equal",

			condition: action.OptionConditionCommonConditionEqual,

			a: "hello",
			b: "hello",

			expectRes: true,
		},
		{
			name: "not equal",

			condition: action.OptionConditionCommonConditionNotEqual,

			a: "hello",
			b: "world",

			expectRes: true,
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
			name: "equal",

			condition: action.OptionConditionCommonConditionEqual,

			a: 123,
			b: 123,

			expectRes: true,
		},
		{
			name: "not equal",

			condition: action.OptionConditionCommonConditionNotEqual,

			a: 123,
			b: 456,

			expectRes: true,
		},
		{
			name: "greater",

			condition: action.OptionConditionCommonConditionGreater,

			a: 456,
			b: 123,

			expectRes: true,
		},
		{
			name: "greater equal(greater) ",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 457,
			b: 456,

			expectRes: true,
		},
		{
			name: "greater equal(equal) ",

			condition: action.OptionConditionCommonConditionGreaterEqual,

			a: 456,
			b: 456,

			expectRes: true,
		},
		{
			name: "less",

			condition: action.OptionConditionCommonConditionLess,

			a: 123,
			b: 456,

			expectRes: true,
		},
		{
			name: "less equal(less) ",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: 123,
			b: 456,

			expectRes: true,
		},
		{
			name: "less equal(equal) ",

			condition: action.OptionConditionCommonConditionLessEqual,

			a: 123,
			b: 123,

			expectRes: true,
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
