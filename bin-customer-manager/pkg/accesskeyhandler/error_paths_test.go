package accesskeyhandler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-customer-manager/models/accesskey"
	"monorepo/bin-customer-manager/pkg/dbhandler"
)

var (
	errPathAccesskeyID = uuid.FromStringOrNil("5b2d0c4f-0000-4000-8000-000000000001")
	errPathCustomerID  = uuid.FromStringOrNil("5b2d0c4f-0000-4000-8000-000000000002")
)

func newErrPathHandler(mc *gomock.Controller) (*accesskeyHandler, *utilhandler.MockUtilHandler, *dbhandler.MockDBHandler, *notifyhandler.MockNotifyHandler) {
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	h := &accesskeyHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	return h, mockUtil, mockDB, mockNotify
}

func Test_List_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, mockDB, _ := newErrPathHandler(mc)
	ctx := context.Background()

	sentinel := fmt.Errorf("list error")
	mockDB.EXPECT().AccesskeyList(ctx, uint64(10), "", gomock.Any()).Return(nil, sentinel)

	res, err := h.List(ctx, 10, "", map[accesskey.Field]any{})
	if res != nil || err != sentinel { //nolint:errorlint // identity check: the error must not be wrapped
		t.Errorf("Wrong match. expect: nil, %v, got: %v, %v", sentinel, res, err)
	}
}

func Test_GetsByCustomerID_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, mockDB, _ := newErrPathHandler(mc)
	ctx := context.Background()

	sentinel := fmt.Errorf("list error")
	expectFilter := map[accesskey.Field]any{accesskey.FieldCustomerID: errPathCustomerID}
	mockDB.EXPECT().AccesskeyList(ctx, uint64(10), "token", expectFilter).Return(nil, sentinel)

	res, err := h.GetsByCustomerID(ctx, 10, "token", errPathCustomerID)
	if res != nil || err != sentinel { //nolint:errorlint // identity check: the error must not be wrapped
		t.Errorf("Wrong match. expect: nil, %v, got: %v, %v", sentinel, res, err)
	}
}

func Test_Get_notFoundIsTyped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, mockDB, _ := newErrPathHandler(mc)
	ctx := context.Background()

	mockDB.EXPECT().AccesskeyGet(ctx, errPathAccesskeyID).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Get(ctx, errPathAccesskeyID)
	if res != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", res)
	}

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("Wrong match. expect: *VoipbinError, got: %v", err)
	}
	if ve.Reason != "ACCESSKEY_NOT_FOUND" || ve.Status != cerrors.StatusNotFound {
		t.Errorf("Wrong match. expect: NOT_FOUND/ACCESSKEY_NOT_FOUND, got: %v/%v", ve.Status, ve.Reason)
	}
	if !errors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("Wrong match. expect the original error to stay in the chain, got: %v", err)
	}
}

func Test_Get_otherErrorPassesThrough(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, mockDB, _ := newErrPathHandler(mc)
	ctx := context.Background()

	sentinel := fmt.Errorf("db unavailable")
	mockDB.EXPECT().AccesskeyGet(ctx, errPathAccesskeyID).Return(nil, sentinel)

	_, err := h.Get(ctx, errPathAccesskeyID)
	if err != sentinel { //nolint:errorlint // identity check: the error must not be wrapped
		t.Errorf("Wrong match. expect: %v, got: %v", sentinel, err)
	}

	var ve *cerrors.VoipbinError
	if errors.As(err, &ve) {
		t.Errorf("Wrong match. expect: untyped error, got: %v", ve)
	}
}

func Test_GetByToken_cases(t *testing.T) {
	one := &accesskey.Accesskey{}
	one.ID = errPathAccesskeyID

	tests := []struct {
		name      string
		listRes   []*accesskey.Accesskey
		listErr   error
		expectRes *accesskey.Accesskey
		expectErr string
	}{
		{
			name:      "list error",
			listErr:   fmt.Errorf("list error"),
			expectErr: "list error",
		},
		{
			name:      "no match",
			listRes:   []*accesskey.Accesskey{},
			expectErr: "accesskey not found",
		},
		{
			name:      "more than one match is ambiguous",
			listRes:   []*accesskey.Accesskey{{}, {}},
			expectErr: "ambiguous token",
		},
		{
			name:      "exactly one match",
			listRes:   []*accesskey.Accesskey{one},
			expectRes: one,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockUtil, mockDB, _ := newErrPathHandler(mc)
			ctx := context.Background()

			mockUtil.EXPECT().HashSHA256Hex("vb_token").Return("hash-value")
			expectFilter := map[accesskey.Field]any{accesskey.FieldTokenHash: "hash-value"}
			mockDB.EXPECT().AccesskeyList(ctx, uint64(100), "", expectFilter).Return(tt.listRes, tt.listErr)

			res, err := h.GetByToken(ctx, "vb_token")
			if tt.expectErr != "" {
				if res != nil || err == nil || err.Error() != tt.expectErr {
					t.Errorf("Wrong match. expect: nil, %q, got: %v, %v", tt.expectErr, res, err)
				}
				return
			}
			if err != nil || res != tt.expectRes {
				t.Errorf("Wrong match. expect: %v, got: %v, %v", tt.expectRes, res, err)
			}
		})
	}
}

