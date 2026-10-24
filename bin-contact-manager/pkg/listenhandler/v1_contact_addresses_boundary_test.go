package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

func Test_processV1ContactAddresses_Rejected(t *testing.T) {
	customerID := uuid.FromStringOrNil("33333333-0071-0071-0071-000000000001")
	contactID := uuid.FromStringOrNil("33333333-0071-0071-0071-000000000002")
	addressID := uuid.FromStringOrNil("33333333-0071-0071-0071-000000000003")

	type fn func(h *listenHandler, ctx context.Context, req *sock.Request) (*sock.Response, error)

	tests := []struct {
		name string
		call fn
		req  *sock.Request
	}{
		{"list unparsable uri", (*listenHandler).processV1ContactAddressesGet, &sock.Request{URI: "/v1/contact_addresses?%zz"}},
		{"list missing customer_id", (*listenHandler).processV1ContactAddressesGet, &sock.Request{URI: "/v1/contact_addresses?type=tel"}},
		{"list malformed customer_id", (*listenHandler).processV1ContactAddressesGet, &sock.Request{URI: "/v1/contact_addresses?customer_id=abc"}},
		{"create malformed body", (*listenHandler).processV1ContactAddressesPost, &sock.Request{URI: "/v1/contact_addresses", Data: []byte(`{bad`)}},
		{"create unresolved without customer_id", (*listenHandler).processV1ContactAddressesPost, &sock.Request{URI: "/v1/contact_addresses", Data: []byte(`{"type":"tel","target":"+15550000001"}`)}},
		{"create unresolved primary", (*listenHandler).processV1ContactAddressesPost, &sock.Request{URI: "/v1/contact_addresses", Data: []byte(fmt.Sprintf(`{"customer_id":"%s","type":"tel","target":"+15550000001","is_primary":true}`, customerID))}},
		{"get unparsable uri", (*listenHandler).processV1ContactAddressesIDGet, &sock.Request{URI: "/v1/contact_addresses/%zz"}},
		{"get missing path id", (*listenHandler).processV1ContactAddressesIDGet, &sock.Request{URI: "/v1/contact_addresses"}},
		{"get missing customer_id", (*listenHandler).processV1ContactAddressesIDGet, &sock.Request{URI: "/v1/contact_addresses/" + addressID.String()}},
		{"update unparsable uri", (*listenHandler).processV1ContactAddressesIDPut, &sock.Request{URI: "/v1/contact_addresses/%zz"}},
		{"update missing path id", (*listenHandler).processV1ContactAddressesIDPut, &sock.Request{URI: "/v1/contact_addresses", Data: []byte(`{}`)}},
		{"update malformed body", (*listenHandler).processV1ContactAddressesIDPut, &sock.Request{URI: "/v1/contact_addresses/" + addressID.String(), Data: []byte(`{bad`)}},
		{"claim short uri", (*listenHandler).processV1ContactAddressesIDClaim, &sock.Request{URI: "/v1/contact_addresses/" + addressID.String()}},
		{"claim missing customer_id", (*listenHandler).processV1ContactAddressesIDClaim, &sock.Request{URI: "/v1/contact_addresses/" + addressID.String() + "/claim", Data: []byte(fmt.Sprintf(`{"contact_id":"%s"}`, contactID))}},
		{"claim malformed body", (*listenHandler).processV1ContactAddressesIDClaim, &sock.Request{URI: "/v1/contact_addresses/" + addressID.String() + "/claim?customer_id=" + customerID.String(), Data: []byte(`{bad`)}},
		{"claim missing contact_id", (*listenHandler).processV1ContactAddressesIDClaim, &sock.Request{URI: "/v1/contact_addresses/" + addressID.String() + "/claim?customer_id=" + customerID.String(), Data: []byte(`{"force":true}`)}},
		{"delete unparsable uri", (*listenHandler).processV1ContactAddressesIDDelete, &sock.Request{URI: "/v1/contact_addresses/%zz"}},
		{"delete missing path id", (*listenHandler).processV1ContactAddressesIDDelete, &sock.Request{URI: "/v1/contact_addresses"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no downstream expectation: strict gomock fails if any handler is reached.
			h, _, _, _ := newTestListenHandlerAll(mc)

			res, err := tt.call(h, context.Background(), tt.req)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != 400 {
				t.Errorf("StatusCode = %d, want 400", res.StatusCode)
			}
		})
	}
}

