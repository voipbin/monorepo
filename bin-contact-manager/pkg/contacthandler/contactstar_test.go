package contacthandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/contactstar"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

var (
	testStarCustomerID = uuid.FromStringOrNil("c1a0b100-1673-4b02-1000-000000000001")
	testStarOtherCust  = uuid.FromStringOrNil("c1a0b100-1673-4b02-1000-000000000002")
	testStarAgentID    = uuid.FromStringOrNil("c1a0b100-1673-4b02-1000-0000000000a1")
	testStarContactID  = uuid.FromStringOrNil("c1a0b100-1673-4b02-1000-0000000000c1")
	testStarRowID      = uuid.FromStringOrNil("c1a0b100-1673-4b02-1000-0000000000d1")
)

func newStarHandler(t *testing.T) (*contactHandler, *dbhandler.MockDBHandler, *utilhandler.MockUtilHandler) {
	t.Helper()

	mc := gomock.NewController(t)
	t.Cleanup(mc.Finish)

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	return &contactHandler{utilHandler: mockUtil, db: mockDB, notifyHandler: mockNotify}, mockDB, mockUtil
}

func starContact(customerID uuid.UUID, deleted bool) *contact.Contact {
	c := &contact.Contact{Identity: commonidentity.Identity{ID: testStarContactID, CustomerID: customerID}}
	if deleted {
		ts := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
		c.TMDelete = &ts
	}

	return c
}

func Test_StarSet(t *testing.T) {
	tests := []struct {
		name string

		contactRes *contact.Contact
		contactErr error

		expectCreate   bool
		expectNotFound bool
	}{
		{name: "normal", contactRes: starContact(testStarCustomerID, false), expectCreate: true},
		{name: "contact of another customer", contactRes: starContact(testStarOtherCust, false), expectNotFound: true},
		{name: "soft-deleted contact", contactRes: starContact(testStarCustomerID, true), expectNotFound: true},
		{name: "missing contact", contactErr: dbhandler.ErrNotFound, expectNotFound: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockDB, mockUtil := newStarHandler(t)
			ctx := context.Background()

			mockDB.EXPECT().ContactGet(ctx, testStarContactID).Return(tt.contactRes, tt.contactErr)
			created := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
			if tt.expectCreate {
				mockUtil.EXPECT().UUIDCreate().Return(testStarRowID)
				mockDB.EXPECT().ContactStarCreate(ctx, &contactstar.ContactStar{
					Identity:  commonidentity.Identity{ID: testStarRowID, CustomerID: testStarCustomerID},
					AgentID:   testStarAgentID,
					ContactID: testStarContactID,
				}).Return(&contactstar.ContactStar{ContactID: testStarContactID, TMCreate: &created}, nil)
			}

			res, err := h.StarSet(ctx, testStarCustomerID, testStarAgentID, testStarContactID)
			if tt.expectNotFound {
				var ve *cerrors.VoipbinError
				if err == nil || res != nil || !stderrors.As(err, &ve) {
					t.Fatalf("Wrong match. expect: not found VoipbinError, got: %v, %v", res, err)
				}
				return
			}
			if err != nil || res == nil || res.ContactID != testStarContactID {
				t.Fatalf("Wrong match. expect: ok, got: %v, %v", res, err)
			}
		})
	}
}

func Test_StarSet_CreateError(t *testing.T) {
	h, mockDB, mockUtil := newStarHandler(t)
	ctx := context.Background()

	mockDB.EXPECT().ContactGet(ctx, testStarContactID).Return(starContact(testStarCustomerID, false), nil)
	mockUtil.EXPECT().UUIDCreate().Return(testStarRowID)
	mockDB.EXPECT().ContactStarCreate(ctx, gomock.Any()).Return(nil, fmt.Errorf("database error"))

	if res, err := h.StarSet(ctx, testStarCustomerID, testStarAgentID, testStarContactID); err == nil || res != nil {
		t.Errorf("Wrong match. expect: error, got: %v, %v", res, err)
	}
}

