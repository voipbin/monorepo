package activeflowhandler

import (
	"context"
	"reflect"
	"testing"

	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/variable"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/pkg/variablehandler"
)

func Test_variableSubstitueAddress(t *testing.T) {

	tests := []struct {
		name string

		address *commonaddress.Address
		v       *variable.Variable

		responseSubName       string
		responseSubDetail     string
		responseSubTarget     string
		responseSubTargetName string

		expectRes *commonaddress.Address
	}{
		{
			name: "normal",

			address: &commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "${test_target}",
				TargetName: "${test_target_name}",
				Name:       "${test_name}",
				Detail:     "${test_detail}",
			},
			v: &variable.Variable{},

			responseSubTarget:     "+821100000001",
			responseSubTargetName: "variable target name",
			responseSubName:       "variable name",
			responseSubDetail:     "variable detail",

			expectRes: &commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+821100000001",
				TargetName: "variable target name",
				Name:       "variable name",
				Detail:     "variable detail",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockVar := variablehandler.NewMockVariableHandler(mc)
			h := &activeflowHandler{
				db:              mockDB,
				variableHandler: mockVar,
			}

			ctx := context.Background()

			mockVar.EXPECT().SubstituteString(ctx, tt.address.Name, tt.v).Return(tt.responseSubName)
			mockVar.EXPECT().SubstituteString(ctx, tt.address.Detail, tt.v).Return(tt.responseSubDetail)
			mockVar.EXPECT().SubstituteString(ctx, tt.address.Target, tt.v).Return(tt.responseSubTarget)
			mockVar.EXPECT().SubstituteString(ctx, tt.address.TargetName, tt.v).Return(tt.responseSubTargetName)

			h.variableSubstitueAddress(ctx, tt.address, tt.v)
			if reflect.DeepEqual(tt.address, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, tt.address)
			}

		})
	}
}
