package outplanhandler

import (
	"context"
	stderrors "errors"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/outplan"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
)

func newErrorsTestHandler(mc *gomock.Controller) (*outplanHandler, *dbhandler.MockDBHandler, *utilhandler.MockUtilHandler) {
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	return &outplanHandler{db: mockDB, util: mockUtil}, mockDB, mockUtil
}

func Test_Get_errors(t *testing.T) {
	id := uuid.FromStringOrNil("e1000000-0000-4000-8000-000000000001")

	tests := []struct {
		name         string
		dbErr        error
		expectTyped  bool
		expectReason string
	}{
		{
			name:         "not found is converted to the typed error with the cause kept",
			dbErr:        dbhandler.ErrNotFound,
			expectTyped:  true,
			expectReason: "OUTPLAN_NOT_FOUND",
		},
		{
			name:  "other errors are returned as they are",
			dbErr: stderrors.New("db unavailable"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _ := newErrorsTestHandler(mc)
			ctx := context.Background()

			mockDB.EXPECT().OutplanGet(ctx, id).Return(nil, tt.dbErr)

			res, err := h.Get(ctx, id)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if !stderrors.Is(err, tt.dbErr) {
				t.Errorf("Wrong match. the cause must be kept. err: %v", err)
			}

			var ve *cerrors.VoipbinError
			isTyped := stderrors.As(err, &ve)
			if isTyped != tt.expectTyped {
				t.Fatalf("Wrong match. expect typed: %v, got: %v", tt.expectTyped, isTyped)
			}
			if tt.expectTyped && (ve.Status != cerrors.StatusNotFound || ve.Reason != tt.expectReason) {
				t.Errorf("Wrong match. expect: %s/%s, got: %s/%s", cerrors.StatusNotFound, tt.expectReason, ve.Status, ve.Reason)
			}
		})
	}
}

func Test_Delete_errors(t *testing.T) {
	id := uuid.FromStringOrNil("e2000000-0000-4000-8000-000000000001")
	deleted := &outplan.Outplan{Identity: commonidentity.Identity{ID: id}}

	tests := []struct {
		name      string
		setup     func(ctx context.Context, mockDB *dbhandler.MockDBHandler)
		expectErr string
	}{
		{
			name: "delete error is returned without reading the outplan again",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().OutplanDelete(ctx, id).Return(stderrors.New("delete error"))
			},
			expectErr: "delete error",
		},
		{
			name: "get error after the delete is returned",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().OutplanDelete(ctx, id).Return(nil)
				mockDB.EXPECT().OutplanGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			expectErr: "get error",
		},
		{
			name: "deleted outplan is returned",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().OutplanDelete(ctx, id).Return(nil)
				mockDB.EXPECT().OutplanGet(ctx, id).Return(deleted, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _ := newErrorsTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, mockDB)

			res, err := h.Delete(ctx, id)
			if tt.expectErr != "" {
				if err == nil || err.Error() != tt.expectErr || res != nil {
					t.Errorf("Wrong match. expect error: %s, got: %v, res: %v", tt.expectErr, err, res)
				}
				return
			}
			if err != nil || res != deleted {
				t.Errorf("Wrong match. expect: the deleted outplan, got: %v, err: %v", res, err)
			}
		})
	}
}

func Test_UpdateBasicInfo_errors(t *testing.T) {
	id := uuid.FromStringOrNil("e3000000-0000-4000-8000-000000000001")

	tests := []struct {
		name      string
		setup     func(ctx context.Context, mockDB *dbhandler.MockDBHandler)
		expectErr string
	}{
		{
			name: "update error is returned without reading the outplan",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().OutplanUpdateBasicInfo(ctx, id, "name", "detail").Return(stderrors.New("update error"))
			},
			expectErr: "update error",
		},
		{
			name: "get error after the update is returned",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().OutplanUpdateBasicInfo(ctx, id, "name", "detail").Return(nil)
				mockDB.EXPECT().OutplanGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			expectErr: "get error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _ := newErrorsTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, mockDB)

			res, err := h.UpdateBasicInfo(ctx, id, "name", "detail")
			if err == nil || err.Error() != tt.expectErr || res != nil {
				t.Errorf("Wrong match. expect error: %s, got: %v, res: %v", tt.expectErr, err, res)
			}
		})
	}
}

