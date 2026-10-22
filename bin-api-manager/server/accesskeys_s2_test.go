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

func Test_AccesskeysHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetAccesskeys unauthenticated", method: http.MethodGet, path: "/accesskeys", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetAccesskeys service error", method: http.MethodGet, path: "/accesskeys?page_size=1000&page_token=tok", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().AccesskeyList(gomock.Any(), a, uint64(100), "tok").Return(nil, s2Err)
		}},
		{name: "PostAccesskeys unauthenticated", method: http.MethodPost, path: "/accesskeys", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostAccesskeys bad json", method: http.MethodPost, path: "/accesskeys", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "GetAccesskeysId unauthenticated", method: http.MethodGet, path: "/accesskeys/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetAccesskeysId nil id", method: http.MethodGet, path: "/accesskeys/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "GetAccesskeysId service error", method: http.MethodGet, path: "/accesskeys/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().AccesskeyGet(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "DeleteAccesskeysId unauthenticated", method: http.MethodDelete, path: "/accesskeys/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "DeleteAccesskeysId nil id", method: http.MethodDelete, path: "/accesskeys/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "DeleteAccesskeysId service error", method: http.MethodDelete, path: "/accesskeys/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().AccesskeyDelete(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "PutAccesskeysId unauthenticated", method: http.MethodPut, path: "/accesskeys/3bc539bc-c68b-11ec-b41f-0776699e7467", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PutAccesskeysId nil id", method: http.MethodPut, path: "/accesskeys/00000000-0000-0000-0000-000000000000", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PutAccesskeysId bad json", method: http.MethodPut, path: "/accesskeys/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	}

	s2RunCases(t, tests)
}
