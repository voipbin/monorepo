package contacthandler

import (
	"context"
	stderrors "errors"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

var (
	errTestBoom = stderrors.New("injected database error")

	epCustomerID = uuid.FromStringOrNil("77777777-0001-0001-0001-000000000001")
	epContactID  = uuid.FromStringOrNil("77777777-0001-0001-0001-000000000002")
	epAddressID  = uuid.FromStringOrNil("77777777-0001-0001-0001-000000000003")
	epTagID      = uuid.FromStringOrNil("77777777-0001-0001-0001-000000000004")
)

// assertVoipbinError checks that err is the typed error with the given reason
// and that the original cause is still reachable.
func assertVoipbinError(t *testing.T, err error, wantReason string, wantCause error) {
	t.Helper()

	if err == nil {
		t.Fatalf("error = nil, want a typed error with reason %s", wantReason)
	}
	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("error = %v, want *cerrors.VoipbinError", err)
	}
	if ve.Reason != wantReason {
		t.Errorf("reason = %s, want %s", ve.Reason, wantReason)
	}
	if wantCause != nil && !stderrors.Is(err, wantCause) {
		t.Errorf("error %v does not wrap %v", err, wantCause)
	}
}

func liveContact() *contact.Contact {
	return &contact.Contact{Identity: commonidentity.Identity{ID: epContactID, CustomerID: epCustomerID}}
}

func deletedContact() *contact.Contact {
	c := liveContact()
	deleted := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	c.TMDelete = &deleted
	return c
}

func Test_Lookups_NotFoundMapping(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		dbErr      error
		call       func(h *contactHandler) (*contact.Contact, error)
		expect     func(m *dbhandler.MockDBHandler, err error)
		wantReason string // empty means the error is passed through untouched
	}{
		{
			name:  "get maps a missing contact to a typed not found",
			dbErr: dbhandler.ErrNotFound,
			call:  func(h *contactHandler) (*contact.Contact, error) { return h.Get(ctx, epContactID) },
			expect: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().ContactGet(ctx, epContactID).Return(nil, err)
			},
			wantReason: "CONTACT_NOT_FOUND",
		},
		{
			name:  "get passes other errors through",
			dbErr: errTestBoom,
			call:  func(h *contactHandler) (*contact.Contact, error) { return h.Get(ctx, epContactID) },
			expect: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().ContactGet(ctx, epContactID).Return(nil, err)
			},
		},
		{
			name:  "phone lookup maps a missing contact to a typed not found",
			dbErr: dbhandler.ErrNotFound,
			call: func(h *contactHandler) (*contact.Contact, error) {
				return h.LookupByPhone(ctx, epCustomerID, "+15550000001")
			},
			expect: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().ContactLookupByPhone(ctx, epCustomerID, gomock.Any()).Return(nil, err)
			},
			wantReason: "CONTACT_NOT_FOUND",
		},
		{
			name:  "phone lookup passes other errors through",
			dbErr: errTestBoom,
			call: func(h *contactHandler) (*contact.Contact, error) {
				return h.LookupByPhone(ctx, epCustomerID, "+15550000001")
			},
			expect: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().ContactLookupByPhone(ctx, epCustomerID, gomock.Any()).Return(nil, err)
			},
		},
		{
			name:  "email lookup maps a missing contact to a typed not found",
			dbErr: dbhandler.ErrNotFound,
			call: func(h *contactHandler) (*contact.Contact, error) {
				return h.LookupByEmail(ctx, epCustomerID, "user@example.com")
			},
			expect: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().ContactLookupByEmail(ctx, epCustomerID, "user@example.com").Return(nil, err)
			},
			wantReason: "CONTACT_NOT_FOUND",
		},
		{
			name:  "email lookup passes other errors through",
			dbErr: errTestBoom,
			call: func(h *contactHandler) (*contact.Contact, error) {
				return h.LookupByEmail(ctx, epCustomerID, "user@example.com")
			},
			expect: func(m *dbhandler.MockDBHandler, err error) {
				m.EXPECT().ContactLookupByEmail(ctx, epCustomerID, "user@example.com").Return(nil, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &contactHandler{db: mockDB}
			tt.expect(mockDB, tt.dbErr)

			res, err := tt.call(h)
			if res != nil {
				t.Errorf("result = %v, want nil", res)
			}
			if tt.wantReason != "" {
				assertVoipbinError(t, err, tt.wantReason, tt.dbErr)
				return
			}
			if err != tt.dbErr {
				t.Errorf("error = %v, want the untouched %v", err, tt.dbErr)
			}
		})
	}
}

