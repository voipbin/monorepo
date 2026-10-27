package groupcallhandler

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/dbhandler"
)

func Test_Delete_cov2(t *testing.T) {
	id := uuid.FromStringOrNil("87141692-f0c6-11ee-966d-5b3abc616460")
	deletedAt := time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC)

	tests := []struct {
		name string

		first     *groupcall.Groupcall
		getErr    error
		statusErr error
		delErr    error

		expectStatus bool
		expectDelete bool
		expectErr    string
		expectRes    *groupcall.Groupcall
	}{
		{name: "get error", getErr: errors.New("g"), expectErr: "Could not get groupcall.: g"},
		{
			name:      "already deleted",
			first:     &groupcall.Groupcall{Identity: commonidentity.Identity{ID: id}, Status: groupcall.StatusHangup, TMDelete: &deletedAt},
			expectRes: &groupcall.Groupcall{Identity: commonidentity.Identity{ID: id}, Status: groupcall.StatusHangup, TMDelete: &deletedAt},
		},
		{
			name:         "hangingup error",
			first:        &groupcall.Groupcall{Identity: commonidentity.Identity{ID: id}, Status: groupcall.StatusProgressing},
			statusErr:    errors.New("s"),
			expectStatus: true,
			expectErr:    "could not update the groupcall status to hangingup",
		},
		{
			name:         "db delete error",
			first:        &groupcall.Groupcall{Identity: commonidentity.Identity{ID: id}, Status: groupcall.StatusHangup},
			delErr:       errors.New("d"),
			expectDelete: true,
			expectErr:    "Could not delete the groupcall id.: d",
		},
		{
			name:         "progressing groupcall is hung up first",
			first:        &groupcall.Groupcall{Identity: commonidentity.Identity{ID: id}, Status: groupcall.StatusProgressing},
			expectStatus: true,
			expectDelete: true,
			expectRes:    &groupcall.Groupcall{Identity: commonidentity.Identity{ID: id}, Status: groupcall.StatusHangingup},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &groupcallHandler{reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify}
			ctx := context.Background()

			if tt.getErr != nil {
				mockDB.EXPECT().GroupcallGet(ctx, id).Return(nil, tt.getErr)
			} else {
				mockDB.EXPECT().GroupcallGet(ctx, id).Return(tt.first, nil)
			}
			if tt.expectStatus {
				mockDB.EXPECT().GroupcallSetStatus(ctx, id, groupcall.StatusHangingup).Return(tt.statusErr)
				if tt.statusErr == nil {
					mockDB.EXPECT().GroupcallGet(ctx, id).Return(&groupcall.Groupcall{Identity: commonidentity.Identity{ID: id}, Status: groupcall.StatusHangingup}, nil)
				}
			}
			if tt.expectDelete {
				mockDB.EXPECT().GroupcallDelete(ctx, id).Return(tt.delErr)
				if tt.delErr == nil {
					deleted := &groupcall.Groupcall{Identity: commonidentity.Identity{ID: id}, Status: groupcall.StatusHangingup}
					mockDB.EXPECT().GroupcallGet(ctx, id).Return(deleted, nil)
					mockNotify.EXPECT().PublishWebhookEvent(ctx, deleted.CustomerID, groupcall.EventTypeGroupcallDeleted, deleted)
				}
			}

			res, err := h.Delete(ctx, id)
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) || res != nil {
					t.Errorf("Wrong match. expect: %q, got: %v / %v", tt.expectErr, res, err)
				}
				return
			}
			if err != nil || !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v, err: %v", tt.expectRes, res, err)
			}
		})
	}
}

func Test_Delete_cov2_getNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	h := &groupcallHandler{db: mockDB}
	ctx := context.Background()
	id := uuid.FromStringOrNil("87141692-f0c6-11ee-966d-5b3abc616460")

	mockDB.EXPECT().GroupcallGet(ctx, id).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Delete(ctx, id)
	if res != nil || err == nil || !errors.Is(err, dbhandler.ErrNotFound) || !strings.Contains(err.Error(), "group call was not found") {
		t.Errorf("Wrong match. got: %v / %v", res, err)
	}
}
