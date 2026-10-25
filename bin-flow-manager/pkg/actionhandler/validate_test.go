package actionhandler

import (
	"errors"
	"strings"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"

	"monorepo/bin-flow-manager/models/action"
)

func Test_ValidateActions(t *testing.T) {
	tests := []struct {
		name    string
		actions []action.Action

		expectedReason string // empty means no error
		expectedInMsg  string
	}{
		{
			name:    "empty list is valid",
			actions: nil,
		},
		{
			name: "supported types without options",
			actions: []action.Action{
				{Type: action.TypeAnswer},
				{Type: action.TypeHangup},
			},
		},
		{
			name: "unsupported type is rejected",
			actions: []action.Action{
				{Type: action.TypeAnswer},
				{Type: action.Type("no_such_type")},
			},
			expectedReason: "INVALID_ACTION_TYPE",
			expectedInMsg:  "no_such_type",
		},
		{
			name:    "empty type is rejected",
			actions: []action.Action{{}},

			expectedReason: "INVALID_ACTION_TYPE",
		},
		{
			name:    "call with a supported anonymous value",
			actions: []action.Action{{Type: action.TypeCall, Option: map[string]any{"anonymous": "yes"}}},
		},
		{
			name:    "call with auto anonymous value",
			actions: []action.Action{{Type: action.TypeCall, Option: map[string]any{"anonymous": "auto"}}},
		},
		{
			name:    "call without anonymous value",
			actions: []action.Action{{Type: action.TypeCall, Option: map[string]any{}}},
		},
		{
			name:    "call with an invalid anonymous value",
			actions: []action.Action{{Type: action.TypeCall, Option: map[string]any{"anonymous": "maybe"}}},

			expectedReason: "INVALID_ANONYMOUS_VALUE",
			expectedInMsg:  "call action: maybe",
		},
		{
			name:    "connect with a supported anonymous value",
			actions: []action.Action{{Type: action.TypeConnect, Option: map[string]any{"anonymous": "no"}}},
		},
		{
			name:    "connect with an invalid anonymous value",
			actions: []action.Action{{Type: action.TypeConnect, Option: map[string]any{"anonymous": "YES"}}},

			expectedReason: "INVALID_ANONYMOUS_VALUE",
			expectedInMsg:  "connect action: YES",
		},
		{
			name:    "invalid anonymous value in a later action is found",
			actions: []action.Action{{Type: action.TypeAnswer}, {Type: action.TypeConnect, Option: map[string]any{"anonymous": "x"}}},

			expectedReason: "INVALID_ANONYMOUS_VALUE",
		},
		{
			name:    "call option that cannot be parsed is skipped",
			actions: []action.Action{{Type: action.TypeCall, Option: map[string]any{"anonymous": 5}}},
		},
		{
			name: "connect option that cannot be parsed is skipped and the next action is still checked",
			actions: []action.Action{
				{Type: action.TypeConnect, Option: map[string]any{"anonymous": 5}},
				{Type: action.Type("no_such_type")},
			},

			expectedReason: "INVALID_ACTION_TYPE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &actionHandler{}

			err := h.ValidateActions(tt.actions)
			if tt.expectedReason == "" {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}

			var ve *cerrors.VoipbinError
			if !errors.As(err, &ve) {
				t.Fatalf("Wrong error type. expect: VoipbinError, got: %v", err)
			}
			if ve.Status != cerrors.StatusInvalidArgument {
				t.Errorf("Wrong status. expect: %v, got: %v", cerrors.StatusInvalidArgument, ve.Status)
			}
			if ve.Reason != tt.expectedReason {
				t.Errorf("Wrong reason. expect: %s, got: %s", tt.expectedReason, ve.Reason)
			}
			if !strings.Contains(ve.Message, tt.expectedInMsg) {
				t.Errorf("Wrong message. expect to contain: %q, got: %q", tt.expectedInMsg, ve.Message)
			}
		})
	}
}

func Test_NewActionHandler(t *testing.T) {
	res := NewActionHandler()
	h, ok := res.(*actionHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.utilHandler == nil {
		t.Errorf("utilHandler is not set")
	}
}