func Test_AddressWrites_ErrorPropagation(t *testing.T) {
	ctx := context.Background()
	telAddress := func() *contact.Address {
		return &contact.Address{Address: commonaddress.Address{Type: contact.AddressTypeTel, Target: "+15550000002"}}
	}

	tests := []struct {
		name       string
		call       func(h *contactHandler) error
		expect     func(m *dbhandler.MockDBHandler, u *utilhandler.MockUtilHandler)
		wantReason string
		wantIs     error
		wantText   string
	}{
		{
			name: "add address rejects an unsupported address type before any write",
			call: func(h *contactHandler) error {
				_, err := h.AddAddress(ctx, epContactID, &contact.Address{Address: commonaddress.Address{Type: commonaddress.Type("fax"), Target: "x"}})
				return err
			},
			expect: func(m *dbhandler.MockDBHandler, _ *utilhandler.MockUtilHandler) {
				m.EXPECT().ContactGet(ctx, epContactID).Return(liveContact(), nil)
			},
			wantReason: "ADDRESS_TYPE_INVALID",
		},
		{
			name: "remove address returns the contact lookup error",
			call: func(h *contactHandler) error {
				_, err := h.RemoveAddress(ctx, epContactID, epAddressID)
				return err
			},
			expect: func(m *dbhandler.MockDBHandler, _ *utilhandler.MockUtilHandler) {
				m.EXPECT().ContactGet(ctx, epContactID).Return(nil, errTestBoom)
			},
			wantIs: errTestBoom,
		},
		{
			name: "remove address returns a missing address as is",
			call: func(h *contactHandler) error {
				_, err := h.RemoveAddress(ctx, epContactID, epAddressID)
				return err
			},
			expect: func(m *dbhandler.MockDBHandler, _ *utilhandler.MockUtilHandler) {
				m.EXPECT().ContactGet(ctx, epContactID).Return(liveContact(), nil)
				m.EXPECT().AddressGet(ctx, epCustomerID, epAddressID).Return(nil, dbhandler.ErrNotFound)
			},
			wantIs: dbhandler.ErrNotFound,
		},
		{
			name: "create unresolved address reports a write failure",
			call: func(h *contactHandler) error {
				_, err := h.CreateUnresolvedAddress(ctx, epCustomerID, telAddress())
				return err
			},
			expect: func(m *dbhandler.MockDBHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().UUIDCreate().Return(epAddressID)
				m.EXPECT().AddressCreate(ctx, gomock.Any()).Return(errTestBoom)
			},
			wantText: "could not create unresolved address",
			wantIs:   errTestBoom,
		},
		{
			name: "create unresolved address reports a read back failure",
			call: func(h *contactHandler) error {
				_, err := h.CreateUnresolvedAddress(ctx, epCustomerID, telAddress())
				return err
			},
			expect: func(m *dbhandler.MockDBHandler, u *utilhandler.MockUtilHandler) {
				u.EXPECT().UUIDCreate().Return(epAddressID)
				m.EXPECT().AddressCreate(ctx, gomock.Any()).Return(nil)
				m.EXPECT().AddressGet(ctx, epCustomerID, epAddressID).Return(nil, errTestBoom)
			},
			wantText: "could not get created address",
			wantIs:   errTestBoom,
		},
		{
			name: "claim address returns the contact lookup error",
			call: func(h *contactHandler) error {
				_, err := h.ClaimAddress(ctx, epCustomerID, epAddressID, epContactID, false)
				return err
			},
			expect: func(m *dbhandler.MockDBHandler, _ *utilhandler.MockUtilHandler) {
				m.EXPECT().ContactGet(ctx, epContactID).Return(nil, errTestBoom)
			},
			wantIs: errTestBoom,
		},
		{
			name: "claim address passes an unclassified write failure through",
			call: func(h *contactHandler) error {
				_, err := h.ClaimAddress(ctx, epCustomerID, epAddressID, epContactID, false)
				return err
			},
			expect: func(m *dbhandler.MockDBHandler, _ *utilhandler.MockUtilHandler) {
				m.EXPECT().ContactGet(ctx, epContactID).Return(liveContact(), nil)
				m.EXPECT().AddressClaim(ctx, epCustomerID, epAddressID, epContactID, false).Return(errTestBoom)
			},
			wantIs: errTestBoom,
		},
		{
			name: "claim address reports a read back failure after the claim",
			call: func(h *contactHandler) error {
				_, err := h.ClaimAddress(ctx, epCustomerID, epAddressID, epContactID, true)
				return err
			},
			expect: func(m *dbhandler.MockDBHandler, _ *utilhandler.MockUtilHandler) {
				m.EXPECT().ContactGet(ctx, epContactID).Return(liveContact(), nil)
				m.EXPECT().AddressClaim(ctx, epCustomerID, epAddressID, epContactID, true).Return(nil)
				m.EXPECT().AddressGet(ctx, epCustomerID, epAddressID).Return(nil, errTestBoom)
			},
			wantText: "could not get claimed address",
			wantIs:   errTestBoom,
		},
		{
			name: "remove tag reports a delete failure and reads nothing back",
			call: func(h *contactHandler) error {
				_, err := h.RemoveTag(ctx, epContactID, epTagID)
				return err
			},
			expect: func(m *dbhandler.MockDBHandler, _ *utilhandler.MockUtilHandler) {
				m.EXPECT().TagAssignmentDelete(ctx, epContactID, epTagID).Return(errTestBoom)
			},
			wantText: "could not delete tag assignment",
			wantIs:   errTestBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &contactHandler{db: mockDB, utilHandler: mockUtil, notifyHandler: notifyhandler.NewMockNotifyHandler(mc)}
			tt.expect(mockDB, mockUtil)

			err := tt.call(h)
			if err == nil {
				t.Fatal("error = nil, want a failure")
			}
			if tt.wantReason != "" {
				assertVoipbinError(t, err, tt.wantReason, tt.wantIs)
				return
			}
			if tt.wantText != "" && !strings.Contains(err.Error(), tt.wantText) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.wantText)
			}
			if tt.wantIs != nil && !stderrors.Is(err, tt.wantIs) {
				t.Errorf("error %v does not wrap %v", err, tt.wantIs)
			}
		})
	}
}

