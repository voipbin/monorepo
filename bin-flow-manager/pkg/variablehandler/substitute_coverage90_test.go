package variablehandler

import (
	"context"
	"errors"
	"testing"

	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/variable"
)

func Test_substituteString_dynamicVariable_coverage90(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c9b0c0de-1e2f-4a3b-8c4d-5e6f7a8b9c01")

	tests := []struct {
		name string

		data string
		v    *variable.Variable

		responseErr error
		expectRes   string
	}{
		{
			// a static variable wins over the dynamic one of the same name, so no request is sent.
			name:      "a static variable is used before the dynamic one",
			data:      "value: ${voipbin.flow.reference_data}",
			v:         &variable.Variable{ID: id, Variables: map[string]string{"voipbin.flow.reference_data": "static"}},
			expectRes: "value: static",
		},
		{
			name:        "a dynamic variable that can not be resolved becomes empty",
			data:        "value: ${voipbin.flow.reference_data}",
			v:           &variable.Variable{ID: id, Variables: map[string]string{}},
			responseErr: errors.New("synthetic dependency failure"),
			expectRes:   "value: ",
		},
		{
			name:      "an unknown variable becomes empty",
			data:      "value: ${unknown}",
			v:         &variable.Variable{ID: id, Variables: map[string]string{}},
			expectRes: "value: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &variableHandler{requestHandler: mockReq}

			if tt.responseErr != nil {
				mockReq.EXPECT().FlowV1ActiveflowGet(ctx, id).Return(nil, tt.responseErr)
			}

			res := h.substituteString(ctx, tt.data, tt.v)
			if res != tt.expectRes {
				t.Errorf("Wrong result.\nexpect: %q\ngot: %q", tt.expectRes, res)
			}
		})
	}
}

func Test_substituteParseDynamic_unsupportedReference_coverage90(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c9b0c0de-1e2f-4a3b-8c4d-5e6f7a8b9c02")

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &variableHandler{requestHandler: mockReq}

	// the reference type has no dynamic data, so nothing else is requested.
	mockReq.EXPECT().FlowV1ActiveflowGet(ctx, id).Return(&activeflow.Activeflow{ReferenceType: activeflow.ReferenceTypeAPI}, nil)

	res, ok := h.substituteParseDynamic(ctx, constVariableReferenceData, &variable.Variable{ID: id})
	if ok || res != "" {
		t.Errorf("Nothing must be resolved. res: %q, ok: %v", res, ok)
	}
}
