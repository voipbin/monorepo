package widgethandler

import (
	"context"
	"errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-webchat-manager/models/widget"
	"monorepo/bin-webchat-manager/pkg/dbhandler"
)

var (
	errTestDB     = errors.New("test db failure")
	errTestDirect = errors.New("test direct failure")

	testWidgetID   = uuid.FromStringOrNil("5a1c3f10-8b0e-4d52-9a4c-1f2e3d4c5b6a")
	testCustomerID = uuid.FromStringOrNil("7d2e4a21-9c1f-4e63-8b5d-2a3f4e5d6c7b")
	testDirectID   = uuid.FromStringOrNil("9f3b5c32-0d2a-4f74-9c6e-3b4a5f6e7d8c")
)

func newErrTestHandler(mc *gomock.Controller) (*widgetHandler, *utilhandler.MockUtilHandler, *requesthandler.MockRequestHandler, *dbhandler.MockDBHandler) {
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	h := &widgetHandler{
		utilHandler: mockUtil,
		reqHandler:  mockReq,
		db:          mockDB,
	}

	return h, mockUtil, mockReq, mockDB
}

func Test_NewWidgetHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	res := NewWidgetHandler(mockReq, mockDB)
	h, ok := res.(*widgetHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.reqHandler != mockReq {
		t.Errorf("Wrong request handler stored.")
	}
	if h.db != mockDB {
		t.Errorf("Wrong db handler stored.")
	}
	if h.utilHandler == nil {
		t.Errorf("Util handler must not be nil.")
	}
}

func Test_Get_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _, mockDB := newErrTestHandler(mc)
	ctx := context.Background()

	mockDB.EXPECT().WidgetGet(ctx, testWidgetID).Return(nil, errTestDB)

	res, err := h.Get(ctx, testWidgetID)
	if !errors.Is(err, errTestDB) {
		t.Errorf("Wrong error. expect: wrapped %v, got: %v", errTestDB, err)
	}
	if res != nil {
		t.Errorf("Wrong result. expect: nil, got: %v", res)
	}
}

func Test_List_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _, mockDB := newErrTestHandler(mc)
	ctx := context.Background()

	filters := map[widget.Field]any{widget.FieldCustomerID: testCustomerID}
	mockDB.EXPECT().WidgetList(ctx, uint64(10), "token-1", filters).Return(nil, errTestDB)

	res, err := h.List(ctx, 10, "token-1", filters)
	if !errors.Is(err, errTestDB) {
		t.Errorf("Wrong error. expect: wrapped %v, got: %v", errTestDB, err)
	}
	if res != nil {
		t.Errorf("Wrong result. expect: nil, got: %v", res)
	}
}