// A failing primary reset is only logged: the address is still created and the
// contact is returned.
func Test_Create_PrimaryResetFailureDoesNotStopTheAddress(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	ctx := context.Background()
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &contactHandler{db: mockDB, utilHandler: mockUtil, notifyHandler: mockNotify}

	addressID := uuid.FromStringOrNil("77777777-0002-0002-0002-000000000001")
	in := &contact.Contact{
		Identity: commonidentity.Identity{ID: epContactID, CustomerID: epCustomerID},
		Addresses: []contact.Address{{
			Address:   commonaddress.Address{Type: contact.AddressTypeTel, Target: "+15550000003"},
			IsPrimary: true,
		}},
	}

	mockDB.EXPECT().ContactCreate(ctx, gomock.Any()).Return(nil)
	mockUtil.EXPECT().UUIDCreate().Return(addressID)
	mockDB.EXPECT().AddressResetPrimary(ctx, epContactID).Return(errTestBoom)
	mockDB.EXPECT().AddressCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, a *contact.Address) error {
		if a.ID != addressID || a.ContactID != epContactID || !a.IsPrimary {
			t.Errorf("address = %+v, want id %s contact %s primary", a, addressID, epContactID)
		}
		return nil
	})
	mockDB.EXPECT().ContactGet(ctx, epContactID).Return(liveContact(), nil)
	mockNotify.EXPECT().PublishEvent(ctx, contact.EventTypeContactCreated, gomock.Any())

	res, err := h.Create(ctx, in)
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if res == nil || res.ID != epContactID {
		t.Errorf("result = %v, want the created contact", res)
	}
}

func Test_InteractionList_LookupFailures(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		contactID  uuid.UUID
		addressID  uuid.UUID
		expect     func(m *dbhandler.MockDBHandler)
		wantReason string
		wantText   string
		wantIs     error
	}{
		{
			name:      "a deleted contact is reported as not found",
			contactID: epContactID,
			expect: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().ContactGet(ctx, epContactID).Return(deletedContact(), nil)
			},
			wantReason: "CONTACT_NOT_FOUND",
		},
		{
			name:      "a contact of another customer is reported as not found",
			contactID: epContactID,
			expect: func(m *dbhandler.MockDBHandler) {
				other := liveContact()
				other.CustomerID = uuid.FromStringOrNil("77777777-0003-0003-0003-000000000001")
				m.EXPECT().ContactGet(ctx, epContactID).Return(other, nil)
			},
			wantReason: "CONTACT_NOT_FOUND",
		},
		{
			name:      "an address listing failure is wrapped",
			contactID: epContactID,
			expect: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().ContactGet(ctx, epContactID).Return(liveContact(), nil)
				m.EXPECT().AddressListAllByContactID(ctx, epContactID).Return(nil, errTestBoom)
			},
			wantText: "could not list contact addresses. InteractionList",
			wantIs:   errTestBoom,
		},
		{
			name:      "an unknown address is reported as not found",
			addressID: epAddressID,
			expect: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().AddressGet(ctx, epCustomerID, epAddressID).Return(nil, dbhandler.ErrNotFound)
			},
			wantReason: "ADDRESS_NOT_FOUND",
		},
		{
			name:      "an address lookup failure is wrapped",
			addressID: epAddressID,
			expect: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().AddressGet(ctx, epCustomerID, epAddressID).Return(nil, errTestBoom)
			},
			wantText: "could not get address. InteractionList",
			wantIs:   errTestBoom,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			// no request handler is set: a lookup failure must return before any RPC
			h := &contactHandler{db: mockDB}
			tt.expect(mockDB)

			res, next, err := h.InteractionList(ctx, epCustomerID, 10, "", "", "", tt.contactID, tt.addressID)
			if res != nil || next != "" {
				t.Errorf("result = %v next = %q, want nothing on error", res, next)
			}
			if tt.wantReason != "" {
				assertVoipbinError(t, err, tt.wantReason, nil)
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantText) {
				t.Fatalf("error = %v, want text %q", err, tt.wantText)
			}
			if !stderrors.Is(err, tt.wantIs) {
				t.Errorf("error %v does not wrap %v", err, tt.wantIs)
			}
		})
	}
}
