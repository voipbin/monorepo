package variablehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
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

			mockDB.EXPECT().VariableUpdate(ctx, tt.variable).Return(tt.expectRes, nil)

			res, err := h.Set(ctx, tt.variable)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_SetVariable(t *testing.T) {

	tests := []struct {
		name string

		id    uuid.UUID
		key   string
		value string

		responseVariable *variable.Variable

		expectRes *variable.Variable
	}{
		{
			"normal",

			uuid.FromStringOrNil("57bbaa4e-cce3-11ec-a843-63d14c3175c9"),
			"key 1",
			"value 1",

			&variable.Variable{
				ID:        uuid.FromStringOrNil("57bbaa4e-cce3-11ec-a843-63d14c3175c9"),
				Variables: map[string]string{},
			},

			&variable.Variable{
				ID: uuid.FromStringOrNil("57bbaa4e-cce3-11ec-a843-63d14c3175c9"),
				Variables: map[string]string{
					"key 1": "value 1",
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
			mockDB.EXPECT().VariableUpdate(ctx, tt.expectRes).Return(tt.expectRes, nil)

			res, err := h.SetVariable(ctx, tt.id, tt.key, tt.value)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
