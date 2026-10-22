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

func Test_AicallsHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "PostAicalls unauthenticated", method: http.MethodPost, path: "/aicalls", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostAicalls bad json", method: http.MethodPost, path: "/aicalls", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "GetAicalls unauthenticated", method: http.MethodGet, path: "/aicalls", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetAicalls service error", method: http.MethodGet, path: "/aicalls?page_size=1000&page_token=tok", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().AIcallGetsByCustomerID(gomock.Any(), a, uint64(100), "tok").Return(nil, s2Err)
		}},
		{name: "GetAicallsId unauthenticated", method: http.MethodGet, path: "/aicalls/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetAicallsId nil id", method: http.MethodGet, path: "/aicalls/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "GetAicallsId service error", method: http.MethodGet, path: "/aicalls/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().AIcallGet(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "DeleteAicallsId unauthenticated", method: http.MethodDelete, path: "/aicalls/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "DeleteAicallsId nil id", method: http.MethodDelete, path: "/aicalls/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "DeleteAicallsId service error", method: http.MethodDelete, path: "/aicalls/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().AIcallDelete(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "PostAicallsIdTerminate unauthenticated", method: http.MethodPost, path: "/aicalls/3bc539bc-c68b-11ec-b41f-0776699e7467/terminate", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostAicallsIdTerminate service error", method: http.MethodPost, path: "/aicalls/3bc539bc-c68b-11ec-b41f-0776699e7467/terminate", body: `{}`, wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().AIcallTerminate(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "GetAicallsIdParticipants unauthenticated", method: http.MethodGet, path: "/aicalls/3bc539bc-c68b-11ec-b41f-0776699e7467/participants", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
	}

	s2RunCases(t, tests)
}