func Test_StarDelete(t *testing.T) {
	tests := []struct {
		name string

		contactRes *contact.Contact
		contactErr error

		expectDelete   bool
		expectNotFound bool
	}{
		{name: "normal", contactRes: starContact(testStarCustomerID, false), expectDelete: true},
		{name: "contact of another customer", contactRes: starContact(testStarOtherCust, false), expectNotFound: true},
		{name: "soft-deleted contact", contactRes: starContact(testStarCustomerID, true), expectNotFound: true},
		{name: "missing contact", contactErr: dbhandler.ErrNotFound, expectNotFound: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockDB, _ := newStarHandler(t)
			ctx := context.Background()

			mockDB.EXPECT().ContactGet(ctx, testStarContactID).Return(tt.contactRes, tt.contactErr)
			if tt.expectDelete {
				mockDB.EXPECT().ContactStarDelete(ctx, testStarCustomerID, testStarAgentID, testStarContactID).Return(nil)
			}

			err := h.StarDelete(ctx, testStarCustomerID, testStarAgentID, testStarContactID)
			if tt.expectNotFound {
				var ve *cerrors.VoipbinError
				if err == nil || !stderrors.As(err, &ve) {
					t.Fatalf("Wrong match. expect: not found VoipbinError, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_StarDelete_DeleteError(t *testing.T) {
	h, mockDB, _ := newStarHandler(t)
	ctx := context.Background()

	mockDB.EXPECT().ContactGet(ctx, testStarContactID).Return(starContact(testStarCustomerID, false), nil)
	mockDB.EXPECT().ContactStarDelete(ctx, testStarCustomerID, testStarAgentID, testStarContactID).Return(fmt.Errorf("database error"))

	if err := h.StarDelete(ctx, testStarCustomerID, testStarAgentID, testStarContactID); err == nil {
		t.Error("Wrong match. expect: error")
	}
}

func Test_StarList(t *testing.T) {
	h, mockDB, _ := newStarHandler(t)
	ctx := context.Background()

	want := []*contactstar.ContactStar{{ContactID: testStarContactID}}
	mockDB.EXPECT().ContactStarList(ctx, testStarCustomerID, testStarAgentID, uint64(10), "tok").Return(want, nil)

	res, err := h.StarList(ctx, testStarCustomerID, testStarAgentID, 10, "tok")
	if err != nil || len(res) != 1 || res[0].ContactID != testStarContactID {
		t.Errorf("Wrong match. got: %v, %v", res, err)
	}
}

func Test_StarList_Errors(t *testing.T) {
	tests := []struct {
		name      string
		dbErr     error
		expectBad bool
	}{
		{name: "invalid token is an invalid argument", dbErr: fmt.Errorf("%w. err: x", dbhandler.ErrInvalidPageToken), expectBad: true},
		{name: "other error is passed through", dbErr: fmt.Errorf("database error")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, mockDB, _ := newStarHandler(t)
			ctx := context.Background()

			mockDB.EXPECT().ContactStarList(ctx, testStarCustomerID, testStarAgentID, uint64(10), "").Return(nil, tt.dbErr)

			_, err := h.StarList(ctx, testStarCustomerID, testStarAgentID, 10, "")
			var ve *cerrors.VoipbinError
			if err == nil || stderrors.As(err, &ve) != tt.expectBad {
				t.Errorf("Wrong match. expectBad: %v, got: %v", tt.expectBad, err)
			}
		})
	}
}

func Test_ListStarred(t *testing.T) {
	h, mockDB, _ := newStarHandler(t)
	ctx := context.Background()

	want := []*contact.Contact{starContact(testStarCustomerID, false)}
	mockDB.EXPECT().ContactListStarred(ctx, testStarCustomerID, testStarAgentID, uint64(10), "tok").Return(want, nil)

	res, err := h.ListStarred(ctx, testStarCustomerID, testStarAgentID, 10, "tok")
	if err != nil || len(res) != 1 || res[0].ID != testStarContactID {
		t.Errorf("Wrong match. got: %v, %v", res, err)
	}

	// an invalid token is an invalid argument
	mockDB.EXPECT().ContactListStarred(ctx, testStarCustomerID, testStarAgentID, uint64(10), "bad").Return(nil, fmt.Errorf("%w. err: x", dbhandler.ErrInvalidPageToken))
	_, err = h.ListStarred(ctx, testStarCustomerID, testStarAgentID, 10, "bad")
	var ve *cerrors.VoipbinError
	if err == nil || !stderrors.As(err, &ve) {
		t.Errorf("Wrong match. expect: VoipbinError, got: %v", err)
	}

	// other errors are passed through
	mockDB.EXPECT().ContactListStarred(ctx, testStarCustomerID, testStarAgentID, uint64(10), "").Return(nil, fmt.Errorf("database error"))
	if _, err := h.ListStarred(ctx, testStarCustomerID, testStarAgentID, 10, ""); err == nil {
		t.Error("Wrong match. expect: error")
	}
}

func Test_EventCustomerDeleted_StarCleanupError(t *testing.T) {
	h, mockDB, _ := newStarHandler(t)
	ctx := context.Background()

	mockDB.EXPECT().ContactDeleteByCustomerID(ctx, testStarCustomerID).Return(nil)
	mockDB.EXPECT().ContactStarDeleteByCustomerID(ctx, testStarCustomerID).Return(fmt.Errorf("database error"))

	if err := h.EventCustomerDeleted(ctx, &cmcustomer.Customer{ID: testStarCustomerID}); err == nil {
		t.Error("Wrong match. expect: error")
	}
}
