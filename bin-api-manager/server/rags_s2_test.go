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

func Test_RagsHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "PostRags unauthenticated", method: http.MethodPost, path: "/rags", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostRags bad json", method: http.MethodPost, path: "/rags", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "GetRags unauthenticated", method: http.MethodGet, path: "/rags", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetRags service error", method: http.MethodGet, path: "/rags?page_size=1000&page_token=tok", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().RagGets(gomock.Any(), a, uint64(100), "tok").Return(nil, s2Err)
		}},
		{name: "GetRagsId unauthenticated", method: http.MethodGet, path: "/rags/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetRagsId service error", method: http.MethodGet, path: "/rags/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().RagGet(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "PutRagsId unauthenticated", method: http.MethodPut, path: "/rags/3bc539bc-c68b-11ec-b41f-0776699e7467", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PutRagsId bad json", method: http.MethodPut, path: "/rags/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PostRagsIdSources unauthenticated", method: http.MethodPost, path: "/rags/3bc539bc-c68b-11ec-b41f-0776699e7467/sources", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostRagsIdSources bad json", method: http.MethodPost, path: "/rags/3bc539bc-c68b-11ec-b41f-0776699e7467/sources", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "DeleteRagsIdSourcesSourceId unauthenticated", method: http.MethodDelete, path: "/rags/3bc539bc-c68b-11ec-b41f-0776699e7467/sources/4cd64acd-d79c-22fd-c52a-1887788f8578", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "DeleteRagsId unauthenticated", method: http.MethodDelete, path: "/rags/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "DeleteRagsId service error", method: http.MethodDelete, path: "/rags/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().RagDelete(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
	}

	s2RunCases(t, tests)
}
