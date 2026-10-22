package server

import (
	"net/http"
	"testing"

	ammcpserver "monorepo/bin-ai-manager/models/mcpserver"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	bmaccount "monorepo/bin-billing-manager/models/account"
	bmbilling "monorepo/bin-billing-manager/models/billing"
	cmoutboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	cerrors "monorepo/bin-common-handler/models/errors"
	rmroute "monorepo/bin-route-manager/models/route"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_BillingAccount_Self(t *testing.T) {
	ret := s4Obj[bmaccount.WebhookMessage](s4ID1, nil)
	var cases []s4Case

	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetBillingAccount", method: http.MethodGet, path: "/billing_account",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().BillingAccountSelfGet(gomock.Any(), a)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PutBillingAccount", method: http.MethodPut, path: "/billing_account",
		body: `{"name":"acct","detail":"acct detail"}`, badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().BillingAccountSelfUpdateBasicInfo(gomock.Any(), a, "acct", "acct detail")
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4Case{
		name: "PutBillingAccount/omitted_fields_become_empty", method: http.MethodPut, path: "/billing_account",
		body: `{}`,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().BillingAccountSelfUpdateBasicInfo(gomock.Any(), a, "", "").Return(ret, nil)
		},
		wantStatus: 200, wantBody: ret,
	})
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PutBillingAccountPaymentInfo", method: http.MethodPut, path: "/billing_account/payment_info",
		body: `{"payment_type":"prepaid","payment_method":"credit card"}`, badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().BillingAccountSelfUpdatePaymentInfo(gomock.Any(), a, bmaccount.PaymentType("prepaid"), bmaccount.PaymentMethod("credit card"))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4Case{
		name: "PutBillingAccountPaymentInfo/omitted_fields_use_none", method: http.MethodPut, path: "/billing_account/payment_info",
		body: `{}`,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().BillingAccountSelfUpdatePaymentInfo(gomock.Any(), a, bmaccount.PaymentTypeNone, bmaccount.PaymentMethodNone).Return(ret, nil)
		},
		wantStatus: 200, wantBody: ret,
	})
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostBillingAccountPaddlePortalSession", method: http.MethodPost, path: "/billing_account/paddle_portal_session",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().BillingAccountSelfCreatePaddlePortalSession(gomock.Any(), a)
		},
		ret: "https://portal.example.com/session/abc", zero: "", okStatus: 200,
		wantBody: map[string]any{"url": "https://portal.example.com/session/abc"},
	})...)
	s4RunCases(t, cases)
}

func Test_BillingAccounts_Admin(t *testing.T) {
	var cases []s4Case
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetBillingAccounts", path: "/billing_accounts",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().BillingAccountList(gomock.Any(), a, size, token, map[string]string{"deleted": "false"})
		},
		mk: s4MkList[bmaccount.Account],
	})...)

	ret := s4Obj[bmaccount.Account](s4ID1, nil)
	bad := "/billing_accounts/not-a-uuid"
	id := s4UUID(s4ID1)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetBillingAccountsId", method: http.MethodGet, path: "/billing_accounts/" + s4ID1,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().BillingAccountGet(gomock.Any(), a, id)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PutBillingAccountsId", method: http.MethodPut, path: "/billing_accounts/" + s4ID1,
		body: `{"name":"acct","detail":"acct detail"}`, badJSON: true,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().BillingAccountUpdateBasicInfo(gomock.Any(), a, id, "acct", "acct detail")
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4Case{
		name: "PutBillingAccountsId/omitted_fields_become_empty", method: http.MethodPut, path: "/billing_accounts/" + s4ID1,
		body: `{}`,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().BillingAccountUpdateBasicInfo(gomock.Any(), a, id, "", "").Return(ret, nil)
		},
		wantStatus: 200, wantBody: ret,
	})
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PutBillingAccountsIdPaymentInfo", method: http.MethodPut, path: "/billing_accounts/" + s4ID1 + "/payment_info",
		body: `{"payment_type":"prepaid","payment_method":"credit card"}`, badJSON: true,
		badIDPath: bad + "/payment_info", badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().BillingAccountUpdatePaymentInfo(gomock.Any(), a, id, bmaccount.PaymentType("prepaid"), bmaccount.PaymentMethod("credit card"))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4Case{
		name: "PutBillingAccountsIdPaymentInfo/omitted_fields_use_none", method: http.MethodPut, path: "/billing_accounts/" + s4ID1 + "/payment_info",
		body: `{}`,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().BillingAccountUpdatePaymentInfo(gomock.Any(), a, id, bmaccount.PaymentTypeNone, bmaccount.PaymentMethodNone).Return(ret, nil)
		},
		wantStatus: 200, wantBody: ret,
	})

	// balance force endpoints: the balance is converted to micro-units.
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostBillingAccountsIdBalanceAddForce", method: http.MethodPost, path: "/billing_accounts/" + s4ID1 + "/balance_add_force",
		body: `{"balance":12.25}`, badJSON: true,
		badIDPath: bad + "/balance_add_force", badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().BillingAccountAddBalanceForce(gomock.Any(), a, id, int64(12250000))
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostBillingAccountsIdBalanceSubtractForce", method: http.MethodPost, path: "/billing_accounts/" + s4ID1 + "/balance_subtract_force",
		body: `{"balance":12.25}`, badJSON: true,
		badIDPath: bad + "/balance_subtract_force", badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().BillingAccountSubtractBalanceForce(gomock.Any(), a, id, int64(12250000))
		},
		ret: ret, okStatus: 200,
	})...)
	for _, ep := range []struct {
		name, path string
		add        bool
	}{
		{"PostBillingAccountsIdBalanceAddForce", "/balance_add_force", true},
		{"PostBillingAccountsIdBalanceSubtractForce", "/balance_subtract_force", false},
	} {
		ep := ep
		cases = append(cases,
			s4Case{
				name: ep.name + "/negative_balance_rejected", method: http.MethodPost, path: "/billing_accounts/" + s4ID1 + ep.path,
				body:          `{"balance":-0.5}`,
				wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT",
			},
			s4Case{
				name: ep.name + "/missing_balance_is_zero", method: http.MethodPost, path: "/billing_accounts/" + s4ID1 + ep.path,
				body: `{}`,
				setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
					if ep.add {
						m.EXPECT().BillingAccountAddBalanceForce(gomock.Any(), a, id, int64(0)).Return(ret, nil)
						return
					}
					m.EXPECT().BillingAccountSubtractBalanceForce(gomock.Any(), a, id, int64(0)).Return(ret, nil)
				},
				wantStatus: 200, wantBody: ret,
			},
		)
	}
	s4RunCases(t, cases)
}

