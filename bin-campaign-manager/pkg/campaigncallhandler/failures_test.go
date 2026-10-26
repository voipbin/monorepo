package campaigncallhandler

import (
	"context"
	stderrors "errors"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	omoutdialtarget "monorepo/bin-outdial-manager/models/outdialtarget"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-campaign-manager/models/campaigncall"
	"monorepo/bin-campaign-manager/pkg/dbhandler"
)

func Test_NewCampaigncallHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)

	res := NewCampaigncallHandler(mockDB, mockReq, mockNotify)

	h, ok := res.(*campaigncallHandler)
	if !ok {
		t.Fatalf("Wrong match. expect: *campaigncallHandler, got: %T", res)
	}
	if h.db != mockDB || h.reqHandler != mockReq || h.notifyHandler != mockNotify {
		t.Errorf("Wrong match. the given dependencies must be kept. handler: %+v", h)
	}
}

func Test_Create_failures(t *testing.T) {
	campaignTargetID := uuid.FromStringOrNil("5c1a0001-0000-4000-8000-000000000001")
	newID := uuid.FromStringOrNil("5c1a0001-0000-4000-8000-000000000002")
	customerID := uuid.FromStringOrNil("5c1a0001-0000-4000-8000-000000000003")
	errDB := stderrors.New("db unavailable")

	tests := []struct {
		name  string
		setup func(mockDB *dbhandler.MockDBHandler, mockNotify *notifyhandler.MockNotifyHandler, mockReq *requesthandler.MockRequestHandler)

		expectErr error
	}{
		{
			name: "database insert fails",
			setup: func(mockDB *dbhandler.MockDBHandler, mockNotify *notifyhandler.MockNotifyHandler, mockReq *requesthandler.MockRequestHandler) {
				mockDB.EXPECT().CampaigncallCreate(gomock.Any(), gomock.Any()).Return(errDB)
			},
			expectErr: errDB,
		},
		{
			name: "reading the created record fails",
			setup: func(mockDB *dbhandler.MockDBHandler, mockNotify *notifyhandler.MockNotifyHandler, mockReq *requesthandler.MockRequestHandler) {
				mockDB.EXPECT().CampaigncallCreate(gomock.Any(), gomock.Any()).Return(nil)
				mockDB.EXPECT().CampaigncallGet(gomock.Any(), newID).Return(nil, errDB)
			},
			expectErr: errDB,
		},
		{
			name: "outdial target update fails and the campaigncall is closed as failed",
			setup: func(mockDB *dbhandler.MockDBHandler, mockNotify *notifyhandler.MockNotifyHandler, mockReq *requesthandler.MockRequestHandler) {
				created := &campaigncall.Campaigncall{
					Identity:        commonidentity.Identity{ID: newID, CustomerID: customerID},
					OutdialTargetID: campaignTargetID,
				}
				done := &campaigncall.Campaigncall{
					Identity:        commonidentity.Identity{ID: newID, CustomerID: customerID},
					OutdialTargetID: campaignTargetID,
					Status:          campaigncall.StatusDone,
					Result:          campaigncall.ResultFail,
				}

				mockDB.EXPECT().CampaigncallCreate(gomock.Any(), gomock.Any()).Return(nil)
				mockDB.EXPECT().CampaigncallGet(gomock.Any(), newID).Return(created, nil)
				mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, campaigncall.EventTypeCampaigncallCreated, created)
				mockReq.EXPECT().OutdialV1OutdialtargetUpdateStatusProgressing(gomock.Any(), campaignTargetID, 0).Return(nil, errDB)

				// the failure closes the campaigncall with the fail result and resets the target
				mockDB.EXPECT().CampaigncallUpdateStatusAndResult(gomock.Any(), newID, campaigncall.StatusDone, campaigncall.ResultFail).Return(nil)
				mockDB.EXPECT().CampaigncallGet(gomock.Any(), newID).Return(done, nil)
				mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), customerID, campaigncall.EventTypeCampaigncallUpdated, done)
				mockReq.EXPECT().OutdialV1OutdialtargetUpdateStatus(gomock.Any(), campaignTargetID, omoutdialtarget.StatusIdle).Return(&omoutdialtarget.OutdialTarget{}, nil)
			},
			expectErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &campaigncallHandler{util: mockUtil, db: mockDB, notifyHandler: mockNotify, reqHandler: mockReq}

			mockUtil.EXPECT().UUIDCreate().Return(newID)
			tt.setup(mockDB, mockNotify, mockReq)

			res, err := h.Create(
				context.Background(), customerID,
				uuid.Nil, uuid.Nil, uuid.Nil, campaignTargetID, uuid.Nil,
				uuid.Nil, uuid.Nil,
				campaigncall.ReferenceTypeCall, uuid.Nil, nil, nil, 0, 0,
			)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if !stderrors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_Delete_failures(t *testing.T) {
	id := uuid.FromStringOrNil("5c1a0002-0000-4000-8000-000000000001")
	customerID := uuid.FromStringOrNil("5c1a0002-0000-4000-8000-000000000002")
	errDB := stderrors.New("db unavailable")

	tests := []struct {
		name  string
		setup func(mockDB *dbhandler.MockDBHandler)

		expectErr error
	}{
		{
			name: "lookup fails",
			setup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().CampaigncallGet(gomock.Any(), id).Return(nil, errDB)
			},
			expectErr: errDB,
		},
		{
			name: "database delete fails",
			setup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().CampaigncallGet(gomock.Any(), id).Return(&campaigncall.Campaigncall{
					Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
					Status:   campaigncall.StatusDone,
				}, nil)
				mockDB.EXPECT().CampaigncallDelete(gomock.Any(), id).Return(errDB)
			},
			expectErr: errDB,
		},
		{
			name: "reading the deleted record fails",
			setup: func(mockDB *dbhandler.MockDBHandler) {
				mockDB.EXPECT().CampaigncallGet(gomock.Any(), id).Return(&campaigncall.Campaigncall{
					Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
					Status:   campaigncall.StatusDone,
				}, nil)
				mockDB.EXPECT().CampaigncallDelete(gomock.Any(), id).Return(nil)
				mockDB.EXPECT().CampaigncallGet(gomock.Any(), id).Return(nil, errDB)
			},
			expectErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _, _ := newErrorsTestHandler(mc)
			tt.setup(mockDB)

			res, err := h.Delete(context.Background(), id)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if !stderrors.Is(err, tt.expectErr) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

// Test_Delete_notDone checks that a campaigncall which is not done is not deleted and
// that neither a webhook event nor a second lookup happens.
func Test_Delete_notDone(t *testing.T) {
	id := uuid.FromStringOrNil("5c1a0003-0000-4000-8000-000000000001")

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _, _ := newErrorsTestHandler(mc)

	// only one lookup is expected. CampaigncallDelete and the webhook publisher have no
	// expectation, so any call to them fails the test.
	mockDB.EXPECT().CampaigncallGet(gomock.Any(), id).Return(&campaigncall.Campaigncall{
		Identity: commonidentity.Identity{ID: id},
		Status:   campaigncall.StatusDialing,
	}, nil)

	res, _ := h.Delete(context.Background(), id)
	if res != nil {
		t.Errorf("Wrong match. expect: nil result, got: %v", res)
	}
}

func Test_EventHandle_doneFailures(t *testing.T) {
	id := uuid.FromStringOrNil("5c1a0004-0000-4000-8000-000000000001")
	errDB := stderrors.New("db unavailable")

	tests := []struct {
		name string
		fn   func(h *campaigncallHandler) (*campaigncall.Campaigncall, error)

		expectResult campaigncall.Result
	}{
		{
			name: "activeflow deleted",
			fn: func(h *campaigncallHandler) (*campaigncall.Campaigncall, error) {
				return h.EventHandleActiveflowDeleted(context.Background(), &campaigncall.Campaigncall{
					Identity: commonidentity.Identity{ID: id},
				})
			},
			expectResult: campaigncall.ResultSuccess,
		},
		{
			name: "reference call hangup",
			fn: func(h *campaigncallHandler) (*campaigncall.Campaigncall, error) {
				return h.EventHandleReferenceCallHungup(
					context.Background(),
					&cmcall.Call{HangupReason: cmcall.HangupReasonBusy},
					&campaigncall.Campaigncall{
						Identity: commonidentity.Identity{ID: id},
						Status:   campaigncall.StatusProgressing,
					},
				)
			},
			expectResult: campaigncall.ResultFail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _, _ := newErrorsTestHandler(mc)

			mockDB.EXPECT().CampaigncallUpdateStatusAndResult(gomock.Any(), id, campaigncall.StatusDone, tt.expectResult).Return(errDB)

			res, err := tt.fn(h)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if !stderrors.Is(err, errDB) {
				t.Errorf("Wrong match. expect: %v, got: %v", errDB, err)
			}
		})
	}
}
