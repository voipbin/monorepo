package accounthandler

import (
	"context"
	stderrors "errors"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"monorepo/bin-storage-manager/models/account"
	"monorepo/bin-storage-manager/pkg/dbhandler"
)

var errTestDB = stderrors.New("simulated db failure")

func newTestAccountHandler(mc *gomock.Controller) (*accountHandler, *utilhandler.MockUtilHandler, *dbhandler.MockDBHandler, *notifyhandler.MockNotifyHandler) {
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	return &accountHandler{utilHandler: mockUtil, db: mockDB, notifyHandler: mockNotify}, mockUtil, mockDB, mockNotify
}

func Test_NewAccountHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	res, ok := NewAccountHandler(mockNotify, mockDB).(*accountHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *accountHandler")
	}
	if res.db != mockDB || res.notifyHandler != mockNotify || res.utilHandler == nil {
		t.Errorf("Wrong match. the dependencies were not stored as given")
	}
}

func Test_Create_errors(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("b16eaf63-0000-0000-0000-000000000001")
	newID := uuid.FromStringOrNil("b16eaf63-0000-0000-0000-000000000002")
	filters := map[account.Field]any{
		account.FieldDeleted:    false,
		account.FieldCustomerID: customerID,
	}

	t.Run("the customer already has an account", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, mockDB, _ := newTestAccountHandler(mc)
		// neither an id nor a record nor an event may be produced for a duplicate.
		mockDB.EXPECT().AccountList(ctx, "", uint64(1), filters).Return([]*account.Account{{ID: newID, CustomerID: customerID}}, nil)

		res, err := h.Create(ctx, customerID)
		if err == nil || res != nil {
			t.Errorf("Wrong match. expect: nil/error, got: %v/%v", res, err)
		}
	})

	t.Run("insert failed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockUtil, mockDB, _ := newTestAccountHandler(mc)
		mockDB.EXPECT().AccountList(ctx, "", uint64(1), filters).Return([]*account.Account{}, nil)
		mockUtil.EXPECT().UUIDCreate().Return(newID)
		mockDB.EXPECT().AccountCreate(ctx, &account.Account{ID: newID, CustomerID: customerID}).Return(errTestDB)

		res, err := h.Create(ctx, customerID)
		if !stderrors.Is(err, errTestDB) || res != nil {
			t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", errTestDB, res, err)
		}
	})

	t.Run("reading back the created account failed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockUtil, mockDB, _ := newTestAccountHandler(mc)
		mockDB.EXPECT().AccountList(ctx, "", uint64(1), filters).Return([]*account.Account{}, nil)
		mockUtil.EXPECT().UUIDCreate().Return(newID)
		mockDB.EXPECT().AccountCreate(ctx, gomock.Any()).Return(nil)
		mockDB.EXPECT().AccountGet(ctx, newID).Return(nil, errTestDB)

		// no PublishEvent expectation: nothing may be published when the read-back failed.
		res, err := h.Create(ctx, customerID)
		if !stderrors.Is(err, errTestDB) || res != nil {
			t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", errTestDB, res, err)
		}
	})
}

func Test_Get_errors(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("c27fb074-0000-0000-0000-000000000001")

	t.Run("not found becomes a typed error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, mockDB, _ := newTestAccountHandler(mc)
		mockDB.EXPECT().AccountGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

		res, err := h.Get(ctx, id)
		var ve *cerrors.VoipbinError
		if res != nil || !stderrors.As(err, &ve) {
			t.Fatalf("Wrong match. expect: nil/*VoipbinError, got: %v/%v", res, err)
		}
		if ve.Reason != "ACCOUNT_NOT_FOUND" || ve.Status != cerrors.StatusNotFound {
			t.Errorf("Wrong match. got reason: %s, status: %v", ve.Reason, ve.Status)
		}
		if !stderrors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("Wrong match. expect the cause to stay reachable through errors.Is")
		}
	})

	t.Run("other errors are wrapped and not classified as not found", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, mockDB, _ := newTestAccountHandler(mc)
		mockDB.EXPECT().AccountGet(ctx, id).Return(nil, errTestDB)

		res, err := h.Get(ctx, id)
		var ve *cerrors.VoipbinError
		if res != nil || !stderrors.Is(err, errTestDB) || stderrors.As(err, &ve) {
			t.Errorf("Wrong match. expect: nil/untyped %v, got: %v/%v", errTestDB, res, err)
		}
	})
}

func Test_List_error(t *testing.T) {
	ctx := context.Background()
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, mockDB, _ := newTestAccountHandler(mc)
	mockDB.EXPECT().AccountList(ctx, "token", uint64(7), gomock.Any()).Return(nil, errTestDB)

	res, err := h.List(ctx, "token", 7, map[account.Field]any{})
	if !stderrors.Is(err, errTestDB) || res != nil {
		t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", errTestDB, res, err)
	}
}

func Test_Delete_errors(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("d38ac185-0000-0000-0000-000000000001")

	t.Run("delete failed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, mockDB, _ := newTestAccountHandler(mc)
		mockDB.EXPECT().AccountDelete(ctx, id).Return(errTestDB)

		res, err := h.Delete(ctx, id)
		if !stderrors.Is(err, errTestDB) || res != nil {
			t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", errTestDB, res, err)
		}
	})

	t.Run("reading back the deleted account failed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, mockDB, _ := newTestAccountHandler(mc)
		mockDB.EXPECT().AccountDelete(ctx, id).Return(nil)
		mockDB.EXPECT().AccountGet(ctx, id).Return(nil, errTestDB)

		res, err := h.Delete(ctx, id)
		if !stderrors.Is(err, errTestDB) || res != nil {
			t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", errTestDB, res, err)
		}
	})
}

