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
	}{
		{
			"call status dialing",
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("5c08dbec-ce3e-11ec-92a8-03d5ad313332"),
				},
				ActiveFlowID: uuid.FromStringOrNil("5c08dbec-ce3e-11ec-92a8-03d5ad313332"),
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

			variables := map[string]string{

				// source
				"voipbin.call.source.name":        tt.call.Source.Name,
				"voipbin.call.source.detail":      tt.call.Source.Detail,
				"voipbin.call.source.target":      tt.call.Source.Target,
				"voipbin.call.source.target_name": tt.call.Source.Target,
				"voipbin.call.source.type":        string(tt.call.Source.Type),

				// destination
				"voipbin.call.destination.name":        tt.call.Destination.Name,
				"voipbin.call.destination.detail":      tt.call.Destination.Detail,
				"voipbin.call.destination.target":      tt.call.Destination.Target,
				"voipbin.call.destination.target_name": tt.call.Destination.TargetName,
				"voipbin.call.destination.type":        string(tt.call.Destination.Type),

				// others
				"voipbin.call.direction":      string(tt.call.Direction),
				"voipbin.call.master_call_id": tt.call.MasterCallID.String(),
				variableCallDigits:            "",
			}

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.call.ActiveFlowID, variables).Return(nil)

			if err := h.setVariablesCall(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
