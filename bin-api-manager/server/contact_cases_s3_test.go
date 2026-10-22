package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	cmkase "monorepo/bin-contact-manager/models/kase"

	"go.uber.org/mock/gomock"
)

func Test_GetContactCasesUnresolved_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/contact_cases/unresolved?page_size=20&page_token=tok", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "GET", path: "/contact_cases/unresolved?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CaseListUnresolved(gomock.Any(), a, uint64(20), "tok").Return(nil, "", s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "page_size_capped", method: "GET", path: "/contact_cases/unresolved?page_size=1000", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CaseListUnresolved(gomock.Any(), a, uint64(100), "").Return(nil, "", s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "empty_list", method: "GET", path: "/contact_cases/unresolved?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CaseListUnresolved(gomock.Any(), a, uint64(20), "tok").Return(nil, "", nil)
		}, wantCode: 200, wantBody: `{"result":[],"next_page_token":""}`},
	})
}

func Test_GetContactCasesId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/contact_cases/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "GET", path: "/contact_cases/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CaseGet(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "GET", path: "/contact_cases/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CaseGet(gomock.Any(), a, s3UUID1).Return(&cmkase.Case{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cmkase.Case{})},
	})
}

func Test_PostContactCasesIdClose_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/contact_cases/00000000-0000-4000-8000-0000000000a1/close", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "POST", path: "/contact_cases/00000000-0000-4000-8000-0000000000a1/close", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CaseClose(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "POST", path: "/contact_cases/00000000-0000-4000-8000-0000000000a1/close", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CaseClose(gomock.Any(), a, s3UUID1).Return(&cmkase.Case{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cmkase.Case{})},
	})
}

func Test_PostContactCasesIdContinue_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/contact_cases/00000000-0000-4000-8000-0000000000a1/continue", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "POST", path: "/contact_cases/00000000-0000-4000-8000-0000000000a1/continue", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CaseContinue(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "POST", path: "/contact_cases/00000000-0000-4000-8000-0000000000a1/continue", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CaseContinue(gomock.Any(), a, s3UUID1).Return(&cmkase.Case{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cmkase.Case{})},
	})
}

func Test_PostContactCasesIdAssign_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/contact_cases/00000000-0000-4000-8000-0000000000a1/assign", body: `{"owner_id":"00000000-0000-4000-8000-0000000000a2"}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_json", method: "POST", path: "/contact_cases/00000000-0000-4000-8000-0000000000a1/assign", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "service_error", method: "POST", path: "/contact_cases/00000000-0000-4000-8000-0000000000a1/assign", body: `{"owner_id":"00000000-0000-4000-8000-0000000000a2"}`, setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CaseAssign(gomock.Any(), a, s3UUID1, s3UUID2).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
	})
}
