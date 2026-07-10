package listenhandler

import (
	"testing"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/addresshandler"
	"monorepo/bin-contact-manager/pkg/contacthandler"
)

// Test_processRequest_ContactAddressesID_Routing is a regression test for
// https://github.com/voipbin/monorepo/issues/1042: GET/PUT/DELETE
// /v1/contact_addresses/{id} requests always carry a trailing
// "?customer_id=..." (or "?contact_id=...") query string, appended by every
// caller in bin-common-handler/pkg/requesthandler/contact_contact_addresses.go.
// regV1ContactAddressesID previously ended in a bare "$" anchor, so it never
// matched these URIs and the request fell through processRequest's switch
// with no matching case, silently returning a 404-equivalent response.
//
// This test dispatches through processRequest (not the handler function
// directly) so the URI-matching routing layer itself is exercised, matching
// the class of bug that unit-testing the handler function alone missed.
func Test_processRequest_ContactAddressesID_Routing(t *testing.T) {
	customerID := uuid.FromStringOrNil("11111111-0002-0002-0002-000000000001")
	contactID := uuid.FromStringOrNil("11111111-0002-0002-0002-000000000002")
	addressID := uuid.FromStringOrNil("11111111-0002-0002-0002-000000000003")

	tests := []struct {
		name    string
		request *sock.Request

		setupMock func(mockAddress *addresshandler.MockAddressHandler, mockContact *contacthandler.MockContactHandler)

		expectCode int
	}{
		{
			name: "GET /contact_addresses/{id}?customer_id=... routes correctly",
			request: &sock.Request{
				URI:    "/v1/contact_addresses/" + addressID.String() + "?customer_id=" + customerID.String(),
				Method: sock.RequestMethodGet,
			},
			setupMock: func(mockAddress *addresshandler.MockAddressHandler, _ *contacthandler.MockContactHandler) {
				mockAddress.EXPECT().GetAddress(gomock.Any(), customerID, addressID).Return(&contact.Address{
					ID:         addressID,
					CustomerID: customerID,
				}, nil)
			},
			expectCode: 200,
		},
		{
			name: "PUT /contact_addresses/{id}?contact_id=... routes correctly",
			request: &sock.Request{
				URI:    "/v1/contact_addresses/" + addressID.String() + "?contact_id=" + contactID.String(),
				Method: sock.RequestMethodPut,
				Data:   []byte(`{"target":"+15551234567"}`),
			},
			setupMock: func(_ *addresshandler.MockAddressHandler, mockContact *contacthandler.MockContactHandler) {
				mockContact.EXPECT().UpdateAddress(gomock.Any(), contactID, addressID, gomock.Any()).Return(&contact.Contact{
					Addresses: []contact.Address{
						{Address: commonaddress.Address{Target: "+15551234567"}, ID: addressID, CustomerID: customerID},
					},
				}, nil)
			},
			expectCode: 200,
		},
		{
			name: "DELETE /contact_addresses/{id}?contact_id=... routes correctly",
			request: &sock.Request{
				URI:    "/v1/contact_addresses/" + addressID.String() + "?contact_id=" + contactID.String(),
				Method: sock.RequestMethodDelete,
			},
			setupMock: func(_ *addresshandler.MockAddressHandler, mockContact *contacthandler.MockContactHandler) {
				mockContact.EXPECT().RemoveAddress(gomock.Any(), contactID, addressID).Return(&contact.Contact{}, nil)
			},
			expectCode: 200,
		},
		{
			// Same bug class as #1042, found in review: the /claim sub-route
			// regex (regV1ContactAddressesIDClaim) also ended in a bare "$"
			// anchor while ContactV1ContactAddressClaim appends
			// "?customer_id=..." to every call, so claim requests 404'd too.
			name: "POST /contact_addresses/{id}/claim?customer_id=... routes correctly",
			request: &sock.Request{
				URI:    "/v1/contact_addresses/" + addressID.String() + "/claim?customer_id=" + customerID.String(),
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"contact_id":"` + contactID.String() + `"}`),
			},
			setupMock: func(_ *addresshandler.MockAddressHandler, mockContact *contacthandler.MockContactHandler) {
				mockContact.EXPECT().ClaimAddress(gomock.Any(), customerID, addressID, contactID).Return(&contact.Address{
					ID:         addressID,
					CustomerID: customerID,
				}, nil)
			},
			expectCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAddress := addresshandler.NewMockAddressHandler(mc)
			mockContact := contacthandler.NewMockContactHandler(mc)
			tt.setupMock(mockAddress, mockContact)

			h := &listenHandler{
				addressHandler: mockAddress,
				contactHandler: mockContact,
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Fatalf("processRequest() error = %v", err)
			}
			if res == nil {
				t.Fatalf("processRequest() returned nil response; request did not match any route (regression of #1042)")
			}
			if res.StatusCode != tt.expectCode {
				t.Errorf("processRequest() StatusCode = %d, want %d", res.StatusCode, tt.expectCode)
			}
		})
	}
}