func Test_IncreaseDecreaseFileInfo_errors(t *testing.T) {
	ctx := context.Background()
	id := uuid.FromStringOrNil("e49bd296-0000-0000-0000-000000000001")

	tests := []struct {
		name    string
		prepare func(m *dbhandler.MockDBHandler, failUpdate bool)
		run     func(h *accountHandler) (*account.Account, error)
	}{
		{
			name: "increase",
			prepare: func(m *dbhandler.MockDBHandler, failUpdate bool) {
				if failUpdate {
					m.EXPECT().AccountIncreaseFileInfo(ctx, id, int64(2), int64(300)).Return(errTestDB)
					return
				}
				m.EXPECT().AccountIncreaseFileInfo(ctx, id, int64(2), int64(300)).Return(nil)
				m.EXPECT().AccountGet(ctx, id).Return(nil, errTestDB)
			},
			run: func(h *accountHandler) (*account.Account, error) { return h.IncreaseFileInfo(ctx, id, 2, 300) },
		},
		{
			name: "decrease",
			prepare: func(m *dbhandler.MockDBHandler, failUpdate bool) {
				if failUpdate {
					m.EXPECT().AccountDecreaseFileInfo(ctx, id, int64(2), int64(300)).Return(errTestDB)
					return
				}
				m.EXPECT().AccountDecreaseFileInfo(ctx, id, int64(2), int64(300)).Return(nil)
				m.EXPECT().AccountGet(ctx, id).Return(nil, errTestDB)
			},
			run: func(h *accountHandler) (*account.Account, error) { return h.DecreaseFileInfo(ctx, id, 2, 300) },
		},
	}

	for _, tt := range tests {
		for _, failUpdate := range []bool{true, false} {
			name := tt.name + " read-back failed"
			if failUpdate {
				name = tt.name + " update failed"
			}

			t.Run(name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				h, _, mockDB, _ := newTestAccountHandler(mc)
				tt.prepare(mockDB, failUpdate)

				// no PublishEvent expectation: an event must not go out on failure.
				res, err := tt.run(h)
				if !stderrors.Is(err, errTestDB) || res != nil {
					t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", errTestDB, res, err)
				}
			})
		}
	}
}

func Test_ValidateFileInfoByCustomerID_limit(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("f5ace3a7-0000-0000-0000-000000000001")
	filters := map[account.Field]any{
		account.FieldDeleted:    false,
		account.FieldCustomerID: customerID,
	}

	tests := []struct {
		name string

		listed  []*account.Account
		listErr error

		filesize int64

		expectOK bool
	}{
		{
			name:     "exactly at the limit is allowed",
			listed:   []*account.Account{{CustomerID: customerID, TotalFileSize: maxFileSize - 100}},
			filesize: 100,
			expectOK: true,
		},
		{
			name:     "one byte over the limit is rejected",
			listed:   []*account.Account{{CustomerID: customerID, TotalFileSize: maxFileSize - 100}},
			filesize: 101,
		},
		{
			name:     "a single file larger than the limit is rejected",
			listed:   []*account.Account{{CustomerID: customerID}},
			filesize: maxFileSize + 1,
		},
		{
			name:     "no account for the customer",
			listed:   []*account.Account{},
			filesize: 1,
		},
		{
			name:     "listing failed",
			listErr:  errTestDB,
			filesize: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, mockDB, _ := newTestAccountHandler(mc)
			mockDB.EXPECT().AccountList(ctx, "", uint64(1), filters).Return(tt.listed, tt.listErr)

			res, err := h.ValidateFileInfoByCustomerID(ctx, customerID, 1, tt.filesize)
			if tt.expectOK {
				if err != nil || res == nil || res.CustomerID != customerID {
					t.Errorf("Wrong match. expect: the account, got: %v/%v", res, err)
				}
				return
			}
			if err == nil || res != nil {
				t.Errorf("Wrong match. expect: nil/error, got: %v/%v", res, err)
			}
		})
	}
}

func Test_EventCustomer_errors(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("06bdf4b8-0000-0000-0000-000000000001")
	accountID := uuid.FromStringOrNil("06bdf4b8-0000-0000-0000-000000000002")
	cu := &cmcustomer.Customer{ID: customerID}
	filters := map[account.Field]any{
		account.FieldDeleted:    false,
		account.FieldCustomerID: customerID,
	}

	t.Run("created: the account already exists", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, mockDB, _ := newTestAccountHandler(mc)
		mockDB.EXPECT().AccountList(ctx, "", uint64(1), filters).Return([]*account.Account{{ID: accountID}}, nil)

		if err := h.EventCustomerCreated(ctx, cu); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("deleted: no account for the customer", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, mockDB, _ := newTestAccountHandler(mc)
		// AccountDelete must not be called when there is nothing to delete.
		mockDB.EXPECT().AccountList(ctx, "", uint64(1), filters).Return([]*account.Account{}, nil)

		if err := h.EventCustomerDeleted(ctx, cu); err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
	})

	t.Run("deleted: delete failed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, mockDB, _ := newTestAccountHandler(mc)
		mockDB.EXPECT().AccountList(ctx, "", uint64(1), filters).Return([]*account.Account{{ID: accountID}}, nil)
		mockDB.EXPECT().AccountDelete(ctx, accountID).Return(errTestDB)

		if err := h.EventCustomerDeleted(ctx, cu); !stderrors.Is(err, errTestDB) {
			t.Errorf("Wrong match. expect: %v, got: %v", errTestDB, err)
		}
	})
}
