package aicallhandler

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_GetSkipCache(t *testing.T) {
	id := uuid.FromStringOrNil("5c2d0b10-2b00-4c00-8000-000000000001")
	otherErr := fmt.Errorf("db down")

	tests := []struct {
		name string

		dbRes *aicall.AIcall
		dbErr error

		expectRes      *aicall.AIcall
		expectNotFound bool
		expectErr      error
	}{
		{
			name:      "found",
			dbRes:     &aicall.AIcall{Identity: commonidentity.Identity{ID: id}},
			expectRes: &aicall.AIcall{Identity: commonidentity.Identity{ID: id}},
		},
		{
			name:           "not found is classified",
			dbErr:          dbhandler.ErrNotFound,
			expectNotFound: true,
		},
		{
			name:      "other error is returned unchanged",
			dbErr:     otherErr,
			expectErr: otherErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &aicallHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().AIcallGetSkipCache(ctx, id).Return(tt.dbRes, tt.dbErr)

			res, err := h.GetSkipCache(ctx, id)

			if tt.expectNotFound {
				var ve *cerrors.VoipbinError
				if !errors.As(err, &ve) || ve.Status != cerrors.StatusNotFound || ve.Reason != "AICALL_NOT_FOUND" {
					t.Fatalf("Wrong error. expect: AICALL_NOT_FOUND, got: %v", err)
				}
				if !errors.Is(err, dbhandler.ErrNotFound) {
					t.Errorf("The original error is not wrapped. got: %v", err)
				}
				if res != nil {
					t.Errorf("Expected nil result, got: %v", res)
				}
				return
			}
			if err != tt.expectErr {
				t.Fatalf("Wrong error. expect: %v, got: %v", tt.expectErr, err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Get_notFoundIsClassified(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &aicallHandler{db: mockDB}
	ctx := context.Background()
	id := uuid.FromStringOrNil("5c2d0b10-2b00-4c00-8000-000000000002")

	mockDB.EXPECT().AIcallGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Get(ctx, id)
	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) || ve.Status != cerrors.StatusNotFound {
		t.Fatalf("Wrong error. expect: not found, got: %v", err)
	}
	if res != nil {
		t.Errorf("Expected nil result, got: %v", res)
	}
}

func Test_Create_errors(t *testing.T) {
	id := uuid.FromStringOrNil("5c2d0b10-2b00-4c00-8000-000000000003")

	tests := []struct {
		name string

		createErr error
		getErr    error
	}{
		{name: "create failure", createErr: fmt.Errorf("insert failed")},
		{name: "get after create failure", getErr: fmt.Errorf("select failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &aicallHandler{db: mockDB, notifyHandler: mockNotify, utilHandler: mockUtil}
			ctx := context.Background()

			mockDB.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(tt.createErr)
			if tt.createErr == nil {
				mockDB.EXPECT().AIcallGet(ctx, id).Return(nil, tt.getErr)
			}

			wantErr := tt.createErr
			if wantErr == nil {
				wantErr = tt.getErr
			}

			a := &ai.AI{Identity: commonidentity.Identity{ID: id}}
			res, err := h.Create(ctx, id, a, aicall.AssistanceTypeAI, id, uuid.Nil, aicall.ReferenceTypeCall, id, uuid.Nil, uuid.Nil, uuid.Nil, nil, nil)
			if err != wantErr {
				t.Errorf("Wrong error. expect: %v, got: %v", wantErr, err)
			}
			if res != nil {
				t.Errorf("Expected nil result, got: %v", res)
			}
		})
	}
}

func Test_CreateByMessaging_errors(t *testing.T) {
	id := uuid.FromStringOrNil("5c2d0b10-2b00-4c00-8000-000000000004")

	tests := []struct {
		name string

		createErr error
		getErr    error
	}{
		{name: "create failure", createErr: fmt.Errorf("insert failed")},
		{name: "get after create failure", getErr: fmt.Errorf("select failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &aicallHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().AIcallCreate(ctx, gomock.Any()).Return(tt.createErr)
			if tt.createErr == nil {
				mockDB.EXPECT().AIcallGet(ctx, id).Return(nil, tt.getErr)
			}

			wantErr := tt.createErr
			if wantErr == nil {
				wantErr = tt.getErr
			}

			a := &ai.AI{Identity: commonidentity.Identity{ID: id}}
			res, err := h.CreateByMessaging(ctx, id, a, aicall.AssistanceTypeAI, id, uuid.Nil, aicall.ReferenceTypeCall, id, uuid.Nil, uuid.Nil, nil, nil)
			if err != wantErr {
				t.Errorf("Wrong error. expect: %v, got: %v", wantErr, err)
			}
			if res != nil {
				t.Errorf("Expected nil result, got: %v", res)
			}
		})
	}
}

func Test_Delete_errors(t *testing.T) {
	id := uuid.FromStringOrNil("5c2d0b10-2b00-4c00-8000-000000000005")

	tests := []struct {
		name string

		deleteErr error
		getErr    error
	}{
		{name: "delete failure", deleteErr: fmt.Errorf("delete failed")},
		{name: "get after delete failure", getErr: fmt.Errorf("select failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &aicallHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().AIcallDelete(ctx, id).Return(tt.deleteErr)
			if tt.deleteErr == nil {
				mockDB.EXPECT().AIcallGet(ctx, id).Return(nil, tt.getErr)
			}

			wantErr := tt.deleteErr
			if wantErr == nil {
				wantErr = tt.getErr
			}

			res, err := h.Delete(ctx, id)
			if err != wantErr {
				t.Errorf("Wrong error. expect: %v, got: %v", wantErr, err)
			}
			if res != nil {
				t.Errorf("Expected nil result, got: %v", res)
			}
		})
	}
}

func Test_GetByReferenceID_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &aicallHandler{db: mockDB}
	ctx := context.Background()
	refID := uuid.FromStringOrNil("5c2d0b10-2b00-4c00-8000-000000000006")
	expectErr := fmt.Errorf("select failed")

	mockDB.EXPECT().AIcallGetByReferenceID(ctx, refID).Return(nil, expectErr)

	res, err := h.GetByReferenceID(ctx, refID)
	if err != expectErr || res != nil {
		t.Errorf("Wrong match. expect: %v/nil, got: %v/%v", expectErr, err, res)
	}
}

func Test_GetByCustomerIDAndReferenceID_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &aicallHandler{db: mockDB}
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("5c2d0b10-2b00-4c00-8000-000000000007")
	refID := uuid.FromStringOrNil("5c2d0b10-2b00-4c00-8000-000000000008")
	expectErr := fmt.Errorf("select failed")

	mockDB.EXPECT().AIcallGetByCustomerIDAndReferenceID(ctx, customerID, refID).Return(nil, expectErr)

	res, err := h.GetByCustomerIDAndReferenceID(ctx, customerID, refID)
	if err != expectErr || res != nil {
		t.Errorf("Wrong match. expect: %v/nil, got: %v/%v", expectErr, err, res)
	}
}

func Test_List_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &aicallHandler{db: mockDB}
	ctx := context.Background()
	filters := map[aicall.Field]any{aicall.FieldDeleted: false}
	expectErr := fmt.Errorf("list failed")

	mockDB.EXPECT().AIcallList(ctx, uint64(10), "token", filters).Return(nil, expectErr)

	res, err := h.List(ctx, 10, "token", filters)
	if err != expectErr || res != nil {
		t.Errorf("Wrong match. expect: %v/nil, got: %v/%v", expectErr, err, res)
	}
}

func Test_UpdatePipecatcallID_errors(t *testing.T) {
	id := uuid.FromStringOrNil("5c2d0b10-2b00-4c00-8000-000000000009")
	pcID := uuid.FromStringOrNil("5c2d0b10-2b00-4c00-8000-00000000000a")

	tests := []struct {
		name string

		updateErr error
		getErr    error
	}{
		{name: "update failure", updateErr: fmt.Errorf("update failed")},
		{name: "get after update failure", getErr: fmt.Errorf("select failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &aicallHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().AIcallUpdate(ctx, id, map[aicall.Field]any{aicall.FieldPipecatcallID: pcID}).Return(tt.updateErr)
			wantErr := tt.updateErr
			if tt.updateErr == nil {
				mockDB.EXPECT().AIcallGet(ctx, id).Return(nil, tt.getErr)
				wantErr = tt.getErr
			}

			res, err := h.UpdatePipecatcallID(ctx, id, pcID)
			if res != nil || !errors.Is(err, wantErr) {
				t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", wantErr, res, err)
			}
		})
	}
}

func Test_UpdateStatus_webhookEventByStatus(t *testing.T) {
	id := uuid.FromStringOrNil("5c2d0b10-2b00-4c00-8000-00000000000b")
	customerID := uuid.FromStringOrNil("5c2d0b10-2b00-4c00-8000-00000000000c")
	created := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string

		status    aicall.Status
		tmCreate  *time.Time
		expectEvt string
	}{
		{name: "progressing", status: aicall.StatusProgressing, expectEvt: aicall.EventTypeStatusProgressing},
		{name: "pausing", status: aicall.StatusPausing, expectEvt: aicall.EventTypeStatusPausing},
		{name: "resuming", status: aicall.StatusResuming, expectEvt: aicall.EventTypeStatusResuming},
		{name: "terminating", status: aicall.StatusTerminating, expectEvt: aicall.EventTypeStatusTerminating},
		{name: "terminated without create time", status: aicall.StatusTerminated, expectEvt: aicall.EventTypeStatusTerminated},
		{name: "terminated with create time", status: aicall.StatusTerminated, tmCreate: &created, expectEvt: aicall.EventTypeStatusTerminated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := &aicallHandler{db: mockDB, notifyHandler: mockNotify, utilHandler: mockUtil}
			ctx := context.Background()

			updated := &aicall.AIcall{
				Identity: commonidentity.Identity{ID: id, CustomerID: customerID},
				Status:   tt.status,
				TMCreate: tt.tmCreate,
			}

			fields := map[aicall.Field]any{aicall.FieldStatus: tt.status}
			if tt.status == aicall.StatusTerminated {
				now := time.Date(2026, 1, 2, 4, 0, 0, 0, time.UTC)
				mockUtil.EXPECT().TimeNow().Return(&now)
				fields[aicall.FieldTMEnd] = &now
			}
			mockDB.EXPECT().AIcallUpdate(ctx, id, fields).Return(nil)
			mockDB.EXPECT().AIcallGet(ctx, id).Return(updated, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, tt.expectEvt, updated)

			res, err := h.UpdateStatus(ctx, id, tt.status)
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if !reflect.DeepEqual(res, updated) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", updated, res)
			}
		})
	}
}

func Test_UpdateStatus_errors(t *testing.T) {
	id := uuid.FromStringOrNil("5c2d0b10-2b00-4c00-8000-00000000000d")

	tests := []struct {
		name string

		updateErr error
		getErr    error
	}{
		{name: "update failure", updateErr: fmt.Errorf("update failed")},
		{name: "get after update failure", getErr: fmt.Errorf("select failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			h := &aicallHandler{db: mockDB}
			ctx := context.Background()

			mockDB.EXPECT().AIcallUpdate(ctx, id, map[aicall.Field]any{aicall.FieldStatus: aicall.StatusProgressing}).Return(tt.updateErr)
			wantErr := tt.updateErr
			if tt.updateErr == nil {
				mockDB.EXPECT().AIcallGet(ctx, id).Return(nil, tt.getErr)
				wantErr = tt.getErr
			}

			res, err := h.UpdateStatus(ctx, id, aicall.StatusProgressing)
			if res != nil || !errors.Is(err, wantErr) {
				t.Errorf("Wrong match. expect: nil/%v, got: %v/%v", wantErr, res, err)
			}
		})
	}
}
