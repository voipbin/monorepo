package callhandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	rmprovider "gitlab.com/voipbin/bin-manager/route-manager.git/models/provider"
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/channel"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/channelhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func Test_CreateCallOutgoing_TypeSIP(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		customerID     uuid.UUID
		flowID         uuid.UUID
		activeflowID   uuid.UUID
		masterCallID   uuid.UUID
		source         commonaddress.Address
		destination    commonaddress.Address
		earlyExecution bool
		connect        bool

		responseActiveflow  *fmactiveflow.Activeflow
		responseUUIDChannel uuid.UUID

		expectDialrouteTarget string
		expectCall            *call.Call
		expectEndpointDst     string
		expectVariables       map[string]string
	}{
		{
			"normal",

			uuid.FromStringOrNil("f1afa9ce-ecb2-11ea-ab94-a768ab787da0"),
			uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
			uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
			uuid.FromStringOrNil("679f0eb2-8c21-41a6-876d-9d778b1b0167"),
			uuid.FromStringOrNil("5935ff8a-8c8f-11ec-b26a-3fee169eaf45"),
			commonaddress.Address{
				Type:       commonaddress.TypeSIP,
				Target:     "testsrc@test.com",
				TargetName: "test",
			},
			commonaddress.Address{
				Type:       commonaddress.TypeSIP,
				Target:     "testoutgoing@test.com",
				TargetName: "test target",
			},
			true,
			true,

			&fmactiveflow.Activeflow{
				CurrentAction: fmaction.Action{
					ID: fmaction.IDStart,
				},
			},
			uuid.FromStringOrNil("80d67b3a-5f3b-11ed-a709-0f2943ef0184"),

			"",
			&call.Call{
				ID:         uuid.FromStringOrNil("f1afa9ce-ecb2-11ea-ab94-a768ab787da0"),
				CustomerID: uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
				ChannelID:  "80d67b3a-5f3b-11ed-a709-0f2943ef0184",
				FlowID:     uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
				Type:       call.TypeFlow,

				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},

				Status:    call.StatusDialing,
				Direction: call.DirectionOutgoing,
				Source: commonaddress.Address{
					Type:       commonaddress.TypeSIP,
					Target:     "testsrc@test.com",
					TargetName: "test",
				},
				Destination: commonaddress.Address{
					Type:       commonaddress.TypeSIP,
					Target:     "testoutgoing@test.com",
					TargetName: "test target",
				},
				Data: map[call.DataType]string{
					call.DataTypeEarlyExecution:            "true",
					call.DataTypeExecuteNextMasterOnHangup: "true",
				},
				Action: fmaction.Action{
					ID: fmaction.IDStart,
				},

				Dialroutes: []rmroute.Route{},

				TMCreate:      "2021-02-19 06:32:14.621",
				TMUpdate:      dbhandler.DefaultTimeStamp,
				TMRinging:     dbhandler.DefaultTimeStamp,
				TMProgressing: dbhandler.DefaultTimeStamp,
				TMHangup:      dbhandler.DefaultTimeStamp,
			},
			"pjsip/call-out/sip:testoutgoing@test.com",
			map[string]string{
				"CALLERID(name)":                        "test",
				"CALLERID(num)":                         "testsrc@test.com",
				"PJSIP_HEADER(add,VBOUT-SDP_Transport)": "RTP/AVP",
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, tt.activeflowID, tt.flowID, fmactiveflow.ReferenceTypeCall, tt.id).Return(tt.responseActiveflow, nil)

			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUIDChannel)
			mockDB.EXPECT().CallCreate(ctx, tt.expectCall).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.expectCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectCall.CustomerID, call.EventTypeCallCreated, tt.expectCall)

			// setVariables
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			if tt.masterCallID != uuid.Nil {
				mockDB.EXPECT().CallTXStart(tt.masterCallID).Return(nil, &call.Call{}, nil)
				mockDB.EXPECT().CallTXAddChainedCallID(gomock.Any(), tt.masterCallID, tt.expectCall.ID).Return(nil)
				mockDB.EXPECT().CallSetMasterCallID(ctx, tt.expectCall.ID, tt.masterCallID).Return(nil)
				mockDB.EXPECT().CallTXFinish(gomock.Any(), true)

				mockDB.EXPECT().CallGet(ctx, tt.masterCallID).Return(&call.Call{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallUpdated, gomock.Any())

				mockDB.EXPECT().CallGet(ctx, tt.expectCall.ID).Return(&call.Call{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallUpdated, gomock.Any())
			}

			mockChannel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, gomock.Any(), fmt.Sprintf("context=%s,call_id=%s", common.ContextOutgoingCall, tt.id), tt.expectEndpointDst, "", "", "", tt.expectVariables).Return(&channel.Channel{}, nil)

			res, err := h.CreateCallOutgoing(ctx, tt.id, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, tt.source, tt.destination, tt.earlyExecution, tt.connect)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectCall) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

