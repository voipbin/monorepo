package callhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/call"
	callapplication "monorepo/bin-call-manager/models/callapplication"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_startServiceFromAMD(t *testing.T) {
	tests := []struct {
		name      string
		channelID string

		responseAMD *callapplication.AMD

		data map[channel.StasisDataType]string
	}{
		{
			"amd result HUMAN",
			"47c4df8c-9ace-11ea-82a2-b7e1b384317c",

			&callapplication.AMD{
				CallID:        uuid.FromStringOrNil("962d6694-ab2f-11ec-a07d-634bebfd48d2"),
				MachineHandle: callapplication.AMDMachineHandleHangup,
				Async:         false,
			},
			map[channel.StasisDataType]string{
				"amd_status": "HUMAN",
			},
		},
		{
			"amd result Machine and continue",
			"47c4df8c-9ace-11ea-82a2-b7e1b384317c",

			&callapplication.AMD{
				CallID:        uuid.FromStringOrNil("962d6694-ab2f-11ec-a07d-634bebfd48d2"),
				MachineHandle: callapplication.AMDMachineHandleContinue,
				Async:         false,
			},
			map[channel.StasisDataType]string{
				"amd_status": "MACHINE",
			},
		},
		{
			"amd result Machine and hangup",
			"d2e4086c-ab30-11ec-9154-0ffe74bbec50",

			&callapplication.AMD{
				CallID:        uuid.FromStringOrNil("d3bd1c6a-ab30-11ec-8b93-879bf5e0ba45"),
				MachineHandle: callapplication.AMDMachineHandleHangup,
				Async:         false,
			},
			map[channel.StasisDataType]string{
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime(.Return(utilhandler.TimeGetCurTime()).AnyTimes()

			mockDB.EXPECT().CallApplicationAMDGet(ctx, tt.channelID.Return(tt.responseAMD, nil)

			if tt.responseAMD.MachineHandle == callapplication.AMDMachineHandleHangup && tt.data[channel.StasisDataTypeServiceAMDStatus] == amdStatusMachine {
				mockDB.EXPECT().CallGet(ctx, tt.responseAMD.CallID.Return(&call.Call{}, nil)
				mockDB.EXPECT().CallSetStatus(ctx, gomock.Any(), call.StatusTerminating.Return(nil)

				tmpCall := &call.Call{
					Status: call.StatusProgressing,
				}
				mockDB.EXPECT().CallGet(ctx, gomock.Any().Return(tmpCall, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), gomock.Any(), gomock.Any())
				mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any().Return(&channel.Channel{TMEnd: dbhandler.DefaultTimeStamp}, nil)
			} else {
				if !tt.responseAMD.Async {
					mockReq.EXPECT().CallV1CallActionNext(ctx, tt.responseAMD.CallID, false)
				}
			}
			mockChannel.EXPECT().HangingUp(ctx, tt.channelID, ari.ChannelCauseNormalClearing.Return(&channel.Channel{}, nil)

			if err := h.startServiceFromAMD(ctx, tt.channelID, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
