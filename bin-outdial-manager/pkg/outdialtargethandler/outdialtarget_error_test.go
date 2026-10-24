package outdialtargethandler

import (
	"context"
	"errors"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-outdial-manager/models/outdialtarget"
	"monorepo/bin-outdial-manager/pkg/dbhandler"
)

var errTestDB = errors.New("test db failure")

func newErrorTestHandler(mc *gomock.Controller) (*outdialTargetHandler, *dbhandler.MockDBHandler) {
	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &outdialTargetHandler{
		db:            mockDB,
		reqHandler:    requesthandler.NewMockRequestHandler(mc),
		notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
	}

	return h, mockDB
}

func Test_Get_error(t *testing.T) {
	id := uuid.FromStringOrNil("81a2b3c4-b9c6-11ef-8b78-718293a4b5c6")

	t.Run("not found is mapped to a typed error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialTargetGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

		res, err := h.Get(ctx, id)
		if res != nil {
			t.Errorf("expected a nil result, got: %+v", res)
		}
		var ve *cerrors.VoipbinError
		if !errors.As(err, &ve) {
			t.Fatalf("expected a VoipbinError, got: %v", err)
		}
		if ve.Reason != "OUTDIAL_TARGET_NOT_FOUND" {
			t.Errorf("wrong reason. expect: OUTDIAL_TARGET_NOT_FOUND, got: %s", ve.Reason)
		}
		if !errors.Is(err, dbhandler.ErrNotFound) {
			t.Errorf("expected the cause ErrNotFound to be kept, got: %v", err)
		}
	})

	t.Run("other error passes through", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialTargetGet(ctx, id).Return(nil, errTestDB)

		res, err := h.Get(ctx, id)
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})
}

func Test_Delete_error(t *testing.T) {
	id := uuid.FromStringOrNil("92b3c4d5-b9c6-11ef-9c89-8293a4b5c6d7")

	t.Run("delete fails and nothing else is called", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialTargetDelete(ctx, id).Return(errTestDB)

		res, err := h.Delete(ctx, id)
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})

	t.Run("get after delete fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialTargetDelete(ctx, id).Return(nil)
		mockDB.EXPECT().OutdialTargetGet(ctx, id).Return(nil, errTestDB)

		res, err := h.Delete(ctx, id)
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})
}

func Test_Create_error(t *testing.T) {
	outdialID := uuid.FromStringOrNil("a3c4d5e6-b9c6-11ef-8d9a-93a4b5c6d7e8")

	t.Run("create fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialTargetCreate(ctx, gomock.Any()).Return(errTestDB)

		res, err := h.Create(ctx, outdialID, "name", "detail", "data", nil, nil, nil, nil, nil)
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})

	t.Run("get created fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialTargetCreate(ctx, gomock.Any()).Return(nil)
		mockDB.EXPECT().OutdialTargetGet(ctx, gomock.Any()).Return(nil, errTestDB)

		res, err := h.Create(ctx, outdialID, "name", "detail", "data", nil, nil, nil, nil, nil)
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})
}

func Test_UpdateProgressing_error(t *testing.T) {
	id := uuid.FromStringOrNil("b4d5e6f7-b9c6-11ef-9eab-a4b5c6d7e8f9")

	t.Run("update fails and the target is not re-fetched", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialTargetUpdateProgressing(ctx, id, 2).Return(errTestDB)

		res, err := h.UpdateProgressing(ctx, id, 2)
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})

	t.Run("get after update fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialTargetUpdateProgressing(ctx, id, 2).Return(nil)
		mockDB.EXPECT().OutdialTargetGet(ctx, id).Return(nil, errTestDB)

		res, err := h.UpdateProgressing(ctx, id, 2)
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})
}

func Test_UpdateStatus_error(t *testing.T) {
	id := uuid.FromStringOrNil("c5e6f708-b9c6-11ef-8fbc-b5c6d7e8f90a")
	expectFields := map[outdialtarget.Field]any{
		outdialtarget.FieldStatus: outdialtarget.StatusDone,
	}

	t.Run("update fails and the target is not re-fetched", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialTargetUpdate(ctx, id, expectFields).Return(errTestDB)

		res, err := h.UpdateStatus(ctx, id, outdialtarget.StatusDone)
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})

	t.Run("get after update fails", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		h, mockDB := newErrorTestHandler(mc)
		ctx := context.Background()

		mockDB.EXPECT().OutdialTargetUpdate(ctx, id, expectFields).Return(nil)
		mockDB.EXPECT().OutdialTargetGet(ctx, id).Return(nil, errTestDB)

		res, err := h.UpdateStatus(ctx, id, outdialtarget.StatusDone)
		if res != nil || err != errTestDB {
			t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
		}
	})
}

func Test_GetAvailable_args(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockDB := newErrorTestHandler(mc)
	ctx := context.Background()

	outdialID := uuid.FromStringOrNil("d6f70819-b9c6-11ef-9acd-c6d7e8f90a1b")

	// distinct try counts so that an argument reorder is detected
	mockDB.EXPECT().OutdialTargetGetAvailable(ctx, outdialID, 1, 2, 3, 4, 5, uint64(7)).Return(nil, errTestDB)

	res, err := h.GetAvailable(ctx, outdialID, 1, 2, 3, 4, 5, 7)
	if res != nil || err != errTestDB {
		t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
	}
}

func Test_GetsByOutdialID_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	h, mockDB := newErrorTestHandler(mc)
	ctx := context.Background()

	outdialID := uuid.FromStringOrNil("e70a192a-b9c6-11ef-8bde-d7e8f90a1b2c")

	expectFilters := map[outdialtarget.Field]any{
		outdialtarget.FieldOutdialID: outdialID,
		outdialtarget.FieldDeleted:   false,
	}
	mockDB.EXPECT().OutdialTargetList(ctx, "token", uint64(5), expectFilters).Return(nil, errTestDB)

	res, err := h.GetsByOutdialID(ctx, outdialID, "token", 5)
	if res != nil || err != errTestDB {
		t.Errorf("expected (nil, db error), got: (%+v, %v)", res, err)
	}
}