func Test_Billings_Matrix(t *testing.T) {
	var cases []s4Case
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetBillings", path: "/billings",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().BillingList(gomock.Any(), a, size, token)
		},
		mk: s4MkList[bmbilling.WebhookMessage],
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetBillingsBillingId", method: http.MethodGet, path: "/billings/" + s4ID1,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().BillingGet(gomock.Any(), a, s4UUID(s4ID1))
		},
		ret: s4Obj[bmbilling.WebhookMessage](s4ID1, nil), okStatus: 200,
	})...)
	s4RunCases(t, cases)
}

func Test_Mcpservers_Matrix(t *testing.T) {
	var cases []s4Case
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetMcpservers", path: "/mcpservers",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().McpServerGetsByCustomerID(gomock.Any(), a, size, token)
		},
		mk: s4MkList[ammcpserver.WebhookMessage],
	})...)

	ret := s4Obj[ammcpserver.WebhookMessage](s4ID1, nil)
	id := s4UUID(s4ID1)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetMcpserversId", method: http.MethodGet, path: "/mcpservers/" + s4ID1,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().McpServerGet(gomock.Any(), a, id)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "DeleteMcpserversId", method: http.MethodDelete, path: "/mcpservers/" + s4ID1,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().McpServerDelete(gomock.Any(), a, id)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostMcpservers", method: http.MethodPost, path: "/mcpservers",
		body:    `{"name":"srv","detail":"srv detail","url":"https://mcp.example.com/sse","auth_type":"api_key","api_key_header":"X-Api-Key","secret":"s3cr3t"}`,
		badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().McpServerCreate(gomock.Any(), a, "srv", "srv detail", "https://mcp.example.com/sse", ammcpserver.AuthTypeAPIKey, "X-Api-Key", "s3cr3t")
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4Case{
		name: "PostMcpservers/optional_fields_default", method: http.MethodPost, path: "/mcpservers",
		body: `{"name":"srv","url":"https://mcp.example.com/sse"}`,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().McpServerCreate(gomock.Any(), a, "srv", "", "https://mcp.example.com/sse", ammcpserver.AuthTypeNone, "", "").Return(ret, nil)
		},
		wantStatus: 200, wantBody: ret,
	})

	name, detail, url, header, secret := "srv2", "d2", "https://mcp.example.com/v2", "X-K", "sec2"
	status := ammcpserver.Status("active")
	authType := ammcpserver.AuthTypeBearer
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PutMcpserversId", method: http.MethodPut, path: "/mcpservers/" + s4ID1,
		body:    `{"name":"srv2","detail":"d2","url":"https://mcp.example.com/v2","status":"active","auth_type":"bearer","api_key_header":"X-K","secret":"sec2"}`,
		badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().McpServerUpdate(gomock.Any(), a, id, &name, &detail, &url, &status, &authType, &header, &secret)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4Case{
		name: "PutMcpserversId/empty_body_leaves_everything_nil", method: http.MethodPut, path: "/mcpservers/" + s4ID1,
		body: `{}`,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().McpServerUpdate(gomock.Any(), a, id, (*string)(nil), (*string)(nil), (*string)(nil),
				(*ammcpserver.Status)(nil), (*ammcpserver.AuthType)(nil), (*string)(nil), (*string)(nil)).Return(ret, nil)
		},
		wantStatus: 200, wantBody: ret,
	})
	s4RunCases(t, cases)
}

