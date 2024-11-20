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

func Test_SubstituteString(t *testing.T) {

	tests := []struct {
		name string

		data string
		v    *variable.Variable

		expectRes string
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

			expectRes: "test data test name",
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

			expectRes: "test data test name and test name",
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

			expectRes: "test data test name and test name and ",
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
			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_SubstituteByte(t *testing.T) {

	tests := []struct {
		name string

		data []byte
		v    *variable.Variable

		expectRes []byte
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

			expectRes: []byte(`{"conversation_id":"7e5116e2-f477-11ec-9c08-b343a05abaee","text":"test message. test name.","sync":true}`),
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

			expectRes: []byte(`test data test name and test name`),
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

			expectRes: []byte(`test data test name and test name and `),
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
			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}
