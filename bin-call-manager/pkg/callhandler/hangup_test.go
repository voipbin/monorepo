package callhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"monorepo/bin-route-manager/models/route"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/bridgehandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
	"monorepo/bin-call-manager/pkg/dbhandler"
	"monorepo/bin-call-manager/pkg/groupcallhandler"
)

func Test_Hangup(t *testing.T) {

	tests := []struct {
		name    string
		channel *channel.Channel

		responseCall    *call.Call
		responseChannel *channel.Channel
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID:          "70271162-1772-11ec-a941-fb10a2f9c2e7",
				AsteriskID:  "80:fa:5b:5e:da:81",
				HangupCause: ari.ChannelCauseNormalClearing,
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("7076de7c-1772-11ec-86f2-835e7382daf2"),
				},
				ChannelID: "70271162-1772-11ec-a941-fb10a2f9c2e7",
				Status:    call.StatusProgressing,
				Action: fmaction.Action{
					Type: fmaction.TypeEcho,
				},
			},
			responseChannel: &channel.Channel{
				TMEnd: dbhandler.DefaultTimeStamp,
			},
		},
		{
			name: "chained calls",

			channel: &channel.Channel{
				ID:          "e3c68930-1778-11ec-8c04-0bcef8a75b4f",
				AsteriskID:  "80:fa:5b:5e:da:81",
				HangupCause: ari.ChannelCauseNormalClearing,
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e37dcd4e-1778-11ec-95c1-5b6f4657bd15"),
				},
				ChannelID: "e3c68930-1778-11ec-8c04-0bcef8a75b4f",
				Status:    call.StatusProgressing,
				Action: fmaction.Action{
					Type: fmaction.TypeEcho,
				},
				ChainedCallIDs: []uuid.UUID{
					uuid.FromStringOrNil("f8913c7a-1778-11ec-bcca-dbdc63ee1e38"),
					uuid.FromStringOrNil("f8e1cf1e-1778-11ec-ba6f-e73cb284ba93"),
				},
			},
			responseChannel: &channel.Channel{
				TMEnd: dbhandler.DefaultTimeStamp,
			},
		},
		{
			name: "has groupcall info",

			channel: &channel.Channel{
				ID:          "09e6139c-d901-11ed-9ec4-c7733d43bc03",
				AsteriskID:  "80:fa:5b:5e:da:81",
				HangupCause: ari.ChannelCauseNormalClearing,
			},

			responseCall: &call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0a3988a6-d901-11ed-9e5a-af6485ff8915"),
				},
				ChannelID: "09e6139c-d901-11ed-9ec4-c7733d43bc03",
				Status:    call.StatusProgressing,
				Action: fmaction.Action{
					Type: fmaction.TypeEcho,
				},
				GroupcallID: uuid.FromStringOrNil("0a660c00-d901-11ed-9d27-eb63c32e1192"),
			},
			responseChannel: &channel.Channel{
				TMEnd: dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)
			mockGroupcall := groupcallhandler.NewMockGroupcallHandler(mc)

			h := &callHandler{
				utilHandler:      mockUtil,
				reqHandler:       mockReq,
				db:               mockDB,
				notifyHandler:    mockNotfiy,
				channelHandler:   mockChannel,
				bridgeHandler:    mockBridge,
				groupcallHandler: mockGroupcall,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime()).AnyTimes()

			mockDB.EXPECT().CallGetByChannelID(ctx, tt.channel.ID).Return(tt.responseCall, nil)
			mockBridge.EXPECT().Destroy(ctx, tt.responseCall.BridgeID).Return(nil)
			mockDB.EXPECT().CallSetHangup(ctx, tt.responseCall.ID, call.HangupReasonNormal, call.HangupByRemote).Return(nil)
			tt.responseCall.Status = call.StatusHangup
			mockDB.EXPECT().CallGet(ctx, tt.responseCall.ID).Return(tt.responseCall, nil)
			mockNotfiy.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallHangup, gomock.Any())
			if tt.responseCall.GroupcallID != uuid.Nil {
				mockReq.EXPECT().CallV1GroupcallHangupCall(ctx, tt.responseCall.GroupcallID).Return(nil)
			}
			mockReq.EXPECT().FlowV1ActiveflowStop(ctx, tt.responseCall.ActiveflowID).Return(&fmactiveflow.Activeflow{}, nil)

			for _, chainedCallID := range tt.responseCall.ChainedCallIDs {
				tmpCall := &call.Call{
					Identity: commonidentity.Identity{
						ID: chainedCallID,
					},
					Status: call.StatusProgressing,
				}
				mockDB.EXPECT().CallGet(ctx, chainedCallID).Return(tmpCall, nil)
				mockDB.EXPECT().CallSetStatus(ctx, tmpCall.ID, call.StatusTerminating).Return(nil)

				tmpCall2 := &call.Call{
					Identity: commonidentity.Identity{
						ID: chainedCallID,
					},
					Status: call.StatusTerminating,
				}
				mockDB.EXPECT().CallGet(ctx, tmpCall.ID).Return(tmpCall2, nil)
				mockNotfiy.EXPECT().PublishWebhookEvent(ctx, tmpCall2.CustomerID, call.EventTypeCallTerminating, gomock.Any())
				mockChannel.EXPECT().HangingUp(ctx, tmpCall2.ChannelID, ari.ChannelCauseNormalClearing).Return(tt.responseChannel, nil)
			}

			res, err := h.Hangup(ctx, tt.channel)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)
			if !reflect.DeepEqual(res, tt.responseCall) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseCall, res)
			}
		})
	}
}

