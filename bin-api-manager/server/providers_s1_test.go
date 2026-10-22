package server

import (
	"fmt"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonrequesthandler "monorepo/bin-common-handler/pkg/requesthandler"
	rmprovider "monorepo/bin-route-manager/models/provider"

	"go.uber.org/mock/gomock"
)

const s1ProviderID = "7e2a1d4f-0003-11f0-8c3b-5a7b2d3e4f03"

func Test_GetProviders_Errors(t *testing.T) {
	s1Run(t, s1ListCases("/providers", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string, err error) {
		m.EXPECT().ProviderList(gomock.Any(), a, size, token).Return(nil, err)
	}))
}

func Test_PostProviders_Errors(t *testing.T) {
	okBody := `{"type":"sip","hostname":"sip.example.com","tech_prefix":"pre","tech_postfix":"post","tech_headers":{"X-A":"b"},"name":"n","detail":"d","codecs":"ulaw"}`
	s1Run(t, []s1Case{
		s1Unauth("POST", "/providers", okBody),
		s1BadJSON("POST", "/providers"),
		{
			name:       "non-string tech_headers value is rejected before the servicehandler",
			method:     "POST",
			path:       "/providers",
			body:       `{"type":"sip","hostname":"sip.example.com","tech_prefix":"","tech_postfix":"","tech_headers":{"X-A":1},"name":"n","detail":"d"}`,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_ARGUMENT",
		},
		s1PermDenied("POST", "/providers", okBody, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ProviderCreate(gomock.Any(), a, rmprovider.Type("sip"), "sip.example.com", "pre", "post", map[string]string{"X-A": "b"}, "n", "d", "ulaw").Return(nil, err)
		}),
	})
}

func Test_DeleteProvidersId_Errors(t *testing.T) {
	path := "/providers/" + s1ProviderID
	expect := func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
		m.EXPECT().ProviderDelete(gomock.Any(), a, s1ID(s1ProviderID)).Return(nil, err)
	}
	s1Run(t, []s1Case{
		s1Unauth("DELETE", path, ""),
		s1BadID("DELETE", "/providers/not-a-uuid", ""),
		s1PermDenied("DELETE", path, "", expect),
	})
}

func Test_GetProvidersId_Errors(t *testing.T) {
	path := "/providers/" + s1ProviderID
	expect := func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
		m.EXPECT().ProviderGet(gomock.Any(), a, s1ID(s1ProviderID)).Return(nil, err)
	}
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1BadID("GET", "/providers/not-a-uuid", ""),
		s1NotFound("GET", path, "", expect),
	})
}

func Test_PutProvidersId_Errors(t *testing.T) {
	path := "/providers/" + s1ProviderID
	body := `{"type":"sip","tech_headers":{"X-A":"b"},"name":"n2"}`
	typ := rmprovider.Type("sip")
	headers := map[string]string{"X-A": "b"}
	name := "n2"
	s1Run(t, []s1Case{
		s1Unauth("PUT", path, body),
		s1BadJSON("PUT", path),
		{
			name:       "non-string tech_headers value is rejected before the servicehandler",
			method:     "PUT",
			path:       path,
			body:       `{"tech_headers":{"X-A":true}}`,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_ARGUMENT",
		},
		s1PermDenied("PUT", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ProviderUpdate(gomock.Any(), a, s1ID(s1ProviderID), &typ, nil, nil, nil, &headers, &name, nil, nil).Return(nil, err)
		}),
	})
}

func Test_PostProvidersSetup(t *testing.T) {
	body := `{"carrier":"telnyx","name":"n","detail":"d","credentials":{"api_key":"key-123"}}`
	bodyNoDetail := `{"carrier":"telnyx","name":"n","credentials":{"api_key":"key-123"}}`
	typedErr := cerrors.PermissionDenied("route-manager", "CARRIER_FORBIDDEN", "carrier says no")

	tests := []s1Case{
		s1Unauth("POST", "/providers/setup", body),
		s1BadJSON("POST", "/providers/setup"),
		{
			name:   "typed upstream error is passed through verbatim",
			method: "POST",
			path:   "/providers/setup",
			body:   body,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ProviderSetup(gomock.Any(), a, "telnyx", "n", "d", "key-123").Return(nil, typedErr)
			},
			wantStatus: cerrors.StatusPermissionDenied,
			wantReason: "CARRIER_FORBIDDEN",
		},
		{
			name:   "legacy 422 is mapped to CARRIER_CREDENTIALS_REJECTED",
			method: "POST",
			path:   "/providers/setup",
			body:   bodyNoDetail,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ProviderSetup(gomock.Any(), a, "telnyx", "n", "", "key-123").Return(nil, fmt.Errorf("setup: %w", commonrequesthandler.ErrUnprocessableEntity))
			},
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "CARRIER_CREDENTIALS_REJECTED",
		},
		{
			name:   "other errors go through the generic translator",
			method: "POST",
			path:   "/providers/setup",
			body:   body,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ProviderSetup(gomock.Any(), a, "telnyx", "n", "d", "key-123").Return(nil, fmt.Errorf("unexpected"))
			},
			wantStatus: cerrors.StatusInternal,
			wantReason: "INTERNAL",
		},
	}
	s1Run(t, tests)
}
