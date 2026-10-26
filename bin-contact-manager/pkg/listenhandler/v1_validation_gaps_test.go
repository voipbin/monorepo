package listenhandler

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/casehandler"
	"monorepo/bin-contact-manager/pkg/contacthandler"
)

func Test_caseURIHelpers(t *testing.T) {
	caseID := uuid.FromStringOrNil("66666666-0001-0001-0001-000000000001")
	subID := uuid.FromStringOrNil("66666666-0001-0001-0001-000000000002")

	tests := []struct {
		name      string
		uri       string
		wantCase  uuid.UUID
		wantSubID uuid.UUID
	}{
		{name: "too few segments for any id", uri: "/v1/cases", wantCase: uuid.Nil, wantSubID: uuid.Nil},
		{name: "case id only", uri: "/v1/cases/" + caseID.String(), wantCase: caseID, wantSubID: uuid.Nil},
		{name: "case id with a sub resource name but no sub id", uri: "/v1/cases/" + caseID.String() + "/notes", wantCase: caseID, wantSubID: uuid.Nil},
		{name: "case id and sub id", uri: "/v1/cases/" + caseID.String() + "/notes/" + subID.String(), wantCase: caseID, wantSubID: subID},
		{name: "malformed ids resolve to the nil uuid", uri: "/v1/cases/not-a-uuid/notes/not-a-uuid", wantCase: uuid.Nil, wantSubID: uuid.Nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := caseIDFromURI(tt.uri); got != tt.wantCase {
				t.Errorf("caseIDFromURI() = %s, want %s", got, tt.wantCase)
			}
			if got := caseSubIDFromURI(tt.uri); got != tt.wantSubID {
				t.Errorf("caseSubIDFromURI() = %s, want %s", got, tt.wantSubID)
			}
		})
	}
}

