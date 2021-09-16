package callhandler

import (
	"fmt"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/conferencehandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/call-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
	"gitlab.com/voipbin/bin-manager/flow-manager.git/models/activeflow"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
)

func TestCreateCallOutgoing(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		db:            mockDB,
		confHandler:   mockConf,
	}

	type test struct {
		name        string
		id          uuid.UUID
		userID      uint64
		flowID      uuid.UUID
		source      address.Address
		destination address.Address

		af                *activeflow.ActiveFlow
		expectCall        *call.Call
		expectEndpointDst string
		expectVariables   map[string]string
	}

	tests := []test{
		{
			"normal",
			uuid.FromStringOrNil("f1afa9ce-ecb2-11ea-ab94-a768ab787da0"),
			1,
			uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
			address.Address{
				Type:   address.TypeSIP,
				Name:   "test",
				Target: "testsrc@test.com",
			},
			address.Address{
				Type:   address.TypeSIP,
				Name:   "test target",
				Target: "testoutgoing@test.com",
			},

			&activeflow.ActiveFlow{
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
			},
			&call.Call{
				ID:        uuid.FromStringOrNil("f1afa9ce-ecb2-11ea-ab94-a768ab787da0"),
				UserID:    1,
				ChannelID: call.TestChannelID,
				FlowID:    uuid.FromStringOrNil("fd5b3234-ecb2-11ea-8f23-4369cba01ddb"),
				Type:      call.TypeFlow,
				Status:    call.StatusDialing,
				Direction: call.DirectionOutgoing,
				Source: address.Address{
					Type:   address.TypeSIP,
					Name:   "test",
					Target: "testsrc@test.com",
				},
				Destination: address.Address{
					Type:   address.TypeSIP,
					Name:   "test target",
					Target: "testoutgoing@test.com",
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
			1,
			uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
			address.Address{
				Type:   address.TypeTel,
				Name:   "test",
				Target: "+99999888",
			},
			address.Address{
				Type:   address.TypeTel,
				Name:   "test target",
				Target: "+123456789",
			},

			&activeflow.ActiveFlow{
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
			},
			&call.Call{
				ID:        uuid.FromStringOrNil("b7c40962-07fb-11eb-bb82-a3bd16bf1bd9"),
				UserID:    1,
				ChannelID: call.TestChannelID,
				FlowID:    uuid.FromStringOrNil("c4f08e1c-07fb-11eb-bd6d-8f92c676d869"),
				Type:      call.TypeFlow,
				Status:    call.StatusDialing,
				Direction: call.DirectionOutgoing,
				Source: address.Address{
					Type:   address.TypeTel,
					Name:   "test",
					Target: "+99999888",
				},
				Destination: address.Address{
					Type:   address.TypeTel,
					Name:   "test target",
					Target: "+123456789",
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
		{
			"callflow has an webhook",
			uuid.FromStringOrNil("4347bd52-8304-11eb-b239-4bec34310838"),
			1,
			uuid.FromStringOrNil("4394ad24-8304-11eb-b397-ff7bf34c829f"),
			address.Address{
				Type:   address.TypeTel,
				Name:   "test",
				Target: "+99999888",
			},
			address.Address{
				Type:   address.TypeTel,
				Name:   "test target",
				Target: "+123456789",
			},

			&activeflow.ActiveFlow{
				WebhookURI: "https://test.com/wwasdd",
				CurrentAction: action.Action{
					ID: action.IDStart,
				},
			},
			&call.Call{
				ID:         uuid.FromStringOrNil("4347bd52-8304-11eb-b239-4bec34310838"),
				UserID:     1,
				ChannelID:  call.TestChannelID,
				FlowID:     uuid.FromStringOrNil("4394ad24-8304-11eb-b397-ff7bf34c829f"),
				Type:       call.TypeFlow,
				Status:     call.StatusDialing,
				Direction:  call.DirectionOutgoing,
				WebhookURI: "https://test.com/wwasdd",
				Source: address.Address{
					Type:   address.TypeTel,
					Name:   "test",
					Target: "+99999888",
				},
				Destination: address.Address{
					Type:   address.TypeTel,
					Name:   "test target",
					Target: "+123456789",
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

			mockDB.EXPECT().CallCreate(gomock.Any(), tt.expectCall).Return(nil)
			mockDB.EXPECT().CallGet(gomock.Any(), tt.id).Return(tt.expectCall, nil)
			mockNotify.EXPECT().NotifyEvent(notifyhandler.EventTypeCallCreated, tt.expectCall)
			mockReq.EXPECT().FlowActvieFlowPost(tt.id, tt.flowID).Return(tt.af, nil)
			mockReq.EXPECT().AstChannelCreate(requesthandler.AsteriskIDCall, gomock.Any(), fmt.Sprintf("context=%s,call_id=%s", ContextOutgoingCall, tt.id), tt.expectEndpointDst, "", "", "", tt.expectVariables).Return(nil)

			res, err := h.CreateCallOutgoing(tt.id, tt.userID, tt.flowID, tt.source, tt.destination)
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
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
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

			res, err := h.getEndpointDestination(*tt.destination)
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
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
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

			res, err := h.getEndpointDestination(*tt.destination)
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
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
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

			mockReq.EXPECT().RMV1ContactsGet(tt.destination.Target).Return(tt.contacts, nil)

			res, err := h.getEndpointDestination(*tt.destination)
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
	mockConf := conferencehandler.NewMockConferenceHandler(mc)

	h := &callHandler{
		reqHandler:  mockReq,
		db:          mockDB,
		confHandler: mockConf,
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

			mockReq.EXPECT().RMV1ContactsGet(tt.destination.Target).Return(tt.contacts, nil)

			_, err := h.getEndpointDestination(*tt.destination)
			if err == nil {
				t.Error("Wrong match. expect: err, got: ok")
			}
		})
	}
}