func Test_CreateCallOutgoing_TypeTel(t *testing.T) {

	tests := []struct {
		name string

		id             uuid.UUID
		customerID     uuid.UUID
		flowID         uuid.UUID
		activeflowID   uuid.UUID
		masterCallID   uuid.UUID
		source         commonaddress.Address
		destination    commonaddress.Address
		earlyExecution bool
		connect        bool

		responseActiveflow  *fmactiveflow.Activeflow
		responseRoutes      []rmroute.Route
		responseUUIDChannel uuid.UUID
		responseProvider    *rmprovider.Provider

		expectDialrouteTarget string
		expectCall            *call.Call
		expectProviderID      uuid.UUID
		expectEndpointDst     string
		expectVariables       map[string]string
	}{
		{
			"have all",

			uuid.FromStringOrNil("b7c40962-07fb-11eb-bb82-a3bd16bf1bd9"),
			uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
			uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
			uuid.FromStringOrNil("11e2bbc8-a181-4ca1-97f7-4e382f128cf6"),
			uuid.FromStringOrNil("61c0fe66-8c8f-11ec-873a-ff90a846a02f"),
			commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+99999888",
				TargetName: "test",
			},
			commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+821121656521",
				TargetName: "test target",
			},
			true,
			true,

			&fmactiveflow.Activeflow{
				ID: uuid.FromStringOrNil("11e2bbc8-a181-4ca1-97f7-4e382f128cf6"),
				CurrentAction: fmaction.Action{
					ID: fmaction.IDStart,
				},
			},
			[]rmroute.Route{
				{
					ID:         uuid.FromStringOrNil("f86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
					ProviderID: uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
				},
			},
			uuid.FromStringOrNil("d948969e-5de3-11ed-94f5-137ec429b6b6"),
			&rmprovider.Provider{
				Hostname: "sip.telnyx.com",
			},

			"+82",
			&call.Call{
				ID:             uuid.FromStringOrNil("b7c40962-07fb-11eb-bb82-a3bd16bf1bd9"),
				CustomerID:     uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
				ChannelID:      "d948969e-5de3-11ed-94f5-137ec429b6b6",
				FlowID:         uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
				ActiveFlowID:   uuid.FromStringOrNil("11e2bbc8-a181-4ca1-97f7-4e382f128cf6"),
				Type:           call.TypeFlow,
				ChainedCallIDs: []uuid.UUID{},
				RecordingIDs:   []uuid.UUID{},
				Status:         call.StatusDialing,
				Data: map[call.DataType]string{
					call.DataTypeEarlyExecution:            "true",
					call.DataTypeExecuteNextMasterOnHangup: "true",
				},
				Direction: call.DirectionOutgoing,
				Source: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+99999888",
					TargetName: "test",
				},
				Destination: commonaddress.Address{
					Type:       commonaddress.TypeTel,
					Target:     "+821121656521",
					TargetName: "test target",
				},
				Action: fmaction.Action{
					ID: fmaction.IDStart,
				},
				DialrouteID: uuid.FromStringOrNil("f86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("f86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
						ProviderID: uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
					},
				},

				TMCreate:      "2021-02-19 06:32:14.621",
				TMUpdate:      dbhandler.DefaultTimeStamp,
				TMRinging:     dbhandler.DefaultTimeStamp,
				TMProgressing: dbhandler.DefaultTimeStamp,
				TMHangup:      dbhandler.DefaultTimeStamp,
			},
			uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
			"pjsip/call-out/sip:+821121656521@sip.telnyx.com;transport=udp",
			map[string]string{
				"CALLERID(name)":                        "test",
				"CALLERID(num)":                         "+99999888",
				"PJSIP_HEADER(add,VBOUT-SDP_Transport)": "RTP/AVP",
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, tt.activeflowID, tt.flowID, fmactiveflow.ReferenceTypeCall, tt.id).Return(tt.responseActiveflow, nil)
			// getDialURI
			mockReq.EXPECT().RouteV1DialrouteGets(ctx, tt.expectCall.CustomerID, tt.expectDialrouteTarget).Return(tt.responseRoutes, nil)

			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUIDChannel)
			mockDB.EXPECT().CallCreate(ctx, tt.expectCall).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.expectCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectCall.CustomerID, call.EventTypeCallCreated, tt.expectCall)

			// setVariables
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

			if tt.masterCallID != uuid.Nil {
				mockDB.EXPECT().CallTXStart(tt.masterCallID).Return(nil, &call.Call{}, nil)
				mockDB.EXPECT().CallTXAddChainedCallID(gomock.Any(), tt.masterCallID, tt.expectCall.ID).Return(nil)
				mockDB.EXPECT().CallSetMasterCallID(ctx, tt.expectCall.ID, tt.masterCallID).Return(nil)
				mockDB.EXPECT().CallTXFinish(gomock.Any(), true)

				mockDB.EXPECT().CallGet(ctx, tt.masterCallID).Return(&call.Call{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallUpdated, gomock.Any())

				mockDB.EXPECT().CallGet(ctx, tt.expectCall.ID).Return(&call.Call{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallUpdated, gomock.Any())
			}

			mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.expectProviderID).Return(tt.responseProvider, nil)

			mockChannel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, gomock.Any(), fmt.Sprintf("context=%s,call_id=%s", common.ContextOutgoingCall, tt.id), tt.expectEndpointDst, "", "", "", tt.expectVariables).Return(&channel.Channel{}, nil)

			res, err := h.CreateCallOutgoing(ctx, tt.id, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, tt.source, tt.destination, tt.earlyExecution, tt.connect)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectCall) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

