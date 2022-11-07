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
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	fmactiveflow "gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	rmprovider "gitlab.com/voipbin/bin-manager/route-manager.git/models/provider"
	rmroute "gitlab.com/voipbin/bin-manager/route-manager.git/models/route"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/util"
)

// func Test_CreateCallOutgoing(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		id           uuid.UUID
// 		customerID   uuid.UUID
// 		flowID       uuid.UUID
// 		activeflowID uuid.UUID
// 		masterCallID uuid.UUID
// 		source       commonaddress.Address
// 		destination  commonaddress.Address

// 		responseActiveflow  *fmactiveflow.Activeflow
// 		responseRoutes      []rmroute.Route
// 		responseUUIDChannel uuid.UUID
// 		responseCurTime     string
// 		responseProvider    *rmprovider.Provider

// 		expectDialrouteTarget string
// 		expectCall            *call.Call
// 		expectEndpointDst     string
// 		expectVariables       map[string]string
// 	}{
// 		// {
// 		// 	"normal",

// 		// 	uuid.FromStringOrNil("f1afa9ce-ecb2-11ea-ab94-a768ab787da0"),
// 		// 	uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
// 		// 	uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
// 		// 	uuid.FromStringOrNil("679f0eb2-8c21-41a6-876d-9d778b1b0167"),
// 		// 	uuid.FromStringOrNil("5935ff8a-8c8f-11ec-b26a-3fee169eaf45"),

// 		// 	commonaddress.Address{
// 		// 		Type:       commonaddress.TypeSIP,
// 		// 		Target:     "testsrc@test.com",
// 		// 		TargetName: "test",
// 		// 	},
// 		// 	commonaddress.Address{
// 		// 		Type:       commonaddress.TypeSIP,
// 		// 		Target:     "testoutgoing@test.com",
// 		// 		TargetName: "test target",
// 		// 	},

// 		// 	&fmactiveflow.Activeflow{
// 		// 		CurrentAction: fmaction.Action{
// 		// 			ID: fmaction.IDStart,
// 		// 		},
// 		// 	},
// 		// 	[]rmroute.Route{},
// 		// 	nil,

// 		// 	"",
// 		// 	&call.Call{
// 		// 		ID:         uuid.FromStringOrNil("f1afa9ce-ecb2-11ea-ab94-a768ab787da0"),
// 		// 		CustomerID: uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
// 		// 		ChannelID:  call.TestChannelID,
// 		// 		FlowID:     uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
// 		// 		Type:       call.TypeFlow,
// 		// 		Status:     call.StatusDialing,
// 		// 		Direction:  call.DirectionOutgoing,
// 		// 		Source: commonaddress.Address{
// 		// 			Type:       commonaddress.TypeSIP,
// 		// 			Target:     "testsrc@test.com",
// 		// 			TargetName: "test",
// 		// 		},
// 		// 		Destination: commonaddress.Address{
// 		// 			Type:       commonaddress.TypeSIP,
// 		// 			Target:     "testoutgoing@test.com",
// 		// 			TargetName: "test target",
// 		// 		},
// 		// 		Action: fmaction.Action{
// 		// 			ID: fmaction.IDStart,
// 		// 		},
// 		// 	},
// 		// 	"pjsip/call-out/sip:testoutgoing@test.com",
// 		// 	map[string]string{
// 		// 		"CALLERID(all)":                         `"test" <sip:testsrc@test.com>`,
// 		// 		"PJSIP_HEADER(add,VBOUT-SDP_Transport)": "RTP/AVP",
// 		// 	},
// 		// },
// 		{
// 			"tel type destination",

// 			uuid.FromStringOrNil("b7c40962-07fb-11eb-bb82-a3bd16bf1bd9"),
// 			uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
// 			uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
// 			uuid.FromStringOrNil("11e2bbc8-a181-4ca1-97f7-4e382f128cf6"),
// 			uuid.FromStringOrNil("61c0fe66-8c8f-11ec-873a-ff90a846a02f"),

// 			commonaddress.Address{
// 				Type:       commonaddress.TypeTel,
// 				Target:     "+99999888",
// 				TargetName: "test",
// 			},
// 			commonaddress.Address{
// 				Type:       commonaddress.TypeTel,
// 				Target:     "+821121656521",
// 				TargetName: "test target",
// 			},

// 			&fmactiveflow.Activeflow{
// 				CurrentAction: fmaction.Action{
// 					ID: fmaction.IDStart,
// 				},
// 			},
// 			[]rmroute.Route{
// 				{
// 					ProviderID: uuid.FromStringOrNil("c213af44-534e-11ed-9a1d-73b0076723b8"),
// 				},
// 			},
// 			uuid.FromStringOrNil("d948969e-5de3-11ed-94f5-137ec429b6b6"),
// 			"2021-02-19 06:32:14.621",
// 			&rmprovider.Provider{
// 				Hostname: "sip.telnyx.com",
// 			},

// 			"+82",
// 			&call.Call{
// 				ID:           uuid.FromStringOrNil("b7c40962-07fb-11eb-bb82-a3bd16bf1bd9"),
// 				CustomerID:   uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
// 				ChannelID:    "d948969e-5de3-11ed-94f5-137ec429b6b6",
// 				FlowID:       uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
// 				ActiveFlowID: uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
// 				Type:         call.TypeFlow,
// 				Status:       call.StatusDialing,
// 				Direction:    call.DirectionOutgoing,
// 				Source: commonaddress.Address{
// 					Type:       commonaddress.TypeTel,
// 					Target:     "+99999888",
// 					TargetName: "test",
// 				},
// 				Destination: commonaddress.Address{
// 					Type:       commonaddress.TypeTel,
// 					Target:     "+821121656521",
// 					TargetName: "test target",
// 				},
// 				Action: fmaction.Action{
// 					ID: fmaction.IDStart,
// 				},
// 			},
// 			"pjsip/call-out/sip:+821121656521@sip.telnyx.com;transport=udp",
// 			map[string]string{
// 				"CALLERID(all)":                         "+99999888",
// 				"PJSIP_HEADER(add,VBOUT-SDP_Transport)": "RTP/AVP",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := util.NewMockUtil(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)