func Test_hangingUpWithCause(t *testing.T) {

	tests := []struct {
		name string

		id    uuid.UUID
		cause ari.ChannelCause

		responseCall    *call.Call
		responseChannel *channel.Channel

		expectCallStatus call.Status
		expectEventType  string
	}{
		{
			"normal terminating",

			uuid.FromStringOrNil("785880aa-1777-11ec-abec-2b721201c1af"),
			ari.ChannelCauseNormalClearing,

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("785880aa-1777-11ec-abec-2b721201c1af"),
				},
				ChannelID: "7877dce8-1777-11ec-b4ea-3bb953ca2fe7",
				Status:    call.StatusProgressing,
				Action: fmaction.Action{
					Type: fmaction.TypeEcho,
				},
			},
			&channel.Channel{
				ID:         "7877dce8-1777-11ec-b4ea-3bb953ca2fe7",
				AsteriskID: "80:fa:5b:5e:da:81",
				TMEnd:      dbhandler.DefaultTimeStamp,
			},

			call.StatusTerminating,
			call.EventTypeCallTerminating,
		},
		{
			"canceling",

			uuid.FromStringOrNil("ac477e50-ab1c-11ec-b50f-7bb28cc97fd4"),
			ari.ChannelCauseNormalClearing,

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ac477e50-ab1c-11ec-b50f-7bb28cc97fd4"),
				},
				ChannelID: "ac7411a4-ab1c-11ec-bce4-e7e983448875",
				Status:    call.StatusDialing,
				Direction: call.DirectionOutgoing,
				Action: fmaction.Action{
					Type: fmaction.TypeEcho,
				},
			},
			&channel.Channel{
				ID:         "ac7411a4-ab1c-11ec-bce4-e7e983448875",
				AsteriskID: "80:fa:5b:5e:da:81",
				TMEnd:      dbhandler.DefaultTimeStamp,
			},

			call.StatusCanceling,
			call.EventTypeCallCanceling,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotfiy,
				channelHandler: mockChannel,
			}
			ctx := context.Background()

			mockUtil.EXPECT().TimeGetCurTime().Return(utilhandler.TimeGetCurTime()).AnyTimes()
			mockDB.EXPECT().CallGet(ctx, tt.responseCall.ID).Return(tt.responseCall, nil)

			// updateStatus
			mockDB.EXPECT().CallSetStatus(ctx, tt.responseCall.ID, tt.expectCallStatus).Return(nil)

			tmpCall := *tt.responseCall
			tmpCall.Status = tt.expectCallStatus
			mockDB.EXPECT().CallGet(ctx, tt.responseCall.ID).Return(&tmpCall, nil)
			mockNotfiy.EXPECT().PublishWebhookEvent(ctx, tmpCall.CustomerID, tt.expectEventType, &tmpCall)

			mockChannel.EXPECT().HangingUp(ctx, tmpCall.ChannelID, tt.cause).Return(tt.responseChannel, nil)

			res, err := h.hangingUpWithCause(ctx, tt.id, tt.cause)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(&tmpCall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tmpCall, res)
			}
		})
	}
}

