package outdialhandler

import (
	"context"
	"errors"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-outdial-manager/models/outdial"
	"monorepo/bin-outdial-manager/pkg/dbhandler"
)

var errTestDB = errors.New("test db failure")

func newErrorTestHandler(mc *gomock.Controller) (*outdialHandler, *dbhandler.MockDBHandler, *notifyhandler.MockNotifyHandler) {
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &outdialHandler{
		db:            mockDB,
		reqHandler:    requesthandler.NewMockRequestHandler(mc),
		notifyHandler: mockNotify,
	}

	return h, mockDB, mockNotify
}

func Test_Get_error(t *testing.T) {
	id := uuid.FromStringOrNil("1a2b3c4d-b9c6-11ef-8a01-0a1b2c3d4e5f")

	t.Run("not found is mapped to a typed error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

		res, err := h.Get(ctx, id)
		if res != nil {
			t.Errorf("expected a nil result, got: %+v", res)
		}
		var ve *cerrors.VoipbinError
		if !errors.As(err, &ve) {
			t.Fatalf("expected a VoipbinError, got: %v", err)
		}
		if ve.Reason != "OUTDIAL_NOT_FOUND" {
			t.Errorf("wrong reason. expect: OUTDIAL_NOT_FOUND, got: %s", ve.Reason)
		}
		if !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("expected the cause ErrNotFound to be kept, got: %v", err)
		}
	})

	t.Run("other error passes through", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialGet(ctx, id).Return(nil, errTestDB)

		res, err := h.Get(ctx, id)
		if res != nil {
			t.Errorf("expected a nil result, got: %+v", res)
		}
		if err != errTestDB {
			t.Errorf("expected the db error unchanged, got: %v", err)
		}
		var ve *cerrors.VoipbinError
		if errors.As(err, &ve) {
			t.Errorf("a generic error must not become a VoipbinError")
		}
	})
}

func Test_Delete_error(t *testing.T) {
	id := uuid.FromStringOrNil("2b3c4d5e-b9c6-11ef-9b12-1b2c3d4e5f60")

	t.Run("delete fails and nothing else is called", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newErrorTestHandler(mc)
		ctx := context.Background()

		// no OutdialGet and no webhook expectation: a call fails the test
		mockDB.EXPECT().OutdialDelete(ctx, id).Return(errTestDB)

		res, err := h.Delete(ctx, id)
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})

	t.Run("get after delete fails and no webhook is published", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialDelete(ctx, id).Return(nil)
		mockDB.EXPECT().OutdialGet(ctx, id).Return(nil, errTestDB)

		res, err := h.Delete(ctx, id)
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})
}

func Test_Create_error(t *testing.T) {
	customerID := uuid.FromStringOrNil("3c4d5e6f-b9c6-11ef-8c23-2c3d4e5f6071")
	campaignID := uuid.FromStringOrNil("4d5e6f70-b9c6-11ef-9d34-3d4e5f607182")

	t.Run("create fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialCreate(ctx, gomock.Any()).Return(errTestDB)

		res, err := h.Create(ctx, customerID, campaignID, "name", "detail", "data")
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})

	t.Run("get created fails and no webhook is published", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialCreate(ctx, gomock.Any()).Return(nil)
		mockDB.EXPECT().OutdialGet(ctx, gomock.Any()).Return(nil, errTestDB)

		res, err := h.Create(ctx, customerID, campaignID, "name", "detail", "data")
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})
}

func Test_Update_error(t *testing.T) {
	id := uuid.FromStringOrNil("5e6f7081-b9c6-11ef-8e45-4e5f60718293")
	campaignID := uuid.FromStringOrNil("6f708192-b9c6-11ef-9f56-5f60718293a4")

	tests := []struct {
		name string
		fn   func(h *outdialHandler, ctx context.Context) (*outdial.Outdial, error)
	}{
		{"basic info", func(h *outdialHandler, ctx context.Context) (*outdial.Outdial, error) {
			return h.UpdateBasicInfo(ctx, id, "name", "detail")
		}},
		{"campaign id", func(h *outdialHandler, ctx context.Context) (*outdial.Outdial, error) {
			return h.UpdateCampaignID(ctx, id, campaignID)
		}},
		{"data", func(h *outdialHandler, ctx context.Context) (*outdial.Outdial, error) {
			return h.UpdateData(ctx, id, "data")
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name+" update fails and nothing else is called", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockDB, _ := newErrorTestHandler(mc)
			ctx := context.Background()

			mockDB.EXPECT().OutdialUpdate(ctx, id, gomock.Any()).Return(errTestDB)

			res, err := tt.fn(h, ctx)
			if res != nil || err != errTestDB {
				t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
			}
		})

		t.Run(tt.name+" get after update fails and no webhook is published", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockDB, _ := newErrorTestHandler(mc)
			ctx := context.Background()

			mockDB.EXPECT().OutdialUpdate(ctx, id, gomock.Any()).Return(nil)
			mockDB.EXPECT().OutdialGet(ctx, id).Return(nil, errTestDB)

			res, err := tt.fn(h, ctx)
			if res != nil || err != errTestDB {
				t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
			}
		})
	}
}

func Test_List_error(t *testing.T) {
	customerID := uuid.FromStringOrNil("708192a3-b9c6-11ef-8a67-60718293a4b5")

	t.Run("list", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialList(ctx, "token", uint64(10), gomock.Any()).Return(nil, errTestDB)

		res, err := h.List(ctx, "token", 10, map[outdial.Field]any{})
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})

	t.Run("gets by customer id passes the customer filter", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB, _ := newErrorTestHandler(mc)
		ctx := context.Background()

		expectFilters := map[outdial.Field]any{
			outdial.FieldCustomerID: customerID,
			outdial.FieldDeleted:    false,
		}
		mockDB.EXPECT().OutdialList(ctx, "token", uint64(10), expectFilters).Return(nil, errTestDB)

		res, err := h.GetsByCustomerID(ctx, customerID, "token", 10)
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})
}
