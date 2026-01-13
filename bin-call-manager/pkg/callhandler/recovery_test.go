package callhandler

import (
	"context"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/recordinghandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_recoveryRun(t *testing.T) {

	tests := []struct {
		name string

		ch *channel.Channel

		responseCall           *call.Call
		responseRecoveryDetail *recoveryDetail
		responseUUID           uuid.UUID
		responseChannel        *channel.Channel

		expectedAppArgs          string
		expectedDialURI          string
		expectedChannelVariables map[string]string
	}{
		{
			name: "normal",

			ch: &channel.Channel{
				ID:        "bce609a6-4822-11f0-846b-afe390a46720",
				Type:      channel.TypeCall,
				SIPCallID: "c99ae0fe-4822-11f0-8d1a-fb8f786822a1",
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c97b20ca-4822-11f0-9345-9b3103d03af7"),
				},
			},
			responseRecoveryDetail: &recoveryDetail{
				RequestURI:   "sip:+821021656521@10.31.35.4:5070;transport=udp",
				Routes:       "<sip:10.164.0.20;transport=tcp;r2=on;lr>, <sip:34.90.68.237:5060;r2=on;lr>, <sip:192.76.120.10;r2=on;lr;ftag=2f41957b-9c9d-45d1-a18c-310ce92516ba>, <sip:10.255.0.1;r2=on;lr;ftag=2f41957b-9c9d-45d1-a18c-310ce92516ba>",
				RecordRoutes: "<sip:10.164.0.20;transport=tcp;r2=on;lr>, <sip:34.90.68.237:5060;r2=on;lr>",
				CallID:       "1ced6b72-70c6-4c45-82e1-078568bf9d45",

				FromDisplay: "Anonymous",
				FromURI:     "sip:anonymous@anonymous.invalid",
				FromTag:     "2f41957b-9c9d-45d1-a18c-310ce92516ba",

				ToDisplay: "",
				ToURI:     "sip:+821021656521@sip.telnyx.com",
				ToTag:     "2cDr76BUDp2SF",
				CSeq:      2595,
			},
			responseUUID: uuid.FromStringOrNil("a9683bae-4824-11f0-872e-f76c0240c5e7"),
			responseChannel: &channel.Channel{
				ID: "a9683bae-4824-11f0-872e-f76c0240c5e7",
			},

			expectedAppArgs: "context_type=call,context=call-recovery,call_id=c97b20ca-4822-11f0-9345-9b3103d03af7",
			expectedDialURI: "pjsip/call-out/sip:+821021656521@10.31.35.4:5070;transport=udp",
			expectedChannelVariables: map[string]string{
				channelVariableRecoveryFromDisplay: "Anonymous",
				channelVariableRecoveryFromURI:     "sip:anonymous@anonymous.invalid",
				channelVariableRecoveryFromTag:     "2f41957b-9c9d-45d1-a18c-310ce92516ba",

				channelVariableRecoveryToDisplay: "",
				channelVariableRecoveryToURI:     "sip:+821021656521@sip.telnyx.com",
				channelVariableRecoveryToTag:     "2cDr76BUDp2SF",

				channelVariableRecoveryCallID: "1ced6b72-70c6-4c45-82e1-078568bf9d45",
				channelVariableRecoveryCSeq:   "2595",

				channelVariableRecoveryRoutes:       "<sip:10.164.0.20;transport=tcp;r2=on;lr>, <sip:34.90.68.237:5060;r2=on;lr>, <sip:192.76.120.10;r2=on;lr;ftag=2f41957b-9c9d-45d1-a18c-310ce92516ba>, <sip:10.255.0.1;r2=on;lr;ftag=2f41957b-9c9d-45d1-a18c-310ce92516ba>",
				channelVariableRecoveryRecordRoutes: "<sip:10.164.0.20;transport=tcp;r2=on;lr>, <sip:34.90.68.237:5060;r2=on;lr>",
				channelVariableRecoveryRequestURI:   "sip:+821021656521@10.31.35.4:5070;transport=udp",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockRecording := recordinghandler.NewMockRecordingHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockRecovery := NewMockRecoveryHandler(mc)

			h := &callHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				notifyHandler:    mockNotify,
				db:               mockDB,
				recordingHandler: mockRecording,
				channelHandler:   mockChannel,
				recoveryHandler:  mockRecovery,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGetByChannelID(ctx, tt.ch.ID.Return(tt.responseCall, nil)
			mockRecovery.EXPECT().GetRecoveryDetail(ctx, tt.ch.SIPCallID.Return(tt.responseRecoveryDetail, nil)
			mockUtil.EXPECT().UUIDCreate(.Return(tt.responseUUID)
			mockChannel.EXPECT().StartChannel(
				ctx,
				requesthandler.AsteriskIDCall,
				tt.responseUUID.String(),
				tt.expectedAppArgs,
				tt.expectedDialURI,
				"",
				"",
				"",
				tt.expectedChannelVariables,
			.Return(tt.responseChannel, nil)

			if errRecovery := h.recoveryRun(ctx, tt.ch); errRecovery != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errRecovery)
			}
		})
	}
}