func Test_Create_errors(t *testing.T) {
	expire := time.Hour
	tmExpire := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name  string
		setup func(ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler)
	}{
		{
			name: "token generation fails",
			setup: func(ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler) {
				mockUtil.EXPECT().StringGenerateRandom(defaultLenToken).Return("", fmt.Errorf("random error"))
			},
		},
		{
			name: "created accesskey lookup fails",
			setup: func(ctx context.Context, mockUtil *utilhandler.MockUtilHandler, mockDB *dbhandler.MockDBHandler) {
				mockUtil.EXPECT().StringGenerateRandom(defaultLenToken).Return("random-part", nil)
				mockUtil.EXPECT().HashSHA256Hex(defaultTokenPrefix + "random-part").Return("hash-value")
				mockDB.EXPECT().AccesskeyCreate(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().AccesskeyGet(ctx, errPathAccesskeyID).Return(nil, fmt.Errorf("get error"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockUtil, mockDB, _ := newErrPathHandler(mc)
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(errPathAccesskeyID)
			mockUtil.EXPECT().TimeNowAdd(expire).Return(&tmExpire)
			tt.setup(ctx, mockUtil, mockDB)

			// no PublishEvent expectation: a failure must not publish
			res, err := h.Create(ctx, errPathCustomerID, "name", "detail", expire)
			if res != nil || err == nil {
				t.Errorf("Wrong match. expect: nil and error, got: %v, %v", res, err)
			}
		})
	}
}

func Test_Delete_errors(t *testing.T) {
	existing := &accesskey.Accesskey{}
	existing.ID = errPathAccesskeyID

	tests := []struct {
		name         string
		setup        func(ctx context.Context, mockDB *dbhandler.MockDBHandler)
		expectOpaque bool
	}{
		{
			name: "delete fails",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().AccesskeyGet(ctx, errPathAccesskeyID).Return(existing, nil)
				mockDB.EXPECT().AccesskeyDelete(ctx, errPathAccesskeyID).Return(fmt.Errorf("delete error"))
			},
		},
		{
			name: "lookup after delete fails and stays opaque",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				gomock.InOrder(
					mockDB.EXPECT().AccesskeyGet(ctx, errPathAccesskeyID).Return(existing, nil),
					mockDB.EXPECT().AccesskeyDelete(ctx, errPathAccesskeyID).Return(nil),
					mockDB.EXPECT().AccesskeyGet(ctx, errPathAccesskeyID).Return(nil, dbhandler.ErrNotFound),
				)
			},
			expectOpaque: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, mockDB, _ := newErrPathHandler(mc)
			ctx := context.Background()
			tt.setup(ctx, mockDB)

			// no PublishEvent expectation: a failure must not publish
			res, err := h.Delete(ctx, errPathAccesskeyID)
			if res != nil || err == nil {
				t.Fatalf("Wrong match. expect: nil and error, got: %v, %v", res, err)
			}

			if tt.expectOpaque {
				var ve *cerrors.VoipbinError
				if errors.As(err, &ve) || errors.Is(err, dbhandler.ErrNotFound) {
					t.Errorf("Wrong match. expect: opaque error, got: %v", err)
				}
			}
		})
	}
}

func Test_UpdateBasicInfo_lookupAfterUpdateError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, mockDB, _ := newErrPathHandler(mc)
	ctx := context.Background()

	gomock.InOrder(
		mockDB.EXPECT().AccesskeyUpdate(ctx, errPathAccesskeyID, gomock.Any()).Return(nil),
		mockDB.EXPECT().AccesskeyGet(ctx, errPathAccesskeyID).Return(nil, fmt.Errorf("get error")),
	)

	// no PublishEvent expectation: a failure must not publish
	res, err := h.UpdateBasicInfo(ctx, errPathAccesskeyID, "name", "detail")
	if res != nil || err == nil {
		t.Errorf("Wrong match. expect: nil and error, got: %v, %v", res, err)
	}
}