const s4McpReturnURL = "https://admin.voipbin.net/#/resources/mcpservers/oauth-return"

func Test_McpserversOauth_Matrix(t *testing.T) {
	var cases []s4Case
	id := s4UUID(s4ID2)

	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostMcpserversOauthStart", method: http.MethodPost, path: "/mcpservers/oauth/start",
		body:    `{"vendor":"notion","mcp_server_id":"` + s4ID2 + `"}`,
		badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().McpOAuthStart(gomock.Any(), a, "notion", &id)
		},
		ret: nil, okStatus: 200,
	})[:2]...)
	cases = append(cases,
		s4Case{
			name: "PostMcpserversOauthStart/service_error", method: http.MethodPost, path: "/mcpservers/oauth/start",
			body: `{"vendor":"notion","mcp_server_id":"` + s4ID2 + `"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().McpOAuthStart(gomock.Any(), a, "notion", &id).Return("", "", s4ErrDenied)
			},
			wantErrStatus: cerrors.StatusPermissionDenied, wantReason: "PERMISSION_DENIED",
		},
		s4Case{
			name: "PostMcpserversOauthStart/success_with_server_id", method: http.MethodPost, path: "/mcpservers/oauth/start",
			body: `{"vendor":"notion","mcp_server_id":"` + s4ID2 + `"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().McpOAuthStart(gomock.Any(), a, "notion", &id).Return("https://auth.example.com/authorize?x=1", "link-token-1", nil)
			},
			wantStatus: 200, wantBody: map[string]any{"authorize_url": "https://auth.example.com/authorize?x=1", "link_token": "link-token-1"},
		},
		s4Case{
			name: "PostMcpserversOauthStart/success_without_server_id", method: http.MethodPost, path: "/mcpservers/oauth/start",
			body: `{"vendor":"notion"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().McpOAuthStart(gomock.Any(), a, "notion", (*uuid.UUID)(nil)).Return("https://auth.example.com/authorize", "link-token-2", nil)
			},
			wantStatus: 200, wantBody: map[string]any{"authorize_url": "https://auth.example.com/authorize", "link_token": "link-token-2"},
		},
	)

	ret := s4Obj[ammcpserver.WebhookMessage](s4ID1, nil)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostMcpserversOauthComplete", method: http.MethodPost, path: "/mcpservers/oauth/complete",
		body:    `{"state":"st-1","code":"code-1"}`,
		badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().McpOAuthComplete(gomock.Any(), a, "st-1", "code-1")
		},
		ret: ret, okStatus: 200,
	})...)

	// Public callback relay: never requires auth, always 302s.
	redirect := func(name, query string, exists bool, svcErr error, wantQuery string) s4Case {
		return s4Case{
			name: "GetMcpserversOauthCallback/" + name, method: http.MethodGet,
			path: "/mcpservers/oauth/callback?" + query, noAuth: true,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().McpOAuthCallback(gomock.Any(), "st-9").Return(exists, svcErr)
			},
			wantStatus:  302,
			wantHeaders: map[string]string{"Location": s4McpReturnURL + "?" + wantQuery},
		}
	}
	cases = append(cases,
		redirect("state_exists_with_code", "state=st-9&code=c-9", true, nil, "code=c-9&state=st-9"),
		redirect("state_exists_without_code", "state=st-9", true, nil, "state=st-9"),
		redirect("state_exists_with_empty_code", "state=st-9&code=", true, nil, "state=st-9"),
		redirect("unknown_state", "state=st-9&code=c-9", false, nil, "error=invalid_state&state=st-9"),
		redirect("service_error_fails_closed", "state=st-9&code=c-9", true, s4ErrDenied, "error=invalid_state&state=st-9"),
	)
	s4RunCases(t, cases)
}

func Test_OutboundConfigs_Matrix(t *testing.T) {
	var cases []s4Case
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetOutboundConfigs", path: "/outbound_configs",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().OutboundConfigList(gomock.Any(), a, size, token)
		},
		mk: s4MkList[cmoutboundconfig.WebhookMessage],
	})...)

	ret := s4Obj[cmoutboundconfig.WebhookMessage](s4ID1, nil)
	id := s4UUID(s4ID1)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetOutboundConfigsId", method: http.MethodGet, path: "/outbound_configs/" + s4ID1,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().OutboundConfigGet(gomock.Any(), a, id)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "DeleteOutboundConfigsId", method: http.MethodDelete, path: "/outbound_configs/" + s4ID1,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().OutboundConfigDelete(gomock.Any(), a, id)
		},
		ret: ret, okStatus: 200,
	})...)

	name, detail, codecs := "cfg", "cfg detail", "ulaw,alaw"
	whitelist := []string{"+15551230001", "+15551230002"}
	srcNum := s4UUID(s4ID3)
	wantReq := &cmoutboundconfig.UpdateRequest{
		Name: &name, Detail: &detail, DestinationWhitelist: &whitelist, Codecs: &codecs, DefaultOutgoingSourceNumberID: &srcNum,
	}
	body := `{"name":"cfg","detail":"cfg detail","destination_whitelist":["+15551230001","+15551230002"],"codecs":"ulaw,alaw","default_outgoing_source_number_id":"` + s4ID3 + `"}`
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostOutboundConfigs", method: http.MethodPost, path: "/outbound_configs", body: body, badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().OutboundConfigCreate(gomock.Any(), a, wantReq)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PutOutboundConfigsId", method: http.MethodPut, path: "/outbound_configs/" + s4ID1, body: body, badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().OutboundConfigUpdate(gomock.Any(), a, id, wantReq)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4Case{
		name: "PostOutboundConfigs/empty_body_gives_empty_request", method: http.MethodPost, path: "/outbound_configs",
		body: `{}`,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().OutboundConfigCreate(gomock.Any(), a, &cmoutboundconfig.UpdateRequest{}).Return(ret, nil)
		},
		wantStatus: 200, wantBody: ret,
	})
	s4RunCases(t, cases)
}

func Test_Routes_Matrix(t *testing.T) {
	var cases []s4Case
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetRoutes", path: "/routes",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().RouteList(gomock.Any(), a, size, token)
		},
		mk: s4MkList[rmroute.Route],
	})...)
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetRoutes_customer_filter", path: "/routes?customer_id=" + s4ID2, sep: "&",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().RouteGetsByCustomerID(gomock.Any(), a, s4UUID(s4ID2), size, token)
		},
		mk: s4MkList[rmroute.Route],
	})...)

	ret := s4Obj[rmroute.Route](s4ID1, nil)
	id := s4UUID(s4ID1)
	bad := "/routes/not-a-uuid"
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetRoutesId", method: http.MethodGet, path: "/routes/" + s4ID1,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().RouteGet(gomock.Any(), a, id)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "DeleteRoutesId", method: http.MethodDelete, path: "/routes/" + s4ID1,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().RouteDelete(gomock.Any(), a, id)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostRoutes", method: http.MethodPost, path: "/routes",
		body:    `{"customer_id":"` + s4ID2 + `","name":"rt","detail":"rt detail","provider_id":"` + s4ID3 + `","priority":5,"target":"+15551230001"}`,
		badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().RouteCreate(gomock.Any(), a, s4UUID(s4ID2), "rt", "rt detail", s4UUID(s4ID3), 5, "+15551230001")
		},
		ret: ret, okStatus: 200,
	})...)

	name, detail, target := "rt2", "rt2 detail", "+15551230002"
	prov := s4UUID(s4ID3)
	prio := 9
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PutRoutesId", method: http.MethodPut, path: "/routes/" + s4ID1,
		body:      `{"name":"rt2","detail":"rt2 detail","provider_id":"` + s4ID3 + `","priority":9,"target":"+15551230002"}`,
		badJSON:   true,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().RouteUpdate(gomock.Any(), a, id, &name, &detail, &prov, &prio, &target)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases,
		s4Case{
			name: "PutRoutesId/invalid_provider_id", method: http.MethodPut, path: "/routes/" + s4ID1,
			body:          `{"provider_id":"not-a-uuid"}`,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT",
		},
		s4Case{
			name: "PutRoutesId/empty_body_all_nil", method: http.MethodPut, path: "/routes/" + s4ID1,
			body: `{}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().RouteUpdate(gomock.Any(), a, id, (*string)(nil), (*string)(nil), (*uuid.UUID)(nil), (*int)(nil), (*string)(nil)).Return(ret, nil)
			},
			wantStatus: 200, wantBody: ret,
		},
	)
	s4RunCases(t, cases)
}