func Test_getDialURI_Tel(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		responseProvider *rmprovider.Provider

		expectProviderID uuid.UUID
		expectTarget     string
		expectRes        string
	}{
		{
			"normal",

			&call.Call{
				ID:         uuid.FromStringOrNil("04e5d530-5d96-11ed-bbc8-cfb95f6d6085"),
				CustomerID: uuid.FromStringOrNil("f7a14b8c-534c-11ed-9fb1-c7c376f2730b"),
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821121656521",
				},
				DialrouteID: uuid.FromStringOrNil("ae237dd0-5db1-11ed-97c4-a7ff6f170fbb"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("ae237dd0-5db1-11ed-97c4-a7ff6f170fbb"),
						ProviderID: uuid.FromStringOrNil("8730a3da-5350-11ed-aa47-7f44741127c1"),
					},
				},
			},

			&rmprovider.Provider{
				ID:       uuid.FromStringOrNil("ae237dd0-5db1-11ed-97c4-a7ff6f170fbb"),
				Hostname: "sip.telnyx.com",
			},

			uuid.FromStringOrNil("8730a3da-5350-11ed-aa47-7f44741127c1"),
			"+82",
			"pjsip/call-out/sip:+821121656521@sip.telnyx.com;transport=udp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.expectProviderID).Return(tt.responseProvider, nil)

			res, err := h.getDialURI(ctx, tt.call)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_getDialURI_SIP(t *testing.T) {

	tests := []struct {
		name string

		c *call.Call

		expectRes string
	}{
		{
			"normal",

			&call.Call{
				ID:         uuid.FromStringOrNil("3c28f5fe-5d96-11ed-bf69-9340492cc88d"),
				CustomerID: uuid.FromStringOrNil("22139104-534d-11ed-aba9-e73d8b8e1c43"),
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "test@test.com",
				},
			},

			"pjsip/call-out/sip:test@test.com",
		},
		{
			"normal",

			&call.Call{
				ID:         uuid.FromStringOrNil("5e0329d8-5d96-11ed-a009-2763e323daa8"),
				CustomerID: uuid.FromStringOrNil("dfd086b2-534c-11ed-b905-93a3b56e1ae8"),
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "test@test.com",
				},
			},

			"pjsip/call-out/sip:test@test.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			res, err := h.getDialURI(ctx, tt.c)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_getDialURI_Endpoint(t *testing.T) {

	tests := []struct {
		name string

		call             *call.Call
		responseContacts []*astcontact.AstContact
		expectRes        string
	}{
		{
			"normal",

			&call.Call{
				ID:         uuid.FromStringOrNil("7e0a846a-5d96-11ed-9005-07794a4f93cb"),
				CustomerID: uuid.FromStringOrNil("6f3fd136-534d-11ed-90a2-ff71219800e5"),
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeEndpoint,
					Target: "test@test.sip.voipbin.net",
				},
			},
			[]*astcontact.AstContact{
				{
					ID:                  "test11@test.sip.voipbin.net^3B@c21de7824c22185a665983170d7028b0",
					URI:                 "sip:test11@211.178.226.108:35551^3Btransport=UDP^3Brinstance=8a1f981a77f30a22",
					ExpirationTime:      1613498199,
					QualifyFrequency:    0,
					OutboundProxy:       "",
					Path:                "",
					UserAgent:           "Z 5.4.9 rv2.10.11.7-mod",
					QualifyTimeout:      3,
					RegServer:           "asterisk-registrar-b46bf4b67-j5rxz",
					AuthenticateQualify: "no",
					ViaAddr:             "192.168.0.20",
					ViaPort:             35551,
					CallID:              "mX4vXXxJZ_gS4QpMapYfwA..",
					Endpoint:            "test@test.sip.voipbin.net",
					PruneOnBoot:         "no",
				},
			},
			"pjsip/call-out/sip:test11@211.178.226.108:35551;transport=UDP;rinstance=8a1f981a77f30a22",
		},
		{
			"2 contacts",

			&call.Call{
				ID:         uuid.FromStringOrNil("2078cd98-5daf-11ed-a2f9-c7b6dae6e3ff"),
				CustomerID: uuid.FromStringOrNil("791a3da4-534d-11ed-9f3a-c3d05994dec2"),
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeEndpoint,
					Target: "test@test.sip.voipbin.net",
				},
			},
			[]*astcontact.AstContact{
				{
					ID:                  "test11@test.sip.voipbin.net^3B@c21de7824c22185a665983170d7028b0",
					URI:                 "sip:test11@211.178.226.108:35551^3Btransport=UDP^3Brinstance=8a1f981a77f30a22",
					ExpirationTime:      1613498199,
					QualifyFrequency:    0,
					OutboundProxy:       "",
					Path:                "",
					UserAgent:           "Z 5.4.9 rv2.10.11.7-mod",
					QualifyTimeout:      3,
					RegServer:           "asterisk-registrar-b46bf4b67-j5rxz",
					AuthenticateQualify: "no",
					ViaAddr:             "192.168.0.20",
					ViaPort:             35551,
					CallID:              "mX4vXXxJZ_gS4QpMapYfwA..",
					Endpoint:            "test@test.sip.voipbin.net",
					PruneOnBoot:         "no",
				},
				{
					ID:                  "test11@test.sip.voipbin.net^3B@c21de7824c22185a665983170d7028b1",
					URI:                 "sip:test11@211.178.226.120:35551^3Btransport=UDP^3Brinstance=8a1f981a77f30a22",
					ExpirationTime:      1613498199,
					QualifyFrequency:    0,
					OutboundProxy:       "",
					Path:                "",
					UserAgent:           "Z 5.4.9 rv2.10.11.7-mod",
					QualifyTimeout:      3,
					RegServer:           "asterisk-registrar-b46bf4b67-j5rxz",
					AuthenticateQualify: "no",
					ViaAddr:             "192.168.0.20",
					ViaPort:             35551,
					CallID:              "mX4vXXxJZ_gS4QpMapYfwA..",
					Endpoint:            "test@test.sip.voipbin.net",
					PruneOnBoot:         "no",
				},
			},
			"pjsip/call-out/sip:test11@211.178.226.108:35551;transport=UDP;rinstance=8a1f981a77f30a22",
		},
		{
			"transport ws",

			&call.Call{
				ID:         uuid.FromStringOrNil("20ac0910-5daf-11ed-994f-27b46cd2e1b8"),
				CustomerID: uuid.FromStringOrNil("81ea0ff4-534d-11ed-af9b-8bfc8edf8627"),
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeEndpoint,
					Target: "test@test.sip.voipbin.net",
				},
			},
			[]*astcontact.AstContact{
				{
					ID:                  "test11@test.sip.voipbin.net^3B@c21de7824c22185a665983170d7028b0",
					URI:                 "sip:test11@211.178.226.108:35551^3Btransport=ws^3Brinstance=8a1f981a77f30a22",
					ExpirationTime:      1613498199,
					QualifyFrequency:    0,
					OutboundProxy:       "",
					Path:                "",
					UserAgent:           "Z 5.4.9 rv2.10.11.7-mod",
					QualifyTimeout:      3,
					RegServer:           "asterisk-registrar-b46bf4b67-j5rxz",
					AuthenticateQualify: "no",
					ViaAddr:             "192.168.0.20",
					ViaPort:             35551,
					CallID:              "mX4vXXxJZ_gS4QpMapYfwA..",
					Endpoint:            "test@test.sip.voipbin.net",
					PruneOnBoot:         "no",
				},
			},
			"pjsip/call-out/sip:test11@211.178.226.108:35551;transport=ws;rinstance=8a1f981a77f30a22",
		},
		{
			"transport wss",

			&call.Call{
				ID:         uuid.FromStringOrNil("e8f15bd6-5db0-11ed-b4e7-5b1307ce9ce9"),
				CustomerID: uuid.FromStringOrNil("89b5324a-534d-11ed-a9da-c3461944cf00"),
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeEndpoint,
					Target: "test@test.sip.voipbin.net",
				},
			},
			[]*astcontact.AstContact{
				{
					ID:                  "test11@test.sip.voipbin.net^3B@c21de7824c22185a665983170d7028b0",
					URI:                 "sip:test11@211.178.226.108:35551^3Btransport=wss^3Brinstance=8a1f981a77f30a22",
					ExpirationTime:      1613498199,
					QualifyFrequency:    0,
					OutboundProxy:       "",
					Path:                "",
					UserAgent:           "Z 5.4.9 rv2.10.11.7-mod",
					QualifyTimeout:      3,
					RegServer:           "asterisk-registrar-b46bf4b67-j5rxz",
					AuthenticateQualify: "no",
					ViaAddr:             "192.168.0.20",
					ViaPort:             35551,
					CallID:              "mX4vXXxJZ_gS4QpMapYfwA..",
					Endpoint:            "test@test.sip.voipbin.net",
					PruneOnBoot:         "no",
				},
			},
			"pjsip/call-out/sip:test11@211.178.226.108:35551;transport=wss;rinstance=8a1f981a77f30a22",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1ContactGets(ctx, tt.call.Destination.Target).Return(tt.responseContacts, nil)

			res, err := h.getDialURI(ctx, tt.call)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectRes {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}

