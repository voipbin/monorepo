package callhandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/pkg/dbhandler"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
)

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseCall *call.Call

		expectRes *call.Call
	}{
		{
			"normal",

			uuid.FromStringOrNil("841c5fa2-f0c2-11ee-834f-53b2b00ec88d"),

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("841c5fa2-f0c2-11ee-834f-53b2b00ec88d"),
				},
				Status:   call.StatusHangup,
				TMDelete: dbhandler.DefaultTimeStamp,
			},

			&call.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("841c5fa2-f0c2-11ee-834f-53b2b00ec88d"),
				},
				Status:   call.StatusHangup,
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

			h := &callHandler{
				reqHandler:    mockReq,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)

			// dbDelete
			mockDB.EXPECT().CallDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().CallGet(ctx, tt.id).Return(tt.responseCall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseCall.CustomerID, call.EventTypeCallDeleted, tt.responseCall)

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
