package server

import (
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	cmcontact "monorepo/bin-contact-manager/models/contact"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

const (
	s1AddressID = "65193cde-0012-11f0-9f22-496ab6c72d12"
	s1ContactID = "762a4def-0013-11f0-8033-5a7bc7d83e13"
	s1NilUUID   = "00000000-0000-0000-0000-000000000000"
)

func s1Address() *cmcontact.Address {
	return &cmcontact.Address{
		ID:         s1ID(s1AddressID),
		CustomerID: s1ID(s1CustomerID),
		ContactID:  s1ID(s1ContactID),
	}
}

func Test_GetContactAddresses_Errors(t *testing.T) {
	addr := s1Address()
	s1Run(t, []s1Case{
		s1Unauth("GET", "/contact_addresses", ""),
		{
			name:   "service error is propagated with filters, token and size",
			method: "GET",
			path:   "/contact_addresses?contact_id=" + s1ContactID + "&type=tel&page_token=tok&page_size=7",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ContactAddressList(gomock.Any(), a, map[string]any{"contact_id": s1ID(s1ContactID), "type": "tel"}, "tok", uint64(7)).Return(nil, s1ServiceUnavailable())
			},
			wantStatus: cerrors.StatusUnavailable,
			wantReason: "SERVICE_UNAVAILABLE",
		},
		{
			name:   "nil contact_id is not added to the filters and default size is 20",
			method: "GET",
			path:   "/contact_addresses?contact_id=" + s1NilUUID,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ContactAddressList(gomock.Any(), a, map[string]any{}, "", uint64(20)).Return(nil, nil)
			},
			wantBody: `{"result":[],"next_page_token":""}`,
		},
		{
			name:   "items are returned with a token from the last tm_create",
			method: "GET",
			path:   "/contact_addresses",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				withTime := *addr
				withTime.TMCreate = s1Time("2020-09-20T03:23:20.995000Z")
				m.EXPECT().ContactAddressList(gomock.Any(), a, map[string]any{}, "", uint64(20)).Return([]cmcontact.Address{withTime}, nil)
			},
			wantBody: s1ContactAddressListBody(t),
		},
	})
}

func Test_PostContactAddresses_Errors(t *testing.T) {
	body := `{"contact_id":"` + s1ContactID + `","type":"tel","target":"+15555550100","is_primary":true,"name":"n","detail":"d"}`
	addr := s1Address()
	s1Run(t, []s1Case{
		s1Unauth("POST", "/contact_addresses", body),
		s1BadJSON("POST", "/contact_addresses"),
		{
			name:       "unresolved address cannot be primary",
			method:     "POST",
			path:       "/contact_addresses",
			body:       `{"type":"tel","target":"+15555550100","is_primary":true}`,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_IS_PRIMARY",
		},
		s1PermDenied("POST", "/contact_addresses", body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ContactAddressCreateIndependent(gomock.Any(), a, s1ID(s1ContactID), "tel", "+15555550100", true, "n", "d").Return(nil, err)
		}),
		{
			name:   "unresolved address success returns the address",
			method: "POST",
			path:   "/contact_addresses",
			body:   `{"type":"email","target":"user@example.com"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ContactAddressCreateIndependent(gomock.Any(), a, uuid.Nil, "email", "user@example.com", false, "", "").Return(addr, nil)
			},
			wantCode: 201,
			wantBody: s1JSON(t, addr),
		},
	})
}

func Test_ContactAddressesId_Errors(t *testing.T) {
	path := "/contact_addresses/" + s1AddressID
	nilPath := "/contact_addresses/" + s1NilUUID
	id := s1ID(s1AddressID)
	addr := s1Address()
	putBody := `{"target":"+15555550101","is_primary":false,"name":"n2","detail":"d2"}`
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", nilPath, ""),
		s1NotFound("GET", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ContactAddressGet(gomock.Any(), a, id).Return(nil, err)
		}),
		{
			name:   "get success returns the address",
			method: "GET",
			path:   path,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ContactAddressGet(gomock.Any(), a, id).Return(addr, nil)
			},
			wantBody: s1JSON(t, addr),
		},
		s1Unauth("PUT", path, putBody),
		s1BadID("PUT", nilPath, putBody),
		s1BadJSON("PUT", path),
		s1PermDenied("PUT", path, putBody, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ContactAddressUpdateIndependent(gomock.Any(), a, id, map[string]any{"target": "+15555550101", "is_primary": false, "name": "n2", "detail": "d2"}).Return(nil, err)
		}),
		s1PermDenied("PUT", path, `{}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ContactAddressUpdateIndependent(gomock.Any(), a, id, map[string]any{}).Return(nil, err)
		}),
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", nilPath, ""),
		s1PermDenied("DELETE", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ContactAddressDeleteIndependent(gomock.Any(), a, id).Return(nil, err)
		}),
	})
}

func Test_PostContactAddressesIdClaim_Errors(t *testing.T) {
	path := "/contact_addresses/" + s1AddressID + "/claim"
	body := `{"contact_id":"` + s1ContactID + `","force":true}`
	addr := s1Address()
	s1Run(t, []s1Case{
		s1Unauth("POST", path, body),
		s1BadID("POST", "/contact_addresses/"+s1NilUUID+"/claim", body),
		s1BadJSON("POST", path),
		{
			name:       "nil contact_id",
			method:     "POST",
			path:       path,
			body:       `{"contact_id":"` + s1NilUUID + `"}`,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_CONTACT_ID",
		},
		s1PermDenied("POST", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ContactAddressClaim(gomock.Any(), a, s1ID(s1AddressID), s1ID(s1ContactID), true).Return(nil, err)
		}),
		{
			name:   "force defaults to false",
			method: "POST",
			path:   path,
			body:   `{"contact_id":"` + s1ContactID + `"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ContactAddressClaim(gomock.Any(), a, s1ID(s1AddressID), s1ID(s1ContactID), false).Return(addr, nil)
			},
			wantBody: s1JSON(t, addr),
		},
	})
}

// s1ContactAddressListBody is the expected list body for the address with a
// tm_create of 2020-09-20T03:23:20.995Z: the item JSON plus the formatted token.
func s1ContactAddressListBody(t *testing.T) string {
	t.Helper()
	item := s1Address()
	item.TMCreate = s1Time("2020-09-20T03:23:20.995000Z")
	return `{"result":[` + s1JSON(t, item) + `],"next_page_token":"2020-09-20T03:23:20.995000Z"}`
}
