package listenhandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

func Test_processV1ContactsAddresses_Rejected(t *testing.T) {
	contactID := uuid.FromStringOrNil("44444444-0081-0081-0081-000000000001")

	type fn func(h *listenHandler, ctx context.Context, req *sock.Request) (*sock.Response, error)

	tests := []struct {
		name string
		call fn
		req  *sock.Request
	}{
		{"list short uri", (*listenHandler).processV1ContactsAddressesGet, &sock.Request{URI: "/v1/contacts/" + contactID.String()}},
		{"create short uri", (*listenHandler).processV1ContactsAddressesPost, &sock.Request{URI: "/v1/contacts/" + contactID.String(), Data: []byte(`{}`)}},
		{"create malformed body", (*listenHandler).processV1ContactsAddressesPost, &sock.Request{URI: "/v1/contacts/" + contactID.String() + "/addresses", Data: []byte(`{bad`)}},
		{"delete short uri", (*listenHandler).processV1ContactsAddressesIDDelete, &sock.Request{URI: "/v1/contacts/" + contactID.String() + "/addresses"}},
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

func Test_processV1ContactsAddressesGet_Result(t *testing.T) {
	contactID := uuid.FromStringOrNil("44444444-0082-0082-0082-000000000001")
	addressID := uuid.FromStringOrNil("44444444-0082-0082-0082-000000000002")
	uri := "/v1/contacts/" + contactID.String() + "/addresses"

	tests := []struct {
		name      string
		contact   *contact.Contact
		expectLen int
	}{
		{"contact with addresses", &contact.Contact{Addresses: []contact.Address{{ID: addressID}}}, 1},
		{"contact without addresses yields an empty list, not null", &contact.Contact{}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockContact, _, _ := newTestListenHandlerAll(mc)

			mockContact.EXPECT().Get(gomock.Any(), contactID).Return(tt.contact, nil)

			res, err := h.processV1ContactsAddressesGet(context.Background(), &sock.Request{URI: uri, Method: sock.RequestMethodGet})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res.StatusCode != 200 || res.DataType != "application/json" {
				t.Fatalf("response = %d %q, want 200 application/json", res.StatusCode, res.DataType)
			}

			var raw map[string]json.RawMessage
			if errUnmarshal := json.Unmarshal(res.Data, &raw); errUnmarshal != nil {
				t.Fatalf("body: %v", errUnmarshal)
			}
			if string(raw["result"]) == "null" {
				t.Fatalf("result must be a list, got null")
			}
			var list []contact.Address
			if errUnmarshal := json.Unmarshal(raw["result"], &list); errUnmarshal != nil || len(list) != tt.expectLen {
				t.Errorf("unexpected result: %s (%v)", raw["result"], errUnmarshal)
			}
		})
	}
}

func Test_processRequest_ContactsAddressesGet_ErrorConversion(t *testing.T) {
	contactID := uuid.FromStringOrNil("44444444-0083-0083-0083-000000000001")
	uri := "/v1/contacts/" + contactID.String() + "/addresses"

	tests := []struct {
		name   string
		err    error
		expect int
	}{
		{"typed not found is mapped to 404", cerrors.NotFound(commonoutline.ServiceNameContactManager, "CONTACT_NOT_FOUND", "no such contact"), 404},
		{"legacy not found is mapped to 404", dbhandler.ErrNotFound, 404},
		{"typed conflict is mapped to 409", cerrors.AlreadyExists(commonoutline.ServiceNameContactManager, "ADDRESS_ALREADY_EXISTS", "exists"), 409},
		{"any other error is a 400", fmt.Errorf("storage failure"), 400},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockContact, _, _ := newTestListenHandlerAll(mc)

			mockContact.EXPECT().Get(gomock.Any(), contactID).Return(nil, tt.err)

			res, err := h.processRequest(&sock.Request{URI: uri, Method: sock.RequestMethodGet})
			if err != nil {
				t.Fatalf("processRequest() must convert the error into a response, got err = %v", err)
			}
			if res.StatusCode != tt.expect {
				t.Errorf("StatusCode = %d, want %d", res.StatusCode, tt.expect)
			}
		})
	}
}

func Test_processV1ContactsAddressesPost_Created(t *testing.T) {
	contactID := uuid.FromStringOrNil("44444444-0084-0084-0084-000000000001")
	addressID := uuid.FromStringOrNil("44444444-0084-0084-0084-000000000002")

	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockContact, _, _ := newTestListenHandlerAll(mc)

	mockContact.EXPECT().AddAddress(gomock.Any(), contactID, gomock.Any()).DoAndReturn(
		func(_ context.Context, _ uuid.UUID, a *contact.Address) (*contact.Address, error) {
			if a.Type != "tel" || a.Target != "+15550000004" || a.Name != "mobile" || a.Detail != "night" || !a.IsPrimary {
				t.Errorf("unexpected address forwarded: %+v", a)
			}
			return &contact.Address{ID: addressID}, nil
		})

	body := `{"type":"tel","target":"+15550000004","name":"mobile","detail":"night","is_primary":true}`
	res, err := h.processV1ContactsAddressesPost(context.Background(), &sock.Request{URI: "/v1/contacts/" + contactID.String() + "/addresses", Data: []byte(body)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var got contact.Address
	if errUnmarshal := json.Unmarshal(res.Data, &got); res.StatusCode != 201 || errUnmarshal != nil || got.ID != addressID {
		t.Errorf("response = %d %s (%v)", res.StatusCode, res.Data, errUnmarshal)
	}
}

func Test_processV1ContactsAddressesIDDelete_Result(t *testing.T) {
	contactID := uuid.FromStringOrNil("44444444-0085-0085-0085-000000000001")
	addressID := uuid.FromStringOrNil("44444444-0085-0085-0085-000000000002")
	uri := "/v1/contacts/" + contactID.String() + "/addresses/" + addressID.String()

	t.Run("success returns the updated contact", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockContact, _, _ := newTestListenHandlerAll(mc)

		mockContact.EXPECT().RemoveAddress(gomock.Any(), contactID, addressID).Return(&contact.Contact{FirstName: "Pat"}, nil)

		res, err := h.processV1ContactsAddressesIDDelete(context.Background(), &sock.Request{URI: uri, Method: sock.RequestMethodDelete})
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

		res, err := h.processV1ContactsAddressesIDDelete(context.Background(), &sock.Request{URI: uri, Method: sock.RequestMethodDelete})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.StatusCode != 400 {
			t.Errorf("StatusCode = %d, want 400", res.StatusCode)
		}
	})
}
