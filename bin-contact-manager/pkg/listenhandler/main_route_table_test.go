package listenhandler

import (
	"fmt"
	"strings"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/casenote"
	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/casehandler"
	"monorepo/bin-contact-manager/pkg/contacthandler"
)

// Test_processRequest_RouteTable verifies that a representative URI and method
// of each route reaches the intended downstream handler method and produces
// the intended response. Every case expects exactly one mock method (gomock is
// strict) and asserts the status code and body markers, so a route wired to a
// handler with a different response shape fails the case.
func Test_processRequest_RouteTable(t *testing.T) {
	customerID := uuid.FromStringOrNil("55555555-0091-0091-0091-000000000001")
	contactID := uuid.FromStringOrNil("55555555-0091-0091-0091-000000000002")
	addressID := uuid.FromStringOrNil("55555555-0091-0091-0091-000000000003")
	caseID := uuid.FromStringOrNil("55555555-0091-0091-0091-000000000004")
	subID := uuid.FromStringOrNil("55555555-0091-0091-0091-000000000005")

	cid := customerID.String()
	aid := addressID.String()
	custBody := []byte(fmt.Sprintf(`{"customer_id":"%s"}`, cid))
	pat := &contact.Contact{FirstName: "Pat", Addresses: []contact.Address{{ID: addressID}}}
	theCase := &kase.Case{ID: caseID, CustomerID: customerID}

	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
		data   []byte
		expect func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler)

		expectCode int
		// bodyHas are substrings the response body must contain; bodyNot must not appear.
		bodyHas []string
		bodyNot []string
	}{
		{
			name: "contact_addresses create", method: sock.RequestMethodPost, uri: "/v1/contact_addresses",
			data: []byte(fmt.Sprintf(`{"contact_id":"%s","type":"tel","target":"+15550000001"}`, contactID)),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().AddAddress(gomock.Any(), contactID, gomock.Any()).Return(&contact.Address{ID: addressID}, nil)
			},
			expectCode: 201, bodyHas: []string{aid}, bodyNot: []string{"first_name"},
		},
		{
			name: "contact_addresses update", method: sock.RequestMethodPut, uri: "/v1/contact_addresses/" + aid + "?contact_id=" + contactID.String(),
			data: []byte(`{"name":"x"}`),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().UpdateAddress(gomock.Any(), contactID, addressID, map[string]any{"name": "x"}).Return(pat, nil)
			},
			expectCode: 200, bodyHas: []string{aid}, bodyNot: []string{"first_name"},
		},
		{
			name: "contact_addresses delete", method: sock.RequestMethodDelete, uri: "/v1/contact_addresses/" + aid + "?contact_id=" + contactID.String(),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().RemoveAddress(gomock.Any(), contactID, addressID).Return(pat, nil)
			},
			expectCode: 200, bodyHas: []string{"Pat"},
		},
		{
			name: "contact_addresses claim", method: sock.RequestMethodPost, uri: "/v1/contact_addresses/" + aid + "/claim?customer_id=" + cid,
			data: []byte(fmt.Sprintf(`{"contact_id":"%s"}`, contactID)),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().ClaimAddress(gomock.Any(), customerID, addressID, contactID, false).Return(&contact.Address{ID: addressID}, nil)
			},
			expectCode: 200, bodyHas: []string{aid}, bodyNot: []string{"first_name"},
		},
		{
			name: "contacts addresses list", method: sock.RequestMethodGet, uri: "/v1/contacts/" + contactID.String() + "/addresses",
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().Get(gomock.Any(), contactID).Return(pat, nil)
			},
			expectCode: 200, bodyHas: []string{`"result"`, aid}, bodyNot: []string{"first_name"},
		},
		{
			name: "contacts addresses create", method: sock.RequestMethodPost, uri: "/v1/contacts/" + contactID.String() + "/addresses",
			data: []byte(`{"type":"tel","target":"+15550000001"}`),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().AddAddress(gomock.Any(), contactID, gomock.Any()).Return(&contact.Address{ID: addressID}, nil)
			},
			expectCode: 201, bodyHas: []string{aid},
		},
		{
			name: "contacts addresses update", method: sock.RequestMethodPut, uri: "/v1/contacts/" + contactID.String() + "/addresses/" + aid,
			data: []byte(`{"name":"x"}`),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().UpdateAddress(gomock.Any(), contactID, addressID, map[string]any{"name": "x"}).Return(pat, nil)
			},
			expectCode: 200, bodyHas: []string{"Pat"},
		},
		{
			name: "contacts addresses delete", method: sock.RequestMethodDelete, uri: "/v1/contacts/" + contactID.String() + "/addresses/" + aid,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().RemoveAddress(gomock.Any(), contactID, addressID).Return(pat, nil)
			},
			expectCode: 200, bodyHas: []string{"Pat"},
		},
		{
			name: "contacts tags add", method: sock.RequestMethodPost, uri: "/v1/contacts/" + contactID.String() + "/tags",
			data: []byte(fmt.Sprintf(`{"tag_id":"%s"}`, subID)),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().AddTag(gomock.Any(), contactID, subID).Return(pat, nil)
			},
			expectCode: 200, bodyHas: []string{"Pat"},
		},
		{
			name: "contacts tags remove", method: sock.RequestMethodDelete, uri: "/v1/contacts/" + contactID.String() + "/tags/" + subID.String(),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().RemoveTag(gomock.Any(), contactID, subID).Return(pat, nil)
			},
			expectCode: 200, bodyHas: []string{"Pat"},
		},
		{
			name: "interactions list", method: sock.RequestMethodGet, uri: "/v1/interactions?contact_id=" + contactID.String(), data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().InteractionList(gomock.Any(), customerID, uint64(0), "", "", "", contactID, uuid.Nil).Return(nil, "", nil)
			},
			expectCode: 200,
		},
		{
			name: "cases list", method: sock.RequestMethodGet, uri: "/v1/cases?status=open", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseList(gomock.Any(), customerID, gomock.Any(), gomock.Any(), "open", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return([]*kase.Case{theCase}, "", nil)
			},
			expectCode: 200, bodyHas: []string{caseID.String()},
		},
		{
			name: "cases unresolved", method: sock.RequestMethodGet, uri: "/v1/cases/unresolved", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseListUnresolved(gomock.Any(), customerID).Return([]*kase.Case{theCase}, nil)
			},
			expectCode: 200, bodyHas: []string{caseID.String()},
		},
		{
			name: "cases get", method: sock.RequestMethodGet, uri: "/v1/cases/" + caseID.String(), data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseGet(gomock.Any(), customerID, caseID).Return(theCase, nil)
			},
			expectCode: 200, bodyHas: []string{caseID.String()},
		},
		{
			name: "cases update contact", method: sock.RequestMethodPut, uri: "/v1/cases/" + caseID.String(),
			data: []byte(fmt.Sprintf(`{"customer_id":"%s","contact_id":"%s"}`, cid, contactID)),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().UpdateContact(gomock.Any(), customerID, caseID, contactID).Return(theCase, nil)
			},
			expectCode: 200, bodyHas: []string{caseID.String()},
		},
		{
			name: "cases close", method: sock.RequestMethodPost, uri: "/v1/cases/" + caseID.String() + "/close", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().Close(gomock.Any(), customerID, caseID, gomock.Any(), gomock.Any()).Return(&casehandler.CloseResult{Case: theCase}, nil)
			},
			expectCode: 200, bodyHas: []string{caseID.String()},
		},
		{
			name: "cases assign", method: sock.RequestMethodPost, uri: "/v1/cases/" + caseID.String() + "/assign", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().Assign(gomock.Any(), customerID, caseID, gomock.Any(), gomock.Any()).Return(theCase, nil)
			},
			expectCode: 200, bodyHas: []string{caseID.String()},
		},
		{
			name: "cases continue", method: sock.RequestMethodPost, uri: "/v1/cases/" + caseID.String() + "/continue", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().Continue(gomock.Any(), customerID, caseID, gomock.Any(), gomock.Any(), false).Return(theCase, nil)
			},
			expectCode: 200, bodyHas: []string{caseID.String()},
		},
		{
			name: "cases notes list", method: sock.RequestMethodGet, uri: "/v1/cases/" + caseID.String() + "/notes", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseNoteListByCase(gomock.Any(), customerID, caseID).Return([]*casenote.CaseNote{{ID: subID, Text: "listed"}}, nil)
			},
			expectCode: 200, bodyHas: []string{"listed"},
		},
		{
			name: "cases notes create", method: sock.RequestMethodPost, uri: "/v1/cases/" + caseID.String() + "/notes", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseNoteCreate(gomock.Any(), customerID, caseID, gomock.Any(), gomock.Any(), gomock.Any()).Return(&casenote.CaseNote{ID: subID, Text: "created"}, nil)
			},
			expectCode: 200, bodyHas: []string{"created"},
		},
		{
			name: "cases notes delete", method: sock.RequestMethodDelete, uri: "/v1/cases/" + caseID.String() + "/notes/" + subID.String(), data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseNoteDelete(gomock.Any(), customerID, caseID, subID).Return(nil)
			},
			expectCode: 200,
		},
		{
			name: "cases tags list", method: sock.RequestMethodGet, uri: "/v1/cases/" + caseID.String() + "/tags", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseTagList(gomock.Any(), customerID, caseID).Return([]uuid.UUID{subID}, nil)
			},
			expectCode: 200, bodyHas: []string{subID.String()},
		},
		{
			name: "cases tags add", method: sock.RequestMethodPost, uri: "/v1/cases/" + caseID.String() + "/tags",
			data: []byte(fmt.Sprintf(`{"customer_id":"%s","tag_id":"%s"}`, cid, subID)),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseTagAdd(gomock.Any(), customerID, caseID, subID).Return(nil)
			},
			expectCode: 200,
		},
		{
			name: "cases tags remove", method: sock.RequestMethodDelete, uri: "/v1/cases/" + caseID.String() + "/tags/" + subID.String(), data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseTagRemove(gomock.Any(), customerID, caseID, subID).Return(nil)
			},
			expectCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockContact, _, mockCase := newTestListenHandlerAll(mc)

			tt.expect(mockContact, mockCase)

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method, Data: tt.data})
			if err != nil {
				t.Fatalf("processRequest() error = %v", err)
			}
			if res.StatusCode != tt.expectCode {
				t.Fatalf("StatusCode = %d, want %d", res.StatusCode, tt.expectCode)
			}
			body := string(res.Data)
			for _, want := range tt.bodyHas {
				if !strings.Contains(body, want) {
					t.Errorf("body %s does not contain %q", body, want)
				}
			}
			for _, not := range tt.bodyNot {
				if strings.Contains(body, not) {
					t.Errorf("body %s must not contain %q", body, not)
				}
			}
		})
	}
}

func Test_processRequest_MethodMismatchIsNotFound(t *testing.T) {
	caseID := uuid.FromStringOrNil("55555555-0092-0092-0092-000000000001")
	addressID := uuid.FromStringOrNil("55555555-0092-0092-0092-000000000002")

	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
	}{
		{"cases close with get", sock.RequestMethodGet, "/v1/cases/" + caseID.String() + "/close"},
		{"cases assign with delete", sock.RequestMethodDelete, "/v1/cases/" + caseID.String() + "/assign"},
		{"interactions with post", sock.RequestMethodPost, "/v1/interactions?contact_id=" + caseID.String()},
		{"contact_addresses claim with get", sock.RequestMethodGet, "/v1/contact_addresses/" + addressID.String() + "/claim"},
		{"unknown version", sock.RequestMethodGet, "/v2/cases"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, _, _, _ := newTestListenHandlerAll(mc)

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method})
			if err != nil {
				t.Fatalf("processRequest() error = %v", err)
			}
			if res.StatusCode != 404 {
				t.Errorf("StatusCode = %d, want 404", res.StatusCode)
			}
		})
	}
}
