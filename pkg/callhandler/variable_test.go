package callhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_setVariables(t *testing.T) {

	tests := []struct {
		name string
		call *call.Call
	}{
		{
			"call status dialing",
			&call.Call{
				ID:           uuid.FromStringOrNil("5c08dbec-ce3e-11ec-92a8-03d5ad313332"),
				ActiveFlowID: uuid.FromStringOrNil("5c08dbec-ce3e-11ec-92a8-03d5ad313332"),
				Source: address.Address{
					Type:       address.TypeTel,
					Target:     "+821100000001",
					TargetName: "test source target name",
					Name:       "test source name",
					Detail:     "test source detail",
				},
				Destination: address.Address{
					Type:       address.TypeTel,
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

			// source
			mockReq.EXPECT().FMV1VariableSetVariable(ctx, tt.call.ActiveFlowID, "voipbin.call.source.name", tt.call.Source.Name).Return(nil)
			mockReq.EXPECT().FMV1VariableSetVariable(ctx, tt.call.ActiveFlowID, "voipbin.call.source.detail", tt.call.Source.Detail).Return(nil)
			mockReq.EXPECT().FMV1VariableSetVariable(ctx, tt.call.ActiveFlowID, "voipbin.call.source.target", tt.call.Source.Target).Return(nil)
			mockReq.EXPECT().FMV1VariableSetVariable(ctx, tt.call.ActiveFlowID, "voipbin.call.source.target_name", tt.call.Source.TargetName).Return(nil)
			mockReq.EXPECT().FMV1VariableSetVariable(ctx, tt.call.ActiveFlowID, "voipbin.call.source.type", string(tt.call.Source.Type)).Return(nil)

			// destination
			mockReq.EXPECT().FMV1VariableSetVariable(ctx, tt.call.ActiveFlowID, "voipbin.call.destination.name", tt.call.Destination.Name).Return(nil)
			mockReq.EXPECT().FMV1VariableSetVariable(ctx, tt.call.ActiveFlowID, "voipbin.call.destination.detail", tt.call.Destination.Detail).Return(nil)
			mockReq.EXPECT().FMV1VariableSetVariable(ctx, tt.call.ActiveFlowID, "voipbin.call.destination.target", tt.call.Destination.Target).Return(nil)
			mockReq.EXPECT().FMV1VariableSetVariable(ctx, tt.call.ActiveFlowID, "voipbin.call.destination.target_name", tt.call.Destination.TargetName).Return(nil)
			mockReq.EXPECT().FMV1VariableSetVariable(ctx, tt.call.ActiveFlowID, "voipbin.call.destination.type", string(tt.call.Destination.Type)).Return(nil)

			if err := h.setVariables(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
