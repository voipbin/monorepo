package agenthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentcall"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/models/agentdial"
	"gitlab.com/voipbin/bin-manager/agent-manager.git/pkg/dbhandler"
)

func TestAgentCallAnswered(t *testing.T) {

	tests := []struct {
		name string

		id        uuid.UUID
		call      *cmcall.Call
		agentCall *agentcall.AgentCall
		agentDial *agentdial.AgentDial
	}{
		{
			"normal",

			uuid.FromStringOrNil("9437722e-53e3-11ec-bbba-ab4a8d821d52"),
			&cmcall.Call{
				ID: uuid.FromStringOrNil("cc8536e4-53e2-11ec-b251-4b5616ce92b1"),
			},
			&agentcall.AgentCall{
				ID:      uuid.FromStringOrNil("cc8536e4-53e2-11ec-b251-4b5616ce92b1"),
				AgentID: uuid.FromStringOrNil("9437722e-53e3-11ec-bbba-ab4a8d821d52"),
			},
			&agentdial.AgentDial{
				AgentID: uuid.FromStringOrNil("9437722e-53e3-11ec-bbba-ab4a8d821d52"),
				AgentCallIDs: []uuid.UUID{
					uuid.FromStringOrNil("cc8536e4-53e2-11ec-b251-4b5616ce92b1"),
				},
			},
		},
		{
			"2 calls",

			uuid.FromStringOrNil("9437722e-53e3-11ec-bbba-ab4a8d821d52"),
			&cmcall.Call{
				ID: uuid.FromStringOrNil("cc8536e4-53e2-11ec-b251-4b5616ce92b1"),
			},
			&agentcall.AgentCall{
				ID:      uuid.FromStringOrNil("cc8536e4-53e2-11ec-b251-4b5616ce92b1"),
				AgentID: uuid.FromStringOrNil("9437722e-53e3-11ec-bbba-ab4a8d821d52"),
			},
			&agentdial.AgentDial{
				AgentID: uuid.FromStringOrNil("9437722e-53e3-11ec-bbba-ab4a8d821d52"),
				AgentCallIDs: []uuid.UUID{
					uuid.FromStringOrNil("cc8536e4-53e2-11ec-b251-4b5616ce92b1"),
					uuid.FromStringOrNil("5b185bd8-53e4-11ec-af8e-abccd8406aa7"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &agentHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().AgentCallGet(gomock.Any(), tt.call.ID).Return(tt.agentCall, nil)
			mockDB.EXPECT().AgentSetStatus(gomock.Any(), tt.agentCall.AgentID, agent.StatusBusy).Return(nil)
			mockDB.EXPECT().AgentDialGet(gomock.Any(), tt.agentCall.AgentDialID).Return(tt.agentDial, nil)
			for _, callID := range tt.agentDial.AgentCallIDs {
				if callID == tt.agentCall.ID {
					continue
				}
				mockReq.EXPECT().CMV1CallHangup(gomock.Any(), callID).Return(nil, nil)
			}

			if err := h.AgentCallAnswered(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