// 			h := &callHandler{
// 				util:          mockUtil,
// 				reqHandler:    mockReq,
// 				notifyHandler: mockNotify,
// 				db:            mockDB,
// 			}

// 			ctx := context.Background()

// 			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, tt.activeflowID, tt.flowID, fmactiveflow.ReferenceTypeCall, tt.id).Return(tt.responseActiveflow, nil)
// 			// getDialURI
// 			if tt.destination.Type == commonaddress.TypeTel {
// 				mockReq.EXPECT().RouteV1DialrouteGets(ctx, tt.expectCall.CustomerID, tt.expectDialrouteTarget).Return(tt.responseRoutes, nil)
// 			}

// 			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUIDChannel)
// 			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
// 			mockDB.EXPECT().CallCreate(ctx, tt.expectCall).Return(nil)
// 			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.expectCall, nil)
// 			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectCall.CustomerID, call.EventTypeCallCreated, tt.expectCall)

// 			// setVariables
// 			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

// 			if tt.masterCallID != uuid.Nil {
// 				mockDB.EXPECT().CallTXStart(tt.masterCallID).Return(nil, &call.Call{}, nil)
// 				mockDB.EXPECT().CallTXAddChainedCallID(gomock.Any(), tt.masterCallID, tt.expectCall.ID).Return(nil)
// 				mockDB.EXPECT().CallSetMasterCallID(ctx, tt.expectCall.ID, tt.masterCallID).Return(nil)
// 				mockDB.EXPECT().CallTXFinish(gomock.Any(), true)

// 				mockDB.EXPECT().CallGet(ctx, tt.masterCallID).Return(&call.Call{}, nil)
// 				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallUpdated, gomock.Any())

// 				mockDB.EXPECT().CallGet(ctx, tt.expectCall.ID).Return(&call.Call{}, nil)
// 				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), call.EventTypeCallUpdated, gomock.Any())
// 			}

// 			mockReq.EXPECT().AstChannelCreate(ctx, requesthandler.AsteriskIDCall, gomock.Any(), fmt.Sprintf("context=%s,call_id=%s", ContextOutgoingCall, tt.id), tt.expectEndpointDst, "", "", "", tt.expectVariables).Return(nil)

// 			res, err := h.CreateCallOutgoing(ctx, tt.id, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, tt.source, tt.destination)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}

// 			if !reflect.DeepEqual(res, tt.expectCall) {
// 				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
// 			}
// 		})
// 	}
// }

func Test_CreateCallOutgoing_TypeTel(t *testing.T) {

	tests := []struct {
		name string

		id           uuid.UUID
		customerID   uuid.UUID
		flowID       uuid.UUID
		activeflowID uuid.UUID
		masterCallID uuid.UUID
		source       commonaddress.Address
		destination  commonaddress.Address

		responseActiveflow  *fmactiveflow.Activeflow
		responseRoutes      []rmroute.Route
		responseUUIDChannel uuid.UUID
		responseCurTime     string
		responseProvider    *rmprovider.Provider

		expectDialrouteTarget string
		expectCall            *call.Call
		expectProviderID      uuid.UUID
		expectEndpointDst     string
		expectVariables       map[string]string
	}{
		{
			"tel type destination",

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
			"2021-02-19 06:32:14.621",
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
				Data:           map[string]string{},
				Direction:      call.DirectionOutgoing,
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
				"CALLERID(all)":                         "+99999888",
				"PJSIP_HEADER(add,VBOUT-SDP_Transport)": "RTP/AVP",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := util.NewMockUtil(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &callHandler{
				util:          mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().FlowV1ActiveflowCreate(ctx, tt.activeflowID, tt.flowID, fmactiveflow.ReferenceTypeCall, tt.id).Return(tt.responseActiveflow, nil)
			// getDialURI
			if tt.destination.Type == commonaddress.TypeTel {
				mockReq.EXPECT().RouteV1DialrouteGets(ctx, tt.expectCall.CustomerID, tt.expectDialrouteTarget).Return(tt.responseRoutes, nil)
			}

			mockUtil.EXPECT().CreateUUID().Return(tt.responseUUIDChannel)
			mockUtil.EXPECT().GetCurTime().Return(tt.responseCurTime)
			mockDB.EXPECT().CallCreate(ctx, tt.expectCall).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.expectCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectCall.CustomerID, call.EventTypeCallCreated, tt.expectCall)

			// setVariables
			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

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

			mockReq.EXPECT().AstChannelCreate(ctx, requesthandler.AsteriskIDCall, gomock.Any(), fmt.Sprintf("context=%s,call_id=%s", ContextOutgoingCall, tt.id), tt.expectEndpointDst, "", "", "", tt.expectVariables).Return(nil)

			res, err := h.CreateCallOutgoing(ctx, tt.id, tt.customerID, tt.flowID, tt.activeflowID, tt.masterCallID, tt.source, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectCall) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

func Test_getDialURITel(t *testing.T) {

	tests := []struct {
		name string

		call *call.Call

		// responseRoutes   []rmroute.Route
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

func Test_getDialURISIP(t *testing.T) {

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

func Test_getDialURIEndpoint(t *testing.T) {

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
