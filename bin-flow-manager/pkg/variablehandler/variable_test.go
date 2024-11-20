package variablehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		activeflowID uuid.UUID
		variables    map[string]string

		expectRes *variable.Variable
	}{
		{
			"normal",

			uuid.FromStringOrNil("58c48f6a-cce2-11ec-9826-af5811eeea09"),
			map[string]string{
				"key1": "val1",
			},

			&variable.Variable{
				ID: uuid.FromStringOrNil("58c48f6a-cce2-11ec-9826-af5811eeea09"),
				Variables: map[string]string{
					"key1": "val1",
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

			mockDB.EXPECT().VariableCreate(ctx, tt.expectRes).Return(nil)
			mockDB.EXPECT().VariableGet(ctx, tt.activeflowID).Return(tt.expectRes, nil)

			res, err := h.Create(ctx, tt.activeflowID, tt.variables)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_Set(t *testing.T) {

	tests := []struct {
		name string

		variable *variable.Variable

		expectRes *variable.Variable
	}{
		{
			"normal",

			&variable.Variable{
				ID: uuid.FromStringOrNil("2204f07c-cce3-11ec-9ea6-6ba29dbf88ff"),
				Variables: map[string]string{
					"key1": "val1",
				},
			},

			&variable.Variable{
				ID: uuid.FromStringOrNil("2204f07c-cce3-11ec-9ea6-6ba29dbf88ff"),
				Variables: map[string]string{
					"key1": "val1",
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

			mockDB.EXPECT().VariableUpdate(ctx, tt.variable).Return(nil)

			if err := h.Set(ctx, tt.variable); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_SetVariable(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		variables map[string]string

		responseVariable *variable.Variable

		updateVariable *variable.Variable
	}{
		{
			"normal",

			uuid.FromStringOrNil("57bbaa4e-cce3-11ec-a843-63d14c3175c9"),
			map[string]string{
				"key 1": "value 1",
				"key 2": "value 2",
			},

			&variable.Variable{
				ID:        uuid.FromStringOrNil("57bbaa4e-cce3-11ec-a843-63d14c3175c9"),
				Variables: map[string]string{},
			},

			&variable.Variable{
				ID: uuid.FromStringOrNil("57bbaa4e-cce3-11ec-a843-63d14c3175c9"),
				Variables: map[string]string{
					"key 1": "value 1",
					"key 2": "value 2",
				},
			},
		},
		{
			"variable reference a variable",

			uuid.FromStringOrNil("ea7d3a8c-dd55-11ec-81be-6b4cb5e0f3a8"),
			map[string]string{
				"key 1": "${test.variable}",
			},

			&variable.Variable{
				ID: uuid.FromStringOrNil("ea7d3a8c-dd55-11ec-81be-6b4cb5e0f3a8"),
				Variables: map[string]string{
					"test.variable": "variable 2",
				},
			},

			&variable.Variable{
				ID: uuid.FromStringOrNil("ea7d3a8c-dd55-11ec-81be-6b4cb5e0f3a8"),
				Variables: map[string]string{
					"test.variable": "variable 2",
					"key 1":         "variable 2",
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

			mockDB.EXPECT().VariableGet(ctx, tt.id).Return(tt.responseVariable, nil)
			mockDB.EXPECT().VariableUpdate(ctx, tt.updateVariable).Return(nil)

			if err := h.SetVariable(ctx, tt.id, tt.variables); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}

func Test_DeleteVariable(t *testing.T) {

	tests := []struct {
		name string

		id  uuid.UUID
		key string

		responseVariable *variable.Variable

		expectupdateVariable *variable.Variable
	}{
		{
			"normal",

			uuid.FromStringOrNil("588f8bc6-db2e-11ec-a327-5374999e8287"),
			"key 1",

			&variable.Variable{
				ID: uuid.FromStringOrNil("588f8bc6-db2e-11ec-a327-5374999e8287"),
				Variables: map[string]string{
					"key 1": "value 1",
					"key 2": "value 2",
				},
			},

			&variable.Variable{
				ID: uuid.FromStringOrNil("588f8bc6-db2e-11ec-a327-5374999e8287"),
				Variables: map[string]string{
					"key 2": "value 2",
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

			mockDB.EXPECT().VariableGet(ctx, tt.id).Return(tt.responseVariable, nil)
			mockDB.EXPECT().VariableUpdate(ctx, tt.expectupdateVariable).Return(nil)

			if err := h.DeleteVariable(ctx, tt.id, tt.key); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