func Test_processV1ContactAddressesGet_Filters(t *testing.T) {
	customerID := uuid.FromStringOrNil("33333333-0072-0072-0072-000000000001")
	contactID := uuid.FromStringOrNil("33333333-0072-0072-0072-000000000002")
	addressID := uuid.FromStringOrNil("33333333-0072-0072-0072-000000000003")

	tests := []struct {
		name          string
		query         string
		expectFilters map[string]any
		expectToken   string
	}{
		{
			name:          "no filters",
			query:         "",
			expectFilters: map[string]any{},
		},
		{
			name:  "all filters",
			query: "&contact_id=" + contactID.String() + "&type=tel&target=%2B15550000001&unresolved=true&page_token=tok1",
			expectFilters: map[string]any{
				"contact_id": contactID,
				"type":       "tel",
				"target":     "+15550000001",
				"unresolved": true,
			},
			expectToken: "tok1",
		},
		{
			name:          "malformed contact_id is dropped and unresolved other than true is ignored",
			query:         "&contact_id=abc&unresolved=false",
			expectFilters: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, _, mockAddr, _ := newTestListenHandlerAll(mc)

			mockAddr.EXPECT().ListAddresses(gomock.Any(), customerID, tt.expectFilters, tt.expectToken, uint64(20)).
				Return([]contact.Address{{ID: addressID, Address: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000001"}}}, nil)

			res, err := h.processV1ContactAddressesGet(context.Background(), &sock.Request{URI: "/v1/contact_addresses?customer_id=" + customerID.String() + tt.query})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != 200 {
				t.Fatalf("StatusCode = %d, want 200", res.StatusCode)
			}
			var got struct {
				Result []contact.Address `json:"result"`
			}
			if errUnmarshal := json.Unmarshal(res.Data, &got); errUnmarshal != nil {
				t.Fatalf("body: %v", errUnmarshal)
			}
			if len(got.Result) != 1 || got.Result[0].ID != addressID {
				t.Errorf("unexpected body: %s", res.Data)
			}
		})
	}
}

func Test_processV1ContactAddressesGet_ListError(t *testing.T) {
	customerID := uuid.FromStringOrNil("33333333-0073-0073-0073-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()
	h, _, mockAddr, _ := newTestListenHandlerAll(mc)

	mockAddr.EXPECT().ListAddresses(gomock.Any(), customerID, map[string]any{}, "", uint64(20)).Return(nil, fmt.Errorf("storage failure"))

	res, err := h.processV1ContactAddressesGet(context.Background(), &sock.Request{URI: "/v1/contact_addresses?customer_id=" + customerID.String()})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.StatusCode != 500 {
		t.Errorf("StatusCode = %d, want 500", res.StatusCode)
	}
}

func Test_processV1ContactAddressesPost_Create(t *testing.T) {
	customerID := uuid.FromStringOrNil("33333333-0074-0074-0074-000000000001")
	contactID := uuid.FromStringOrNil("33333333-0074-0074-0074-000000000002")
	addressID := uuid.FromStringOrNil("33333333-0074-0074-0074-000000000003")

	t.Run("attached to a contact returns 201 with the address", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockContact, _, _ := newTestListenHandlerAll(mc)

		expectAddr := &contact.Address{
			Address:   commonaddress.Address{Type: commonaddress.TypeEmail, Target: "user-a@example.test", Name: "work", Detail: "desk"},
			IsPrimary: true,
		}
		mockContact.EXPECT().AddAddress(gomock.Any(), contactID, expectAddr).Return(&contact.Address{ID: addressID}, nil)

		body := fmt.Sprintf(`{"contact_id":"%s","type":"email","target":"user-a@example.test","name":"work","detail":"desk","is_primary":true}`, contactID)
		res, err := h.processV1ContactAddressesPost(context.Background(), &sock.Request{URI: "/v1/contact_addresses", Data: []byte(body)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.StatusCode != 201 || res.DataType != "application/json" {
			t.Fatalf("response = %d %q, want 201 application/json", res.StatusCode, res.DataType)
		}
		var got contact.Address
		if errUnmarshal := json.Unmarshal(res.Data, &got); errUnmarshal != nil || got.ID != addressID {
			t.Errorf("unexpected body: %s (%v)", res.Data, errUnmarshal)
		}
	})

	t.Run("unresolved address create error is a 500", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockContact, _, _ := newTestListenHandlerAll(mc)

		mockContact.EXPECT().CreateUnresolvedAddress(gomock.Any(), customerID, gomock.Any()).Return(nil, fmt.Errorf("storage failure"))

		body := fmt.Sprintf(`{"customer_id":"%s","type":"tel","target":"+15550000001"}`, customerID)
		res, err := h.processV1ContactAddressesPost(context.Background(), &sock.Request{URI: "/v1/contact_addresses", Data: []byte(body)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.StatusCode != 500 {
			t.Errorf("StatusCode = %d, want 500", res.StatusCode)
		}
	})

	t.Run("unresolved address create success", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockContact, _, _ := newTestListenHandlerAll(mc)

		expectAddr := &contact.Address{Address: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000001"}}
		mockContact.EXPECT().CreateUnresolvedAddress(gomock.Any(), customerID, expectAddr).Return(&contact.Address{ID: addressID}, nil)

		body := fmt.Sprintf(`{"customer_id":"%s","type":"tel","target":"+15550000001"}`, customerID)
		res, err := h.processV1ContactAddressesPost(context.Background(), &sock.Request{URI: "/v1/contact_addresses", Data: []byte(body)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var got contact.Address
		if errUnmarshal := json.Unmarshal(res.Data, &got); res.StatusCode != 201 || errUnmarshal != nil || got.ID != addressID {
			t.Errorf("response = %d %s (%v)", res.StatusCode, res.Data, errUnmarshal)
		}
	})
}

func Test_processV1ContactAddressesIDGet(t *testing.T) {
	customerID := uuid.FromStringOrNil("33333333-0075-0075-0075-000000000001")
	addressID := uuid.FromStringOrNil("33333333-0075-0075-0075-000000000002")
	uri := "/v1/contact_addresses/" + addressID.String() + "?customer_id=" + customerID.String()

	t.Run("success", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, _, mockAddr, _ := newTestListenHandlerAll(mc)

		mockAddr.EXPECT().GetAddress(gomock.Any(), customerID, addressID).Return(&contact.Address{ID: addressID}, nil)

		res, err := h.processV1ContactAddressesIDGet(context.Background(), &sock.Request{URI: uri})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var got contact.Address
		if errUnmarshal := json.Unmarshal(res.Data, &got); res.StatusCode != 200 || errUnmarshal != nil || got.ID != addressID {
			t.Errorf("response = %d %s (%v)", res.StatusCode, res.Data, errUnmarshal)
		}
	})

	t.Run("not found maps to 404", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, _, mockAddr, _ := newTestListenHandlerAll(mc)

		mockAddr.EXPECT().GetAddress(gomock.Any(), customerID, addressID).Return(nil, dbhandler.ErrNotFound)

		res, err := h.processV1ContactAddressesIDGet(context.Background(), &sock.Request{URI: uri})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.StatusCode != 404 {
			t.Errorf("StatusCode = %d, want 404", res.StatusCode)
		}
	})
}

func Test_processV1ContactAddressesIDPut(t *testing.T) {
	customerID := uuid.FromStringOrNil("33333333-0076-0076-0076-000000000001")
	contactID := uuid.FromStringOrNil("33333333-0076-0076-0076-000000000002")
	addressID := uuid.FromStringOrNil("33333333-0076-0076-0076-000000000003")
	uri := "/v1/contact_addresses/" + addressID.String() + "?contact_id=" + contactID.String() + "&customer_id=" + customerID.String()
	body := []byte(`{"target":"+15550000009","name":"desk","detail":"floor 2","is_primary":true}`)
	expectFields := map[string]any{"target": "+15550000009", "name": "desk", "detail": "floor 2", "is_primary": true}

	t.Run("returns the updated address found on the contact", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockContact, _, _ := newTestListenHandlerAll(mc)

		mockContact.EXPECT().UpdateAddress(gomock.Any(), contactID, addressID, expectFields).
			Return(&contact.Contact{Addresses: []contact.Address{{ID: uuid.Must(uuid.NewV4())}, {ID: addressID, Address: commonaddress.Address{Target: "+15550000009"}}}}, nil)

		res, err := h.processV1ContactAddressesIDPut(context.Background(), &sock.Request{URI: uri, Data: body})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var got contact.Address
		if errUnmarshal := json.Unmarshal(res.Data, &got); res.StatusCode != 200 || errUnmarshal != nil || got.ID != addressID || got.Target != "+15550000009" {
			t.Errorf("response = %d %s (%v)", res.StatusCode, res.Data, errUnmarshal)
		}
	})

	t.Run("falls back to the address handler when the contact does not list it", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockContact, mockAddr, _ := newTestListenHandlerAll(mc)

		mockContact.EXPECT().UpdateAddress(gomock.Any(), contactID, addressID, expectFields).Return(&contact.Contact{}, nil)
		mockAddr.EXPECT().GetAddress(gomock.Any(), customerID, addressID).Return(&contact.Address{ID: addressID}, nil)

		res, err := h.processV1ContactAddressesIDPut(context.Background(), &sock.Request{URI: uri, Data: body})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.StatusCode != 200 {
			t.Errorf("StatusCode = %d, want 200", res.StatusCode)
		}
	})

	t.Run("fallback lookup failure is a 500", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockContact, mockAddr, _ := newTestListenHandlerAll(mc)

		mockContact.EXPECT().UpdateAddress(gomock.Any(), contactID, addressID, expectFields).Return(&contact.Contact{}, nil)
		mockAddr.EXPECT().GetAddress(gomock.Any(), customerID, addressID).Return(nil, fmt.Errorf("storage failure"))

		res, err := h.processV1ContactAddressesIDPut(context.Background(), &sock.Request{URI: uri, Data: body})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.StatusCode != 500 {
			t.Errorf("StatusCode = %d, want 500", res.StatusCode)
		}
	})

	t.Run("update failure is a 500 and only supplied fields are forwarded", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockContact, _, _ := newTestListenHandlerAll(mc)

		mockContact.EXPECT().UpdateAddress(gomock.Any(), contactID, addressID, map[string]any{"name": "desk"}).Return(nil, fmt.Errorf("storage failure"))

		res, err := h.processV1ContactAddressesIDPut(context.Background(), &sock.Request{URI: uri, Data: []byte(`{"name":"desk"}`)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.StatusCode != 500 {
			t.Errorf("StatusCode = %d, want 500", res.StatusCode)
		}
	})
}

func Test_processV1ContactAddressesIDClaim_Result(t *testing.T) {
	customerID := uuid.FromStringOrNil("33333333-0077-0077-0077-000000000001")
	contactID := uuid.FromStringOrNil("33333333-0077-0077-0077-000000000002")
	addressID := uuid.FromStringOrNil("33333333-0077-0077-0077-000000000003")
	uri := "/v1/contact_addresses/" + addressID.String() + "/claim?customer_id=" + customerID.String()
	body := []byte(fmt.Sprintf(`{"contact_id":"%s","force":true}`, contactID))

	t.Run("force flag and ids are forwarded", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockContact, _, _ := newTestListenHandlerAll(mc)

		mockContact.EXPECT().ClaimAddress(gomock.Any(), customerID, addressID, contactID, true).Return(&contact.Address{ID: addressID}, nil)

		res, err := h.processV1ContactAddressesIDClaim(context.Background(), &sock.Request{URI: uri, Data: body})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.StatusCode != 200 {
			t.Errorf("StatusCode = %d, want 200", res.StatusCode)
		}
	})

	t.Run("not found maps to 404", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockContact, _, _ := newTestListenHandlerAll(mc)

		mockContact.EXPECT().ClaimAddress(gomock.Any(), customerID, addressID, contactID, true).Return(nil, dbhandler.ErrNotFound)

		res, err := h.processV1ContactAddressesIDClaim(context.Background(), &sock.Request{URI: uri, Data: body})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.StatusCode != 404 {
			t.Errorf("StatusCode = %d, want 404", res.StatusCode)
		}
	})
}

func Test_processV1ContactAddressesIDDelete_Result(t *testing.T) {
	contactID := uuid.FromStringOrNil("33333333-0078-0078-0078-000000000001")
	addressID := uuid.FromStringOrNil("33333333-0078-0078-0078-000000000002")
	uri := "/v1/contact_addresses/" + addressID.String() + "?contact_id=" + contactID.String()

	t.Run("success returns the contact", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockContact, _, _ := newTestListenHandlerAll(mc)

		mockContact.EXPECT().RemoveAddress(gomock.Any(), contactID, addressID).Return(&contact.Contact{FirstName: "Pat"}, nil)

		res, err := h.processV1ContactAddressesIDDelete(context.Background(), &sock.Request{URI: uri})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var got contact.Contact
		if errUnmarshal := json.Unmarshal(res.Data, &got); res.StatusCode != 200 || errUnmarshal != nil || got.FirstName != "Pat" {
			t.Errorf("response = %d %s (%v)", res.StatusCode, res.Data, errUnmarshal)
		}
	})

	t.Run("remove failure is reported as 400", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockContact, _, _ := newTestListenHandlerAll(mc)

		mockContact.EXPECT().RemoveAddress(gomock.Any(), contactID, addressID).Return(nil, fmt.Errorf("storage failure"))

		res, err := h.processV1ContactAddressesIDDelete(context.Background(), &sock.Request{URI: uri})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.StatusCode != 400 {
			t.Errorf("StatusCode = %d, want 400", res.StatusCode)
		}
	})
}