// Test_processV1_InputValidationAndErrorMapping covers rejected requests and
// the mapping of business handler errors to response codes. gomock is strict:
// a case without an expectation fails if the request reaches a handler.
func Test_processV1_InputValidationAndErrorMapping(t *testing.T) {
	customerID := uuid.FromStringOrNil("66666666-0002-0002-0002-000000000001")
	contactID := uuid.FromStringOrNil("66666666-0002-0002-0002-000000000002")
	addressID := uuid.FromStringOrNil("66666666-0002-0002-0002-000000000003")
	caseID := uuid.FromStringOrNil("66666666-0002-0002-0002-000000000004")

	custBody := fmt.Sprintf(`{"customer_id":"%s"}`, customerID)
	caseURI := "/v1/cases/" + caseID.String()
	notFound := cerrors.NotFound(commonoutline.ServiceNameContactManager, "CASE_NOT_FOUND", "not found")

	type deps struct {
		contact *contacthandler.MockContactHandler
		kase    *casehandler.MockCaseHandler
	}

	tests := []struct {
		name       string
		call       func(h *listenHandler, ctx context.Context, req *sock.Request) (*sock.Response, error)
		uri        string
		method     sock.RequestMethod
		body       string
		expect     func(d deps)
		wantStatus int
		wantErr    bool
	}{
		{
			name:       "cases list rejects a malformed URI",
			call:       (*listenHandler).processV1CasesGet,
			uri:        "/v1/cases/%zz",
			method:     sock.RequestMethodGet,
			body:       custBody,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "cases list maps a typed handler error",
			call:   (*listenHandler).processV1CasesGet,
			uri:    "/v1/cases?page_size=5",
			method: sock.RequestMethodGet,
			body:   custBody,
			expect: func(d deps) {
				d.kase.EXPECT().CaseList(gomock.Any(), customerID, uint64(5), "", "", gomock.Any(), uuid.Nil, uuid.Nil, "").Return(nil, "", notFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "cases create rejects a malformed body",
			call:       (*listenHandler).processV1CasesPost,
			uri:        "/v1/cases",
			method:     sock.RequestMethodPost,
			body:       `{"customer_id":`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "cases create maps a typed handler error",
			call:   (*listenHandler).processV1CasesPost,
			uri:    "/v1/cases",
			method: sock.RequestMethodPost,
			body:   custBody,
			expect: func(d deps) {
				d.kase.EXPECT().Create(gomock.Any(), customerID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, notFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "unresolved cases require a customer",
			call:       (*listenHandler).processV1CasesUnresolvedGet,
			uri:        "/v1/cases/unresolved",
			method:     sock.RequestMethodGet,
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "unresolved cases map an untyped handler error to 500",
			call:   (*listenHandler).processV1CasesUnresolvedGet,
			uri:    "/v1/cases/unresolved",
			method: sock.RequestMethodGet,
			body:   custBody,
			expect: func(d deps) {
				d.kase.EXPECT().CaseListUnresolved(gomock.Any(), customerID).Return(nil, fmt.Errorf("boom"))
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:       "case get rejects a URI without an id",
			call:       (*listenHandler).processV1CasesIDGet,
			uri:        "/v1/cases",
			method:     sock.RequestMethodGet,
			body:       custBody,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "case get requires a customer",
			call:       (*listenHandler).processV1CasesIDGet,
			uri:        caseURI,
			method:     sock.RequestMethodGet,
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "case continue rejects a URI without an id",
			call:       (*listenHandler).processV1CasesIDContinuePost,
			uri:        "/v1/cases",
			method:     sock.RequestMethodPost,
			body:       custBody,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "case continue rejects a malformed body",
			call:       (*listenHandler).processV1CasesIDContinuePost,
			uri:        caseURI + "/continue",
			method:     sock.RequestMethodPost,
			body:       `{"customer_id"`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "case continue requires a customer",
			call:       (*listenHandler).processV1CasesIDContinuePost,
			uri:        caseURI + "/continue",
			method:     sock.RequestMethodPost,
			body:       `{}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "contacts list rejects a filter with an invalid value",
			call:       (*listenHandler).processV1ContactsGet,
			uri:        "/v1/contacts?page_size=5",
			method:     sock.RequestMethodGet,
			body:       `{"customer_id":"not-a-uuid"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "contacts list returns the URI parse error",
			call:    (*listenHandler).processV1ContactsGet,
			uri:     "/v1/contacts/%zz",
			method:  sock.RequestMethodGet,
			wantErr: true,
		},
		{
			name:       "contact get rejects a URI without an id",
			call:       (*listenHandler).processV1ContactsIDGet,
			uri:        "/v1/contacts",
			method:     sock.RequestMethodGet,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "contact update rejects a URI without an id",
			call:       (*listenHandler).processV1ContactsIDPut,
			uri:        "/v1/contacts",
			method:     sock.RequestMethodPut,
			body:       `{"first_name":"A"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "contact delete rejects a URI without an id",
			call:       (*listenHandler).processV1ContactsIDDelete,
			uri:        "/v1/contacts",
			method:     sock.RequestMethodDelete,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:    "contact lookup returns the URI parse error",
			call:    (*listenHandler).processV1ContactsLookupGet,
			uri:     "/v1/contacts/lookup%zz",
			method:  sock.RequestMethodGet,
			wantErr: true,
		},
		{
			name:       "contact tag add rejects a URI without the tag segment",
			call:       (*listenHandler).processV1ContactsTagsPost,
			uri:        "/v1/contacts/" + contactID.String(),
			method:     sock.RequestMethodPost,
			body:       `{"tag_id":"x"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "contact tag remove rejects a URI without the tag id",
			call:       (*listenHandler).processV1ContactsTagsIDDelete,
			uri:        "/v1/contacts/" + contactID.String() + "/tags",
			method:     sock.RequestMethodDelete,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "contact address list rejects a malformed URI",
			call:       (*listenHandler).processV1ContactAddressesGet,
			uri:        "/v1/contact_addresses/%zz",
			method:     sock.RequestMethodGet,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "contact address claim rejects a URI without the claim segment",
			call:       (*listenHandler).processV1ContactAddressesIDClaim,
			uri:        "/v1/contact_addresses/" + addressID.String(),
			method:     sock.RequestMethodPost,
			body:       `{"contact_id":"x"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "contact address claim rejects a malformed URI",
			call:       (*listenHandler).processV1ContactAddressesIDClaim,
			uri:        "/v1/contact_addresses/" + addressID.String() + "/claim%zz",
			method:     sock.RequestMethodPost,
			body:       `{"contact_id":"x"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "contact address update under a contact rejects a short URI",
			call:       (*listenHandler).processV1ContactsAddressesIDPut,
			uri:        "/v1/contacts/" + contactID.String() + "/addresses",
			method:     sock.RequestMethodPut,
			body:       `{"name":"x"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "interactions list rejects a malformed URI",
			call:       (*listenHandler).processV1InteractionsGet,
			uri:        "/v1/interactions%zz",
			method:     sock.RequestMethodGet,
			body:       custBody,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockContact, _, mockCase := newTestListenHandlerAll(mc)
			if tt.expect != nil {
				tt.expect(deps{contact: mockContact, kase: mockCase})
			}

			res, err := tt.call(h, context.Background(), &sock.Request{URI: tt.uri, Method: tt.method, DataType: "application/json", Data: []byte(tt.body)})
			if tt.wantErr {
				if err == nil {
					t.Fatalf("error = nil, want the URI parse error")
				}
				if res != nil {
					t.Errorf("response = %v, want nil on error", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if res.StatusCode != tt.wantStatus {
				t.Errorf("status = %d, want %d", res.StatusCode, tt.wantStatus)
			}
		})
	}
}

// The contact address detail field is forwarded to the handler as given.
func Test_processV1ContactsAddressesIDPut_Detail(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	contactID := uuid.FromStringOrNil("66666666-0003-0003-0003-000000000001")
	addressID := uuid.FromStringOrNil("66666666-0003-0003-0003-000000000002")
	h, mockContact, _, _ := newTestListenHandlerAll(mc)

	mockContact.EXPECT().
		UpdateAddress(gomock.Any(), contactID, addressID, map[string]any{"detail": "front desk"}).
		Return(&contact.Contact{FirstName: "Pat"}, nil)

	res, err := h.processV1ContactsAddressesIDPut(context.Background(), &sock.Request{
		URI:    "/v1/contacts/" + contactID.String() + "/addresses/" + addressID.String(),
		Method: sock.RequestMethodPut,
		Data:   []byte(`{"detail":"front desk"}`),
	})
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
}

// Requests for the contact address list and for case creation reach their
// handlers through the request router.
func Test_processRequest_AddressListAndCaseCreateRoutes(t *testing.T) {
	tests := []struct {
		name   string
		uri    string
		method sock.RequestMethod
	}{
		{name: "contact address list without a customer", uri: "/v1/contact_addresses?contact_id=none", method: sock.RequestMethodGet},
		{name: "case create with a malformed body", uri: "/v1/cases", method: sock.RequestMethodPost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _, _ := newTestListenHandlerAll(mc)

			// both handlers answer 400 for these inputs before calling any
			// business handler, which the strict mocks verify
			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method, Data: []byte(`{`)})
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if res.StatusCode != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", res.StatusCode, http.StatusBadRequest)
			}
		})
	}
}