func Test_Create_errors(t *testing.T) {
	id := uuid.FromStringOrNil("e4000000-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("e4000000-0000-4000-8000-000000000002")
	source := &commonaddress.Address{Type: commonaddress.TypeTel, Target: "+15550000001"}

	expect := &outplan.Outplan{
		Identity:     commonidentity.Identity{ID: id, CustomerID: customerID},
		Name:         "n",
		Detail:       "d",
		Source:       source,
		DialTimeout:  30000,
		TryInterval:  60000,
		MaxTryCount0: 1,
		MaxTryCount1: 2,
		MaxTryCount2: 3,
		MaxTryCount3: 4,
		MaxTryCount4: 5,
	}

	tests := []struct {
		name      string
		setup     func(ctx context.Context, mockDB *dbhandler.MockDBHandler)
		expectErr string
	}{
		{
			name: "create error is returned without reading the outplan",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().OutplanCreate(ctx, expect).Return(stderrors.New("create error"))
			},
			expectErr: "create error",
		},
		{
			name: "get error after the create is returned",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().OutplanCreate(ctx, expect).Return(nil)
				mockDB.EXPECT().OutplanGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			expectErr: "get error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, mockUtil := newErrorsTestHandler(mc)
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(id)
			tt.setup(ctx, mockDB)

			res, err := h.Create(ctx, customerID, "n", "d", source, 30000, 60000, 1, 2, 3, 4, 5)
			if err == nil || err.Error() != tt.expectErr || res != nil {
				t.Errorf("Wrong match. expect error: %s, got: %v, res: %v", tt.expectErr, err, res)
			}
		})
	}
}

func Test_List_errors(t *testing.T) {
	customerID := uuid.FromStringOrNil("e5000000-0000-4000-8000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _ := newErrorsTestHandler(mc)
	ctx := context.Background()

	mockDB.EXPECT().OutplanList(ctx, "token", uint64(10), gomock.Any()).Return(nil, stderrors.New("list error"))
	if res, err := h.List(ctx, "token", 10, map[outplan.Field]any{outplan.FieldCustomerID: customerID}); err == nil || res != nil {
		t.Errorf("Wrong match. expect: list error, got: %v, res: %v", err, res)
	}

	mockDB.EXPECT().OutplanListByCustomerID(ctx, customerID, "token", uint64(10)).Return(nil, stderrors.New("list error"))
	if res, err := h.ListByCustomerID(ctx, customerID, "token", 10); err == nil || res != nil {
		t.Errorf("Wrong match. expect: list error, got: %v, res: %v", err, res)
	}
}

func Test_UpdateDialInfo_errors(t *testing.T) {
	id := uuid.FromStringOrNil("e6000000-0000-4000-8000-000000000001")
	timeout := 45000
	retries := 3

	tests := []struct {
		name      string
		setup     func(ctx context.Context, mockDB *dbhandler.MockDBHandler)
		expectErr string
	}{
		{
			name: "update error is returned without reading the outplan",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().OutplanUpdate(ctx, id, map[outplan.Field]any{
					outplan.FieldDialTimeout:  timeout,
					outplan.FieldMaxTryCount2: retries,
				}).Return(stderrors.New("update error"))
			},
			expectErr: "update error",
		},
		{
			name: "get error after the update is returned",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().OutplanUpdate(ctx, id, gomock.Any()).Return(nil)
				mockDB.EXPECT().OutplanGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			expectErr: "get error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _ := newErrorsTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, mockDB)

			res, err := h.UpdateDialInfo(ctx, id, nil, &timeout, nil, nil, nil, &retries, nil, nil)
			if err == nil || err.Error() != tt.expectErr || res != nil {
				t.Errorf("Wrong match. expect error: %s, got: %v, res: %v", tt.expectErr, err, res)
			}
		})
	}
}
