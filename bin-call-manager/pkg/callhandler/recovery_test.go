package callhandler

import (
	"context"
	"fmt"
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
	"time"

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

		expectedRole             asteriskRole
		expectedAppArgs          string
		expectedDialURI          string
		expectedChannelVariables map[string]string
	}{
		{
			name: "outgoing call",

			ch: &channel.Channel{
				ID:        "bce609a6-4822-11f0-846b-afe390a46720",
				Type:      channel.TypeCall,
				SIPCallID: "c99ae0fe-4822-11f0-8d1a-fb8f786822a1",
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c97b20ca-4822-11f0-9345-9b3103d03af7"),
				},
				Status:    call.StatusProgressing,
				Direction: call.DirectionOutgoing,
			},
			responseRecoveryDetail: &recoveryDetail{
				RequestURI:   "sip:+821****6521@10.31.35.4:5070;transport=udp",
				Routes:       "<sip:10.164.0.20;transport=tcp;r2=on;lr>, <sip:34.90.68.237:5060;r2=on;lr>",
				RecordRoutes: "<sip:34.90.68.237:5060;r2=on;lr>, <sip:10.164.0.20;transport=tcp;r2=on;lr>",
				CallID:       "1ced6b72-70c6-4c45-82e1-078568bf9d45",

				FromDisplay: "Anonymous",
				FromURI:     "sip:anonymous@anonymous.invalid",
				FromTag:     "2f41957b-9c9d-45d1-a18c-310ce92516ba",

				ToDisplay: "",
				ToURI:     "sip:+821****6521@sip.telnyx.com",
				ToTag:     "2cDr76BUDp2SF",
				CSeq:      2595,
			},
			responseUUID: uuid.FromStringOrNil("a9683bae-4824-11f0-872e-f76c0240c5e7"),
			responseChannel: &channel.Channel{
				ID: "a9683bae-4824-11f0-872e-f76c0240c5e7",
			},

			expectedRole:    asteriskRoleUAC,
			expectedAppArgs: "context_type=call,context=call-recovery,call_id=c97b20ca-4822-11f0-9345-9b3103d03af7,recovery_channel_id=bce609a6-4822-11f0-846b-afe390a46720",
			expectedDialURI: "pjsip/call-out/sip:+821****6521@10.31.35.4:5070;transport=udp",
			expectedChannelVariables: map[string]string{
				channelVariableRecoveryFromDisplay: "Anonymous",
				channelVariableRecoveryFromURI:     "sip:anonymous@anonymous.invalid",
				channelVariableRecoveryFromTag:     "2f41957b-9c9d-45d1-a18c-310ce92516ba",

				channelVariableRecoveryToDisplay: "",
				channelVariableRecoveryToURI:     "sip:+821****6521@sip.telnyx.com",
				channelVariableRecoveryToTag:     "2cDr76BUDp2SF",

				channelVariableRecoveryCallID: "1ced6b72-70c6-4c45-82e1-078568bf9d45",
				channelVariableRecoveryCSeq:   "2595",

				channelVariableRecoveryRoutes:       "<sip:10.164.0.20;transport=tcp;r2=on;lr>, <sip:34.90.68.237:5060;r2=on;lr>",
				channelVariableRecoveryRecordRoutes: "<sip:34.90.68.237:5060;r2=on;lr>, <sip:10.164.0.20;transport=tcp;r2=on;lr>",
				channelVariableRecoveryRequestURI:   "sip:+821****6521@10.31.35.4:5070;transport=udp",
			},
		},
		{
			name: "incoming call without a known local CSeq",

			ch: &channel.Channel{
				ID:        "4e0b6f1a-9f72-11f1-9c3d-2a1b3c4d5e01",
				Type:      channel.TypeCall,
				SIPCallID: "4e0b6f1a-sip",
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4e4a2c8e-9f72-11f1-8d4e-3b2c4d5e6f02"),
				},
				Status:    call.StatusProgressing,
				Direction: call.DirectionIncoming,
			},
			responseRecoveryDetail: &recoveryDetail{
				RequestURI:   "sip:caller@198.51.100.10:5060",
				Routes:       "<sip:199.127.61.42;r2=on;lr>, <sip:172.24.0.246;transport=tcp;r2=on;lr>",
				RecordRoutes: "<sip:199.127.61.42;r2=on;lr>, <sip:172.24.0.246;transport=tcp;r2=on;lr>",
				CallID:       "4e0b6f1a-sip",

				FromURI: "sip:2000@example.voipbin.net",
				FromTag: "as-local",
				ToURI:   "sip:caller@example.com",
				ToTag:   "remote-tag",
				CSeq:    0,
			},
			responseUUID: uuid.FromStringOrNil("4e88f3a0-9f72-11f1-9e5f-4c3d5e6f7a03"),
			responseChannel: &channel.Channel{
				ID: "4e88f3a0-9f72-11f1-9e5f-4c3d5e6f7a03",
			},

			expectedRole:    asteriskRoleUAS,
			expectedAppArgs: "context_type=call,context=call-recovery,call_id=4e4a2c8e-9f72-11f1-8d4e-3b2c4d5e6f02,recovery_channel_id=4e0b6f1a-9f72-11f1-9c3d-2a1b3c4d5e01",
			expectedDialURI: "pjsip/call-out/sip:caller@198.51.100.10:5060",
			expectedChannelVariables: map[string]string{
				channelVariableRecoveryFromDisplay: "",
				channelVariableRecoveryFromURI:     "sip:2000@example.voipbin.net",
				channelVariableRecoveryFromTag:     "as-local",

				channelVariableRecoveryToDisplay: "",
				channelVariableRecoveryToURI:     "sip:caller@example.com",
				channelVariableRecoveryToTag:     "remote-tag",

				channelVariableRecoveryCallID: "4e0b6f1a-sip",

				channelVariableRecoveryRoutes:       "<sip:199.127.61.42;r2=on;lr>, <sip:172.24.0.246;transport=tcp;r2=on;lr>",
				channelVariableRecoveryRecordRoutes: "<sip:199.127.61.42;r2=on;lr>, <sip:172.24.0.246;transport=tcp;r2=on;lr>",
				channelVariableRecoveryRequestURI:   "sip:caller@198.51.100.10:5060",
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

			mockDB.EXPECT().CallGetByChannelID(ctx, tt.ch.ID).Return(tt.responseCall, nil)
			mockDB.EXPECT().CallRecoveryClaim(ctx, tt.responseCall.ID, recoveryClaimTTL).Return(true, nil)
			mockRecovery.EXPECT().GetRecoveryDetail(ctx, tt.ch.SIPCallID, tt.expectedRole).Return(tt.responseRecoveryDetail, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
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
			).Return(tt.responseChannel, nil)

			if errRecovery := h.recoveryRun(ctx, tt.ch); errRecovery != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errRecovery)
			}
		})
	}
}

