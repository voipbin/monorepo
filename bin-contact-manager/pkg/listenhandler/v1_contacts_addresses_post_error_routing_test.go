package listenhandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/pkg/contacthandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_processV1ContactsAddressesPost_ErrorRouting is a regression test for
// https://github.com/voipbin/monorepo/issues/1044: POST
// /v1/contacts/{id}/addresses collapsed every AddAddress error into a bare
// 500, giving callers no way to distinguish "this address already exists
// on this contact" (a typed conflict) from a genuine infrastructure
// failure or a not-found contact. processV1ContactsAddressesPost now
// routes AddAddress errors through errorResponse instead of an
// unconditional simpleResponse(500).
func Test_processV1ContactsAddressesPost_ErrorRouting(t *testing.T) {
	contactID := uuid.FromStringOrNil("11111111-0004-0004-0004-000000000001")

	tests := []struct {
		name string

		addAddressErr error
		expectCode    int
	}{
		{
			name: "duplicate address -> 409, not 500",
			addAddressErr: cerrors.AlreadyExists(
				commonoutline.ServiceNameContactManager,
				"ADDRESS_ALREADY_EXISTS",
				"An address with this type and target already exists for this customer.",
			),
			expectCode: 409,
		},
		{
			name:          "nonexistent contact -> 404, not 500",
			addAddressErr: dbhandler.ErrNotFound,
			expectCode:    404,
		},
		{
			name:          "genuine infrastructure failure -> 500",
			addAddressErr: cerrors.Internal(commonoutline.ServiceNameContactManager, "DB_ERROR", "database is unavailable"),
			expectCode:    500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockContact := contacthandler.NewMockContactHandler(mc)
			h := &listenHandler{
				contactHandler: mockContact,
			}

			mockContact.EXPECT().AddAddress(gomock.Any(), contactID, gomock.Any()).Return(nil, tt.addAddressErr)

			request := &sock.Request{
				URI:      "/v1/contacts/" + contactID.String() + "/addresses",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"type":"tel","target":"+1-555-000-0000"}`),
			}

			res, err := h.processRequest(request)
			if err != nil {
				t.Fatalf("processRequest() error = %v", err)
			}
			if res == nil {
				t.Fatalf("processRequest() returned nil response; request did not match any route")
			}
			if res.StatusCode != tt.expectCode {
				t.Errorf("processRequest() StatusCode = %d, want %d", res.StatusCode, tt.expectCode)
			}
		})
	}
}

// Test_processV1ContactAddressesPost_ErrorRouting covers the sibling
// independent-resource endpoint (POST /v1/contact_addresses with
// contact_id in the body), which shares the exact same AddAddress call and
// the exact same bug for issue #1044, even though only the
// /contacts/{id}/addresses endpoint was named in the issue text.
func Test_processV1ContactAddressesPost_ErrorRouting(t *testing.T) {
	contactID := uuid.FromStringOrNil("11111111-0004-0004-0004-000000000002")

	tests := []struct {
		name string

		addAddressErr error
		expectCode    int
	}{
		{
			name: "duplicate address -> 409, not 500",
			addAddressErr: cerrors.AlreadyExists(
				commonoutline.ServiceNameContactManager,
				"ADDRESS_ALREADY_EXISTS",
				"An address with this type and target already exists for this customer.",
			),
			expectCode: 409,
		},
		{
			name:          "nonexistent contact -> 404, not 500",
			addAddressErr: dbhandler.ErrNotFound,
			expectCode:    404,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockContact := contacthandler.NewMockContactHandler(mc)
			h := &listenHandler{
				contactHandler: mockContact,
			}

			mockContact.EXPECT().AddAddress(gomock.Any(), contactID, gomock.Any()).Return(nil, tt.addAddressErr)

			request := &sock.Request{
				URI:      "/v1/contact_addresses",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"contact_id":"` + contactID.String() + `","type":"tel","target":"+1-555-000-0000"}`),
			}

			res, err := h.processRequest(request)
			if err != nil {
				t.Fatalf("processRequest() error = %v", err)
			}
			if res == nil {
				t.Fatalf("processRequest() returned nil response; request did not match any route")
			}
			if res.StatusCode != tt.expectCode {
				t.Errorf("processRequest() StatusCode = %d, want %d", res.StatusCode, tt.expectCode)
			}
		})
	}
}