func Test_hangingupWithReference(t *testing.T) {

	tests := []struct {
		name string

		call        *call.Call
		referenceID uuid.UUID

		responseReferenceCall    *call.Call
		responseReferenceChannel *channel.Channel
		responseCall             *call.Call
	}{
		{
			"normal",

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("045e6cd0-41f7-4b24-833d-f17b0236b9a6"),
				},
				Status: call.StatusProgressing,
			},
			uuid.FromStringOrNil("0bd3152c-1a1e-4464-9c60-395bdcafa6bd"),

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("0bd3152c-1a1e-4464-9c60-395bdcafa6bd"),
				},
				Status:    call.StatusHangup,
				ChannelID: "19b1bc03-cf90-47b9-9fbd-5fef6d9393a4",
			},
			&channel.Channel{
				ID:          "19b1bc03-cf90-47b9-9fbd-5fef6d9393a4",
				HangupCause: ari.ChannelCauseNoAnswer,
				TMEnd:       dbhandler.DefaultTimeStamp,
			},
			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("045e6cd0-41f7-4b24-833d-f17b0236b9a6"),
				},
				Status: call.StatusTerminating,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotfiy,
				channelHandler: mockChannel,
			}
			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.referenceID).Return(tt.responseReferenceCall, nil)
			mockChannel.EXPECT().Get(ctx, tt.responseReferenceCall.ChannelID).Return(tt.responseReferenceChannel, nil)

			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.call, nil)
			mockDB.EXPECT().CallSetStatus(ctx, tt.call.ID, call.StatusTerminating).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.responseCall, nil)
			mockNotfiy.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallTerminating, tt.responseCall)

			mockChannel.EXPECT().HangingUp(ctx, tt.responseCall.ChannelID, tt.responseReferenceChannel.HangupCause).Return(tt.responseReferenceChannel, nil)

			res, err := h.hangingupWithReference(ctx, tt.call, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseCall, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseCall, res)
			}
		})
	}
}

func Test_isRetryable(t *testing.T) {

	tests := []struct {
		name string

		call    *call.Call
		channel *channel.Channel

		expectRes bool
	}{
		{
			name: "call needs retry",

			call: &call.Call{
				Status:    call.StatusRinging,
				Direction: call.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeTel,
				},
				DialrouteID: uuid.FromStringOrNil("acda1e28-e1d9-11ed-a059-2bb9e2d99f68"),
				Dialroutes: []route.Route{
					{
						ID: uuid.FromStringOrNil("acda1e28-e1d9-11ed-a059-2bb9e2d99f68"),
					},
					{
						ID: uuid.FromStringOrNil("ad1edd92-e1d9-11ed-b07f-43651ba67795"),
					},
				},
			},
			channel: &channel.Channel{
				HangupCause: ari.ChannelCauseInterworking,
			},

			expectRes: true,
		},
		{
			name: "call direction is not outgoing",

			call: &call.Call{
				Direction: call.DirectionIncoming,
			},
			channel: &channel.Channel{},

			expectRes: false,
		},
		{
			name: "destination type is not tel type",

			call: &call.Call{
				Direction: call.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeAgent,
				},
			},
			channel: &channel.Channel{},

			expectRes: false,
		},
		{
			name: "early media and ringing",

			call: &call.Call{
				Status:    call.StatusRinging,
				Direction: call.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeTel,
				},
				Data: map[call.DataType]string{
					call.DataTypeEarlyExecution: "true",
				},
			},
			channel: &channel.Channel{},

			expectRes: false,
		},
		{
			name: "not retryable codes",

			call: &call.Call{
				Status:    call.StatusRinging,
				Direction: call.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeTel,
				},
			},
			channel: &channel.Channel{
				HangupCause: ari.ChannelCauseNormalClearing,
			},

			expectRes: false,
		},
		{
			name: "call status is not retryable",

			call: &call.Call{
				Status:    call.StatusProgressing,
				Direction: call.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeTel,
				},
			},
			channel: &channel.Channel{
				HangupCause: ari.ChannelCauseCallDurationTimeout,
			},

			expectRes: false,
		},
		{
			name: "call has no dial route",

			call: &call.Call{
				Status:    call.StatusRinging,
				Direction: call.DirectionOutgoing,
				Destination: commonaddress.Address{
					Type: commonaddress.TypeTel,
				},
			},
			channel: &channel.Channel{
				HangupCause: ari.ChannelCauseUserBusy,
			},

			expectRes: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotfiy := notifyhandler.NewMockNotifyHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotfiy,
				channelHandler: mockChannel,
			}
			ctx := context.Background()

			res := h.isRetryable(ctx, tt.call, tt.channel)
			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}
