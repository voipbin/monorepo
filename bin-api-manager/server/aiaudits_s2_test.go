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

func Test_AiauditsHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "PostAiaudits unauthenticated", method: http.MethodPost, path: "/aiaudits", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostAiaudits bad json", method: http.MethodPost, path: "/aiaudits", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "GetAiaudits unauthenticated", method: http.MethodGet, path: "/aiaudits", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetAiauditsId unauthenticated", method: http.MethodGet, path: "/aiaudits/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetAiauditsId service error", method: http.MethodGet, path: "/aiaudits/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().AIAuditGet(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "DeleteAiauditsId unauthenticated", method: http.MethodDelete, path: "/aiaudits/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "DeleteAiauditsId service error", method: http.MethodDelete, path: "/aiaudits/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().AIAuditDelete(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
	}

	s2RunCases(t, tests)
}
