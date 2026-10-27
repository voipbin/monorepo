package server

import (
	"net/http"
	"testing"
	"time"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	wcmessage "monorepo/bin-webchat-manager/models/message"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_WebchatMessagesHandlers_Success(t *testing.T) {
	msgID := uuid.FromStringOrNil("44444444-0000-4000-8000-000000000001")
	sessionID := uuid.FromStringOrNil("44444444-0000-4000-8000-000000000002")
	tm := time.Date(2024, 7, 8, 9, 10, 11, 0, time.UTC)

	msg := &wcmessage.WebhookMessage{Identity: commonidentity.Identity{ID: msgID}, SessionID: sessionID, Direction: wcmessage.DirectionOutbound, TMCreate: &tm}

	tests := []okCase{
		{
			name: "list with session filter", method: http.MethodGet, path: "/webchat_messages?page_size=4&page_token=tok&session_id=44444444-0000-4000-8000-000000000002", wantStatus: 200,
			wantFields: map[string]any{"next_page_token": "2024-07-08T09:10:11.000000Z"},
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().WebchatMessageList(gomock.Any(), a, uint64(4), "tok", sessionID).Return([]*wcmessage.WebhookMessage{msg}, nil)
			},
		},
		{
			name: "list defaults", method: http.MethodGet, path: "/webchat_messages", wantStatus: 200,
			wantJSON: `{"result":[],"next_page_token":""}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().WebchatMessageList(gomock.Any(), a, uint64(100), "", uuid.Nil).Return(nil, nil)
			},
		},
		{
			name: "list clamps page size just above the limit", method: http.MethodGet, path: "/webchat_messages?page_size=101", wantStatus: 200,
			wantJSON: `{"result":[],"next_page_token":""}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().WebchatMessageList(gomock.Any(), a, uint64(100), "", uuid.Nil).Return(nil, nil)
			},
		},
		{
			name: "create", method: http.MethodPost, path: "/webchat_messages", wantStatus: 200,
			body:       `{"session_id":"44444444-0000-4000-8000-000000000002","direction":"outbound","text":"hi"}`,
			wantFields: map[string]any{"id": msgID.String(), "direction": "outbound"},
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().WebchatMessageCreate(gomock.Any(), a, sessionID, wcmessage.DirectionOutbound, "hi").Return(msg, nil)
			},
		},
		{
			name: "get", method: http.MethodGet, path: "/webchat_messages/44444444-0000-4000-8000-000000000001", wantStatus: 200,
			wantFields: map[string]any{"id": msgID.String()},
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().WebchatMessageGet(gomock.Any(), a, msgID).Return(msg, nil)
			},
		},
		{
			name: "delete", method: http.MethodDelete, path: "/webchat_messages/44444444-0000-4000-8000-000000000001", wantStatus: 200,
			wantFields: map[string]any{"id": msgID.String()},
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().WebchatMessageDelete(gomock.Any(), a, msgID).Return(msg, nil)
			},
		},
	}

	okRunCases(t, tests)
}

func Test_WebchatMessagesHandlers_Errors(t *testing.T) {
	tests := []s2Case{
		{name: "list service error", method: http.MethodGet, path: "/webchat_messages", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().WebchatMessageList(gomock.Any(), a, uint64(100), "", uuid.Nil).Return(nil, s2Err)
			}},
		{name: "create invalid session id", method: http.MethodPost, path: "/webchat_messages", body: `{"session_id":"not-a-uuid","direction":"outbound","text":"hi"}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_SESSION_ID"},
		{name: "create service error", method: http.MethodPost, path: "/webchat_messages", body: `{"session_id":"44444444-0000-4000-8000-000000000002","direction":"inbound","text":"hi"}`, wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().WebchatMessageCreate(gomock.Any(), a, gomock.Any(), wcmessage.DirectionInbound, "hi").Return(nil, s2Err)
			}},
	}

	s2RunCases(t, tests)
}
