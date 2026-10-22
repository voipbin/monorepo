package server

import (
	"net/http"
	"testing"

	amaipromptproposal "monorepo/bin-ai-manager/models/aipromptproposal"
	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/servicehandler"
	cmcall "monorepo/bin-call-manager/models/call"
	cacampaigncall "monorepo/bin-campaign-manager/models/campaigncall"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	mmmessage "monorepo/bin-message-manager/models/message"
	tmtag "monorepo/bin-tag-manager/models/tag"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Aipromptproposals_Matrix(t *testing.T) {
	var cases []s4Case
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetAipromptproposals", path: "/aipromptproposals",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().AIPromptProposalGetsByCustomerID(gomock.Any(), a, size, token, uuid.Nil, amaipromptproposal.Status(""))
		},
		mk: s4MkList[amaipromptproposal.WebhookMessage],
	})...)
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetAipromptproposals_filters", path: "/aipromptproposals?ai_id=" + s4ID2 + "&status=pending", sep: "&",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().AIPromptProposalGetsByCustomerID(gomock.Any(), a, size, token, s4UUID(s4ID2), amaipromptproposal.Status("pending"))
		},
		mk: s4MkList[amaipromptproposal.WebhookMessage],
	})...)

	ret := s4Obj[amaipromptproposal.WebhookMessage](s4ID1, nil)
	id := s4UUID(s4ID1)
	for _, ep := range []struct {
		name, method, path string
		call               func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call
	}{
		{"GetAipromptproposalsId", http.MethodGet, "/aipromptproposals/" + s4ID1, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().AIPromptProposalGet(gomock.Any(), a, id)
		}},
		{"PostAipromptproposalsIdAccept", http.MethodPost, "/aipromptproposals/" + s4ID1 + "/accept", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().AIPromptProposalAccept(gomock.Any(), a, id)
		}},
		{"PostAipromptproposalsIdReject", http.MethodPost, "/aipromptproposals/" + s4ID1 + "/reject", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().AIPromptProposalReject(gomock.Any(), a, id)
		}},
		{"DeleteAipromptproposalsId", http.MethodDelete, "/aipromptproposals/" + s4ID1, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().AIPromptProposalDelete(gomock.Any(), a, id)
		}},
	} {
		cases = append(cases, s4ItemCases(s4ItemEP{name: ep.name, method: ep.method, path: ep.path, call: ep.call, ret: ret, okStatus: 200})...)
	}

	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostAipromptproposals", method: http.MethodPost, path: "/aipromptproposals",
		body:    `{"ai_id":"` + s4ID1 + `","audit_ids":["` + s4ID2 + `","` + s4ID3 + `"],"language":"ko-KR"}`,
		badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().AIPromptProposalCreate(gomock.Any(), a, id, []uuid.UUID{s4UUID(s4ID2), s4UUID(s4ID3)}, "ko-KR")
		},
		ret: ret, okStatus: 202,
	})...)
	cases = append(cases,
		s4Case{
			name: "PostAipromptproposals/invalid_ai_id", method: http.MethodPost, path: "/aipromptproposals",
			body:          `{"ai_id":"nope","audit_ids":["` + s4ID2 + `"]}`,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID",
		},
		s4Case{
			name: "PostAipromptproposals/invalid_audit_id", method: http.MethodPost, path: "/aipromptproposals",
			body:          `{"ai_id":"` + s4ID1 + `","audit_ids":["` + s4ID2 + `","nope"]}`,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID",
		},
		s4Case{
			name: "PostAipromptproposals/language_defaults_to_empty", method: http.MethodPost, path: "/aipromptproposals",
			body: `{"ai_id":"` + s4ID1 + `","audit_ids":["` + s4ID2 + `"]}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().AIPromptProposalCreate(gomock.Any(), a, id, []uuid.UUID{s4UUID(s4ID2)}, "").Return(ret, nil)
			},
			wantStatus: 202, wantBody: ret,
		},
	)
	s4RunCases(t, cases)
}

func Test_Messages_Matrix(t *testing.T) {
	var cases []s4Case
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetMessages", path: "/messages",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().MessageList(gomock.Any(), a, size, token)
		},
		mk: s4MkList[mmmessage.WebhookMessage],
	})...)

	ret := s4Obj[mmmessage.WebhookMessage](s4ID1, nil)
	id := s4UUID(s4ID1)
	bad := "/messages/not-a-uuid"
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetMessagesId", method: http.MethodGet, path: "/messages/" + s4ID1,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().MessageGet(gomock.Any(), a, id)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "DeleteMessagesId", method: http.MethodDelete, path: "/messages/" + s4ID1,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().MessageDelete(gomock.Any(), a, id)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostMessages", method: http.MethodPost, path: "/messages",
		body:    `{"source":{"type":"tel","target":"+15551230001"},"destinations":[{"type":"tel","target":"+15551230002"},{"type":"tel","target":"+15551230003"}],"text":"hello"}`,
		badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().MessageSend(gomock.Any(), a,
				&commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15551230001"},
				[]commonaddress.Address{
					{Type: commonaddress.TypeTel, Target: "+15551230002"},
					{Type: commonaddress.TypeTel, Target: "+15551230003"},
				}, "hello")
		},
		ret: ret, okStatus: 200,
	})...)
	s4RunCases(t, cases)
}

func Test_Campaigncalls_Matrix(t *testing.T) {
	var cases []s4Case
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetCampaigncalls", path: "/campaigncalls",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().CampaigncallList(gomock.Any(), a, size, token)
		},
		mk: s4MkList[cacampaigncall.WebhookMessage],
	})...)

	ret := s4Obj[cacampaigncall.WebhookMessage](s4ID1, nil)
	id := s4UUID(s4ID1)
	bad := "/campaigncalls/not-a-uuid"
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetCampaigncallsId", method: http.MethodGet, path: "/campaigncalls/" + s4ID1,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().CampaigncallGet(gomock.Any(), a, id)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "DeleteCampaigncallsId", method: http.MethodDelete, path: "/campaigncalls/" + s4ID1,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().CampaigncallDelete(gomock.Any(), a, id)
		},
		ret: ret, okStatus: 200,
	})...)
	s4RunCases(t, cases)
}

func Test_ServiceAgentsTagsAndCalls_Matrix(t *testing.T) {
	var cases []s4Case
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetServiceAgentsTags", path: "/service_agents/tags",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().ServiceAgentTagList(gomock.Any(), a, size, token)
		},
		mk: s4MkList[tmtag.WebhookMessage],
	})...)
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetServiceAgentsCalls", path: "/service_agents/calls",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().ServiceAgentCallList(gomock.Any(), a, size, token)
		},
		mk: s4MkList[cmcall.WebhookMessage],
	})...)

	id := s4UUID(s4ID1)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetServiceAgentsTagsId", method: http.MethodGet, path: "/service_agents/tags/" + s4ID1,
		badIDPath: "/service_agents/tags/not-a-uuid", badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().ServiceAgentTagGet(gomock.Any(), a, id)
		},
		ret: s4Obj[tmtag.WebhookMessage](s4ID1, nil), okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetServiceAgentsCallsId", method: http.MethodGet, path: "/service_agents/calls/" + s4ID1,
		badIDPath: "/service_agents/calls/not-a-uuid", badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().ServiceAgentCallGet(gomock.Any(), a, id)
		},
		ret: s4Obj[cmcall.WebhookMessage](s4ID1, nil), okStatus: 200,
	})...)
	s4RunCases(t, cases)
}

func Test_ServiceAgentsContacts_Matrix(t *testing.T) {
	a0 := s4Agent()
	filters := map[string]string{"customer_id": a0.CustomerID.String(), "deleted": "false"}

	var cases []s4Case
	cases = append(cases, s4ListCases(s4ListEP{
		name: "GetServiceAgentsContacts", path: "/service_agents/contacts",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, size uint64, token string) *gomock.Call {
			return m.EXPECT().ServiceAgentContactList(gomock.Any(), a, size, token, filters)
		},
		mk: s4MkList[cmcontact.WebhookMessage],
	})...)

	ret := s4Obj[cmcontact.WebhookMessage](s4ID1, nil)
	id := s4UUID(s4ID1)
	bad := "/service_agents/contacts/not-a-uuid"
	base := "/service_agents/contacts/" + s4ID1

	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "GetServiceAgentsContactsId", method: http.MethodGet, path: base,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().ServiceAgentContactGet(gomock.Any(), a, id)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "DeleteServiceAgentsContactsId", method: http.MethodDelete, path: base,
		badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().ServiceAgentContactDelete(gomock.Any(), a, id)
		},
		ret: ret, okStatus: 200,
	})...)

	fn, ln, dn, co, jt, ext, notes := "Ann", "Lee", "Ann Lee", "Example Co", "CTO", "ext-1", "some notes"
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PutServiceAgentsContactsId", method: http.MethodPut, path: base,
		body:    `{"first_name":"Ann","last_name":"Lee","display_name":"Ann Lee","company":"Example Co","job_title":"CTO","external_id":"ext-1","notes":"some notes"}`,
		badJSON: true, badIDPath: bad, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().ServiceAgentContactUpdate(gomock.Any(), a, id, &fn, &ln, &dn, &co, &jt, &ext, &notes)
		},
		ret: ret, okStatus: 200,
	})...)

	// create
	createBody := `{"first_name":"Ann","last_name":"Lee","display_name":"Ann Lee","company":"Example Co","job_title":"CTO","source":"manual","external_id":"ext-1","notes":"some notes",` +
		`"addresses":[{"type":"tel","target":"+15551230001","is_primary":true,"name":"mobile","detail":"work"},{"type":"email","target":"ann@example.com"}],` +
		`"tag_ids":["` + s4ID2 + `"]}`
	wantAddrs := []cmcontact.AddressInput{
		{Address: commonaddress.Address{Type: "tel", Target: "+15551230001", Name: "mobile", Detail: "work"}, IsPrimary: true},
		{Address: commonaddress.Address{Type: "email", Target: "ann@example.com"}},
	}
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostServiceAgentsContacts", method: http.MethodPost, path: "/service_agents/contacts", body: createBody, badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().ServiceAgentContactCreate(gomock.Any(), a, "Ann", "Lee", "Ann Lee", "Example Co", "CTO", "manual", "ext-1", "some notes",
				wantAddrs, []uuid.UUID{s4UUID(s4ID2)})
		},
		ret: ret, okStatus: 201,
	})...)
	cases = append(cases, s4Case{
		name: "PostServiceAgentsContacts/empty_body_defaults", method: http.MethodPost, path: "/service_agents/contacts",
		body: `{}`,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentContactCreate(gomock.Any(), a, "", "", "", "", "", "", "", "",
				[]cmcontact.AddressInput{}, []uuid.UUID{}).Return(ret, nil)
		},
		wantStatus: 201, wantBody: ret,
	})
	addrErr := func(name, addr string, reason string) s4Case {
		return s4Case{
			name: "PostServiceAgentsContacts/" + name, method: http.MethodPost, path: "/service_agents/contacts",
			body:          `{"addresses":[` + addr + `]}`,
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: reason,
		}
	}
	cases = append(cases,
		addrErr("address_missing_type", `{"target":"+15551230001"}`, "INVALID_ADDRESS_TYPE"),
		addrErr("address_empty_type", `{"type":"","target":"+15551230001"}`, "INVALID_ADDRESS_TYPE"),
		addrErr("address_unsupported_type", `{"type":"fax","target":"+15551230001"}`, "INVALID_ADDRESS_TYPE"),
		addrErr("address_missing_target", `{"type":"tel"}`, "INVALID_ADDRESS_TARGET"),
		addrErr("address_empty_target", `{"type":"tel","target":""}`, "INVALID_ADDRESS_TARGET"),
	)

	// lookup
	cases = append(cases,
		s4Case{
			name: "GetServiceAgentsContactsLookup/unauthenticated", method: http.MethodGet, path: "/service_agents/contacts/lookup?email=a@example.com", noAuth: true,
			wantErrStatus: cerrors.StatusUnauthenticated, wantReason: "AUTHENTICATION_REQUIRED",
		},
		s4Case{
			name: "GetServiceAgentsContactsLookup/no_params", method: http.MethodGet, path: "/service_agents/contacts/lookup",
			wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ARGUMENT",
		},
		s4Case{
			name: "GetServiceAgentsContactsLookup/phone", method: http.MethodGet, path: "/service_agents/contacts/lookup?phone=%2B15551230001",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentContactLookup(gomock.Any(), a, "+15551230001", "").Return(ret, nil)
			},
			wantStatus: 200, wantBody: ret,
		},
		s4Case{
			name: "GetServiceAgentsContactsLookup/email", method: http.MethodGet, path: "/service_agents/contacts/lookup?email=ann%40example.com",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentContactLookup(gomock.Any(), a, "", "ann@example.com").Return(ret, nil)
			},
			wantStatus: 200, wantBody: ret,
		},
		s4Case{
			name: "GetServiceAgentsContactsLookup/service_error", method: http.MethodGet, path: "/service_agents/contacts/lookup?email=ann%40example.com",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().ServiceAgentContactLookup(gomock.Any(), a, "", "ann@example.com").Return(nil, s4ErrDenied)
			},
			wantErrStatus: cerrors.StatusPermissionDenied, wantReason: "PERMISSION_DENIED",
		},
	)

	// addresses
	addrID := s4UUID(s4ID2)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostServiceAgentsContactsIdAddresses", method: http.MethodPost, path: base + "/addresses",
		body:    `{"type":"tel","target":"+15551230001","is_primary":true,"name":"mobile","detail":"work"}`,
		badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().ServiceAgentContactAddressCreate(gomock.Any(), a, id, "tel", "+15551230001", true, "mobile", "work")
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4Case{
		name: "PostServiceAgentsContactsIdAddresses/optional_fields_default", method: http.MethodPost, path: base + "/addresses",
		body: `{"type":"email","target":"ann@example.com"}`,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentContactAddressCreate(gomock.Any(), a, id, "email", "ann@example.com", false, "", "").Return(ret, nil)
		},
		wantStatus: 200, wantBody: ret,
	})
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PutServiceAgentsContactsIdAddressesAddressId", method: http.MethodPut, path: base + "/addresses/" + s4ID2,
		body:    `{"target":"+15551230009","is_primary":false,"name":"home","detail":"d"}`,
		badJSON: true,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().ServiceAgentContactAddressUpdate(gomock.Any(), a, id, addrID,
				map[string]any{"target": "+15551230009", "is_primary": false, "name": "home", "detail": "d"})
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4Case{
		name: "PutServiceAgentsContactsIdAddressesAddressId/empty_body_empty_fields", method: http.MethodPut, path: base + "/addresses/" + s4ID2,
		body: `{}`,
		setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
			m.EXPECT().ServiceAgentContactAddressUpdate(gomock.Any(), a, id, addrID, map[string]any{}).Return(ret, nil)
		},
		wantStatus: 200, wantBody: ret,
	})
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "DeleteServiceAgentsContactsIdAddressesAddressId", method: http.MethodDelete, path: base + "/addresses/" + s4ID2,
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().ServiceAgentContactAddressDelete(gomock.Any(), a, id, addrID)
		},
		ret: ret, okStatus: 200,
	})...)

	// tags
	tagID := s4UUID(s4ID3)
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "PostServiceAgentsContactsIdTags", method: http.MethodPost, path: base + "/tags",
		body:    `{"tag_id":"` + s4ID3 + `"}`,
		badJSON: true, badIDPath: bad + "/tags", badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().ServiceAgentContactTagAdd(gomock.Any(), a, id, tagID)
		},
		ret: ret, okStatus: 201,
	})...)
	cases = append(cases, s4Case{
		name: "PostServiceAgentsContactsIdTags/nil_tag_id", method: http.MethodPost, path: base + "/tags",
		body:          `{"tag_id":"00000000-0000-0000-0000-000000000000"}`,
		wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID",
	})
	cases = append(cases, s4ItemCases(s4ItemEP{
		name: "DeleteServiceAgentsContactsIdTagsTagId", method: http.MethodDelete, path: base + "/tags/" + s4ID3,
		badIDPath: bad + "/tags/" + s4ID3, badIDReason: "INVALID_ID",
		call: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) *gomock.Call {
			return m.EXPECT().ServiceAgentContactTagRemove(gomock.Any(), a, id, tagID)
		},
		ret: ret, okStatus: 200,
	})...)
	cases = append(cases, s4Case{
		name: "DeleteServiceAgentsContactsIdTagsTagId/invalid_tag_id", method: http.MethodDelete, path: base + "/tags/not-a-uuid",
		wantErrStatus: cerrors.StatusInvalidArgument, wantReason: "INVALID_ID",
	})
	s4RunCases(t, cases)
}
