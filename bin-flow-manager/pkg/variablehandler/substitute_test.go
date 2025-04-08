package variablehandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

func Test_Substitute(t *testing.T) {

	tests := []struct {
		name string

		id   uuid.UUID
		data string

		responseVariable *variable.Variable

		expectedRes string
	}{
		{
			name: "normal",

			id:   uuid.FromStringOrNil("202edeaa-f78f-11ef-a7df-639fb04c39b2"),
			data: `{"conversation_id":"${voipbin.test.id}","text":"test message. ${voipbin.test.name}.","sync":true}`,

			responseVariable: &variable.Variable{
				ID: uuid.FromStringOrNil("202edeaa-f78f-11ef-a7df-639fb04c39b2"),
				Variables: map[string]string{
					"voipbin.test.id":   "7e5116e2-f477-11ec-9c08-b343a05abaee",
					"voipbin.test.name": "test name",
				},
			},

			expectedRes: `{"conversation_id":"7e5116e2-f477-11ec-9c08-b343a05abaee","text":"test message. test name.","sync":true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &variableHandler{
				db: mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().VariableGet(ctx, tt.id).Return(tt.responseVariable, nil)

			res, err := h.Substitute(ctx, tt.id, string(tt.data))
			if err != nil {
				t.Errorf("Wrong match. expect:ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedRes, res)
			}

		})
	}
}

func Test_SubstituteString(t *testing.T) {

	tests := []struct {
		name string

		data string
		v    *variable.Variable

		expectedRes string
	}{
		{
			name: "normal",

			data: "test data ${voipbin.test.name}",
			v: &variable.Variable{
				ID: uuid.FromStringOrNil("5072a680-dd54-11ec-aeff-7b54e7355667"),
				Variables: map[string]string{
					"voipbin.test.name": "test name",
				},
			},

			expectedRes: "test data test name",
		},
		{
			name: "data has same variable",

			data: "test data ${voipbin.test.name} and ${voipbin.test.name}",
			v: &variable.Variable{
				ID: uuid.FromStringOrNil("5072a680-dd54-11ec-aeff-7b54e7355667"),
				Variables: map[string]string{
					"voipbin.test.name": "test name",
				},
			},

			expectedRes: "test data test name and test name",
		},
		{
			name: "data has same empty variable",

			data: "test data ${voipbin.test.name} and ${voipbin.test.name} and ${voipbin.test.none}",
			v: &variable.Variable{
				ID: uuid.FromStringOrNil("5072a680-dd54-11ec-aeff-7b54e7355667"),
				Variables: map[string]string{
					"voipbin.test.name": "test name",
				},
			},

			expectedRes: "test data test name and test name and ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &variableHandler{
				db: mockDB,
			}

			ctx := context.Background()

			res := h.SubstituteString(ctx, tt.data, tt.v)
			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedRes, res)
			}

		})
	}
}

func Test_SubstituteByte(t *testing.T) {

	tests := []struct {
		name string

		data []byte
		v    *variable.Variable

		expectedRes []byte
	}{
		{
			name: "normal",

			data: []byte(`{"conversation_id":"${voipbin.test.id}","text":"test message. ${voipbin.test.name}.","sync":true}`),
			v: &variable.Variable{
				ID: uuid.FromStringOrNil("5072a680-dd54-11ec-aeff-7b54e7355667"),
				Variables: map[string]string{
					"voipbin.test.id":   "7e5116e2-f477-11ec-9c08-b343a05abaee",
					"voipbin.test.name": "test name",
				},
			},

			expectedRes: []byte(`{"conversation_id":"7e5116e2-f477-11ec-9c08-b343a05abaee","text":"test message. test name.","sync":true}`),
		},
		{
			name: "data has same variable",

			data: []byte(`test data ${voipbin.test.name} and ${voipbin.test.name}`),
			v: &variable.Variable{
				ID: uuid.FromStringOrNil("5072a680-dd54-11ec-aeff-7b54e7355667"),
				Variables: map[string]string{
					"voipbin.test.name": "test name",
				},
			},

			expectedRes: []byte(`test data test name and test name`),
		},
		{
			name: "data has same empty variable",

			data: []byte(`test data ${voipbin.test.name} and ${voipbin.test.name} and ${voipbin.test.none}`),
			v: &variable.Variable{
				ID: uuid.FromStringOrNil("5072a680-dd54-11ec-aeff-7b54e7355667"),
				Variables: map[string]string{
					"voipbin.test.name": "test name",
				},
			},

			expectedRes: []byte(`test data test name and test name and `),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &variableHandler{
				db: mockDB,
			}

			ctx := context.Background()

			res := h.SubstituteByte(ctx, tt.data, tt.v)
			if reflect.DeepEqual(res, tt.expectedRes) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedRes, res)
			}

		})
	}
}

func Test_SubstituteOption(t *testing.T) {

	tests := []struct {
		name string

		data map[string]any
		v    *variable.Variable

		expectedRes map[string]any
	}{
		{
			name: "normal",

			data: map[string]any{
				"conversation_id": "${voipbin.test.id}",
				"text":            "test message. ${voipbin.test.name}.",
				"bytes":           []byte("test message. ${voipbin.test.name}."),
				"sync":            true,
				"list string": []string{
					"${voipbin.test.id}",
					"${voipbin.test.name}",
				},
				"list map": []map[string]any{
					{
						"key1": "${voipbin.test.id}",
					},
				},
			},
			v: &variable.Variable{
				ID: uuid.FromStringOrNil("5072a680-dd54-11ec-aeff-7b54e7355667"),
				Variables: map[string]string{
					"voipbin.test.id":   "7e5116e2-f477-11ec-9c08-b343a05abaee",
					"voipbin.test.name": "test name",
				},
			},

			expectedRes: map[string]any{
				"conversation_id": "7e5116e2-f477-11ec-9c08-b343a05abaee",
				"text":            "test message. test name.",
				"bytes":           []byte("test message. test name."),
				"sync":            true,
				"list string": []string{
					"7e5116e2-f477-11ec-9c08-b343a05abaee",
					"test name",
				},
				"list map": []map[string]any{
					{
						"key1": "7e5116e2-f477-11ec-9c08-b343a05abaee",
					},
				},
			},
		},
		{
			name: "data has same variable",

			data: map[string]any{
				"test1": "${voipbin.test.name}",
				"test2": "${voipbin.test.name}",
			},

			v: &variable.Variable{
				ID: uuid.FromStringOrNil("5072a680-dd54-11ec-aeff-7b54e7355667"),
				Variables: map[string]string{
					"voipbin.test.name": "test name",
				},
			},

			expectedRes: map[string]any{
				"test1": "test name",
				"test2": "test name",
			},
		},
		{
			name: "data has same empty variable",

			data: map[string]any{
				"test1": "${voipbin.test.name}",
				"test2": "${voipbin.test.name}",
				"test3": "${voipbin.test.none}",
			},
			v: &variable.Variable{
				ID: uuid.FromStringOrNil("5072a680-dd54-11ec-aeff-7b54e7355667"),
				Variables: map[string]string{
					"voipbin.test.name": "test name",
				},
			},

			expectedRes: map[string]any{
				"test1": "test name",
				"test2": "test name",
				"test3": "",
			},
		},
		{
			name: "nested data",

			data: map[string]any{
				"test1": "${voipbin.test.name}",
				"test2": "${voipbin.test.name}",
				"test3": "${voipbin.test.none}",
				"nested": map[string]any{
					"nested1": "${voipbin.test.name}",
					"nested2": map[string]any{
						"nested2-1": "${voipbin.test.name}",
					},
				},
			},
			v: &variable.Variable{
				ID: uuid.FromStringOrNil("5072a680-dd54-11ec-aeff-7b54e7355667"),
				Variables: map[string]string{
					"voipbin.test.name": "test name",
				},
			},

			expectedRes: map[string]any{
				"test1": "test name",
				"test2": "test name",
				"test3": "",
				"nested": map[string]any{
					"nested1": "test name",
					"nested2": map[string]any{
						"nested2-1": "test name",
					},
				},
			},
		},
		{
			name: "nested structs",

			data: map[string]any{
				"nestedMapList": []map[string]any{
					{
						"key1": "${voipbin.test.name}",
					},
				},
				"stringList": []string{
					"${voipbin.test.name}",
					"${voipbin.test.id}",
				},
			},
			v: &variable.Variable{
				ID: uuid.FromStringOrNil("5072a680-dd54-11ec-aeff-7b54e7355667"),
				Variables: map[string]string{
					"voipbin.test.name": "test name",
					"voipbin.test.id":   "7e5116e2-f477-11ec-9c08-b343a05abaee",
				},
			},

			expectedRes: map[string]any{
				"nestedMapList": []map[string]any{
					{
						"key1": "test name",
					},
				},
				"stringList": []string{
					"test name",
					"7e5116e2-f477-11ec-9c08-b343a05abaee",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &variableHandler{
				db: mockDB,
			}

			ctx := context.Background()

			// Run the substitute function
			h.SubstituteOption(ctx, tt.data, tt.v)

			// Compare the results
			if !reflect.DeepEqual(tt.data, tt.expectedRes) {
				t.Errorf("Test %s failed: expected %v, got %v", tt.name, tt.expectedRes, tt.data)
			}
		})
	}
}
