package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	fmflow "monorepo/bin-flow-manager/models/flow"

	"go.uber.org/mock/gomock"
)

func Test_GetFlows_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/flows?page_size=20&page_token=tok", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "GET", path: "/flows?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().FlowList(gomock.Any(), a, uint64(20), "tok").Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "page_size_capped", method: "GET", path: "/flows?page_size=1000", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().FlowList(gomock.Any(), a, uint64(100), "").Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "empty_list", method: "GET", path: "/flows?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().FlowList(gomock.Any(), a, uint64(20), "tok").Return(nil, nil)
		}, wantCode: 200, wantBody: `{"result":[],"next_page_token":""}`},
	})
}

func Test_GetFlowsId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/flows/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "GET", path: "/flows/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "GET", path: "/flows/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().FlowGet(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "GET", path: "/flows/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().FlowGet(gomock.Any(), a, s3UUID1).Return(&fmflow.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&fmflow.WebhookMessage{})},
	})
}

func Test_PutFlowsId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "PUT", path: "/flows/00000000-0000-4000-8000-0000000000a1", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "PUT", path: "/flows/00000000-0000-0000-0000-000000000000", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
	})
}

func Test_DeleteFlowsId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "DELETE", path: "/flows/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "DELETE", path: "/flows/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "DELETE", path: "/flows/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().FlowDelete(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "DELETE", path: "/flows/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().FlowDelete(gomock.Any(), a, s3UUID1).Return(&fmflow.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&fmflow.WebhookMessage{})},
	})
}

func Test_PostFlowsIdDirectHashRegenerate_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/flows/00000000-0000-4000-8000-0000000000a1/direct-hash-regenerate", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "POST", path: "/flows/00000000-0000-4000-8000-0000000000a1/direct-hash-regenerate", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().FlowDirectHashRegenerate(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "POST", path: "/flows/00000000-0000-4000-8000-0000000000a1/direct-hash-regenerate", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().FlowDirectHashRegenerate(gomock.Any(), a, s3UUID1).Return(&fmflow.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&fmflow.WebhookMessage{})},
	})
}
