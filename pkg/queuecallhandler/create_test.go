package queuecallhandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
)

func TestCreate(t *testing.T) {
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

	tests := []struct {
		name string

		customerID    uuid.UUID
		queueID       uuid.UUID
		referenceType queuecall.ReferenceType
		referenceID   uuid.UUID

		flowID          uuid.UUID
		forwardActionID uuid.UUID
		exitActionID    uuid.UUID
		confbridgeID    uuid.UUID

		webhookURI    string
		webhookMethod string
		source        cmaddress.Address
		routingMethod queue.RoutingMethod
		tagIDs        []uuid.UUID

		timeoutWait    int
		timeoutService int

		queuecall *queuecall.Queuecall

		expectRes *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("c910ccc8-7f55-11ec-9c6e-a356bdf34421"),
			uuid.FromStringOrNil("9b75a91c-5e5a-11ec-883b-ab05ca15277b"),
			queuecall.ReferenceTypeCall,
			uuid.FromStringOrNil("a875b472-5e5a-11ec-9467-8f2c600000f3"),

			uuid.FromStringOrNil("c9e87138-7699-11ec-aa80-0321af12db91"),
			uuid.FromStringOrNil("a89d0acc-5e5a-11ec-8f3b-274070e9fa26"),
			uuid.FromStringOrNil("a8bd43fa-5e5a-11ec-8e43-236c955d6691"),
			uuid.FromStringOrNil("a8dca420-5e5a-11ec-87e3-eff5c9e3d170"),

			"test.com",
			"POST",
			cmaddress.Address{
				Type:   cmaddress.TypeTel,
				Target: "+821021656521",
			},
			queue.RoutingMethodRandom,
			[]uuid.UUID{
				uuid.FromStringOrNil("a8f7abf8-5e5a-11ec-b03a-0f722823a0ca"),
			},
			100000,
			1000000,

			&queuecall.Queuecall{
				WebhookURI: "test.com",
				Source:     cmaddress.Address{},
				TagIDs:     []uuid.UUID{},
			},

			&queuecall.Queuecall{
				WebhookURI: "test.com",
				Source:     cmaddress.Address{},
				TagIDs:     []uuid.UUID{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().QueuecallCreate(gomock.Any(), gomock.Any()).Return(nil)
			mockDB.EXPECT().QueuecallGet(gomock.Any(), gomock.Any()).Return(tt.queuecall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), queuecall.EventTypeQueuecallCreated, tt.queuecall.WebhookURI, tt.queuecall)

			res, err := h.Create(
				ctx,
				tt.customerID,
				tt.queueID,
				tt.referenceType,
				tt.referenceID,
				tt.flowID,
				tt.forwardActionID,
				tt.exitActionID,
				tt.confbridgeID,
				tt.webhookURI,
				tt.webhookMethod,
				tt.source,
				tt.routingMethod,
				tt.tagIDs,
				tt.timeoutWait,
				tt.timeoutService,
			)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
