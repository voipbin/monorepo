package server

import (
	"fmt"
	"net/http"
	"testing"

	"monorepo/bin-api-manager/models/auth"
	"monorepo/bin-api-manager/pkg/serviceerrors"
	"monorepo/bin-api-manager/pkg/servicehandler"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	cmcasenote "monorepo/bin-contact-manager/models/casenote"
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_PutCustomer_ServiceErrors(t *testing.T) {
	name, email, uri := "n", "user@example.com", "https://example.com/hook"
	method := cscustomer.WebhookMethod("POST")
	s1Run(t, []s1Case{
		s1PermDenied("PUT", "/customer", `{"name":"n","email":"user@example.com","webhook_method":"POST","webhook_uri":"https://example.com/hook"}`,
			func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
				m.EXPECT().CustomerSelfUpdate(gomock.Any(), a, &name, (*string)(nil), &email, (*string)(nil), (*string)(nil), &method, &uri).Return(nil, err)
			}),
		s1PermDenied("PUT", "/customer", `{}`,
			func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
				m.EXPECT().CustomerSelfUpdate(gomock.Any(), a, (*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil), (*string)(nil), (*cscustomer.WebhookMethod)(nil), (*string)(nil)).Return(nil, err)
			}),
	})
}

func Test_PutCustomerMetadata_Errors(t *testing.T) {
	s1Run(t, []s1Case{
		s1Unauth("PUT", "/customer/metadata", `{"rtp_debug":true}`),
		s1BadJSON("PUT", "/customer/metadata"),
		s1PermDenied("PUT", "/customer/metadata", `{"rtp_debug":true}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CustomerSelfUpdateMetadata(gomock.Any(), a, cscustomer.Metadata{RTPDebug: true}).Return(nil, err)
		}),
		s1PermDenied("PUT", "/customer/metadata", `{}`, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CustomerSelfUpdateMetadata(gomock.Any(), a, cscustomer.Metadata{RTPDebug: false}).Return(nil, err)
		}),
	})
}

func Test_PutCustomerBillingAccountId_Errors(t *testing.T) {
	path := "/customer/billing_account_id"
	body := `{"billing_account_id":"` + s1RefID + `"}`
	s1Run(t, []s1Case{
		s1Unauth("PUT", path, body),
		s1BadJSON("PUT", path),
		s1BadID("PUT", path, `{"billing_account_id":"not-a-uuid"}`),
		s1PermDenied("PUT", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CustomerSelfUpdateBillingAccountID(gomock.Any(), a, s1ID(s1RefID)).Return(nil, err)
		}),
	})
}

func Test_ContactCasesIdNotes_Errors(t *testing.T) {
	path := "/contact_cases/" + s1CaseID + "/notes"
	id := s1ID(s1CaseID)
	authorID := s1ID(s1AgentID)
	body := `{"author_type":"agent","author_id":"` + s1AgentID + `","text":"note"}`
	note := &cmcasenote.CaseNote{ID: s1ID(s1NoteID), CaseID: id, Text: "note"}
	s1Run(t, []s1Case{
		s1Unauth("GET", path, ""),
		s1PermDenied("GET", path, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CaseNoteList(gomock.Any(), a, id).Return(nil, err)
		}),
		s1Unauth("POST", path, body),
		s1BadJSON("POST", path),
		s1PermDenied("POST", path, body, func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CaseNoteCreate(gomock.Any(), a, id, "agent", &authorID, "note").Return(nil, err)
		}),
		{
			name:   "post without author_id passes a nil author",
			method: "POST",
			path:   path,
			body:   `{"author_type":"agent","text":"note"}`,
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().CaseNoteCreate(gomock.Any(), a, id, "agent", (*uuid.UUID)(nil), "note").Return(note, nil)
			},
			wantBody: s1JSON(t, note),
		},
		s1Unauth("DELETE", path+"/"+s1NoteID, ""),
		s1PermDenied("DELETE", path+"/"+s1NoteID, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().CaseNoteDelete(gomock.Any(), a, id, s1ID(s1NoteID)).Return(err)
		}),
	})
}

func Test_GetContactInteractions_Errors(t *testing.T) {
	base := "/contact_interactions"
	badFilter := func(name, q string) s1Case {
		return s1Case{
			name:       name,
			method:     "GET",
			path:       base + q,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_FILTER",
		}
	}
	s1Run(t, []s1Case{
		s1Unauth("GET", base+"?contact_id="+s1ContactID, ""),
		badFilter("no filter", ""),
		badFilter("two filters", "?contact_id="+s1ContactID+"&address_id="+s1AddressID),
		badFilter("peer filter plus contact", "?peer_type=tel&peer_target=%2B15555550100&contact_id="+s1ContactID),
		s1PermDenied("GET", base+"?contact_id="+s1ContactID+"&page_size=500&page_token=tok", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().InteractionList(gomock.Any(), a, uint64(100), "tok", "", "", s1ID(s1ContactID), uuid.Nil).Return(nil, "", err)
		}),
		s1PermDenied("GET", base+"?peer_type=tel&peer_target=%2B15555550100&page_size=0", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().InteractionList(gomock.Any(), a, uint64(100), "", "tel", "+15555550100", uuid.Nil, uuid.Nil).Return(nil, "", err)
		}),
		s1PermDenied("GET", base+"?address_id="+s1AddressID, "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().InteractionList(gomock.Any(), a, uint64(100), "", "", "", uuid.Nil, s1ID(s1AddressID)).Return(nil, "", err)
		}),
	})
}

func Test_GetContactPeerEvents_Errors(t *testing.T) {
	base := "/contact_peer_events"
	badFilter := func(name, q string) s1Case {
		return s1Case{
			name:       name,
			method:     "GET",
			path:       base + q,
			wantStatus: cerrors.StatusInvalidArgument,
			wantReason: "INVALID_FILTER",
		}
	}
	peer := &commonaddress.Address{Type: commonaddress.Type("tel"), Target: "+15555550100"}
	s1Run(t, []s1Case{
		s1Unauth("GET", base+"?contact_id="+s1ContactID, ""),
		badFilter("no filter", ""),
		badFilter("contact plus peer filter", "?contact_id="+s1ContactID+"&peer_type=tel&peer_target=%2B15555550100"),
		s1PermDenied("GET", base+"?contact_id="+s1ContactID+"&page_size=500&page_token=tok", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().PeerEventList(gomock.Any(), a, s1ID(s1ContactID), (*commonaddress.Address)(nil), "tok", uint64(100)).Return(nil, "", err)
		}),
		s1PermDenied("GET", base+"?peer_type=tel&peer_target=%2B15555550100&page_size=0", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().PeerEventList(gomock.Any(), a, uuid.Nil, peer, "", uint64(100)).Return(nil, "", err)
		}),
	})
}

func Test_GetStorageAccount_Errors(t *testing.T) {
	s1Run(t, []s1Case{
		s1Unauth("GET", "/storage_account", ""),
		s1NotFound("GET", "/storage_account", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().StorageAccountGetByCustomerID(gomock.Any(), a).Return(nil, err)
		}),
	})
}

func Test_GetServiceAgentsCustomer_Errors(t *testing.T) {
	s1Run(t, []s1Case{
		s1Unauth("GET", "/service_agents/customer", ""),
		s1PermDenied("GET", "/service_agents/customer", "", func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity, err error) {
			m.EXPECT().ServiceAgentCustomerGet(gomock.Any(), a).Return(nil, err)
		}),
	})
}

func Test_GetWs_Errors(t *testing.T) {
	isWriter := gomock.Cond(func(x any) bool { _, ok := x.(http.ResponseWriter); return ok })
	isWsReq := gomock.Cond(func(x any) bool {
		r, ok := x.(*http.Request)
		return ok && r.URL.Path == "/ws"
	})
	s1Run(t, []s1Case{
		s1Unauth("GET", "/ws", ""),
		{
			// The handler only logs a websocket failure; the response is
			// whatever the servicehandler already wrote (nothing here).
			name:   "websocket error is logged and no error body is written",
			method: "GET",
			path:   "/ws",
			setup: func(m *servicehandler.MockServiceHandler, a *auth.AuthIdentity) {
				m.EXPECT().WebsockCreate(gomock.Any(), a, isWriter, isWsReq).Return(fmt.Errorf("%w: upgrade failed", serviceerrors.ErrInternal))
			},
			wantBody: "",
		},
	})
}

func Test_PostActiveflowsIdStop_InvalidID(t *testing.T) {
	s1Run(t, []s1Case{
		s1BadID("POST", "/activeflows/not-a-uuid/stop", ""),
	})
}
