package callhandler

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/bridgehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/flow"
	rmdomain "gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
)

func Test_startIncomingDomainTypeSIPDestinationTypeAgent(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource      *commonaddress.Address
		responseDestination *commonaddress.Address
		responseDomain      *rmdomain.Domain
		responseAgent       *amagent.Agent
		responseFlow        *fmflow.Flow

		expectDomainName string
		expectAgentID    uuid.UUID
		expectActions    []fmaction.Action
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675216038.171",

				DestinationName:   "",
				DestinationNumber: "agent-eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
				SourceName:        "",
				SourceNumber:      "test01",

				StasisData: map[string]string{
					"context": "call-in",
					"domain":  "test.sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "test01",
			},
			responseDestination: &commonaddress.Address{
				Type:   commonaddress.TypeAgent,
				Target: "eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b",
			},
			responseDomain: &rmdomain.Domain{
				ID:         uuid.FromStringOrNil("76f0fa45-0b43-455c-afe2-e572ff0b76c9"),
				CustomerID: uuid.FromStringOrNil("a7be89e0-8170-4f48-ac01-a81a31c6e344"),
			},
			responseAgent: &amagent.Agent{
				ID:         uuid.FromStringOrNil("eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"),
				CustomerID: uuid.FromStringOrNil("a7be89e0-8170-4f48-ac01-a81a31c6e344"),
			},
			responseFlow: &fmflow.Flow{
				ID: uuid.FromStringOrNil("1d82f6c0-e6a6-4718-8f23-720f845a8fbe"),
			},

			expectDomainName: "test",
			expectAgentID:    uuid.FromStringOrNil("eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"),
			expectActions: []fmaction.Action{
				{
					Type:   fmaction.TypeAgentCall,
					Option: []byte(`{"agent_id":"eb1ac5c0-ff63-47e2-bcdb-5da9c336eb4b"}`),
				},
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
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeSIP).Return(tt.responseSource)
			mockChannel.EXPECT().AddressGetDestinationWithoutSpecificType(tt.channel).Return(tt.responseDestination)
			mockReq.EXPECT().RegistrarV1DomainGetByDomainName(ctx, tt.expectDomainName).Return(tt.responseDomain, nil)

			mockReq.EXPECT().AgentV1AgentGet(ctx, tt.expectAgentID).Return(tt.responseAgent, nil)
			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.responseDomain.CustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.expectActions, false).Return(tt.responseFlow, nil)

			// startCallTypeFlow
			mockUtil.EXPECT().CreateUUID().Return(utilhandler.CreateUUID())
			mockUtil.EXPECT().CreateUUID().Return(utilhandler.CreateUUID())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIPDestinationTypeConference(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource      *commonaddress.Address
		responseDestination *commonaddress.Address
		responseDomain      *rmdomain.Domain
		responseConference  *cfconference.Conference
		responseFlow        *fmflow.Flow

		expectDomainName   string
		expectConferenceID uuid.UUID
		expectActions      []fmaction.Action
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675220154.178",

				DestinationName:   "",
				DestinationNumber: "conference-99accfb7-c0dd-4a54-997d-dd18af7bc280",
				SourceName:        "",
				SourceNumber:      "test01",

				StasisData: map[string]string{
					"context": "call-in",
					"domain":  "test.sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "test01",
			},
			responseDestination: &commonaddress.Address{
				Type:   commonaddress.TypeConference,
				Target: "99accfb7-c0dd-4a54-997d-dd18af7bc280",
			},
			responseDomain: &rmdomain.Domain{
				ID:         uuid.FromStringOrNil("76f0fa45-0b43-455c-afe2-e572ff0b76c9"),
				CustomerID: uuid.FromStringOrNil("a7be89e0-8170-4f48-ac01-a81a31c6e344"),
			},
			responseConference: &cfconference.Conference{
				ID:         uuid.FromStringOrNil("99accfb7-c0dd-4a54-997d-dd18af7bc280"),
				CustomerID: uuid.FromStringOrNil("a7be89e0-8170-4f48-ac01-a81a31c6e344"),
				FlowID:     uuid.FromStringOrNil("90f05e61-408b-429b-85fb-0ee3d2d77c21"),
			},
			responseFlow: &fmflow.Flow{
				ID: uuid.FromStringOrNil("531912e6-8a0d-4d9b-a03b-6760275bb0dd"),
			},

			expectDomainName:   "test",
			expectConferenceID: uuid.FromStringOrNil("99accfb7-c0dd-4a54-997d-dd18af7bc280"),
			expectActions: []fmaction.Action{
				{
					Type:   fmaction.TypeConferenceJoin,
					Option: []byte(`{"conference_id":"99accfb7-c0dd-4a54-997d-dd18af7bc280"}`),
				},
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
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeSIP).Return(tt.responseSource)
			mockChannel.EXPECT().AddressGetDestinationWithoutSpecificType(tt.channel).Return(tt.responseDestination)
			mockReq.EXPECT().RegistrarV1DomainGetByDomainName(ctx, tt.expectDomainName).Return(tt.responseDomain, nil)

			mockReq.EXPECT().ConferenceV1ConferenceGet(ctx, tt.expectConferenceID).Return(tt.responseConference, nil)
			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.responseDomain.CustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.expectActions, false).Return(tt.responseFlow, nil)

			// startCallTypeFlow
			mockUtil.EXPECT().CreateUUID().Return(utilhandler.CreateUUID())
			mockUtil.EXPECT().CreateUUID().Return(utilhandler.CreateUUID())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_startIncomingDomainTypeSIPDestinationTypeTel(t *testing.T) {
	tests := []struct {
		name string

		channel *channel.Channel

		responseSource      *commonaddress.Address
		responseDestination *commonaddress.Address
		responseDomain      *rmdomain.Domain
		responseFlow        *fmflow.Flow

		expectDomainName   string
		expectConferenceID uuid.UUID
		expectActions      []fmaction.Action
	}{
		{
			name: "normal",

			channel: &channel.Channel{
				ID: "asterisk-call-58f54b64c7-2kwmb-1675220876.181",

				DestinationName:   "",
				DestinationNumber: "+821100000001",
				SourceName:        "",
				SourceNumber:      "test01",

				StasisData: map[string]string{
					"context": "call-in",
					"domain":  "test.sip.voipbin.net",
					"source":  "222.112.233.190",
				},
			},

			responseSource: &commonaddress.Address{
				Type:   commonaddress.TypeSIP,
				Target: "test01",
			},
			responseDestination: &commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			responseDomain: &rmdomain.Domain{
				ID:         uuid.FromStringOrNil("76f0fa45-0b43-455c-afe2-e572ff0b76c9"),
				CustomerID: uuid.FromStringOrNil("a7be89e0-8170-4f48-ac01-a81a31c6e344"),
			},
			responseFlow: &fmflow.Flow{
				ID: uuid.FromStringOrNil("531912e6-8a0d-4d9b-a03b-6760275bb0dd"),
			},

			expectDomainName:   "test",
			expectConferenceID: uuid.FromStringOrNil("99accfb7-c0dd-4a54-997d-dd18af7bc280"),
			expectActions: []fmaction.Action{
				{
					Type:   fmaction.TypeConnect,
					Option: []byte(`{"source":{"type":"sip","target":"test01","target_name":"","name":"","detail":""},"destinations":[{"type":"tel","target":"+821100000001","target_name":"","name":"","detail":""}],"unchained":false}`),
				},
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
			mockBridge := bridgehandler.NewMockBridgeHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				notifyHandler:  mockNotify,
				channelHandler: mockChannel,
				bridgeHandler:  mockBridge,
			}

			ctx := context.Background()

			mockChannel.EXPECT().AddressGetSource(tt.channel, commonaddress.TypeSIP).Return(tt.responseSource)
			mockChannel.EXPECT().AddressGetDestinationWithoutSpecificType(tt.channel).Return(tt.responseDestination)
			mockReq.EXPECT().RegistrarV1DomainGetByDomainName(ctx, tt.expectDomainName).Return(tt.responseDomain, nil)

			mockReq.EXPECT().FlowV1FlowCreate(ctx, tt.responseDomain.CustomerID, fmflow.TypeFlow, gomock.Any(), gomock.Any(), tt.expectActions, false).Return(tt.responseFlow, nil)

			// startCallTypeFlow
			mockUtil.EXPECT().CreateUUID().Return(utilhandler.CreateUUID())
			mockUtil.EXPECT().CreateUUID().Return(utilhandler.CreateUUID())
			mockBridge.EXPECT().Start(ctx, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fmt.Errorf(""))
			mockChannel.EXPECT().HangingUp(ctx, gomock.Any(), gomock.Any()).Return(&channel.Channel{}, nil)

			if err := h.startIncomingDomainTypeSIP(ctx, tt.channel); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
