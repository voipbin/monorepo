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

func Test_ConversationsHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetConversations unauthenticated", method: http.MethodGet, path: "/conversations", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetConversationsId unauthenticated", method: http.MethodGet, path: "/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetConversationsId nil id", method: http.MethodGet, path: "/conversations/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "GetConversationsId service error", method: http.MethodGet, path: "/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ConversationGet(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "PutConversationsId unauthenticated", method: http.MethodPut, path: "/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PutConversationsId nil id", method: http.MethodPut, path: "/conversations/00000000-0000-0000-0000-000000000000", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PutConversationsId bad json", method: http.MethodPut, path: "/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "GetConversationsIdMessages unauthenticated", method: http.MethodGet, path: "/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467/messages", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetConversationsIdMessages nil id", method: http.MethodGet, path: "/conversations/00000000-0000-0000-0000-000000000000/messages", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PostConversationsIdMessages unauthenticated", method: http.MethodPost, path: "/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467/messages", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostConversationsIdMessages nil id", method: http.MethodPost, path: "/conversations/00000000-0000-0000-0000-000000000000/messages", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PostConversationsIdMessages bad json", method: http.MethodPost, path: "/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467/messages", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PostConversationsIdUnassign unauthenticated", method: http.MethodPost, path: "/conversations/3bc539bc-c68b-11ec-b41f-0776699e7467/unassign", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
	}

	s2RunCases(t, tests)
}
