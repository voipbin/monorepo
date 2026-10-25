package variablehandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	cmcall "monorepo/bin-call-manager/models/call"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

func testVars() *variable.Variable {
	return &variable.Variable{
		ID: uuid.FromStringOrNil("5072a680-dd54-11ec-aeff-7b54e7355667"),
		Variables: map[string]string{
			"voipbin.test.id":   "7e5116e2",
			"voipbin.test.name": "test name",
		},
	}
}

func Test_Substitute_getError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &variableHandler{db: mockDB}
	id := uuid.FromStringOrNil("5072a680-dd54-11ec-aeff-7b54e7355667")

	mockDB.EXPECT().VariableGet(gomock.Any(), id).Return(nil, errSynthetic)

	res, err := h.Substitute(context.Background(), id, "hello ${voipbin.test.name}")
	if res != "" {
		t.Errorf("Wrong result. expect: empty, got: %s", res)
	}
	if !errors.Is(err, errSynthetic) {
		t.Errorf("Wrong error. expect: %v, got: %v", errSynthetic, err)
	}
}

func Test_substituteString_unresolved(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		expectedRes string
	}{
		{"unknown name becomes empty", "a=${no.such.name};b=${voipbin.test.id}", "a=;b=7e5116e2"},
		{"no placeholder is kept", "plain text", "plain text"},
		{"unterminated placeholder is kept", "${voipbin.test.id", "${voipbin.test.id"},
		{"empty name becomes empty", "x${}y", "xy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &variableHandler{}
			res := h.substituteString(context.Background(), tt.data, testVars())
			if res != tt.expectedRes {
				t.Errorf("Wrong match. expect: %q, got: %q", tt.expectedRes, res)
			}
		})
	}
}

func Test_SubstituteOption_types(t *testing.T) {
	str := "p ${voipbin.test.name}"
	nested := map[string]any{"deep": "${voipbin.test.id}"}

	data := map[string]any{
		"nested map":  map[string]any{"inner": "${voipbin.test.id}", "inner list": []any{"${voipbin.test.name}"}},
		"any list":    []any{"${voipbin.test.id}", []byte("${voipbin.test.name}"), 7, true, nil, map[string]any{"k": "${voipbin.test.id}"}, []string{"${voipbin.test.name}"}},
		"string ptr":  &str,
		"map ptr":     &nested,
		"unsupported": 42,
		"plain":       "${voipbin.test.id}",
	}

	h := &variableHandler{}
	h.SubstituteOption(context.Background(), data, testVars())

	expected := map[string]any{
		"nested map":  map[string]any{"inner": "7e5116e2", "inner list": []any{"test name"}},
		"any list":    []any{"7e5116e2", []byte("test name"), "7", "true", "<nil>", map[string]any{"k": "7e5116e2"}, []string{"test name"}},
		"string ptr":  "p test name",
		"map ptr":     map[string]any{"deep": "7e5116e2"},
		"unsupported": 42,
		"plain":       "7e5116e2",
	}
	if !reflect.DeepEqual(data, expected) {
		t.Errorf("Wrong match.\nexpect: %#v\ngot: %#v", expected, data)
	}

	// the map behind the pointer is substituted in place as well
	if nested["deep"] != "7e5116e2" {
		t.Errorf("Wrong in-place result. expect: 7e5116e2, got: %v", nested["deep"])
	}
}

func Test_resolveValue(t *testing.T) {
	str := "${voipbin.test.name}"
	mp := map[string]any{"k": "${voipbin.test.id}"}

	tests := []struct {
		name        string
		value       any
		expectedRes any
	}{
		{"string", "id=${voipbin.test.id}", "id=7e5116e2"},
		{"bytes", []byte("id=${voipbin.test.id}"), []byte("id=7e5116e2")},
		{"map", map[string]any{"k": "${voipbin.test.id}"}, map[string]any{"k": "7e5116e2"}},
		{"any slice", []any{"${voipbin.test.id}", 3}, []any{"7e5116e2", "3"}},
		{"string slice", []string{"${voipbin.test.id}", "x"}, []string{"7e5116e2", "x"}},
		{"string pointer", &str, "test name"},
		{"map pointer", &mp, map[string]any{"k": "7e5116e2"}},
		{"int falls back to its text", 12, "12"},
		{"bool falls back to its text", false, "false"},
		{"nil falls back to its text", nil, "<nil>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &variableHandler{}
			res := h.resolveValue(context.Background(), tt.value, testVars())
			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match. expect: %#v, got: %#v", tt.expectedRes, res)
			}
		})
	}
}

func Test_substituteParseDynamic_unresolved(t *testing.T) {
	afID := uuid.FromStringOrNil("df85d5c6-c05f-11f0-9ad8-4355d58efb8a")
	refID := uuid.FromStringOrNil("dfb89132-c05f-11f0-a2ad-8beb167b2ddd")
	v := &variable.Variable{ID: afID}

	tests := []struct {
		name         string
		variableName string
		setup        func(mockReq *requesthandler.MockRequestHandler)
	}{
		{
			name:         "other name is not dynamic and nothing is requested",
			variableName: "voipbin.flow.other",
			setup:        func(mockReq *requesthandler.MockRequestHandler) {},
		},
		{
			name:         "activeflow lookup fails",
			variableName: constVariableReferenceData,
			setup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(nil, errSynthetic)
			},
		},
		{
			name:         "call lookup fails",
			variableName: constVariableReferenceData,
			setup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(&activeflow.Activeflow{ReferenceType: activeflow.ReferenceTypeCall, ReferenceID: refID}, nil)
				mockReq.EXPECT().CallV1CallGet(gomock.Any(), refID).Return((*cmcall.Call)(nil), errSynthetic)
			},
		},
		{
			name:         "conversation lookup fails",
			variableName: constVariableReferenceData,
			setup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(&activeflow.Activeflow{ReferenceType: activeflow.ReferenceTypeConversation, ReferenceID: refID}, nil)
				mockReq.EXPECT().ConversationV1ConversationGet(gomock.Any(), refID).Return(nil, errSynthetic)
			},
		},
		{
			name:         "reference type without data",
			variableName: constVariableReferenceData,
			setup: func(mockReq *requesthandler.MockRequestHandler) {
				mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), afID).Return(&activeflow.Activeflow{ReferenceType: activeflow.ReferenceTypeAPI, ReferenceID: refID}, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &variableHandler{requestHandler: mockReq}
			tt.setup(mockReq)

			res, found := h.substituteParseDynamic(context.Background(), tt.variableName, v)
			if found || res != "" {
				t.Errorf("Wrong match. expect: (empty, false), got: (%q, %v)", res, found)
			}
		})
	}
}

func Test_substituteString_dynamicFailureBecomesEmpty(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &variableHandler{requestHandler: mockReq}
	v := &variable.Variable{ID: uuid.FromStringOrNil("df85d5c6-c05f-11f0-9ad8-4355d58efb8a")}

	mockReq.EXPECT().FlowV1ActiveflowGet(gomock.Any(), v.ID).Return(nil, errSynthetic)

	res := h.substituteString(context.Background(), "ref=[${voipbin.flow.reference_data}]", v)
	if res != "ref=[]" {
		t.Errorf("Wrong match. expect: ref=[], got: %s", res)
	}
}
