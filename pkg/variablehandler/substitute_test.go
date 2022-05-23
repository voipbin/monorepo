package variablehandler

import (
	"context"
	"reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

func Test_Substitue(t *testing.T) {

	tests := []struct {
		name string

		data      string
		variables map[string]string

		expectRes string
	}{
		{
			name: "normal",

			data: "test data ${voipbin.test.name}",
			variables: map[string]string{
				"voipbin.test.name": "test name",
			},

			expectRes: "test data test name",
		},
		{
			name: "data has same variable",

			data: "test data ${voipbin.test.name} and ${voipbin.test.name}",
			variables: map[string]string{
				"voipbin.test.name": "test name",
			},

			expectRes: "test data test name and test name",
		},
		{
			name: "data has same empty variable",

			data: "test data ${voipbin.test.name} and ${voipbin.test.name} and ${voipbin.test.none}",
			variables: map[string]string{
				"voipbin.test.name": "test name",
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

			res := h.Substitue(ctx, tt.data, tt.variables)
			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}
