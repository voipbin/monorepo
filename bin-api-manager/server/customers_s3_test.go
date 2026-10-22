package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	cucustomer "monorepo/bin-customer-manager/models/customer"

	"go.uber.org/mock/gomock"
)

func Test_PostCustomers_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/customers", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_json", method: "POST", path: "/customers", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	})
}

func Test_GetCustomers_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/customers?page_size=20&page_token=tok", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
	})
}

func Test_GetCustomersId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/customers/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "GET", path: "/customers/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "GET", path: "/customers/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CustomerGet(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "GET", path: "/customers/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CustomerGet(gomock.Any(), a, s3UUID1).Return(&cucustomer.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cucustomer.WebhookMessage{})},
	})
}

func Test_PutCustomersId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "PUT", path: "/customers/00000000-0000-4000-8000-0000000000a1", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "PUT", path: "/customers/00000000-0000-0000-0000-000000000000", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "PUT", path: "/customers/00000000-0000-4000-8000-0000000000a1", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	})
}

func Test_DeleteCustomersId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "DELETE", path: "/customers/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "DELETE", path: "/customers/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "DELETE", path: "/customers/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CustomerDelete(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "DELETE", path: "/customers/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CustomerDelete(gomock.Any(), a, s3UUID1).Return(&cucustomer.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cucustomer.WebhookMessage{})},
	})
}

func Test_PutCustomersIdBillingAccountId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "PUT", path: "/customers/00000000-0000-4000-8000-0000000000a1/billing_account_id", body: `{"billing_account_id":"00000000-0000-4000-8000-0000000000a2"}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "PUT", path: "/customers/00000000-0000-0000-0000-000000000000/billing_account_id", body: `{"billing_account_id":"00000000-0000-4000-8000-0000000000a2"}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "PUT", path: "/customers/00000000-0000-4000-8000-0000000000a1/billing_account_id", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "nil_billing_account_id", method: "PUT", path: "/customers/00000000-0000-4000-8000-0000000000a1/billing_account_id", body: `{"billing_account_id":"00000000-0000-0000-0000-000000000000"}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "PUT", path: "/customers/00000000-0000-4000-8000-0000000000a1/billing_account_id", body: `{"billing_account_id":"00000000-0000-4000-8000-0000000000a2"}`, setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CustomerUpdateBillingAccountID(gomock.Any(), a, s3UUID1, s3UUID2).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
	})
}

func Test_PutCustomersIdMetadata_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "PUT", path: "/customers/00000000-0000-4000-8000-0000000000a1/metadata", body: `{"rtp_debug":true}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "PUT", path: "/customers/00000000-0000-0000-0000-000000000000/metadata", body: `{"rtp_debug":true}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "PUT", path: "/customers/00000000-0000-4000-8000-0000000000a1/metadata", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "service_error", method: "PUT", path: "/customers/00000000-0000-4000-8000-0000000000a1/metadata", body: `{"rtp_debug":true}`, setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CustomerUpdateMetadata(gomock.Any(), a, s3UUID1, cucustomer.Metadata{RTPDebug: true}).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
	})
}

func Test_PostCustomersIdFreeze_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/customers/00000000-0000-4000-8000-0000000000a1/freeze", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "POST", path: "/customers/00000000-0000-0000-0000-000000000000/freeze", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "POST", path: "/customers/00000000-0000-4000-8000-0000000000a1/freeze", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CustomerFreeze(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "POST", path: "/customers/00000000-0000-4000-8000-0000000000a1/freeze", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CustomerFreeze(gomock.Any(), a, s3UUID1).Return(&cucustomer.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cucustomer.WebhookMessage{})},
	})
}

func Test_PostCustomersIdRecover_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/customers/00000000-0000-4000-8000-0000000000a1/recover", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "POST", path: "/customers/00000000-0000-0000-0000-000000000000/recover", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "POST", path: "/customers/00000000-0000-4000-8000-0000000000a1/recover", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CustomerRecover(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "POST", path: "/customers/00000000-0000-4000-8000-0000000000a1/recover", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().CustomerRecover(gomock.Any(), a, s3UUID1).Return(&cucustomer.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cucustomer.WebhookMessage{})},
	})
}