func Test_Create_errorPaths(t *testing.T) {
	direct := &dmdirect.Direct{
		Identity: commonidentity.Identity{ID: testDirectID, CustomerID: testCustomerID},
		Hash:     "synthetic-hash",
	}

	tests := []struct {
		name string

		directCreateErr error
		widgetCreateErr error
		directDeleteErr error
		widgetGetErr    error

		expectDirectDelete bool
		expectWidgetGet    bool
		expectErr          error
	}{
		{
			name:            "direct create fails and nothing is stored",
			directCreateErr: errTestDirect,
			expectErr:       errTestDirect,
		},
		{
			name:               "store fails and the orphan direct is deleted",
			widgetCreateErr:    errTestDB,
			expectDirectDelete: true,
			expectErr:          errTestDB,
		},
		{
			name:               "store fails and the orphan cleanup failure is tolerated",
			widgetCreateErr:    errTestDB,
			directDeleteErr:    errTestDirect,
			expectDirectDelete: true,
			expectErr:          errTestDB,
		},
		{
			name:            "re-read fails after store",
			widgetGetErr:    errTestDB,
			expectWidgetGet: true,
			expectErr:       errTestDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockUtil, mockReq, mockDB := newErrTestHandler(mc)
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(testWidgetID)
			if tt.directCreateErr != nil {
				mockReq.EXPECT().DirectV1DirectCreate(ctx, testCustomerID, dmdirect.ResourceTypeWebchatWidget, testWidgetID).Return(nil, tt.directCreateErr)
			} else {
				mockReq.EXPECT().DirectV1DirectCreate(ctx, testCustomerID, dmdirect.ResourceTypeWebchatWidget, testWidgetID).Return(direct, nil)
				mockDB.EXPECT().WidgetCreate(ctx, gomock.Any()).Return(tt.widgetCreateErr)
			}
			if tt.expectDirectDelete {
				mockReq.EXPECT().DirectV1DirectDelete(ctx, testDirectID).Return(nil, tt.directDeleteErr)
			}
			if tt.expectWidgetGet {
				mockDB.EXPECT().WidgetGet(ctx, testWidgetID).Return(nil, tt.widgetGetErr)
			}

			res, err := h.Create(ctx, testCustomerID, "name", uuid.Nil, uuid.Nil, 600, nil)
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_Create_idleTimeoutDefault(t *testing.T) {
	tests := []struct {
		name          string
		timeout       int
		expectTimeout int
	}{
		{name: "zero falls back to the default", timeout: 0, expectTimeout: 1800},
		{name: "negative falls back to the default", timeout: -5, expectTimeout: 1800},
		{name: "positive value is kept", timeout: 600, expectTimeout: 600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockUtil, mockReq, mockDB := newErrTestHandler(mc)
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(testWidgetID)
			mockReq.EXPECT().DirectV1DirectCreate(ctx, testCustomerID, dmdirect.ResourceTypeWebchatWidget, testWidgetID).
				Return(&dmdirect.Direct{Identity: commonidentity.Identity{ID: testDirectID}, Hash: "synthetic-hash"}, nil)

			var stored *widget.Widget
			mockDB.EXPECT().WidgetCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, w *widget.Widget) error {
				stored = w
				return nil
			})
			mockDB.EXPECT().WidgetGet(ctx, testWidgetID).Return(&widget.Widget{Identity: commonidentity.Identity{ID: testWidgetID}}, nil)

			if _, err := h.Create(ctx, testCustomerID, "name", uuid.Nil, uuid.Nil, tt.timeout, nil); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if stored == nil {
				t.Fatalf("Widget was not stored.")
			}
			if stored.SessionIdleTimeout != tt.expectTimeout {
				t.Errorf("Wrong idle timeout. expect: %d, got: %d", tt.expectTimeout, stored.SessionIdleTimeout)
			}
			if stored.DirectID != testDirectID || stored.Hash != "synthetic-hash" || stored.Status != widget.StatusActive {
				t.Errorf("Wrong stored widget. direct_id: %s, hash: %s, status: %s", stored.DirectID, stored.Hash, stored.Status)
			}
		})
	}
}

