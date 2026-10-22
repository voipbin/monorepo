package server

import (
	"testing"

	uuid "github.com/gofrs/uuid"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	cmcontact "monorepo/bin-contact-manager/models/contact"

	"go.uber.org/mock/gomock"
)

func Test_GetServiceAgentsContactAddresses_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/service_agents/contact_addresses?page_size=20&page_token=tok", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "service_error", method: "GET", path: "/service_agents/contact_addresses?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentContactAddressList(gomock.Any(), a, map[string]any{}, "tok", uint64(20)).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "empty_list", method: "GET", path: "/service_agents/contact_addresses?page_size=20&page_token=tok", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentContactAddressList(gomock.Any(), a, map[string]any{}, "tok", uint64(20)).Return(nil, nil)
		}, wantCode: 200, wantBody: `{"result":[],"next_page_token":""}`},
	})
}

func Test_PostServiceAgentsContactAddresses_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/service_agents/contact_addresses", body: `{"type":"tel","target":"+15550001","name":"n"}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_json", method: "POST", path: "/service_agents/contact_addresses", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "primary_without_contact", method: "POST", path: "/service_agents/contact_addresses", body: `{"type":"tel","target":"+15550001","is_primary":true}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_IS_PRIMARY"},
		{name: "service_error", method: "POST", path: "/service_agents/contact_addresses", body: `{"type":"tel","target":"+15550001","name":"n"}`, setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentContactAddressCreateIndependent(gomock.Any(), a, uuid.Nil, "tel", "+15550001", false, "n", "").Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
	})
}

func Test_GetServiceAgentsContactAddressesId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "GET", path: "/service_agents/contact_addresses/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "GET", path: "/service_agents/contact_addresses/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "GET", path: "/service_agents/contact_addresses/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentContactAddressGet(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "GET", path: "/service_agents/contact_addresses/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentContactAddressGet(gomock.Any(), a, s3UUID1).Return(&cmcontact.Address{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cmcontact.Address{})},
	})
}

func Test_PutServiceAgentsContactAddressesId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "PUT", path: "/service_agents/contact_addresses/00000000-0000-4000-8000-0000000000a1", body: `{"name":"n"}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "PUT", path: "/service_agents/contact_addresses/00000000-0000-0000-0000-000000000000", body: `{"name":"n"}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "PUT", path: "/service_agents/contact_addresses/00000000-0000-4000-8000-0000000000a1", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "service_error", method: "PUT", path: "/service_agents/contact_addresses/00000000-0000-4000-8000-0000000000a1", body: `{"name":"n"}`, setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentContactAddressUpdateIndependent(gomock.Any(), a, s3UUID1, map[string]any{"name": "n"}).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
	})
}

func Test_DeleteServiceAgentsContactAddressesId_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "DELETE", path: "/service_agents/contact_addresses/00000000-0000-4000-8000-0000000000a1", noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "DELETE", path: "/service_agents/contact_addresses/00000000-0000-0000-0000-000000000000", wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "service_error", method: "DELETE", path: "/service_agents/contact_addresses/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentContactAddressDeleteIndependent(gomock.Any(), a, s3UUID1).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
		{name: "success", method: "DELETE", path: "/service_agents/contact_addresses/00000000-0000-4000-8000-0000000000a1", setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentContactAddressDeleteIndependent(gomock.Any(), a, s3UUID1).Return(&cmcontact.Address{}, nil)
		}, wantCode: 200, wantBody: s3JSON(&cmcontact.Address{})},
	})
}

func Test_PostServiceAgentsContactAddressesIdClaim_S3(t *testing.T) {
	s3Run(t, []s3Case{
		{name: "unauthenticated", method: "POST", path: "/service_agents/contact_addresses/00000000-0000-4000-8000-0000000000a1/claim", body: `{"contact_id":"00000000-0000-4000-8000-0000000000a2"}`, noAuth: true, wantStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED"},
		{name: "invalid_id", method: "POST", path: "/service_agents/contact_addresses/00000000-0000-0000-0000-000000000000/claim", body: `{"contact_id":"00000000-0000-4000-8000-0000000000a2"}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID"},
		{name: "invalid_json", method: "POST", path: "/service_agents/contact_addresses/00000000-0000-4000-8000-0000000000a1/claim", body: `{`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_JSON_BODY"},
		{name: "nil_contact_id", method: "POST", path: "/service_agents/contact_addresses/00000000-0000-4000-8000-0000000000a1/claim", body: `{"contact_id":"00000000-0000-0000-0000-000000000000"}`, wantStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_CONTACT_ID"},
		{name: "service_error", method: "POST", path: "/service_agents/contact_addresses/00000000-0000-4000-8000-0000000000a1/claim", body: `{"contact_id":"00000000-0000-4000-8000-0000000000a2"}`, setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentContactAddressClaim(gomock.Any(), a, s3UUID1, s3UUID2).Return(nil, s3SvcErr())
		}, wantStatus: cerrors.StatusNotFound, wantReason: "S3_SVC_NOT_FOUND"},
	})
}
