package channelhandler

import (
	"context"
	"reflect"
	"testing"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-call-manager/models/ari"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_ARIStasisStart(t *testing.T) {

	type test struct {
		name string

		event *ari.StasisStart

		responseChannel *channel.Channel

		expectID         string
		expectType       channel.Type
		expectStasisName string
		expectStasisData map[channel.StasisDataType]string
		expectDirection  channel.Direction

		expectSIPCallID    string
		expectSIPPai       string
		expectSIPPrivacy   string
		expectSIPTransport channel.SIPTransport

		expectRes *channel.Channel
	}

	tests := []test{
		{
			name: "normal",

			event: &ari.StasisStart{
				Event: ari.Event{
					Type:        ari.EventTypeStasisStart,
					Application: "voipbin",
					Timestamp:   "2020-04-25T00:27:18.342",
					AsteriskID:  "42:01:0a:a4:00:03",
				},
				Args: ari.ArgsMap{
					string(channel.StasisDataTypeContextType): "call",
					string(channel.StasisDataTypeContext):     "call-in",
					string(channel.StasisDataTypeDomain):      "test.trunk.voipbin.net",
					string(channel.StasisDataTypeSource):      "213.127.79.161",
					string(channel.StasisDataTypeDirection):   "incoming",
					string(channel.StasisDataTypeSIPCallID):   "8juJJyujlS",
					string(channel.StasisDataTypeSIPPAI):      "tel:+821100000001",
					string(channel.StasisDataTypeSIPPrivacy):  "id",
					string(channel.StasisDataTypeTransport):   "udp",
				},
				Channel: ari.Channel{
					ID:           "1587774438.2390",
					Name:         "PJSIP/in-voipbin-00001948",
					Language:     "en",
					CreationTime: "2020-04-25T00:27:18.341",
					State:        ari.ChannelStateRing,
					Caller: ari.CallerID{
						Name:   "tttt",
						Number: "pchero",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "1234234324",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "voipbin,context_type=call,context=call-in,domain=test.trunk.voipbin.net,source=213.127.79.161,direction=incoming,sip_call_id=8juJJyujlS,sip_pai=,sip_privacy=",
					},
				},
			},

			responseChannel: &channel.Channel{
				ID: "1587774438.2390",
				StasisData: map[channel.StasisDataType]string{
					channel.StasisDataTypeContextType: "call",
					channel.StasisDataTypeContext:     "call-in",
					channel.StasisDataTypeDomain:      "test.trunk.voipbin.net",
					channel.StasisDataTypeSource:      "213.127.79.161",
					channel.StasisDataTypeDirection:   "incoming",
					channel.StasisDataTypeSIPCallID:   "8juJJyujlS",
					channel.StasisDataTypeSIPPAI:      "tel:+821100000001",
					channel.StasisDataTypeSIPPrivacy:  "id",
					channel.StasisDataTypeTransport:   "udp",
				},
			},

			expectID:         "1587774438.2390",
			expectType:       channel.TypeCall,
			expectStasisName: "voipbin",
			expectStasisData: map[channel.StasisDataType]string{
				channel.StasisDataTypeContextType: "call",
				channel.StasisDataTypeContext:     "call-in",
				channel.StasisDataTypeDomain:      "test.trunk.voipbin.net",
				channel.StasisDataTypeSource:      "213.127.79.161",
				channel.StasisDataTypeDirection:   "incoming",
				channel.StasisDataTypeSIPCallID:   "8juJJyujlS",
				channel.StasisDataTypeSIPPAI:      "tel:+821100000001",
				channel.StasisDataTypeSIPPrivacy:  "id",
				channel.StasisDataTypeTransport:   "udp",
			},
			expectDirection: channel.DirectionIncoming,

			expectSIPCallID:    "8juJJyujlS",
			expectSIPPai:       "tel:+821100000001",
			expectSIPPrivacy:   "id",
			expectSIPTransport: "udp",

			expectRes: &channel.Channel{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ChannelSetStasisInfo(ctx, tt.expectID, tt.expectType, tt.expectStasisName, tt.expectStasisData, tt.expectDirection).Return(nil)
			mockDB.EXPECT().ChannelGet(ctx, tt.expectID).Return(tt.responseChannel, nil)

			mockDB.EXPECT().ChannelSetSIPCallID(ctx, tt.expectID, tt.expectSIPCallID).Return(nil)
			mockDB.EXPECT().ChannelSetDataItem(ctx, tt.expectID, "sip_pai", tt.expectSIPPai).Return(nil)
			mockDB.EXPECT().ChannelSetDataItem(ctx, tt.expectID, "sip_privacy", tt.expectSIPPrivacy).Return(nil)
			mockDB.EXPECT().ChannelSetSIPTransport(ctx, tt.expectID, tt.expectSIPTransport).Return(nil)
			mockDB.EXPECT().ChannelGet(ctx, tt.expectID).Return(tt.responseChannel, nil)

			mockDB.EXPECT().ChannelGet(ctx, tt.expectID).Return(tt.responseChannel, nil)

			res, err := h.ARIStasisStart(ctx, tt.event)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if reflect.DeepEqual(tt.responseChannel, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChannel, res)
			}
		})
	}
}