func Test_UpdateBasicInfo_errorPaths(t *testing.T) {
	tests := []struct {
		name string

		updateErr error
		getErr    error

		expectGet bool
		expectErr error
	}{
		{name: "update fails", updateErr: errTestDB, expectErr: errTestDB},
		{name: "re-read fails", getErr: errTestDB, expectGet: true, expectErr: errTestDB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _, mockDB := newErrTestHandler(mc)
			ctx := context.Background()

			mockDB.EXPECT().WidgetUpdate(ctx, testWidgetID, gomock.Any()).Return(tt.updateErr)
			if tt.expectGet {
				mockDB.EXPECT().WidgetGet(ctx, testWidgetID).Return(nil, tt.getErr)
			}

			res, err := h.UpdateBasicInfo(ctx, testWidgetID, "name", uuid.Nil, uuid.Nil, 600, nil)
			if !errors.Is(err, tt.expectErr) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_UpdateBasicInfo_idleTimeoutDefault(t *testing.T) {
	tests := []struct {
		name          string
		timeout       int
		expectTimeout int
	}{
		{name: "zero falls back to the default", timeout: 0, expectTimeout: 1800},
		{name: "negative falls back to the default", timeout: -1, expectTimeout: 1800},
		{name: "positive value is kept", timeout: 90, expectTimeout: 90},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _, mockDB := newErrTestHandler(mc)
			ctx := context.Background()

			var fields map[widget.Field]any
			mockDB.EXPECT().WidgetUpdate(ctx, testWidgetID, gomock.Any()).DoAndReturn(func(_ context.Context, _ uuid.UUID, f map[widget.Field]any) error {
				fields = f
				return nil
			})
			mockDB.EXPECT().WidgetGet(ctx, testWidgetID).Return(&widget.Widget{Identity: commonidentity.Identity{ID: testWidgetID}}, nil)

			if _, err := h.UpdateBasicInfo(ctx, testWidgetID, "name", uuid.Nil, uuid.Nil, tt.timeout, nil); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if got := fields[widget.FieldSessionIdleTimeout]; got != tt.expectTimeout {
				t.Errorf("Wrong idle timeout. expect: %d, got: %v", tt.expectTimeout, got)
			}
		})
	}
}

func Test_Delete_errorPaths(t *testing.T) {
	withDirect := &widget.Widget{Identity: commonidentity.Identity{ID: testWidgetID}, DirectID: testDirectID}

	tests := []struct {
		name string

		firstGetErr     error
		directDeleteErr error
		deleteErr       error
		secondGetErr    error

		expectDirectDelete bool
		expectDelete       bool
		expectSecondGet    bool
	}{
		{
			name:        "first read fails and nothing is deleted",
			firstGetErr: errTestDB,
		},
		{
			name:               "direct delete failure does not block the deletion",
			directDeleteErr:    errTestDirect,
			expectDirectDelete: true,
			expectDelete:       true,
			deleteErr:          errTestDB,
		},
		{
			name:               "delete fails",
			expectDirectDelete: true,
			expectDelete:       true,
			deleteErr:          errTestDB,
		},
		{
			name:               "re-read fails",
			expectDirectDelete: true,
			expectDelete:       true,
			expectSecondGet:    true,
			secondGetErr:       errTestDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, mockReq, mockDB := newErrTestHandler(mc)
			ctx := context.Background()

			if tt.firstGetErr != nil {
				mockDB.EXPECT().WidgetGet(ctx, testWidgetID).Return(nil, tt.firstGetErr)
			} else {
				mockDB.EXPECT().WidgetGet(ctx, testWidgetID).Return(withDirect, nil)
			}
			if tt.expectDirectDelete {
				mockReq.EXPECT().DirectV1DirectDelete(ctx, testDirectID).Return(nil, tt.directDeleteErr)
			}
			if tt.expectDelete {
				mockDB.EXPECT().WidgetDelete(ctx, testWidgetID).Return(tt.deleteErr)
			}
			if tt.expectSecondGet {
				mockDB.EXPECT().WidgetGet(ctx, testWidgetID).Return(nil, tt.secondGetErr)
			}

			res, err := h.Delete(ctx, testWidgetID)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
		})
	}
}

func Test_DirectHashRegenerate_errorPaths(t *testing.T) {
	withDirect := &widget.Widget{Identity: commonidentity.Identity{ID: testWidgetID}, DirectID: testDirectID}
	regenerated := &dmdirect.Direct{Identity: commonidentity.Identity{ID: testDirectID}, Hash: "synthetic-new-hash"}

	tests := []struct {
		name string

		firstGetErr     error
		regenerateErr   error
		updateErr       error
		secondGetErr    error
		expectRegen     bool
		expectUpdate    bool
		expectSecondGet bool
	}{
		{name: "first read fails", firstGetErr: errTestDB},
		{name: "regenerate rpc fails", regenerateErr: errTestDirect, expectRegen: true},
		{name: "hash update fails", expectRegen: true, expectUpdate: true, updateErr: errTestDB},
		{name: "re-read fails", expectRegen: true, expectUpdate: true, expectSecondGet: true, secondGetErr: errTestDB},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, mockReq, mockDB := newErrTestHandler(mc)
			ctx := context.Background()

			if tt.firstGetErr != nil {
				mockDB.EXPECT().WidgetGet(ctx, testWidgetID).Return(nil, tt.firstGetErr)
			} else {
				mockDB.EXPECT().WidgetGet(ctx, testWidgetID).Return(withDirect, nil)
			}
			if tt.expectRegen {
				if tt.regenerateErr != nil {
					mockReq.EXPECT().DirectV1DirectRegenerate(ctx, testDirectID).Return(nil, tt.regenerateErr)
				} else {
					mockReq.EXPECT().DirectV1DirectRegenerate(ctx, testDirectID).Return(regenerated, nil)
				}
			}
			if tt.expectUpdate {
				mockDB.EXPECT().WidgetUpdate(ctx, testWidgetID, map[widget.Field]any{widget.FieldHash: "synthetic-new-hash"}).Return(tt.updateErr)
			}
			if tt.expectSecondGet {
				mockDB.EXPECT().WidgetGet(ctx, testWidgetID).Return(nil, tt.secondGetErr)
			}

			res, err := h.DirectHashRegenerate(ctx, testWidgetID)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
		})
	}
}
