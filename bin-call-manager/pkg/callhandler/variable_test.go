package callhandler

import (
	"context"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_setVariables(t *testing.T) {

	tests := []struct {
		name string
		call *call.Call

		expectedVariables map[string]string
	}{
		{
			name: "call status dialing",
			call: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5c08dbec-ce3e-11ec-92a8-03d5ad313332"),
				},
				ActiveflowID: uuid.FromStringOrNil("5c08dbec-ce3e-11ec-92a8-03d5ad313332"),
				Source: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000001",
					TargetName: "test source target name",
					Name:       "test source name",
					Detail:     "test source detail",
				},
				Destination: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821100000002",
					TargetName: "test destination target name",
					Name:       "test destination name",
					Detail:     "test destination detail",
				},
				Direction:    call.DirectionOutgoing,
				MasterCallID: uuid.FromStringOrNil("8a86b740-055a-11f0-8900-6bfe8037ecb4"),
			},

			expectedVariables: map[string]string{
				variableCallID: "5c08dbec-ce3e-11ec-92a8-03d5ad313332",

				variableCallSourceName:       "test source name",
				variableCallSourceDetail:     "test source detail",
				variableCallSourceTarget:     "+821100000001",
				variableCallSourceTargetName: "test source target name",
				variableCallSourceType:       "tel",

				variableCallDestinationName:       "test destination name",
				variableCallDestinationDetail:     "test destination detail",
				variableCallDestinationTarget:     "+821100000002",
				variableCallDestinationTargetName: "test destination target name",
				variableCallDestinationType:       "tel",

				// others
				variableCallDirection:    "outgoing",
				variableCallMasterCallID: "8a86b740-055a-11f0-8900-6bfe8037ecb4",
				variableCallDigits:       "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &callHandler{
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.call.ActiveflowID, tt.expectedVariables).Return(nil)

			if err := h.setVariablesCall(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
