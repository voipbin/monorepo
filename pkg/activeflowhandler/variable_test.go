package activeflowhandler

import (
	"context"
	"reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
)

func Test_variableSubstitue(t *testing.T) {

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
			h := &activeflowHandler{
				db: mockDB,
			}

			ctx := context.Background()

			res := h.variableSubstitue(ctx, tt.data, tt.variables)
			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}

		})
	}
}

func Test_variableSubstitueAddress(t *testing.T) {

	tests := []struct {
		name string

		address   *cmaddress.Address
		variables map[string]string

		expectRes *cmaddress.Address
	}{
		{
			name: "normal",

			address: &cmaddress.Address{
				Type:       cmaddress.TypeTel,
				Target:     "${test_target}",
				TargetName: "${test_target_name}",
				Name:       "${test_name}",
				Detail:     "${test_detail}",
			},
			variables: map[string]string{
				"test_target":      "+821100000001",
				"test_target_name": "variable target name",
				"test_name":        "variable name",
				"test_detail":      "variable detail",
			},

			expectRes: &cmaddress.Address{
				Type:       cmaddress.TypeTel,
				Target:     "+821100000001",
				TargetName: "variable target name",
				Name:       "variable name",
				Detail:     "variable detail",
			},
		},
		{
			name: "have no variables",

			address: &cmaddress.Address{
				Type:       cmaddress.TypeTel,
				Target:     "+821100000002",
				TargetName: "test target name",
				Name:       "test name",
				Detail:     "test detail",
			},
			variables: map[string]string{
				"test_target":      "+821100000001",
				"test_target_name": "variable target name",
				"test_name":        "variable name",
				"test_detail":      "variable detail",
			},

			expectRes: &cmaddress.Address{
				Type:       cmaddress.TypeTel,
				Target:     "+821100000002",
				TargetName: "test target name",
				Name:       "test name",
				Detail:     "test detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &activeflowHandler{
				db: mockDB,
			}

			ctx := context.Background()

			h.variableSubstitueAddress(ctx, tt.address, tt.variables)
			if reflect.DeepEqual(tt.address, tt.expectRes) != true {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, tt.address)
			}

		})
	}
}
