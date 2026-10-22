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

func Test_WebchatMessagesHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetWebchatMessages unauthenticated", method: http.MethodGet, path: "/webchat_messages", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostWebchatMessages unauthenticated", method: http.MethodPost, path: "/webchat_messages", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostWebchatMessages bad json", method: http.MethodPost, path: "/webchat_messages", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "DeleteWebchatMessagesId unauthenticated", method: http.MethodDelete, path: "/webchat_messages/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "DeleteWebchatMessagesId nil id", method: http.MethodDelete, path: "/webchat_messages/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "DeleteWebchatMessagesId service error", method: http.MethodDelete, path: "/webchat_messages/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().WebchatMessageDelete(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "GetWebchatMessagesId unauthenticated", method: http.MethodGet, path: "/webchat_messages/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetWebchatMessagesId nil id", method: http.MethodGet, path: "/webchat_messages/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "GetWebchatMessagesId service error", method: http.MethodGet, path: "/webchat_messages/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().WebchatMessageGet(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
	}

	s2RunCases(t, tests)
}
