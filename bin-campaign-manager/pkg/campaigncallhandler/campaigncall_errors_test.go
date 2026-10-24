package campaigncallhandler

import (
	"context"
	stderrors "errors"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
)

func newErrorsTestHandler(mc *gomock.Controller) (*campaigncallHandler, *dbhandler.MockDBHandler, *notifyhandler.MockNotifyHandler, *requesthandler.MockRequestHandler) {
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	return &campaigncallHandler{db: mockDB, notifyHandler: mockNotify, reqHandler: mockReq}, mockDB, mockNotify, mockReq
}

func Test_Get_errors(t *testing.T) {
	id := uuid.FromStringOrNil("f1000000-0000-4000-8000-000000000001")

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
			expectReason: "CAMPAIGNCALL_NOT_FOUND",
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

			h, mockDB, _, _ := newErrorsTestHandler(mc)
			ctx := context.Background()

			mockDB.EXPECT().CampaigncallGet(ctx, id).Return(nil, tt.dbErr)

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

func Test_GetBy_lookups(t *testing.T) {
	id := uuid.FromStringOrNil("f2000000-0000-4000-8000-000000000001")
	found := &campaigncall.Campaigncall{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("f2000000-0000-4000-8000-000000000002")}}
	dbErr := stderrors.New("db error")

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _, _ := newErrorsTestHandler(mc)
	ctx := context.Background()

	mockDB.EXPECT().CampaigncallGetByReferenceID(ctx, id).Return(found, nil)
	if res, err := h.GetByReferenceID(ctx, id); err != nil || res != found {
		t.Errorf("Wrong match. expect: the campaigncall, got: %v, err: %v", res, err)
	}
	mockDB.EXPECT().CampaigncallGetByReferenceID(ctx, id).Return(nil, dbErr)
	if res, err := h.GetByReferenceID(ctx, id); !stderrors.Is(err, dbErr) || res != nil {
		t.Errorf("Wrong match. expect: db error, got: %v, res: %v", err, res)
	}

	mockDB.EXPECT().CampaigncallGetByActiveflowID(ctx, id).Return(found, nil)
	if res, err := h.GetByActiveflowID(ctx, id); err != nil || res != found {
		t.Errorf("Wrong match. expect: the campaigncall, got: %v, err: %v", res, err)
	}
	mockDB.EXPECT().CampaigncallGetByActiveflowID(ctx, id).Return(nil, dbErr)
	if res, err := h.GetByActiveflowID(ctx, id); !stderrors.Is(err, dbErr) || res != nil {
		t.Errorf("Wrong match. expect: db error, got: %v, res: %v", err, res)
	}
}

func Test_List_variants(t *testing.T) {
	id := uuid.FromStringOrNil("f3000000-0000-4000-8000-000000000001")
	items := []*campaigncall.Campaigncall{{Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("f3000000-0000-4000-8000-000000000002")}}}
	dbErr := stderrors.New("db error")
	filters := map[campaigncall.Field]any{campaigncall.FieldCampaignID: id}

	tests := []struct {
		name   string
		expect func(ctx context.Context, mockDB *dbhandler.MockDBHandler, res []*campaigncall.Campaigncall, err error)
		call   func(ctx context.Context, h *campaigncallHandler) ([]*campaigncall.Campaigncall, error)
	}{
		{
			name: "list with filters",
			expect: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, res []*campaigncall.Campaigncall, err error) {
				mockDB.EXPECT().CampaigncallList(ctx, "token", uint64(7), filters).Return(res, err)
			},
			call: func(ctx context.Context, h *campaigncallHandler) ([]*campaigncall.Campaigncall, error) {
				return h.List(ctx, "token", 7, filters)
			},
		},
		{
			name: "list by customer id",
			expect: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, res []*campaigncall.Campaigncall, err error) {
				mockDB.EXPECT().CampaigncallListByCustomerID(ctx, id, "token", uint64(7)).Return(res, err)
			},
			call: func(ctx context.Context, h *campaigncallHandler) ([]*campaigncall.Campaigncall, error) {
				return h.ListByCustomerID(ctx, id, "token", 7)
			},
		},
		{
			name: "list by campaign id",
			expect: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, res []*campaigncall.Campaigncall, err error) {
				mockDB.EXPECT().CampaigncallListByCampaignID(ctx, id, "token", uint64(7)).Return(res, err)
			},
			call: func(ctx context.Context, h *campaigncallHandler) ([]*campaigncall.Campaigncall, error) {
				return h.ListByCampaignID(ctx, id, "token", 7)
			},
		},
		{
			name: "list by campaign id and status",
			expect: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, res []*campaigncall.Campaigncall, err error) {
				mockDB.EXPECT().CampaigncallListByCampaignIDAndStatus(ctx, id, campaigncall.StatusDialing, "token", uint64(7)).Return(res, err)
			},
			call: func(ctx context.Context, h *campaigncallHandler) ([]*campaigncall.Campaigncall, error) {
				return h.ListByCampaignIDAndStatus(ctx, id, campaigncall.StatusDialing, "token", 7)
			},
		},
		{
			name: "list ongoing by campaign id",
			expect: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, res []*campaigncall.Campaigncall, err error) {
				mockDB.EXPECT().CampaigncallListOngoingByCampaignID(ctx, id, "token", uint64(7)).Return(res, err)
			},
			call: func(ctx context.Context, h *campaigncallHandler) ([]*campaigncall.Campaigncall, error) {
				return h.ListOngoingByCampaignID(ctx, id, "token", 7)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name+" success", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _, _ := newErrorsTestHandler(mc)
			ctx := context.Background()

			tt.expect(ctx, mockDB, items, nil)
			res, err := tt.call(ctx, h)
			if err != nil || len(res) != 1 || res[0] != items[0] {
				t.Errorf("Wrong match. expect: the listed items, got: %v, err: %v", res, err)
			}
		})
		t.Run(tt.name+" error", func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _, _ := newErrorsTestHandler(mc)
			ctx := context.Background()

			tt.expect(ctx, mockDB, nil, dbErr)
			res, err := tt.call(ctx, h)
			if !stderrors.Is(err, dbErr) || res != nil {
				t.Errorf("Wrong match. expect: db error, got: %v, res: %v", err, res)
			}
		})
	}
}

