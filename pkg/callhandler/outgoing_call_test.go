package callhandler

import (
	"context"
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
)

func TestCreateCallOutgoing(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
	}

	type test struct {
		name         string
		id           uuid.UUID
		customerID   uuid.UUID
		flowID       uuid.UUID
		masterCallID uuid.UUID
		source       address.Address
		destination  address.Address

		af                *activeflow.ActiveFlow
		expectCall        *call.Call
		expectEndpointDst string
		expectVariables   map[string]string
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("f1afa9ce-ecb2-11ea-ab94-a768ab787da0"),
			uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
			uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
			uuid.FromStringOrNil("5935ff8a-8c8f-11ec-b26a-3fee169eaf45"),
			address.Address{
				Type:       address.TypeSIP,
				Target:     "testsrc@test.com",
				TargetName: "test",
			},
			address.Address{
				Type:       address.TypeSIP,
				Target:     "testoutgoing@test.com",
				TargetName: "test target",
			},

			&activeflow.ActiveFlow{
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("f1afa9ce-ecb2-11ea-ab94-a768ab787da0"),
				CustomerID: uuid.FromStringOrNil("5999f628-7f44-11ec-801f-173217f33e3f"),
				ChannelID:  call.TestChannelID,
				FlowID:     uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
				Type:       call.TypeFlow,
				Status:     call.StatusDialing,
				Direction:  call.DirectionOutgoing,
				Source: address.Address{
					Type:       address.TypeSIP,
					Target:     "testsrc@test.com",
					TargetName: "test",
				},
				Destination: address.Address{
					Type:       address.TypeSIP,
					Target:     "testoutgoing@test.com",
					TargetName: "test target",
				},
				Action: action.Action{
					ID: action.IDStart,
				},
			},
			"pjsip/call-out/sip:testoutgoing@test.com",
			map[string]string{
				"CALLERID(all)":                         `"test" <sip:testsrc@test.com>`,
				"PJSIP_HEADER(add,VBOUT-SDP_Transport)": "RTP/AVP",
			},
		},
		{
			"tel type destination",
			uuid.FromStringOrNil("b7c40962-07fb-11eb-bb82-a3bd16bf1bd9"),
			uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
			uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
			uuid.FromStringOrNil("61c0fe66-8c8f-11ec-873a-ff90a846a02f"),
			address.Address{
				Type:       address.TypeTel,
				Target:     "+99999888",
				TargetName: "test",
			},
			address.Address{
				Type:       address.TypeTel,
				Target:     "+123456789",
				TargetName: "test target",
			},

			&activeflow.ActiveFlow{
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("b7c40962-07fb-11eb-bb82-a3bd16bf1bd9"),
				CustomerID: uuid.FromStringOrNil("68c94bbc-7f44-11ec-9be4-77cb8e61c513"),
				ChannelID:  call.TestChannelID,
				FlowID:     uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
				Type:       call.TypeFlow,
				Status:     call.StatusDialing,
				Direction:  call.DirectionOutgoing,
				Source: address.Address{
					Type:       address.TypeTel,
					Target:     "+99999888",
					TargetName: "test",
				},
				Destination: address.Address{
					Type:       address.TypeTel,
					Target:     "+123456789",
					TargetName: "test target",
				},
				Action: action.Action{
					ID: action.IDStart,
				},
			},
			// "pjsip/call-out/sip:+123456789@voipbin.pstn.twilio.com",
			"pjsip/call-out/sip:+123456789@sip.telnyx.com;transport=udp",
			map[string]string{
				"CALLERID(all)":                         "+99999888",
				"PJSIP_HEADER(add,VBOUT-SDP_Transport)": "RTP/AVP",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().FMV1ActvieFlowCreate(gomock.Any(), tt.id, tt.flowID).Return(tt.af, nil)
			mockDB.EXPECT().CallCreate(gomock.Any(), tt.expectCall).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.id).Return(tt.expectCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), tt.expectCall.CustomerID, call.EventTypeCallCreated, tt.expectCall)

			if tt.masterCallID != uuid.Nil {
				mockDB.EXPECT().CallTXStart(tt.masterCallID).Return(nil, &call.Call{}, nil)
				mockDB.EXPECT().CallTXAddChainedCallID(gomock.Any(), tt.masterCallID, tt.expectCall.ID).Return(nil)
				mockDB.EXPECT().CallSetMasterCallID(gomock.Any(), tt.expectCall.ID, tt.masterCallID).Return(nil)
				mockDB.EXPECT().CallTXFinish(gomock.Any(), true)
			}

			mockReq.EXPECT().AstChannelCreate(gomock.Any(), requesthandler.AsteriskIDCall, gomock.Any(), fmt.Sprintf("context=%s,call_id=%s", ContextOutgoingCall, tt.id), tt.expectEndpointDst, "", "", "", tt.expectVariables).Return(nil)

			res, err := h.CreateCallOutgoing(context.Background(), tt.id, tt.customerID, tt.flowID, tt.masterCallID, tt.source, tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectCall) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectCall, res)
			}
		})
	}
}

