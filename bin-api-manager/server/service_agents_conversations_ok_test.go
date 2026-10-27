package server

import (
	"net/http"
	"testing"
	"time"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmedia "monorepo/bin-conversation-manager/models/media"
	cvmessage "monorepo/bin-conversation-manager/models/message"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentsConversationsHandlers_Success(t *testing.T) {
	convID := uuid.FromStringOrNil("33333333-0000-4000-8000-000000000001")
	ownerID := uuid.FromStringOrNil("33333333-0000-4000-8000-000000000002")
	tm := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)

	conv := &cvconversation.WebhookMessage{Identity: commonidentity.Identity{ID: convID}, Name: "conv", TMCreate: &tm}
	msg := &cvmessage.WebhookMessage{Identity: commonidentity.Identity{ID: ownerID}, ConversationID: convID, TMCreate: &tm}
	base := "/service_agents/conversations/33333333-0000-4000-8000-000000000001"

	tests := []okCase{
		{
			name: "list", method: http.MethodGet, path: "/service_agents/conversations?page_size=3&page_token=tok", wantStatus: 200,
			wantFields: map[string]any{"next_page_token": "2024-05-06T07:08:09.000000Z"},
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentConversationList(gomock.Any(), a, uint64(3), "tok").Return([]*cvconversation.WebhookMessage{conv}, nil)
			},
		},
		{
			name: "list clamps oversized page size", method: http.MethodGet, path: "/service_agents/conversations?page_size=500", wantStatus: 200,
			wantJSON: `{"result":[],"next_page_token":""}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentConversationList(gomock.Any(), a, uint64(100), "").Return(nil, nil)
			},
		},
		{
			name: "get", method: http.MethodGet, path: base, wantStatus: 200,
			wantFields: map[string]any{"id": convID.String(), "name": "conv"},
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentConversationGet(gomock.Any(), a, convID).Return(conv, nil)
			},
		},
		{
			name: "message list", method: http.MethodGet, path: base + "/messages?page_size=2", wantStatus: 200,
			wantFields: map[string]any{"next_page_token": "2024-05-06T07:08:09.000000Z"},
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentConversationMessageList(gomock.Any(), a, convID, uint64(2), "").Return([]*cvmessage.WebhookMessage{msg}, nil)
			},
		},
		{
			name: "message send", method: http.MethodPost, path: base + "/messages", body: `{"text":"hello","medias":[]}`, wantStatus: 200,
			wantFields: map[string]any{"conversation_id": convID.String()},
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentConversationMessageSend(gomock.Any(), a, convID, "hello", []cvmedia.Media{}).Return(msg, nil)
			},
		},
		{
			name: "update name", method: http.MethodPut, path: base, body: `{"name":"renamed"}`, wantStatus: 200,
			wantFields: map[string]any{"id": convID.String()},
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentConversationUpdate(gomock.Any(), a, convID, map[cvconversation.Field]any{cvconversation.FieldName: "renamed"}).Return(conv, nil)
			},
		},
		{
			name: "unassign", method: http.MethodPost, path: base + "/unassign", wantStatus: 200,
			wantFields: map[string]any{"id": convID.String()},
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentConversationUnassign(gomock.Any(), a, convID).Return(conv, nil)
			},
		},
	}

	okRunCases(t, tests)
}

func Test_ServiceAgentsConversationsHandlers_Errors(t *testing.T) {
	base := "/service_agents/conversations/33333333-0000-4000-8000-000000000001"
	nilBase := "/service_agents/conversations/00000000-0000-0000-0000-000000000000"
	denied := cerrors.StatusPermissionDenied

	tests := []s2Case{
		{name: "list service error", method: http.MethodGet, path: "/service_agents/conversations", wantStatus: denied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentConversationList(gomock.Any(), a, uint64(100), "").Return(nil, s2Err)
			}},
		{name: "get nil id", method: http.MethodGet, path: nilBase, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "get service error", method: http.MethodGet, path: base, wantStatus: denied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentConversationGet(gomock.Any(), a, gomock.Any()).Return(nil, s2Err)
			}},
		{name: "message list nil id", method: http.MethodGet, path: nilBase + "/messages", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "message list service error", method: http.MethodGet, path: base + "/messages", wantStatus: denied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentConversationMessageList(gomock.Any(), a, gomock.Any(), uint64(100), "").Return(nil, s2Err)
			}},
		{name: "message send nil id", method: http.MethodPost, path: nilBase + "/messages", body: `{"text":"x","medias":[]}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "message send bad json", method: http.MethodPost, path: base + "/messages", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "message send service error", method: http.MethodPost, path: base + "/messages", body: `{"text":"x","medias":[]}`, wantStatus: denied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentConversationMessageSend(gomock.Any(), a, gomock.Any(), "x", []cvmedia.Media{}).Return(nil, s2Err)
			}},
		{name: "update unauthenticated", method: http.MethodPut, path: base, body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "update bad json", method: http.MethodPut, path: base, body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "update service error", method: http.MethodPut, path: base, body: `{"detail":"d"}`, wantStatus: denied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentConversationUpdate(gomock.Any(), a, gomock.Any(), map[cvconversation.Field]any{cvconversation.FieldDetail: "d"}).Return(nil, s2Err)
			}},
		{name: "unassign unauthenticated", method: http.MethodPost, path: base + "/unassign", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "unassign service error", method: http.MethodPost, path: base + "/unassign", wantStatus: denied, wantReason: "S2_DENIED",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentConversationUnassign(gomock.Any(), a, gomock.Any()).Return(nil, s2Err)
			}},
	}

	s2RunCases(t, tests)
}