func Test_Done_errors(t *testing.T) {
	id := uuid.FromStringOrNil("f4000000-0000-4000-8000-000000000001")
	targetID := uuid.FromStringOrNil("f4000000-0000-4000-8000-000000000002")
	cc := &campaigncall.Campaigncall{
		Identity:        commonidentity.Identity{ID: id, CustomerID: uuid.FromStringOrNil("f4000000-0000-4000-8000-000000000003")},
		OutdialTargetID: targetID,
	}

	tests := []struct {
		name      string
		result    campaigncall.Result
		setup     func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockNotify *notifyhandler.MockNotifyHandler, mockReq *requesthandler.MockRequestHandler)
		expectErr string
	}{
		{
			name:   "status update error is returned before anything else is called",
			result: campaigncall.ResultFail,
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, _ *notifyhandler.MockNotifyHandler, _ *requesthandler.MockRequestHandler) {
				mockDB.EXPECT().CampaigncallUpdateStatusAndResult(ctx, id, campaigncall.StatusDone, campaigncall.ResultFail).Return(stderrors.New("update error"))
			},
			expectErr: "update error",
		},
		{
			name:   "read back error is returned before the outdial target is touched",
			result: campaigncall.ResultFail,
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, _ *notifyhandler.MockNotifyHandler, _ *requesthandler.MockRequestHandler) {
				mockDB.EXPECT().CampaigncallUpdateStatusAndResult(ctx, id, campaigncall.StatusDone, campaigncall.ResultFail).Return(nil)
				mockDB.EXPECT().CampaigncallGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			expectErr: "get error",
		},
		{
			name:   "unknown result is rejected before the outdial target is touched",
			result: campaigncall.Result("unknown"),
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockNotify *notifyhandler.MockNotifyHandler, _ *requesthandler.MockRequestHandler) {
				mockDB.EXPECT().CampaigncallUpdateStatusAndResult(ctx, id, campaigncall.StatusDone, campaigncall.Result("unknown")).Return(nil)
				mockDB.EXPECT().CampaigncallGet(ctx, id).Return(cc, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, cc.CustomerID, campaigncall.EventTypeCampaigncallUpdated, cc)
			},
			expectErr: "status not found",
		},
		{
			name:   "outdial target update error is returned",
			result: campaigncall.ResultSuccess,
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler, mockNotify *notifyhandler.MockNotifyHandler, mockReq *requesthandler.MockRequestHandler) {
				mockDB.EXPECT().CampaigncallUpdateStatusAndResult(ctx, id, campaigncall.StatusDone, campaigncall.ResultSuccess).Return(nil)
				mockDB.EXPECT().CampaigncallGet(ctx, id).Return(cc, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, cc.CustomerID, campaigncall.EventTypeCampaigncallUpdated, cc)
				mockReq.EXPECT().OutdialV1OutdialtargetUpdateStatus(ctx, targetID, omoutdialtarget.StatusDone).Return(nil, stderrors.New("outdial error"))
			},
			expectErr: "outdial error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, mockNotify, mockReq := newErrorsTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, mockDB, mockNotify, mockReq)

			res, err := h.Done(ctx, id, tt.result)
			if err == nil || err.Error() != tt.expectErr || res != nil {
				t.Errorf("Wrong match. expect error: %s, got: %v, res: %v", tt.expectErr, err, res)
			}
		})
	}
}

func Test_Progressing_errors(t *testing.T) {
	id := uuid.FromStringOrNil("f5000000-0000-4000-8000-000000000001")

	tests := []struct {
		name      string
		setup     func(ctx context.Context, mockDB *dbhandler.MockDBHandler)
		expectErr string
	}{
		{
			name: "status update error is returned",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().CampaigncallUpdateStatus(ctx, id, campaigncall.StatusProgressing).Return(stderrors.New("update error"))
			},
			expectErr: "update error",
		},
		{
			name: "read back error is returned",
			setup: func(ctx context.Context, mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().CampaigncallUpdateStatus(ctx, id, campaigncall.StatusProgressing).Return(nil)
				mockDB.EXPECT().CampaigncallGet(ctx, id).Return(nil, stderrors.New("get error"))
			},
			expectErr: "get error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _, _ := newErrorsTestHandler(mc)
			ctx := context.Background()

			tt.setup(ctx, mockDB)

			res, err := h.Progressing(ctx, id)
			if err == nil || err.Error() != tt.expectErr || res != nil {
				t.Errorf("Wrong match. expect error: %s, got: %v, res: %v", tt.expectErr, err, res)
			}
		})
	}
}
