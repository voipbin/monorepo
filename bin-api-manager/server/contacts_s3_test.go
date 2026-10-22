package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	cmcontact "monorepo/bin-contact-manager/models/contact"

	"go.uber.org/mock/gomock"
)

func Test_GetContacts_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/contacts?page_size=20&page_token=tok", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "GET", path: "/contacts?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ContactList(gomock.Any(), a, uint64(20), "tok", map[string]string{"customer_id": "00000000-0000-4000-8000-0000000000c1", "deleted": "false"}).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "page_size_capped", method: "GET", path: "/contacts?page_size=1000", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ContactList(gomock.Any(), a, uint64(100), "", map[string]string{"customer_id": "00000000-0000-4000-8000-0000000000c1", "deleted": "false"}).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "empty_list", method: "GET", path: "/contacts?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ContactList(gomock.Any(), a, uint64(20), "tok", map[string]string{"customer_id": "00000000-0000-4000-8000-0000000000c1", "deleted": "false"}).Return(nil, nil)
		}, wantCode: 200, wantBody: `{"result":[],"next_page_token":""}`},
	})
}

func Test_PostContacts_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/contacts", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_json", method: "POST", path: "/contacts", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "address_missing_type", method: "POST", path: "/contacts", body: `{"first_name":"f","addresses":[{"target":"x"}]}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ADDRESS_TYPE"},
		{name: "address_bad_type", method: "POST", path: "/contacts", body: `{"first_name":"f","addresses":[{"type":"fax","target":"x"}]}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ADDRESS_TYPE"},
		{name: "address_missing_target", method: "POST", path: "/contacts", body: `{"first_name":"f","addresses":[{"type":"tel"}]}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ADDRESS_TARGET"},
	})
}

func Test_GetContactsId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/contacts/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "GET", path: "/contacts/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "GET", path: "/contacts/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ContactGet(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "GET", path: "/contacts/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ContactGet(gomock.Any(), a, s3UUID1).Return(&cmcontact.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cmcontact.WebhookMessage{})},
	})
}

func Test_PutContactsId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "PUT", path: "/contacts/00000000-0000-4000-8000-0000000000a1", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "PUT", path: "/contacts/00000000-0000-0000-0000-000000000000", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "PUT", path: "/contacts/00000000-0000-4000-8000-0000000000a1", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	})
}

func Test_DeleteContactsId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "DELETE", path: "/contacts/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "DELETE", path: "/contacts/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "DELETE", path: "/contacts/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ContactDelete(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "DELETE", path: "/contacts/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ContactDelete(gomock.Any(), a, s3UUID1).Return(&cmcontact.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cmcontact.WebhookMessage{})},
	})
}

func Test_GetContactsLookup_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/contacts/lookup?phone=%2B15551230001", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "missing_phone_and_email", method: "GET", path: "/contacts/lookup", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT"},
		{name: "service_error", method: "GET", path: "/contacts/lookup?phone=%2B15551230001", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ContactLookup(gomock.Any(), a, "+15551230001", "").Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
	})
}

func Test_PostContactsIdAddresses_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/contacts/00000000-0000-4000-8000-0000000000a1/addresses", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "POST", path: "/contacts/00000000-0000-0000-0000-000000000000/addresses", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "POST", path: "/contacts/00000000-0000-4000-8000-0000000000a1/addresses", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	})
}

func Test_PutContactsIdAddressesAddressId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "PUT", path: "/contacts/00000000-0000-4000-8000-0000000000a1/addresses/00000000-0000-4000-8000-0000000000a2", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "PUT", path: "/contacts/00000000-0000-0000-0000-000000000000/addresses/00000000-0000-4000-8000-0000000000a2", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_address_id", method: "PUT", path: "/contacts/00000000-0000-4000-8000-0000000000a1/addresses/00000000-0000-0000-0000-000000000000", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "PUT", path: "/contacts/00000000-0000-4000-8000-0000000000a1/addresses/00000000-0000-4000-8000-0000000000a2", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
	})
}

func Test_DeleteContactsIdAddressesAddressId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "DELETE", path: "/contacts/00000000-0000-4000-8000-0000000000a1/addresses/00000000-0000-4000-8000-0000000000a2", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "DELETE", path: "/contacts/00000000-0000-0000-0000-000000000000/addresses/00000000-0000-4000-8000-0000000000a2", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_address_id", method: "DELETE", path: "/contacts/00000000-0000-4000-8000-0000000000a1/addresses/00000000-0000-0000-0000-000000000000", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "DELETE", path: "/contacts/00000000-0000-4000-8000-0000000000a1/addresses/00000000-0000-4000-8000-0000000000a2", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ContactAddressDelete(gomock.Any(), a, s3UUID1, s3UUID2).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "DELETE", path: "/contacts/00000000-0000-4000-8000-0000000000a1/addresses/00000000-0000-4000-8000-0000000000a2", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ContactAddressDelete(gomock.Any(), a, s3UUID1, s3UUID2).Return(&cmcontact.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cmcontact.WebhookMessage{})},
	})
}

func Test_PostContactsIdTags_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/contacts/00000000-0000-4000-8000-0000000000a1/tags", body: `{}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "POST", path: "/contacts/00000000-0000-0000-0000-000000000000/tags", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "POST", path: "/contacts/00000000-0000-4000-8000-0000000000a1/tags", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "invalid_tag_id", method: "POST", path: "/contacts/00000000-0000-4000-8000-0000000000a1/tags", body: `{"tag_id":"00000000-0000-0000-0000-000000000000"}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
	})
}

func Test_DeleteContactsIdTagsTagId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "DELETE", path: "/contacts/00000000-0000-4000-8000-0000000000a1/tags/00000000-0000-4000-8000-0000000000a3", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "DELETE", path: "/contacts/00000000-0000-0000-0000-000000000000/tags/00000000-0000-4000-8000-0000000000a3", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_tag_id", method: "DELETE", path: "/contacts/00000000-0000-4000-8000-0000000000a1/tags/00000000-0000-0000-0000-000000000000", body: `{}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "DELETE", path: "/contacts/00000000-0000-4000-8000-0000000000a1/tags/00000000-0000-4000-8000-0000000000a3", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ContactTagRemove(gomock.Any(), a, s3UUID1, s3UUID3).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "DELETE", path: "/contacts/00000000-0000-4000-8000-0000000000a1/tags/00000000-0000-4000-8000-0000000000a3", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ContactTagRemove(gomock.Any(), a, s3UUID1, s3UUID3).Return(&cmcontact.WebhookMessage{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cmcontact.WebhookMessage{})},
	})
}