func Test_getDialURIError(t *testing.T) {

	tests := []struct {
		name string

		call             *call.Call
		responseContacts []*astcontact.AstContact
	}{
		{
			"no contact",

			&call.Call{
				ID:         uuid.FromStringOrNil("10d6da04-5db1-11ed-ada1-53cfbee7570c"),
				CustomerID: uuid.FromStringOrNil("9c1d4850-534d-11ed-87aa-bb08e4fa1db5"),
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeEndpoint,
					Target: "test@test.sip.voipbin.net",
				},
			},
			[]*astcontact.AstContact{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				reqHandler: mockReq,
				db:         mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RegistrarV1ContactGets(ctx, tt.call.Destination.Target).Return(tt.responseContacts, nil)

			_, err := h.getDialURI(ctx, tt.call)
			if err == nil {
				t.Error("Wrong match. expect: err, got: ok")
			}
		})
	}
}

func Test_createChannel(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		responseProvider *rmprovider.Provider

		expectProviderID uuid.UUID
		expectArgs       string
		expectDialURI    string
		expectVariables  map[string]string
	}{
		{
			"normal",

			&call.Call{
				ID:         uuid.FromStringOrNil("7e0a846a-5d96-11ed-9005-07794a4f93cb"),
				CustomerID: uuid.FromStringOrNil("6f3fd136-534d-11ed-90a2-ff71219800e5"),

				ChannelID: "7fd0947a-5f39-11ed-8ce5-7b34330a9d7c",
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DialrouteID: uuid.FromStringOrNil("f86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("f86d48aa-5de6-11ed-a69e-9f3df36c7aa8"),
						ProviderID: uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
					},
				},
			},

			&rmprovider.Provider{
				ID:       uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
				Hostname: "test.com",
			},

			uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
			"context=call-out,call_id=7e0a846a-5d96-11ed-9005-07794a4f93cb",
			"pjsip/call-out/sip:+821100000001@test.com;transport=udp",
			map[string]string{
				"CALLERID(name)":                        "",
				"CALLERID(num)":                         "+821100000002",
				"PJSIP_HEADER(add,VBOUT-SDP_Transport)": "RTP/AVP",
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
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				db:             mockDB,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.expectProviderID).Return(tt.responseProvider, nil)
			mockChannel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, tt.call.ChannelID, tt.expectArgs, tt.expectDialURI, "", "", "", tt.expectVariables).Return(&channel.Channel{}, nil)

			if err := h.createChannel(ctx, tt.call); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_createFailoverChannel(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		responseUUIDChannel uuid.UUID
		responseCall        *call.Call
		responseProvider    *rmprovider.Provider

		expectDialrouteID uuid.UUID
		expectProviderID  uuid.UUID
		expectArgs        string
		expectDialURI     string
		expectVariables   map[string]string

		expectRes *call.Call
	}{
		{
			"normal",

			&call.Call{
				ID:         uuid.FromStringOrNil("25c7a29a-5f7d-11ed-86cc-bb999f3cccaf"),
				CustomerID: uuid.FromStringOrNil("260b56c0-5f7d-11ed-8930-cbd2b7cb46ff"),

				ChannelID: "263bc88c-5f7d-11ed-a26c-735426e79b33",
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DialrouteID: uuid.FromStringOrNil("266caba0-5f7d-11ed-b8bc-c3e291d85720"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("266caba0-5f7d-11ed-b8bc-c3e291d85720"),
						ProviderID: uuid.FromStringOrNil("269dad7c-5f7d-11ed-aa09-c30aa616e3ea"),
					},
					{
						ID:         uuid.FromStringOrNil("c403ec52-5f7d-11ed-9b6f-5b9ada249a57"),
						ProviderID: uuid.FromStringOrNil("cc3d77a8-5f7d-11ed-9232-03d402cb4d34"),
					},
				},
			},

			uuid.FromStringOrNil("2902a512-5f7e-11ed-8cb0-ef43ac5ddea8"),
			&call.Call{
				ID:         uuid.FromStringOrNil("25c7a29a-5f7d-11ed-86cc-bb999f3cccaf"),
				CustomerID: uuid.FromStringOrNil("260b56c0-5f7d-11ed-8930-cbd2b7cb46ff"),

				ChannelID: "2902a512-5f7e-11ed-8cb0-ef43ac5ddea8",
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DialrouteID: uuid.FromStringOrNil("c403ec52-5f7d-11ed-9b6f-5b9ada249a57"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("266caba0-5f7d-11ed-b8bc-c3e291d85720"),
						ProviderID: uuid.FromStringOrNil("269dad7c-5f7d-11ed-aa09-c30aa616e3ea"),
					},
					{
						ID:         uuid.FromStringOrNil("c403ec52-5f7d-11ed-9b6f-5b9ada249a57"),
						ProviderID: uuid.FromStringOrNil("cc3d77a8-5f7d-11ed-9232-03d402cb4d34"),
					},
				},
			},
			&rmprovider.Provider{
				ID:       uuid.FromStringOrNil("cc3d77a8-5f7d-11ed-9232-03d402cb4d34"),
				Hostname: "test.com",
			},

			uuid.FromStringOrNil("c403ec52-5f7d-11ed-9b6f-5b9ada249a57"),
			uuid.FromStringOrNil("cc3d77a8-5f7d-11ed-9232-03d402cb4d34"),
			"context=call-out,call_id=25c7a29a-5f7d-11ed-86cc-bb999f3cccaf",
			"pjsip/call-out/sip:+821100000001@test.com;transport=udp",
			map[string]string{
				"CALLERID(name)":                        "",
				"CALLERID(num)":                         "+821100000002",
				"PJSIP_HEADER(add,VBOUT-SDP_Transport)": "RTP/AVP",
			},

			&call.Call{
				ID:         uuid.FromStringOrNil("25c7a29a-5f7d-11ed-86cc-bb999f3cccaf"),
				CustomerID: uuid.FromStringOrNil("260b56c0-5f7d-11ed-8930-cbd2b7cb46ff"),

				ChannelID: "2902a512-5f7e-11ed-8cb0-ef43ac5ddea8",
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000002",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				DialrouteID: uuid.FromStringOrNil("c403ec52-5f7d-11ed-9b6f-5b9ada249a57"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("266caba0-5f7d-11ed-b8bc-c3e291d85720"),
						ProviderID: uuid.FromStringOrNil("269dad7c-5f7d-11ed-aa09-c30aa616e3ea"),
					},
					{
						ID:         uuid.FromStringOrNil("c403ec52-5f7d-11ed-9b6f-5b9ada249a57"),
						ProviderID: uuid.FromStringOrNil("cc3d77a8-5f7d-11ed-9232-03d402cb4d34"),
					},
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChannel := channelhandler.NewMockChannelHandler(mc)

			h := &callHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				channelHandler: mockChannel,
			}

			ctx := context.Background()

			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUIDChannel)

			// updateForRouteFailover
			mockDB.EXPECT().CallSetForRouteFailover(ctx, tt.call.ID, tt.responseUUIDChannel.String(), tt.expectDialrouteID).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.call.ID).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallUpdated, tt.responseCall)

			mockReq.EXPECT().RouteV1ProviderGet(ctx, tt.expectProviderID).Return(tt.responseProvider, nil)
			mockChannel.EXPECT().StartChannel(ctx, requesthandler.AsteriskIDCall, tt.responseCall.ChannelID, tt.expectArgs, tt.expectDialURI, "", "", "", tt.expectVariables).Return(&channel.Channel{}, nil)

			res, err := h.createFailoverChannel(ctx, tt.call)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getNextDialroute(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		expectRes *rmroute.Route
	}{
		{
			"normal",

			&call.Call{
				DialrouteID: uuid.FromStringOrNil("d1112a98-5f7f-11ed-944b-3f78d26bca39"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("d1112a98-5f7f-11ed-944b-3f78d26bca39"),
						ProviderID: uuid.FromStringOrNil("d14afce6-5f7f-11ed-baff-6b5dd84ca88b"),
					},
					{
						ID:         uuid.FromStringOrNil("d1798e58-5f7f-11ed-bb82-770f515db7bb"),
						ProviderID: uuid.FromStringOrNil("d1adc1be-5f7f-11ed-860e-efd0fa6f95c5"),
					},
				},
			},

			&rmroute.Route{
				ID:         uuid.FromStringOrNil("d1798e58-5f7f-11ed-bb82-770f515db7bb"),
				ProviderID: uuid.FromStringOrNil("d1adc1be-5f7f-11ed-860e-efd0fa6f95c5"),
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

			h := &callHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			res, err := h.getNextDialroute(ctx, tt.call)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getNextDialroute_error(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call
	}{
		{
			"empty routes",

			&call.Call{
				DialrouteID: uuid.Nil,
				Dialroutes:  []rmroute.Route{},
			},
		},
		{
			"no more dialroutes left",

			&call.Call{
				DialrouteID: uuid.FromStringOrNil("8bd1aba0-5f80-11ed-84f3-13ad177e24a7"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("9c86e082-5f80-11ed-8345-2b30b7c6d70e"),
						ProviderID: uuid.FromStringOrNil("9cad5a1e-5f80-11ed-87a1-73640a77ce52"),
					},
					{
						ID:         uuid.FromStringOrNil("8bd1aba0-5f80-11ed-84f3-13ad177e24a7"),
						ProviderID: uuid.FromStringOrNil("9cf9531a-5f80-11ed-b3d4-5bc4b0bec0c2"),
					},
				},
			},
		},
		{
			"dialroute id does not exist in the dialroutes",

			&call.Call{
				DialrouteID: uuid.FromStringOrNil("b86c9170-5f80-11ed-9429-ff3d42d871e9"),
				Dialroutes: []rmroute.Route{
					{
						ID:         uuid.FromStringOrNil("b8941d9e-5f80-11ed-ba54-f3c6180e2331"),
						ProviderID: uuid.FromStringOrNil("b8bc7244-5f80-11ed-b9c9-cb656548f971"),
					},
					{
						ID:         uuid.FromStringOrNil("b8e66464-5f80-11ed-be77-dffb61c583e3"),
						ProviderID: uuid.FromStringOrNil("b911a7be-5f80-11ed-a251-ef2d84919cb7"),
					},
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			_, err := h.getNextDialroute(ctx, tt.call)
			if err == nil {
				t.Error("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func Test_getDialroutes(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		destination *commonaddress.Address

		responseDialroutes []rmroute.Route

		expectTarget string

		expectRes []rmroute.Route
	}{
		{
			"normal",

			uuid.FromStringOrNil("551562fe-5f81-11ed-b9b3-535fe8d67b80"),
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},

			[]rmroute.Route{
				{
					ID: uuid.FromStringOrNil("b8d6da7a-5f81-11ed-9274-9313db0184ad"),
				},
			},

			"+82",

			[]rmroute.Route{
				{
					ID: uuid.FromStringOrNil("b8d6da7a-5f81-11ed-9274-9313db0184ad"),
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
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().RouteV1DialrouteGets(ctx, tt.customerID, tt.expectTarget).Return(tt.responseDialroutes, nil)

			res, err := h.getDialroutes(ctx, tt.customerID, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getVariablesCallerID(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		expectRes map[string]string
	}{
		{
			"destination type tel and source target is anonymous",

			&call.Call{
				Source: commonaddress.Address{
					Target: "anonymous",
				},
				Destination: commonaddress.Address{
					Type: commonaddress.TypeTel,
				},
			},

			map[string]string{
				"CALLERID(pres)":                        "prohib",
				"PJSIP_HEADER(add,P-Asserted-Identity)": "\"Anonymous\" <sip:+821100000001@pstn.voipbin.net>",
				"PJSIP_HEADER(add,Privacy)":             "id",
			},
		},
		{
			"destination type is sip",

			&call.Call{
				Source: commonaddress.Address{
					Type:   commonaddress.TypeTel,
					Target: "+821100000001",
				},
				Destination: commonaddress.Address{
					Type:   commonaddress.TypeSIP,
					Target: "sip:test@test.sip.voipbin.net",
				},
			},

			map[string]string{
				"CALLERID(name)": "",
				"CALLERID(num)":  "+821100000001",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := map[string]string{}
			setChannelVariablesCallerID(res, tt.call)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_getSourceForOutgoingCall(t *testing.T) {

	tests := []struct {
		name string

		source      *commonaddress.Address
		destination *commonaddress.Address

		expectRes *commonaddress.Address
	}{
		{
			"destination type tel and source target is anonymous",

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000001",
			},
		},
		{
			"destination type is tel but source has + prefix",

			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "821100000001",
			},
			&commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+821100000002",
			},

			&commonaddress.Address{
				Type:       commonaddress.TypeTel,
				TargetName: "Anonymous",
				Target:     "anonymous",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := getSourceForOutgoingCall(tt.source, tt.destination)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
