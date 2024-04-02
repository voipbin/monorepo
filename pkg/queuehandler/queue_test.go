package queuehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		queueID uuid.UUID

		responseQueue *queue.Queue
	}{
		{
			"normal",

			uuid.FromStringOrNil("013d3b56-f090-11ee-bbc1-a359f9e014d8"),

			&queue.Queue{
				ID: uuid.FromStringOrNil("013d3b56-f090-11ee-bbc1-a359f9e014d8"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &queueHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().QueueSetExecute(ctx, tt.queueID, queue.ExecuteStop).Return(nil)

			// dbDelete
			mockDB.EXPECT().QueueDelete(ctx, tt.queueID).Return(nil)
			mockDB.EXPECT().QueueGet(ctx, tt.queueID).Return(tt.responseQueue, nil)
			mockNotify.EXPECT().PublishEvent(ctx, queue.EventTypeQueueDeleted, tt.responseQueue)

			res, err := h.Delete(ctx, tt.queueID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.responseQueue, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseQueue, res)
			}
		})
	}
}
