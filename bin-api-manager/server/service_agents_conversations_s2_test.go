package server

import (
	"net/http"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentsConversationsHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetServiceAgentsConversations unauthenticated", method: http.MethodGet, path: "/service_agents/conversations", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetServiceAgentsConversations service error", method: http.MethodGet, path: "/service_agents/conversations?page_size=1000&page_token=tok", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentConversationList(gomock.Any(), a, uint64(100), "tok").Return(nil, s2Err)
		}},
		{name: "GetServiceAgentsConversationsId unauthenticated", method: http.MethodGet, path: "/service_agents/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetServiceAgentsConversationsId nil id", method: http.MethodGet, path: "/service_agents/conversations/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "GetServiceAgentsConversationsId service error", method: http.MethodGet, path: "/service_agents/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentConversationGet(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "GetServiceAgentsConversationsIdMessages unauthenticated", method: http.MethodGet, path: "/service_agents/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467/messages", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetServiceAgentsConversationsIdMessages nil id", method: http.MethodGet, path: "/service_agents/conversations/00000000-0000-0000-0000-000000000000/messages", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PostServiceAgentsConversationsIdMessages unauthenticated", method: http.MethodPost, path: "/service_agents/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467/messages", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostServiceAgentsConversationsIdMessages nil id", method: http.MethodPost, path: "/service_agents/conversations/00000000-0000-0000-0000-000000000000/messages", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PostServiceAgentsConversationsIdMessages bad json", method: http.MethodPost, path: "/service_agents/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467/messages", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PutServiceAgentsConversationsId unauthenticated", method: http.MethodPut, path: "/service_agents/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PutServiceAgentsConversationsId bad json", method: http.MethodPut, path: "/service_agents/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PostServiceAgentsConversationsIdUnassign unauthenticated", method: http.MethodPost, path: "/service_agents/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467/unassign", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
	}

	s2RunCases(t, tests)
}