func TestGetEndpointDestinationTypeTel(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name               string
		destination        *address.Address
		expectEndpointDest string
	}

	tests := []test{
		{
			"normal",
			&address.Address{
				Type:   address.TypeTel,
				Target: "+1234567890",
			},
			"pjsip/call-out/sip:+1234567890@sip.telnyx.com;transport=udp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res, err := h.getEndpointDestination(context.Background(), *tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectEndpointDest {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectEndpointDest, res)
			}
		})
	}
}

func TestGetEndpointDestinationTypeSIP(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name               string
		destination        *address.Address
		expectEndpointDest string
	}

	tests := []test{
		{
			"normal",
			&address.Address{
				Type:   address.TypeSIP,
				Target: "test@test.com",
			},
			"pjsip/call-out/sip:test@test.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res, err := h.getEndpointDestination(context.Background(), *tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectEndpointDest {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectEndpointDest, res)
			}
		})
	}
}

func TestGetEndpointDestinationTypeSIPVoIPBIN(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name               string
		destination        *address.Address
		contacts           []*astcontact.AstContact
		expectEndpointDest string
	}

	tests := []test{
		{
			"normal",
			&address.Address{
				Type:   address.TypeSIP,
				Target: "test@test.sip.voipbin.net",
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
			&address.Address{
				Type:   address.TypeSIP,
				Target: "test@test.sip.voipbin.net",
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
			&address.Address{
				Type:   address.TypeSIP,
				Target: "test@test.sip.voipbin.net",
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
			&address.Address{
				Type:   address.TypeSIP,
				Target: "test@test.sip.voipbin.net",
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

			mockReq.EXPECT().RMV1ContactGets(gomock.Any(), tt.destination.Target).Return(tt.contacts, nil)

			res, err := h.getEndpointDestination(context.Background(), *tt.destination)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectEndpointDest {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectEndpointDest, res)
			}
		})
	}
}

func TestGetEndpointDestinationTypeSIPVoIPBINError(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &callHandler{
		reqHandler: mockReq,
		db:         mockDB,
	}

	type test struct {
		name        string
		destination *address.Address
		contacts    []*astcontact.AstContact
	}

	tests := []test{
		{
			"no contact",
			&address.Address{
				Type:   address.TypeSIP,
				Target: "test@test.sip.voipbin.net",
			},
			[]*astcontact.AstContact{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockReq.EXPECT().RMV1ContactGets(gomock.Any(), tt.destination.Target).Return(tt.contacts, nil)

			_, err := h.getEndpointDestination(context.Background(), *tt.destination)
			if err == nil {
				t.Error("Wrong match. expect: err, got: ok")
			}
		})
	}
}
