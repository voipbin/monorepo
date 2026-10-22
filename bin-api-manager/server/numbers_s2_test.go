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

func Test_NumbersHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetNumbers unauthenticated", method: http.MethodGet, path: "/numbers", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetNumbers service error", method: http.MethodGet, path: "/numbers?page_size=1000&page_token=tok", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().NumberList(gomock.Any(), a, uint64(100), "tok").Return(nil, s2Err)
		}},
		{name: "GetNumbersId unauthenticated", method: http.MethodGet, path: "/numbers/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetNumbersId nil id", method: http.MethodGet, path: "/numbers/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "GetNumbersId service error", method: http.MethodGet, path: "/numbers/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().NumberGet(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "PostNumbers unauthenticated", method: http.MethodPost, path: "/numbers", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostNumbers bad json", method: http.MethodPost, path: "/numbers", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "DeleteNumbersId unauthenticated", method: http.MethodDelete, path: "/numbers/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "DeleteNumbersId nil id", method: http.MethodDelete, path: "/numbers/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "DeleteNumbersId service error", method: http.MethodDelete, path: "/numbers/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().NumberDelete(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "PutNumbersId unauthenticated", method: http.MethodPut, path: "/numbers/3bc539bc-c68b-11ec-b41f-0776699e7467", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PutNumbersId nil id", method: http.MethodPut, path: "/numbers/00000000-0000-0000-0000-000000000000", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PutNumbersId bad json", method: http.MethodPut, path: "/numbers/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PutNumbersIdFlowIds unauthenticated", method: http.MethodPut, path: "/numbers/3bc539bc-c68b-11ec-b41f-0776699e7467/flow_ids", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PutNumbersIdFlowIds nil id", method: http.MethodPut, path: "/numbers/00000000-0000-0000-0000-000000000000/flow_ids", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PutNumbersIdFlowIds bad json", method: http.MethodPut, path: "/numbers/3bc539bc-c68b-11ec-b41f-0776699e7467/flow_ids", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PutNumbersIdMetadata unauthenticated", method: http.MethodPut, path: "/numbers/3bc539bc-c68b-11ec-b41f-0776699e7467/metadata", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PutNumbersIdMetadata nil id", method: http.MethodPut, path: "/numbers/00000000-0000-0000-0000-000000000000/metadata", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "PutNumbersIdMetadata bad json", method: http.MethodPut, path: "/numbers/3bc539bc-c68b-11ec-b41f-0776699e7467/metadata", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "PostNumbersRenew unauthenticated", method: http.MethodPost, path: "/numbers/renew", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostNumbersRenew bad json", method: http.MethodPost, path: "/numbers/renew", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	}

	s2RunCases(t, tests)
}
