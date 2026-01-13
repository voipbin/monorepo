package queuecallhandler

import (
	"context"
	"strconv"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-queue-manager/models/queue"
	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/pkg/dbhandler"
)

func Test_setVariables(t *testing.T) {

	tests := []struct {
		name string

		queue     *queue.Queue
		queuecall *queuecall.Queuecall
	}{
		{
			name: "normal",

			queue: &queue.Queue{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9b75a91c-5e5a-11ec-883b-ab05ca15277b"),
					CustomerID: uuid.FromStringOrNil("c910ccc8-7f55-11ec-9c6e-a356bdf34421"),
				},

				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a8f7abf8-5e5a-11ec-b03a-0f722823a0ca"),
				},

				WaitTimeout:    100000,
				ServiceTimeout: 1000000,
			},
			queuecall: &queuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ec57c480-db57-11ec-bb3e-b36e382aaec6"),
				},
				ReferenceActiveflowID: uuid.FromStringOrNil("48acb876-db58-11ec-a465-3fdb0b80f24c"),

				TimeoutWait:    100000,
				TimeoutService: 1000000,
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

			h := &queuecallHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			variables := map[string]string{
				"voipbin.queue.id":     tt.queue.ID.String(),
				"voipbin.queue.name":   tt.queue.Name,
				"voipbin.queue.detail": tt.queue.Detail,

				"voipbin.queuecall.id":              tt.queuecall.ID.String(),
				"voipbin.queuecall.timeout_wait":    strconv.Itoa(tt.queuecall.TimeoutWait),
				"voipbin.queuecall.timeout_service": strconv.Itoa(tt.queuecall.TimeoutService),
			}

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.queuecall.ReferenceActiveflowID, variables.Return(nil)

			if err := h.setVariables(ctx, tt.queue, tt.queuecall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_deleteVariables(t *testing.T) {

	tests := []struct {
		name string

		queue     *queue.Queue
		queuecall *queuecall.Queuecall
	}{
		{
			name: "normal",

			queue: &queue.Queue{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("9b75a91c-5e5a-11ec-883b-ab05ca15277b"),
					CustomerID: uuid.FromStringOrNil("c910ccc8-7f55-11ec-9c6e-a356bdf34421"),
				},

				RoutingMethod: queue.RoutingMethodRandom,
				TagIDs: []uuid.UUID{
					uuid.FromStringOrNil("a8f7abf8-5e5a-11ec-b03a-0f722823a0ca"),
				},

				WaitTimeout:    100000,
				ServiceTimeout: 1000000,
			},
			queuecall: &queuecall.Queuecall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ec57c480-db57-11ec-bb3e-b36e382aaec6"),
				},
				ReferenceActiveflowID: uuid.FromStringOrNil("360c2e5e-db58-11ec-a454-13204c0093d8"),

				TimeoutWait:    100000,
				TimeoutService: 1000000,
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

			h := &queuecallHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyhandler: mockNotify,
			}

			ctx := context.Background()

			variables := []string{
				"voipbin.queue.id",
				"voipbin.queue.name",
				"voipbin.queue.detail",

				"voipbin.queuecall.id",
				"voipbin.queuecall.timeout_wait",
				"voipbin.queuecall.timeout_service",
			}

			for _, key := range variables {
				mockReq.EXPECT().FlowV1VariableDeleteVariable(ctx, tt.queuecall.ReferenceActiveflowID, key.Return(nil)
			}

			if err := h.deleteVariables(ctx, tt.queuecall); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
