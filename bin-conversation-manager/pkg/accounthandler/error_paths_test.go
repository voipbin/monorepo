package accounthandler

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
)

func newErrorPathHandler(mc *gomock.Controller) (*accountHandler, *dbhandler.MockDBHandler, *notifyhandler.MockNotifyHandler, *utilhandler.MockUtilHandler) {
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := &accountHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	return h, mockDB, mockNotify, mockUtil
}

func Test_Get_errorMapping(t *testing.T) {
	id := uuid.FromStringOrNil("74a879e6-fe49-11ed-98e7-576bc17c7b79")

	tests := []struct {
		name string

		dbErr error

		expectTyped bool
	}{
		{"not found becomes a typed not found error", dbhandler.ErrNotFound, true},
		{"other errors are not typed", fmt.Errorf("connection reset"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _, _ := newErrorPathHandler(mc)
			mockDB.EXPECT().AccountGet(gomock.Any(), id).Return(nil, tt.dbErr)

			res, err := h.Get(context.Background(), id)
			if err == nil || res != nil {
				t.Fatalf("expected error and nil result, got: %v, %v", res, err)
			}

			var ve *cerrors.VoipbinError
			if stderrors.As(err, &ve) != tt.expectTyped {
				t.Errorf("Wrong typed error. expect: %v, got err: %v", tt.expectTyped, err)
			}
			if tt.expectTyped && ve.Status != cerrors.StatusNotFound {
				t.Errorf("Wrong status. expect: %v, got: %v", cerrors.StatusNotFound, ve.Status)
			}
			if !stderrors.Is(err, tt.dbErr) {
				t.Errorf("the cause must stay reachable. err: %v", err)
			}
		})
	}
}

func Test_List_dbError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mockDB, _, _ := newErrorPathHandler(mc)
	mockDB.EXPECT().AccountList(gomock.Any(), uint64(10), "", gomock.Any()).Return(nil, fmt.Errorf("db down"))

	res, err := h.List(context.Background(), "", 10, map[account.Field]any{})
	if err == nil || res != nil {
		t.Errorf("expected error and nil result, got: %v, %v", res, err)
	}
}

func Test_Create_dbErrors(t *testing.T) {
	customerID := uuid.FromStringOrNil("3b24255a-e60b-11ec-9815-5f679b51ac4d")
	id := uuid.FromStringOrNil("4f187bba-fdf7-11ed-87b2-d7cc82900fb6")

	tests := []struct {
		name string

		createErr error
		getErr    error
	}{
		{"insert failure stops before reading back", fmt.Errorf("insert failed"), nil},
		{"read back failure stops before the webhook event", nil, fmt.Errorf("read failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _, mockUtil := newErrorPathHandler(mc)

			mockUtil.EXPECT().UUIDCreate().Return(id)
			// sms accounts need no external setup
			mockDB.EXPECT().AccountCreate(gomock.Any(), gomock.Any()).Return(tt.createErr)
			if tt.createErr == nil {
				mockDB.EXPECT().AccountGet(gomock.Any(), id).Return(nil, tt.getErr)
			}
			// no notify expectation: a webhook event must not be published

			res, err := h.Create(context.Background(), customerID, account.TypeSMS, "n", "d", "s", "t", uuid.Nil, json.RawMessage(nil))
			if err == nil || res != nil {
				t.Errorf("expected error and nil result, got: %v, %v", res, err)
			}
		})
	}
}

func Test_Update_errors(t *testing.T) {
	id := uuid.FromStringOrNil("74a879e6-fe49-11ed-98e7-576bc17c7b79")
	fields := map[account.Field]any{account.FieldName: "new name"}

	tests := []struct {
		name string

		updateErr error
		getErr    error
		accType   account.Type
	}{
		{"update failure stops before reading back", fmt.Errorf("update failed"), nil, account.TypeSMS},
		{"read back failure stops before setup", nil, fmt.Errorf("read failed"), account.TypeSMS},
		{"setup failure stops before the webhook event", nil, nil, account.Type("unsupported")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _, _ := newErrorPathHandler(mc)

			mockDB.EXPECT().AccountUpdate(gomock.Any(), id, fields).Return(tt.updateErr)
			if tt.updateErr == nil {
				if tt.getErr != nil {
					mockDB.EXPECT().AccountGet(gomock.Any(), id).Return(nil, tt.getErr)
				} else {
					mockDB.EXPECT().AccountGet(gomock.Any(), id).Return(&account.Account{
						Identity: commonidentity.Identity{ID: id},
						Type:     tt.accType,
					}, nil)
				}
			}

			res, err := h.Update(context.Background(), id, fields)
			if err == nil || res != nil {
				t.Errorf("expected error and nil result, got: %v, %v", res, err)
			}
		})
	}
}

func Test_Delete_errors(t *testing.T) {
	id := uuid.FromStringOrNil("74a879e6-fe49-11ed-98e7-576bc17c7b79")
	existing := &account.Account{
		Identity: commonidentity.Identity{ID: id},
		Type:     account.TypeLine,
	}

	tests := []struct {
		name string

		firstGetErr error
		deleteErr   error
		secondGet   error
	}{
		{"lookup failure skips teardown and delete", fmt.Errorf("lookup failed"), nil, nil},
		{"delete failure stops before reading back", nil, fmt.Errorf("delete failed"), nil},
		{"read back failure stops before the webhook event", nil, nil, fmt.Errorf("read failed")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, mockDB, _, _ := newErrorPathHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h.lineHandler = mockLine

			if tt.firstGetErr != nil {
				mockDB.EXPECT().AccountGet(gomock.Any(), id).Return(nil, tt.firstGetErr)
			} else {
				mockDB.EXPECT().AccountGet(gomock.Any(), id).Return(existing, nil)
				mockLine.EXPECT().Teardown(gomock.Any(), existing).Return(nil)
				mockDB.EXPECT().AccountDelete(gomock.Any(), id).Return(tt.deleteErr)
				if tt.deleteErr == nil {
					mockDB.EXPECT().AccountGet(gomock.Any(), id).Return(nil, tt.secondGet)
				}
			}

			res, err := h.Delete(context.Background(), id)
			if err == nil || res != nil {
				t.Errorf("expected error and nil result, got: %v, %v", res, err)
			}
		})
	}
}
