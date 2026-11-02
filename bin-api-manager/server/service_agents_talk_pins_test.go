package server

import (
	"fmt"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	tkmessage "monorepo/bin-talk-manager/models/message"
	tkpin "monorepo/bin-talk-manager/models/pin"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const (
	pinsPath     = "/service_agents/talk_pins"
	pinsPinID    = "6f1c1a52-8d3b-4c1e-9a5e-0b7f2d9a1c10"
	pinsTargetID = "0d4b7a3e-2f5c-4b8e-8a21-9c3e6f1d7a42"
	pinsChatID   = "9a2b5c7d-1e3f-4a6b-8c9d-0e1f2a3b4c5d"
)

func pinsTestPin(targetType tkpin.TargetType, withMessage bool) *tkpin.WebhookMessage {
	res := &tkpin.WebhookMessage{
		ID:         s1ID(pinsPinID),
		TargetType: targetType,
		TargetID:   s1ID(pinsTargetID),
		ChatID:     s1ID(pinsChatID),
		TMCreate:   s1Time("2026-11-02T09:15:30.123456Z"),
	}
	if withMessage {
		res.Message = &tkmessage.WebhookMessage{
			Identity: commonidentity.Identity{ID: s1ID(pinsTargetID), CustomerID: s1ID(s1CustomerID)},
			Owner:    commonidentity.Owner{OwnerType: "agent", OwnerID: s1ID(s1AgentID)},
			ChatID:   s1ID(pinsChatID),
			Type:     tkmessage.TypeNormal,
			Text:     "hello",
		}
	}
	return res
}

func Test_PostServiceAgentsTalkPins(t *testing.T) {
	chatPin := pinsTestPin(tkpin.TargetTypeChat, false)

	s1Run(t, []s1Case{
		s1Unauth("POST", pinsPath, `{"target_type":"chat","target_id":"`+pinsTargetID+`"}`),
		s1BadJSON("POST", pinsPath),
		{
			name: "target id that is not a uuid is rejected as a bad body", method: "POST", path: pinsPath,
			body:       `{"target_type":"chat","target_id":"x"}`,
			wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY",
		},
		{
			name: "chat pin", method: "POST", path: pinsPath,
			body: `{"target_type":"chat","target_id":"` + pinsTargetID + `"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentTalkPinCreate(gomock.Any(), a, tkpin.TargetTypeChat, s1ID(pinsTargetID)).Return(chatPin, nil)
			},
			wantBody: s1JSON(t, chatPin),
		},
		{
			name: "unknown target type reaches the service, which rejects it", method: "POST", path: pinsPath,
			body: `{"target_type":"call","target_id":"` + pinsTargetID + `"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentTalkPinCreate(gomock.Any(), a, tkpin.TargetType("call"), s1ID(pinsTargetID)).Return(nil, fmt.Errorf("%w: target_type must be chat or message", serviceerrors.ErrInvalidArgument))
			},
			wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT",
		},
		s1PermDenied("POST", pinsPath, `{"target_type":"message","target_id":"`+pinsTargetID+`"}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkPinCreate(gomock.Any(), a, tkpin.TargetTypeMessage, s1ID(pinsTargetID)).Return(nil, err)
		}),
		s1NotFound("POST", pinsPath, `{"target_type":"message","target_id":"`+pinsTargetID+`"}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkPinCreate(gomock.Any(), a, tkpin.TargetTypeMessage, s1ID(pinsTargetID)).Return(nil, err)
		}),
	})
}

func Test_GetServiceAgentsTalkPins(t *testing.T) {
	msgPin := pinsTestPin(tkpin.TargetTypeMessage, true)
	older := pinsTestPin(tkpin.TargetTypeChat, false)
	older.TMCreate = s1Time("2026-11-01T08:00:00.5Z")

	list := []*tkpin.WebhookMessage{msgPin, older}
	wantList := `{"result":` + s1JSON(t, list) + `,"next_page_token":"2026-11-01T08:00:00.500000Z"}`

	cases := s1ListCases(pinsPath, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error) {
		m.EXPECT().ServiceAgentTalkPinList(gomock.Any(), a, tkpin.TargetType(""), uuid.Nil, size, token).Return(nil, err)
	})
	cases = append(cases,
		s1Case{
			name: "default page size and the next page token is the last tm_create", method: "GET", path: pinsPath,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentTalkPinList(gomock.Any(), a, tkpin.TargetType(""), uuid.Nil, uint64(100), "").Return(list, nil)
			},
			wantBody: wantList,
		},
		s1Case{
			name: "filters are forwarded", method: "GET", path: pinsPath + "?target_type=message&chat_id=" + pinsChatID + "&page_size=5",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentTalkPinList(gomock.Any(), a, tkpin.TargetTypeMessage, s1ID(pinsChatID), uint64(5), "").Return(list, nil)
			},
			wantBody: wantList,
		},
		s1Case{
			name: "empty result", method: "GET", path: pinsPath,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentTalkPinList(gomock.Any(), a, tkpin.TargetType(""), uuid.Nil, uint64(100), "").Return(nil, nil)
			},
			wantBody: `{"result":[],"next_page_token":""}`,
		},
		s1Case{
			// the binding error of the generated wrapper; the service is not called
			name: "chat id that is not a uuid is a bad parameter", method: "GET", path: pinsPath + "?chat_id=x",
			wantCode: 400, wantBody: `{"msg":"Invalid format for parameter chat_id: error unmarshaling 'x' text as *uuid.UUID: invalid UUID length: 1"}`,
		},
		s1Case{
			name: "unknown target type reaches the service, which rejects it", method: "GET", path: pinsPath + "?target_type=call",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentTalkPinList(gomock.Any(), a, tkpin.TargetType("call"), uuid.Nil, uint64(100), "").Return(nil, fmt.Errorf("%w: target_type must be chat or message", serviceerrors.ErrInvalidArgument))
			},
			wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT",
		},
	)
	s1Run(t, cases)
}

func Test_DeleteServiceAgentsTalkPinsId(t *testing.T) {
	chatPin := pinsTestPin(tkpin.TargetTypeChat, false)
	path := pinsPath + "/" + pinsPinID

	s1Run(t, []s1Case{
		s1Unauth("DELETE", path, ""),
		{
			// the binding error of the generated wrapper; the service is not called
			name: "pin id that is not a uuid is a bad parameter", method: "DELETE", path: pinsPath + "/x",
			wantCode: 400, wantBody: `{"msg":"Invalid format for parameter id: error unmarshaling 'x' text as *uuid.UUID: invalid UUID length: 1"}`,
		},
		{
			name: "removed pin", method: "DELETE", path: path,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentTalkPinDelete(gomock.Any(), a, s1ID(pinsPinID)).Return(chatPin, nil)
			},
			wantBody: s1JSON(t, chatPin),
		},
		s1NotFound("DELETE", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkPinDelete(gomock.Any(), a, s1ID(pinsPinID)).Return(nil, err)
		}),
		s1PermDenied("DELETE", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentTalkPinDelete(gomock.Any(), a, s1ID(pinsPinID)).Return(nil, err)
		}),
	})
}
