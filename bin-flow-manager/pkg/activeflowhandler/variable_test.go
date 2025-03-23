package activeflowhandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

func Test_variableCreate(t *testing.T) {

	tests := []struct {
		name string

		activeflow *activeflow.Activeflow

		responseReferenceActiveflowVariable *variable.Variable
		responseVariable                    *variable.Variable

		expectedVariables map[string]string
	}{
		{
			name: "normal",

			activeflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7953a60e-07f3-11f0-98bc-93c8a022b396"),
				},
				ReferenceType:         activeflow.ReferenceTypeCall,
				ReferenceID:           uuid.FromStringOrNil("79cf3b5c-07f3-11f0-b1f5-ab8e50a8c6f5"),
				ReferenceActiveflowID: uuid.FromStringOrNil("7a0a2938-07f3-11f0-8103-0b6f3ab09e0c"),
				FlowID:                uuid.FromStringOrNil("7a78656a-07f3-11f0-a8b5-5faa92a202a8"),
			},

			responseReferenceActiveflowVariable: &variable.Variable{
				Variables: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
			},
			responseVariable: &variable.Variable{
				ID: uuid.FromStringOrNil("7a419b16-07f3-11f0-ac8f-93809a7c98ce"),
			},

			expectedVariables: map[string]string{
				"key1":                                  "value1",
				"key2":                                  "value2",
				variableActiveflowID:                    "7953a60e-07f3-11f0-98bc-93c8a022b396",
				variableActiveflowReferenceType:         "call",
				variableActiveflowReferenceID:           "79cf3b5c-07f3-11f0-b1f5-ab8e50a8c6f5",
				variableActiveflowReferenceActiveflowID: "7a0a2938-07f3-11f0-8103-0b6f3ab09e0c",
				variableActiveflowFlowID:                "7a78656a-07f3-11f0-a8b5-5faa92a202a8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &activeflowHandler{
				db:              mockDB,
				reqHandler:      mockReq,
				variableHandler: mockVar,
			}
			ctx := context.Background()

			if tt.activeflow.ReferenceActiveflowID != uuid.Nil {
				mockVar.EXPECT().Get(ctx, tt.activeflow.ReferenceActiveflowID).Return(tt.responseReferenceActiveflowVariable, nil)
			}

			mockVar.EXPECT().Create(ctx, tt.activeflow.ID, tt.expectedVariables).Return(tt.responseVariable, nil)

			res, err := h.variableCreate(ctx, tt.activeflow)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", nil, err)
			}

			if reflect.DeepEqual(res, tt.responseVariable) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseVariable, res)
			}

		})
	}
}
