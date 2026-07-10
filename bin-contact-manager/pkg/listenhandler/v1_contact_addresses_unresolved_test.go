package listenhandler

import (
	"context"
	"testing"

	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/addresshandler"
	"monorepo/bin-contact-manager/pkg/contacthandler"
)

// Test_processV1ContactAddressesPost_Unresolved covers §5.1: creation
// without contact_id succeeds when customer_id is present, 400s when
// customer_id is also absent, and 400s when is_primary=true is combined
// with no contact_id.
func Test_processV1ContactAddressesPost_Unresolved(t *testing.T) {
	customerID := uuid.FromStringOrNil("11111111-0001-0001-0001-000000000001")
	addressID := uuid.FromStringOrNil("11111111-0001-0001-0001-000000000002")

	tests := []struct {
		name string
		body []byte

		expectCall bool
		expectCode int
	}{
		{
			name:       "unresolved create succeeds with customer_id",
			body:       []byte(`{"customer_id":"11111111-0001-0001-0001-000000000001","type":"tel","target":"+15559998888"}`),
			expectCall: true,
			expectCode: 201,
		},
		{
			name:       "missing customer_id is a 400",
			body:       []byte(`{"type":"tel","target":"+15559998888"}`),
			expectCall: false,
			expectCode: 400,
		},
		{
			name:       "is_primary true with no contact_id is a 400",
			body:       []byte(`{"customer_id":"11111111-0001-0001-0001-000000000001","type":"tel","target":"+15559998888","is_primary":true}`),
			expectCall: false,
			expectCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockContact := contacthandler.NewMockContactHandler(mc)
			h := &listenHandler{contactHandler: mockContact}

			if tt.expectCall {
				mockContact.EXPECT().CreateUnresolvedAddress(gomock.Any(), customerID, gomock.Any()).Return(&contact.Address{
					Address: commonaddress.Address{
						Type:   "tel",
						Target: "+15559998888",
					},
					ID:         addressID,
					CustomerID: customerID,
				}, nil)
			}

			res, err := h.processV1ContactAddressesPost(context.Background(), &sock.Request{
				URI:    "/v1/contact_addresses",
				Method: sock.RequestMethodPost,
				Data:   tt.body,
			})
			if err != nil {
				t.Fatalf("processV1ContactAddressesPost() error = %v", err)
			}
			if res.StatusCode != tt.expectCode {
				t.Errorf("processV1ContactAddressesPost() StatusCode = %d, want %d", res.StatusCode, tt.expectCode)
			}
		})
	}
}

// Test_processV1ContactAddressesGet_Unresolved covers §5.2: the
// unresolved=true query param is threaded into the ListAddresses filters.
func Test_processV1ContactAddressesGet_Unresolved(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockAddress := addresshandler.NewMockAddressHandler(mc)
	h := &listenHandler{addressHandler: mockAddress}

	customerID := uuid.FromStringOrNil("11111111-0002-0002-0002-000000000001")

	mockAddress.EXPECT().ListAddresses(gomock.Any(), customerID, map[string]any{"unresolved": true}, "", uint64(20)).Return([]contact.Address{}, nil)

	res, err := h.processV1ContactAddressesGet(context.Background(), &sock.Request{
		URI:    "/v1/contact_addresses?customer_id=11111111-0002-0002-0002-000000000001&unresolved=true",
		Method: sock.RequestMethodGet,
	})
	if err != nil {
		t.Fatalf("processV1ContactAddressesGet() error = %v", err)
	}
	if res.StatusCode != 200 {
		t.Errorf("processV1ContactAddressesGet() StatusCode = %d, want 200", res.StatusCode)
	}
}

// Test_processV1ContactAddressesIDClaim covers §5.3's response matrix.
func Test_processV1ContactAddressesIDClaim(t *testing.T) {
	addressID := uuid.FromStringOrNil("11111111-0003-0003-0003-000000000001")
	customerID := uuid.FromStringOrNil("11111111-0003-0003-0003-000000000002")
	contactID := uuid.FromStringOrNil("11111111-0003-0003-0003-000000000003")

	uri := "/v1/contact_addresses/" + addressID.String() + "/claim?customer_id=" + customerID.String()

	tests := []struct {
		name string
		body []byte

		setupMock  func(m *contacthandler.MockContactHandler)
		expectCode int
	}{
		{
			name: "200 claimed successfully",
			body: []byte(`{"contact_id":"` + contactID.String() + `"}`),
			setupMock: func(m *contacthandler.MockContactHandler) {
				m.EXPECT().ClaimAddress(gomock.Any(), customerID, addressID, contactID).Return(&contact.Address{
					ID:         addressID,
					CustomerID: customerID,
					ContactID:  contactID,
				}, nil)
			},
			expectCode: 200,
		},
		{
			name:       "400 missing contact_id in body",
			body:       []byte(`{}`),
			setupMock:  func(m *contacthandler.MockContactHandler) {},
			expectCode: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockContact := contacthandler.NewMockContactHandler(mc)
			tt.setupMock(mockContact)
			h := &listenHandler{contactHandler: mockContact}

			res, err := h.processV1ContactAddressesIDClaim(context.Background(), &sock.Request{
				URI:    uri,
				Method: sock.RequestMethodPost,
				Data:   tt.body,
			})
			if err != nil {
				t.Fatalf("processV1ContactAddressesIDClaim() error = %v", err)
			}
			if res.StatusCode != tt.expectCode {
				t.Errorf("processV1ContactAddressesIDClaim() StatusCode = %d, want %d", res.StatusCode, tt.expectCode)
			}
		})
	}
}
