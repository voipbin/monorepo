package dbhandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/cachehandler"
)

func Test_VariableCreate(t *testing.T) {

	tests := []struct {
		name     string
		variable *variable.Variable

		expectRes *variable.Variable
	}{
		{
			"have no actions",
			&variable.Variable{
				ID: uuid.FromStringOrNil("8ae9e942-cce0-11ec-9471-af401e37420e"),
			},
			&variable.Variable{
				ID: uuid.FromStringOrNil("8ae9e942-cce0-11ec-9471-af401e37420e"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := handler{
				db:    dbTest,
				cache: mockCache,
			}

			ctx := context.Background()

			mockCache.EXPECT().VariableSet(ctx, tt.variable)
			if err := h.VariableCreate(ctx, tt.variable); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().VariableGet(ctx, tt.variable.ID).Return(tt.expectRes, nil)
			res, err := h.VariableGet(ctx, tt.variable.ID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			t.Logf("Created flow. flow: %v", res)

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_VariableUpdate(t *testing.T) {

	tests := []struct {
		name     string
		variable *variable.Variable

		updateVariable *variable.Variable
	}{
		{
			"test normal",
			&variable.Variable{
				ID: uuid.FromStringOrNil("585b7a74-18a0-48ac-b4c5-1ba5ddea87ae"),
			},

			&variable.Variable{
				ID: uuid.FromStringOrNil("585b7a74-18a0-48ac-b4c5-1ba5ddea87ae"),
				Variables: map[string]string{
					"test variable": "test value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCache := cachehandler.NewMockCacheHandler(mc)
			h := NewHandler(dbTest, mockCache)

			ctx := context.Background()

			mockCache.EXPECT().VariableSet(ctx, tt.variable)
			if err := h.VariableCreate(context.Background(), tt.variable); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			mockCache.EXPECT().VariableSet(ctx, tt.updateVariable).Return(nil)
			if err := h.VariableUpdate(context.Background(), tt.updateVariable); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

		})
	}
}
