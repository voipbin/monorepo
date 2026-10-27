package addresshandler

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

func Test_GetAddress(t *testing.T) {
	customerID := uuid.FromStringOrNil("0a1b2c3d-1659-4000-8000-000000000001")
	addressID := uuid.FromStringOrNil("0a1b2c3d-1659-4000-8000-000000000002")
	errBoom := errors.New("injected failure")

	tests := []struct {
		name string

		responseAddress *contact.Address
		responseErr     error

		expectRes *contact.Address
		expectErr error
	}{
		{
			name:            "normal",
			responseAddress: &contact.Address{ID: addressID, CustomerID: customerID, Target: "+155****0001"},
			expectRes:       &contact.Address{ID: addressID, CustomerID: customerID, Target: "+155****0001"},
		},
		{
			name:        "not found is passed through",
			responseErr: dbhandler.ErrNotFound,
			expectErr:   dbhandler.ErrNotFound,
		},
		{
			name:        "database error is passed through",
			responseErr: errBoom,
			expectErr:   errBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &addressHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().AddressGet(ctx, customerID, addressID).Return(tt.responseAddress, tt.responseErr)

			res, err := h.GetAddress(ctx, customerID, addressID)
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ListAddresses(t *testing.T) {
	customerID := uuid.FromStringOrNil("0a1b2c3d-1659-4000-8000-000000000001")
	contactID := uuid.FromStringOrNil("0a1b2c3d-1659-4000-8000-000000000003")
	errBoom := errors.New("injected failure")

	tests := []struct {
		name string

		filters   map[string]any
		pageToken string
		pageSize  uint64

		responseAddresses []contact.Address
		responseErr       error

		expectRes []contact.Address
		expectErr error
	}{
		{
			name:              "filters and paging are forwarded",
			filters:           map[string]any{"contact_id": contactID, "type": "tel"},
			pageToken:         "2026-01-01T00:00:00.000000Z",
			pageSize:          25,
			responseAddresses: []contact.Address{{Target: "+155****0001"}, {Target: "+155****0002"}},
			expectRes:         []contact.Address{{Target: "+155****0001"}, {Target: "+155****0002"}},
		},
		{
			name:        "database error is passed through",
			filters:     map[string]any{},
			pageSize:    10,
			responseErr: errBoom,
			expectErr:   errBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &addressHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().AddressList(ctx, customerID, tt.filters, tt.pageToken, tt.pageSize).Return(tt.responseAddresses, tt.responseErr)

			res, err := h.ListAddresses(ctx, customerID, tt.filters, tt.pageToken, tt.pageSize)
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_NewAddressHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)

	res := NewAddressHandler(mockDB)

	h, ok := res.(*addressHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *addressHandler, got: %T", res)
	}
	if h.db != dbhandler.DBHandler(mockDB) {
		t.Errorf("Wrong match. expect: injected db handler, got: %v", h.db)
	}
}
