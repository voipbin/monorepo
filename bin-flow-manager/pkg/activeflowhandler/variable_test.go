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
			name: "has reference activeflow id",

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
					"key1":                          "value1",
					"key2":                          "value2",
					variableActiveflowCompleteCount: "2",
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
				variableActiveflowCompleteCount:         "3",
			},
		},
		{
			name: "no reference activeflow id",

			activeflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("053a6bb0-cf29-11f0-be55-ebfa876b98e5"),
				},
				ReferenceType: activeflow.ReferenceTypeAI,
				ReferenceID:   uuid.FromStringOrNil("05689a9e-cf29-11f0-a598-437d69259ef2"),
				FlowID:        uuid.FromStringOrNil("0591416a-cf29-11f0-9cb9-c76b4fe51f98"),
			},

			responseVariable: &variable.Variable{
				ID: uuid.FromStringOrNil("8bb1e4f4-cce0-11ec-a4d3-5f6e7c8b9d0e"),
			},
			expectedVariables: map[string]string{
				variableActiveflowID:                    "053a6bb0-cf29-11f0-be55-ebfa876b98e5",
				variableActiveflowReferenceType:         "ai",
				variableActiveflowReferenceID:           "05689a9e-cf29-11f0-a598-437d69259ef2",
				variableActiveflowReferenceActiveflowID: "00000000-0000-0000-0000-000000000000",
				variableActiveflowFlowID:                "0591416a-cf29-11f0-9cb9-c76b4fe51f98",
				variableActiveflowCompleteCount:         "0",
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
				mockVar.EXPECT().Get(ctx, tt.activeflow.ReferenceActiveflowID.Return(tt.responseReferenceActiveflowVariable, nil)
			}

			mockVar.EXPECT().Create(ctx, tt.activeflow.ID, tt.expectedVariables.Return(tt.responseVariable, nil)

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

func Test_variableCreate_error(t *testing.T) {

	tests := []struct {
		name string

		activeflow *activeflow.Activeflow

		responseReferenceActiveflowVariable *variable.Variable
	}{
		{
			name: "exceed max complete count",

			activeflow: &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("27850090-cf74-11f0-a41e-736faa98174a"),
				},
				ReferenceActiveflowID: uuid.FromStringOrNil("27e57524-cf74-11f0-8119-3f63b551357d"),
			},

			responseReferenceActiveflowVariable: &variable.Variable{
				Variables: map[string]string{
					variableActiveflowCompleteCount: "4",
				},
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

			mockVar.EXPECT().Get(ctx, tt.activeflow.ReferenceActiveflowID.Return(tt.responseReferenceActiveflowVariable, nil)

			_, err := h.variableCreate(ctx, tt.activeflow)
			if err == nil {
				t.Errorf("Wrong match.\nexpect: error\ngot: ok\n")
			}
		})
	}
}

func Test_variableSetFromReferenceActiveflow(t *testing.T) {

	tests := []struct {
		name string

		variables             map[string]string
		referenceActiveflowID uuid.UUID

		responseVariable *variable.Variable

		expectedVariables map[string]string
	}{
		{
			name: "normal",

			variables: map[string]string{
				"key1": "value1",
			},
			referenceActiveflowID: uuid.FromStringOrNil("cc988c54-cf48-11f0-a59a-d734702c4854"),

			responseVariable: &variable.Variable{
				ID: uuid.FromStringOrNil("cc988c54-cf48-11f0-a59a-d734702c4854"),
				Variables: map[string]string{
					"key2":                          "value2",
					variableActiveflowCompleteCount: "2",
				},
			},

			expectedVariables: map[string]string{
				"key1":                          "value1",
				"key2":                          "value2",
				variableActiveflowCompleteCount: "2",
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

			mockVar.EXPECT().Get(ctx, tt.referenceActiveflowID.Return(tt.responseVariable, nil)

			err := h.variableSetFromReferenceActiveflow(ctx, tt.variables, tt.referenceActiveflowID)
			if err != nil {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", nil, err)
			}

			if reflect.DeepEqual(tt.variables, tt.expectedVariables) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedVariables, tt.variables)
			}
		})
	}
}

func Test_variableParseCompleteCount(t *testing.T) {

	tests := []struct {
		name string

		variables map[string]string

		expectedError bool
		expectedRes   int
	}{
		{
			name: "normal",

			variables: map[string]string{
				"key1":                          "value1",
				variableActiveflowCompleteCount: "2",
			},

			expectedRes: 2,
		},
		{
			name: "value is float",

			variables: map[string]string{
				"key1":                          "value1",
				variableActiveflowCompleteCount: "2.1",
			},

			expectedRes: 2,
		},
		{
			name: "error - value is string char",

			variables: map[string]string{
				"key1":                          "value1",
				variableActiveflowCompleteCount: "abc",
			},

			expectedError: true,
			expectedRes:   0,
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

			res, err := h.variableParseCompleteCount(tt.variables)
			if (err != nil) != tt.expectedError {
				t.Errorf("Wrong match.\nexpect error: %v\ngot error: %v\n", tt.expectedError, err != nil)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}
