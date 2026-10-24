package listenhandler

import (
	"fmt"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/pkg/casehandler"
	"monorepo/bin-contact-manager/pkg/contacthandler"
)

// Test_processRequest_RouteTable verifies that a representative URI and method
// of each route reaches the intended downstream handler method. Every case
// expects exactly one distinct mock method and gomock is strict, so a route
// wired to another handler fails the case.
func Test_processRequest_RouteTable(t *testing.T) {
	customerID := uuid.FromStringOrNil("55555555-0091-0091-0091-000000000001")
	contactID := uuid.FromStringOrNil("55555555-0091-0091-0091-000000000002")
	addressID := uuid.FromStringOrNil("55555555-0091-0091-0091-000000000003")
	caseID := uuid.FromStringOrNil("55555555-0091-0091-0091-000000000004")
	subID := uuid.FromStringOrNil("55555555-0091-0091-0091-000000000005")

	cid := customerID.String()
	custBody := []byte(fmt.Sprintf(`{"customer_id":"%s"}`, cid))
	fail := fmt.Errorf("downstream failure")

	tests := []struct {
		name   string
		method sock.RequestMethod
		uri    string
		data   []byte
		expect func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler)
	}{
		{
			name: "contact_addresses create", method: sock.RequestMethodPost, uri: "/v1/contact_addresses",
			data: []byte(fmt.Sprintf(`{"contact_id":"%s","type":"tel","target":"+15550000001"}`, contactID)),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().AddAddress(gomock.Any(), contactID, gomock.Any()).Return(nil, fail)
			},
		},
		{
			name: "contact_addresses update", method: sock.RequestMethodPut, uri: "/v1/contact_addresses/" + addressID.String() + "?contact_id=" + contactID.String(),
			data: []byte(`{"name":"x"}`),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().UpdateAddress(gomock.Any(), contactID, addressID, map[string]any{"name": "x"}).Return(nil, fail)
			},
		},
		{
			name: "contact_addresses delete", method: sock.RequestMethodDelete, uri: "/v1/contact_addresses/" + addressID.String() + "?contact_id=" + contactID.String(),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().RemoveAddress(gomock.Any(), contactID, addressID).Return(nil, fail)
			},
		},
		{
			name: "contact_addresses claim", method: sock.RequestMethodPost, uri: "/v1/contact_addresses/" + addressID.String() + "/claim?customer_id=" + cid,
			data: []byte(fmt.Sprintf(`{"contact_id":"%s"}`, contactID)),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().ClaimAddress(gomock.Any(), customerID, addressID, contactID, false).Return(nil, fail)
			},
		},
		{
			name: "contacts addresses list", method: sock.RequestMethodGet, uri: "/v1/contacts/" + contactID.String() + "/addresses",
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().Get(gomock.Any(), contactID).Return(nil, fail)
			},
		},
		{
			name: "contacts addresses create", method: sock.RequestMethodPost, uri: "/v1/contacts/" + contactID.String() + "/addresses",
			data: []byte(`{"type":"tel","target":"+15550000001"}`),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().AddAddress(gomock.Any(), contactID, gomock.Any()).Return(nil, fail)
			},
		},
		{
			name: "contacts addresses update", method: sock.RequestMethodPut, uri: "/v1/contacts/" + contactID.String() + "/addresses/" + addressID.String(),
			data: []byte(`{"name":"x"}`),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().UpdateAddress(gomock.Any(), contactID, addressID, map[string]any{"name": "x"}).Return(nil, fail)
			},
		},
		{
			name: "contacts addresses delete", method: sock.RequestMethodDelete, uri: "/v1/contacts/" + contactID.String() + "/addresses/" + addressID.String(),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().RemoveAddress(gomock.Any(), contactID, addressID).Return(nil, fail)
			},
		},
		{
			name: "contacts tags add", method: sock.RequestMethodPost, uri: "/v1/contacts/" + contactID.String() + "/tags",
			data: []byte(fmt.Sprintf(`{"tag_id":"%s"}`, subID)),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().AddTag(gomock.Any(), contactID, subID).Return(nil, fail)
			},
		},
		{
			name: "contacts tags remove", method: sock.RequestMethodDelete, uri: "/v1/contacts/" + contactID.String() + "/tags/" + subID.String(),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().RemoveTag(gomock.Any(), contactID, subID).Return(nil, fail)
			},
		},
		{
			name: "interactions list", method: sock.RequestMethodGet, uri: "/v1/interactions?contact_id=" + contactID.String(), data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				c.EXPECT().InteractionList(gomock.Any(), customerID, uint64(0), "", "", "", contactID, uuid.Nil).Return(nil, "", fail)
			},
		},
		{
			name: "cases list", method: sock.RequestMethodGet, uri: "/v1/cases?status=open", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseList(gomock.Any(), customerID, gomock.Any(), gomock.Any(), "open", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, "", fail)
			},
		},
		{
			name: "cases unresolved", method: sock.RequestMethodGet, uri: "/v1/cases/unresolved", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseListUnresolved(gomock.Any(), customerID).Return(nil, fail)
			},
		},
		{
			name: "cases get", method: sock.RequestMethodGet, uri: "/v1/cases/" + caseID.String(), data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseGet(gomock.Any(), customerID, caseID).Return(nil, fail)
			},
		},
		{
			name: "cases update contact", method: sock.RequestMethodPut, uri: "/v1/cases/" + caseID.String(),
			data: []byte(fmt.Sprintf(`{"customer_id":"%s","contact_id":"%s"}`, cid, contactID)),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().UpdateContact(gomock.Any(), customerID, caseID, contactID).Return(nil, fail)
			},
		},
		{
			name: "cases close", method: sock.RequestMethodPost, uri: "/v1/cases/" + caseID.String() + "/close", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().Close(gomock.Any(), customerID, caseID, gomock.Any(), gomock.Any()).Return(nil, fail)
			},
		},
		{
			name: "cases assign", method: sock.RequestMethodPost, uri: "/v1/cases/" + caseID.String() + "/assign", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().Assign(gomock.Any(), customerID, caseID, gomock.Any(), gomock.Any()).Return(nil, fail)
			},
		},
		{
			name: "cases continue", method: sock.RequestMethodPost, uri: "/v1/cases/" + caseID.String() + "/continue", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().Continue(gomock.Any(), customerID, caseID, gomock.Any(), gomock.Any(), false).Return(nil, fail)
			},
		},
		{
			name: "cases notes list", method: sock.RequestMethodGet, uri: "/v1/cases/" + caseID.String() + "/notes", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseNoteListByCase(gomock.Any(), customerID, caseID).Return(nil, fail)
			},
		},
		{
			name: "cases notes create", method: sock.RequestMethodPost, uri: "/v1/cases/" + caseID.String() + "/notes", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseNoteCreate(gomock.Any(), customerID, caseID, gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, fail)
			},
		},
		{
			name: "cases notes delete", method: sock.RequestMethodDelete, uri: "/v1/cases/" + caseID.String() + "/notes/" + subID.String(), data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseNoteDelete(gomock.Any(), customerID, caseID, subID).Return(fail)
			},
		},
		{
			name: "cases tags list", method: sock.RequestMethodGet, uri: "/v1/cases/" + caseID.String() + "/tags", data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseTagList(gomock.Any(), customerID, caseID).Return(nil, fail)
			},
		},
		{
			name: "cases tags add", method: sock.RequestMethodPost, uri: "/v1/cases/" + caseID.String() + "/tags",
			data: []byte(fmt.Sprintf(`{"customer_id":"%s","tag_id":"%s"}`, cid, subID)),
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseTagAdd(gomock.Any(), customerID, caseID, subID).Return(fail)
			},
		},
		{
			name: "cases tags remove", method: sock.RequestMethodDelete, uri: "/v1/cases/" + caseID.String() + "/tags/" + subID.String(), data: custBody,
			expect: func(c *contacthandler.MockContactHandler, k *casehandler.MockCaseHandler) {
				k.EXPECT().CaseTagRemove(gomock.Any(), customerID, caseID, subID).Return(fail)
			},
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
			if res == nil {
				t.Fatalf("processRequest() returned no response")
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
