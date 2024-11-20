package groupcallhandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-call-manager/models/groupcall"
	"monorepo/bin-call-manager/pkg/dbhandler"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseGroupcall *groupcall.Groupcall

		expectRes *groupcall.Groupcall
	}{
		{
			"normal",

			uuid.FromStringOrNil("87141692-f0c6-11ee-966d-5b3abc616460"),

			&groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("87141692-f0c6-11ee-966d-5b3abc616460"),
				},
				Status:   groupcall.StatusHangup,
				TMDelete: dbhandler.DefaultTimeStamp,
			},

			&groupcall.Groupcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("87141692-f0c6-11ee-966d-5b3abc616460"),
				},
				Status:   groupcall.StatusHangup,
				TMDelete: dbhandler.DefaultTimeStamp,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &groupcallHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)

			// dbDelete
			mockDB.EXPECT().GroupcallDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().GroupcallGet(ctx, tt.id).Return(tt.responseGroupcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseGroupcall.CustomerID, groupcall.EventTypeGroupcallDeleted, tt.responseGroupcall)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

		})
	}
}