// Test_recoveryRun_skip checks the calls recovery does not attempt: the strict mocks are the assertion
// (no claim, no Homer query, no channel).
func Test_recoveryRun_skip(t *testing.T) {

	base := func() *call.Call {
		return &call.Call{
			Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("6b2c1d0e-9f72-11f1-8a1b-5d4e6f7a8b01")},
			Status:    call.StatusProgressing,
			Direction: call.DirectionIncoming,
		}
	}

	tests := []struct {
		name string
		call func() *call.Call
	}{
		{
			name: "not progressing",
			call: func() *call.Call { c := base(); c.Status = call.StatusTerminating; return c },
		},
		{
			name: "unknown direction",
			call: func() *call.Call { c := base(); c.Direction = ""; return c },
		},
		{
			name: "in a confbridge",
			call: func() *call.Call {
				c := base()
				c.ConfbridgeID = uuid.FromStringOrNil("6b6a0e5c-9f72-11f1-9b2c-6e5f7a8b9c02")
				return c
			},
		},
		{
			name: "has chained calls",
			call: func() *call.Call {
				c := base()
				c.ChainedCallIDs = []uuid.UUID{uuid.FromStringOrNil("6ba7d3b8-9f72-11f1-ac3d-7f6a8b9c0d03")}
				return c
			},
		},
		{
			name: "has a master call",
			call: func() *call.Call {
				c := base()
				c.MasterCallID = uuid.FromStringOrNil("6be4c9f4-9f72-11f1-bd4e-8a7b9c0d1e04")
				return c
			},
		},
		{
			name: "groupcall leg",
			call: func() *call.Call {
				c := base()
				c.GroupcallID = uuid.FromStringOrNil("6c21bf30-9f72-11f1-8e5f-9b8c0d1e2f05")
				return c
			},
		},
		{
			name: "has external media",
			call: func() *call.Call {
				c := base()
				c.ExternalMediaIDs = []uuid.UUID{uuid.FromStringOrNil("6c5eb46c-9f72-11f1-9f6a-0c9d1e2f3a06")}
				return c
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockRecovery := NewMockRecoveryHandler(mc)

			h := &callHandler{
				db:              mockDB,
				channelHandler:  mockChannel,
				recoveryHandler: mockRecovery,
			}
			ctx := context.Background()

			ch := &channel.Channel{ID: "6c9ba9a8-9f72-11f1-a07b-1d0e2f3a4b07", Type: channel.TypeCall, SIPCallID: "sip-call-id"}
			mockDB.EXPECT().CallGetByChannelID(ctx, ch.ID).Return(tt.call(), nil)

			if err := h.recoveryRun(ctx, ch); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_recoveryRun_claim checks that a recovery runs only with the claim: not acquired and a claim error both skip.
func Test_recoveryRun_claim(t *testing.T) {

	tests := []struct {
		name string

		responseClaimed bool
		responseErr     error
	}{
		{
			name:            "held by another recovery",
			responseClaimed: false,
		},
		{
			name:        "claim error",
			responseErr: fmt.Errorf("redis down"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockRecovery := NewMockRecoveryHandler(mc)

			h := &callHandler{
				db:              mockDB,
				channelHandler:  mockChannel,
				recoveryHandler: mockRecovery,
			}
			ctx := context.Background()

			ch := &channel.Channel{ID: "7a0e1f2a-9f72-11f1-b18c-2e1f3a4b5c08", Type: channel.TypeCall, SIPCallID: "sip-call-id"}
			c := &call.Call{
				Identity:  commonidentity.Identity{ID: uuid.FromStringOrNil("7a4b0c66-9f72-11f1-829d-3f2a4b5c6d09")},
				Status:    call.StatusProgressing,
				Direction: call.DirectionOutgoing,
			}
			mockDB.EXPECT().CallGetByChannelID(ctx, ch.ID).Return(c, nil)
			mockDB.EXPECT().CallRecoveryClaim(ctx, c.ID, recoveryClaimTTL).Return(tt.responseClaimed, tt.responseErr)

			if err := h.recoveryRun(ctx, ch); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

// Test_RecoveryStart_enabled checks that RecoveryStart looks up channels for the given asterisk
// ID (the only entry point of call recovery).
func Test_RecoveryStart_enabled(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockChannel := channelhandler.NewMockChannelHandler(mc)

	h := &callHandler{
		utilHandler:    mockUtil,
		channelHandler: mockChannel,
	}

	ctx := context.Background()
	startTime := time.Date(2026, 8, 31, 10, 0, 0, 0, time.UTC)
	endTime := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)

	mockUtil.EXPECT().TimeNowAdd(-(time.Hour * 24)).Return(&startTime)
	mockUtil.EXPECT().TimeNow().Return(&endTime)
	mockChannel.EXPECT().
		GetChannelsForRecovery(ctx, "3e:50:6b:43:bb:32", channel.TypeCall, &startTime, &endTime, defaultRecoveryChannelLimit).
		Return([]*channel.Channel{}, nil)

	if err := h.RecoveryStart(ctx, "3e:50:6b:43:bb:32"); err != nil {
		t.Errorf("Wrong match. expect: ok, got: %v", err)
	}
}
