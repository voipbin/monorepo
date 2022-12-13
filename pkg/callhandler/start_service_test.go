package callhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/ari"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	callapplication "gitlab.com/voipbin/bin-manager/call-manager.git/models/callapplication"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_startServiceFromAMD(t *testing.T) {
	tests := []struct {
		name    string
		channel *channel.Channel

		responseAMD *callapplication.AMD

		data map[string]string
	}{
		{
			"amd result HUMAN",
			&channel.Channel{
				ID:         "47c4df8c-9ace-11ea-82a2-b7e1b384317c",
				AsteriskID: "80:fa:5b:5e:da:81",
			},

			&callapplication.AMD{
				CallID:        uuid.FromStringOrNil("962d6694-ab2f-11ec-a07d-634bebfd48d2"),
				MachineHandle: callapplication.AMDMachineHandleHangup,
				Async:         false,
			},
			map[string]string{
				"amd_status": "HUMAN",
			},
		},
		{
			"amd result Machine and continue",
			&channel.Channel{
				ID:         "47c4df8c-9ace-11ea-82a2-b7e1b384317c",
				AsteriskID: "80:fa:5b:5e:da:81",
			},

			&callapplication.AMD{
				CallID:        uuid.FromStringOrNil("962d6694-ab2f-11ec-a07d-634bebfd48d2"),
				MachineHandle: callapplication.AMDMachineHandleContinue,
				Async:         false,
			},
			map[string]string{
				"amd_status": "MACHINE",
			},
		},
		{
			"amd result Machine and hangup",
			&channel.Channel{
				ID:         "d2e4086c-ab30-11ec-9154-0ffe74bbec50",
				AsteriskID: "80:fa:5b:5e:da:81",
			},

			&callapplication.AMD{
				CallID:        uuid.FromStringOrNil("d3bd1c6a-ab30-11ec-8b93-879bf5e0ba45"),
				MachineHandle: callapplication.AMDMachineHandleHangup,
				Async:         false,
			},
			map[string]string{
				"amd_status": "MACHINE",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &callHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockUtil.EXPECT().GetCurTime().Return(utilhandler.GetCurTime()).AnyTimes()

			mockDB.EXPECT().CallApplicationAMDGet(ctx, tt.channel.ID).Return(tt.responseAMD, nil)

			if tt.responseAMD.MachineHandle == callapplication.AMDMachineHandleHangup && tt.data["amd_status"] == amdStatusMachine {
				mockDB.EXPECT().CallGet(ctx, tt.responseAMD.CallID).Return(&call.Call{}, nil)
				mockDB.EXPECT().CallSetStatus(ctx, gomock.Any(), call.StatusTerminating).Return(nil)
				mockDB.EXPECT().CallGet(ctx, gomock.Any()).Return(&call.Call{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())
				mockReq.EXPECT().AstChannelHangup(ctx, gomock.Any(), gomock.Any(), ari.ChannelCauseCallAMD, 0).Return(nil)
			} else {
				if !tt.responseAMD.Async {
					mockReq.EXPECT().CallV1CallActionNext(ctx, tt.responseAMD.CallID, false)
				}
			}
			mockReq.EXPECT().AstChannelHangup(ctx, tt.channel.AsteriskID, tt.channel.ID, ari.ChannelCauseNormalClearing, 0).Return(nil)

			if err := h.startServiceFromAMD(ctx, tt.channel, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