func Test_getChannelType(t *testing.T) {

	type test struct {
		name string

		context channel.Context

		expectRes channel.Type
	}

	tests := []test{
		{
			name: "application",

			context:   channel.ContextApplication,
			expectRes: channel.TypeApplication,
		},
		{
			name: "conference incoming",

			context:   channel.ContextConfIncoming,
			expectRes: channel.TypeConfbridge,
		},
		{
			name: "conference outoing",

			context:   channel.ContextConfOutgoing,
			expectRes: channel.TypeConfbridge,
		},
		{
			name: "external media",

			context:   channel.ContextExternalMedia,
			expectRes: channel.TypeExternal,
		},
		{
			name: "call incoming",

			context:   channel.ContextCallIncoming,
			expectRes: channel.TypeCall,
		},
		{
			name: "call outgoing",

			context:   channel.ContextCallOutgoing,
			expectRes: channel.TypeCall,
		},
		{
			name: "call service",

			context:   channel.ContextCallService,
			expectRes: channel.TypeCall,
		},
		{
			name: "join call",

			context:   channel.ContextJoinCall,
			expectRes: channel.TypeJoin,
		},
		{
			name: "recording",

			context:   channel.ContextRecording,
			expectRes: channel.TypeRecording,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := channelHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
			}

			res := h.getChannelType(tt.context)

			if reflect.DeepEqual(tt.expectRes, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ARIChannelStateChange_outgoing_statusUp(t *testing.T) {

	tests := []struct {
		name  string
		event *ari.ChannelStateChange

		responseChannel    *channel.Channel
		responseValCallID  string
		responseValPai     string
		responseValPrivacy string

		expectTransport channel.SIPTransport
	}{
		{
			name: "normal",
			event: &ari.ChannelStateChange{
				Event: ari.Event{
					Type:        ari.EventTypeChannelStateChange,
					Application: "voipbin",
					Timestamp:   "2020-04-25T19:17:13.786",
					AsteriskID:  "42:01:0a:a4:00:05",
				},
				Channel: ari.Channel{
					ID:           "1587842233.10219",
					Name:         "PJSIP/in-voipbin-000026ef",
					Language:     "en",
					CreationTime: "2020-04-25T19:17:13.585",
					State:        ari.ChannelStateUp,
					Caller: ari.CallerID{
						Number: "586737682",
					},
					Dialplan: ari.DialplanCEP{
						Context:  "in-voipbin",
						Exten:    "46842002310",
						Priority: 2,
						AppName:  "Stasis",
						AppData:  "voipbin,CONTEXT=in-voipbin,SIP_CALLID=1491366011-850848062-1281392838,SIP_PAI=,SIP_PRIVACY=,DOMAIN=sip-service.voipbin.net,SOURCE=45.151.255.178",
					},
				},
			},

			responseChannel: &channel.Channel{
				ID:         "1587842233.10219",
				AsteriskID: "42:01:0a:a4:00:03",

				Direction: channel.DirectionOutgoing,
				State:     ari.ChannelStateUp,

				TMDelete: dbhandler.DefaultTimeStamp,
			},
			responseValCallID:  "286c4e9c-5b95-4bf8-a435-345883c99d27",
			responseValPai:     "tel:+31616818985",
			responseValPrivacy: "id",

			expectTransport: channel.SIPTransportNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockRequest := requesthandler.NewMockRequestHandler(mc)
			h := channelHandler{
				db:         mockDB,
				reqHandler: mockRequest,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChannelSetStateAnswer(ctx, tt.event.Channel.ID, tt.event.Channel.State).Return(nil)
			mockDB.EXPECT().ChannelGet(ctx, tt.event.Channel.ID).Return(tt.responseChannel, nil)

			// set sip_call_id
			mockRequest.EXPECT().AstChannelVariableGet(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, `CHANNEL(pjsip,Call-ID)`).Return(tt.responseValCallID, nil)
			mockDB.EXPECT().ChannelSetSIPCallID(ctx, tt.responseChannel.ID, tt.responseValCallID).Return(nil)

			// set sip_pai
			mockRequest.EXPECT().AstChannelVariableGet(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, `CHANNEL(pjsip,P-Asserted-Identity)`).Return(tt.responseValPai, nil)
			mockDB.EXPECT().ChannelSetDataItem(ctx, tt.responseChannel.ID, "sip_pai", tt.responseValPai).Return(nil)

			// set sip_privacy
			mockRequest.EXPECT().AstChannelVariableGet(ctx, tt.responseChannel.AsteriskID, tt.responseChannel.ID, `CHANNEL(pjsip,Privacy)`).Return(tt.responseValPrivacy, nil)
			mockDB.EXPECT().ChannelSetDataItem(ctx, tt.responseChannel.ID, "sip_privacy", tt.responseValPrivacy).Return(nil)

			// set sip transport
			mockDB.EXPECT().ChannelSetSIPTransport(ctx, tt.responseChannel.ID, tt.expectTransport).Return(nil)
			mockDB.EXPECT().ChannelGet(ctx, tt.responseChannel.ID).Return(tt.responseChannel, nil)

			mockDB.EXPECT().ChannelGet(ctx, tt.responseChannel.ID).Return(tt.responseChannel, nil)

			res, err := h.ARIChannelStateChange(ctx, tt.event)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if !reflect.DeepEqual(res, tt.responseChannel) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChannel, res)
			}
		})
	}
}
