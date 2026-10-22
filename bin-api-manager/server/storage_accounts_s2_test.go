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

func Test_StorageAccountsHandlers_ErrorPaths(t *testing.T) {
	tests := []s2Case{
		{name: "GetStorageAccounts unauthenticated", method: http.MethodGet, path: "/storage_accounts", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetStorageAccounts service error", method: http.MethodGet, path: "/storage_accounts?page_size=1000&page_token=tok", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().StorageAccountList(gomock.Any(), a, uint64(100), "tok").Return(nil, s2Err)
		}},
		{name: "PostStorageAccounts unauthenticated", method: http.MethodPost, path: "/storage_accounts", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "PostStorageAccounts bad json", method: http.MethodPost, path: "/storage_accounts", body: "{", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "GetStorageAccountsId unauthenticated", method: http.MethodGet, path: "/storage_accounts/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "GetStorageAccountsId nil id", method: http.MethodGet, path: "/storage_accounts/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "GetStorageAccountsId service error", method: http.MethodGet, path: "/storage_accounts/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().StorageAccountGet(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
		{name: "DeleteStorageAccountsId unauthenticated", method: http.MethodDelete, path: "/storage_accounts/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "DeleteStorageAccountsId nil id", method: http.MethodDelete, path: "/storage_accounts/00000000-0000-0000-0000-000000000000", body: "", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "DeleteStorageAccountsId service error", method: http.MethodDelete, path: "/storage_accounts/3bc539bc-c68b-11ec-b41f-0776699e7467", body: "", wantStatus: cerrors.StatusPermissionDenied, wantReason: "S2_DENIED", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().StorageAccountDelete(gomock.Any(), a, uuid.FromStringOrNil("3bc539bc-c68b-11ec-b41f-0776699e7467")).Return(nil, s2Err)
		}},
	}

	s2RunCases(t, tests)
}
