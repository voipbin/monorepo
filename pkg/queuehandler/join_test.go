package queuehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
	cmcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/call"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queue"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallreferencehandler"
)

func TestJoin(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)
	mockQueuecallReference := queuecallreferencehandler.NewMockQueuecallReferenceHandler(mc)

	h := &queueHandler{
		db:                        mockDB,
		reqHandler:                mockReq,
		notifyhandler:             mockNotify,
		queuecallHandler:          mockQueuecall,
		queuecallReferenceHandler: mockQueuecallReference,
	}

	tests := []struct {
		name string

		queueID       uuid.UUID
		referenceType queuecall.ReferenceType
		referenceID   uuid.UUID
		exitActionID  uuid.UUID

		queue     *queue.Queue
		call      *cmcall.Call
		queuecall *queuecall.Queuecall

		expectRes *queuecall.Queuecall
	}{
		{
			"normal",

			uuid.FromStringOrNil("8e8c729e-60e9-11ec-ae8e-130047a0c46f"),
			queuecall.ReferenceTypeCall,
			uuid.FromStringOrNil("8efbb17c-60e9-11ec-8d51-2f2d74388fff"),
			uuid.FromStringOrNil("8f29be1e-60e9-11ec-8032-977e00b8b523"),

			&queue.Queue{
				ID:              uuid.FromStringOrNil("8e8c729e-60e9-11ec-ae8e-130047a0c46f"),
				UserID:          1,
				ForwardActionID: uuid.FromStringOrNil("1fb424a0-60eb-11ec-aae3-731297df86c1"),
				ConfbridgeID:    uuid.FromStringOrNil("2b7daf7c-60eb-11ec-9d2e-ebec25af18e8"),
			},
			&cmcall.Call{
				ID:        uuid.FromStringOrNil("8efbb17c-60e9-11ec-8d51-2f2d74388fff"),
				Direction: cmcall.DirectionIncoming,
				Source: cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821021656521",
				},
			},
			&queuecall.Queuecall{
				ID:      uuid.FromStringOrNil("20e2616a-60ec-11ec-912a-978318aa1f5e"),
				QueueID: uuid.FromStringOrNil("8e8c729e-60e9-11ec-ae8e-130047a0c46f"),
			},

			&queuecall.Queuecall{
				ID:      uuid.FromStringOrNil("20e2616a-60ec-11ec-912a-978318aa1f5e"),
				QueueID: uuid.FromStringOrNil("8e8c729e-60e9-11ec-ae8e-130047a0c46f"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			mockDB.EXPECT().QueueGet(gomock.Any(), tt.queueID).Return(tt.queue, nil)
			mockReq.EXPECT().CMV1CallGet(gomock.Any(), tt.referenceID).Return(tt.call, nil)

			var source cmaddress.Address
			if tt.call.Direction == cmcall.DirectionIncoming {
				source = tt.call.Source
			} else {
				source = tt.call.Destination
			}

			mockQueuecall.EXPECT().Create(
				gomock.Any(),
				tt.queue.UserID,
				tt.queue.ID,
				tt.referenceType,
				tt.referenceID,
				tt.queue.ForwardActionID,
				tt.exitActionID,
				tt.queue.ConfbridgeID,
				tt.queue.WebhookURI,
				tt.queue.WebhookMethod,
				source,
				tt.queue.RoutingMethod,
				tt.queue.TagIDs,
				tt.queue.WaitTimeout,
				tt.queue.ServiceTimeout,
			).Return(tt.queuecall, nil)

			mockQueuecallReference.EXPECT().SetCurrentQueuecallID(gomock.Any(), tt.referenceID, tt.referenceType, tt.queuecall.ID).Return(nil)
			mockDB.EXPECT().QueueAddQueueCallID(gomock.Any(), tt.queuecall.QueueID, tt.queuecall.ID).Return(nil)

			res, err := h.Join(ctx, tt.queueID, tt.referenceType, tt.referenceID, tt.exitActionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func TestGetSource(t *testing.T) {

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)
	mockQueuecallReference := queuecallreferencehandler.NewMockQueuecallReferenceHandler(mc)

	h := &queueHandler{
		db:                        mockDB,
		reqHandler:                mockReq,
		notifyhandler:             mockNotify,
		queuecallHandler:          mockQueuecall,
		queuecallReferenceHandler: mockQueuecallReference,
	}

	tests := []struct {
		name string

		call *cmcall.Call

		expectRes cmaddress.Address
	}{
		{
			"incoming tel type",

			&cmcall.Call{
				ID:        uuid.FromStringOrNil("8efbb17c-60e9-11ec-8d51-2f2d74388fff"),
				Direction: cmcall.DirectionIncoming,
				Source: cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821021656522",
				},
			},
			cmaddress.Address{
				Type:       cmaddress.TypeTel,
				Target:     "+821021656522",
				TargetName: "",
				Name:       "",
				Detail:     "",
			},
		},
		{
			"incoming sip type",

			&cmcall.Call{
				ID:        uuid.FromStringOrNil("8efbb17c-60e9-11ec-8d51-2f2d74388fff"),
				Direction: cmcall.DirectionIncoming,
				Source: cmaddress.Address{
					Type:   cmaddress.TypeSIP,
					Target: "test@voipbin.net",
				},
			},
			cmaddress.Address{
				Type:       defaultSourceType,
				Target:     defaultSourceTarget,
				TargetName: "",
				Name:       "",
				Detail:     "",
			},
		},
		{
			"outgoing tel type",

			&cmcall.Call{
				ID:        uuid.FromStringOrNil("8efbb17c-60e9-11ec-8d51-2f2d74388fff"),
				Direction: cmcall.DirectionOutgoing,
				Destination: cmaddress.Address{
					Type:   cmaddress.TypeTel,
					Target: "+821021656522",
				},
			},
			cmaddress.Address{
				Type:       cmaddress.TypeTel,
				Target:     "+821021656522",
				TargetName: "",
				Name:       "",
				Detail:     "",
			},
		},
		{
			"outgoing sip type",

			&cmcall.Call{
				ID:        uuid.FromStringOrNil("8efbb17c-60e9-11ec-8d51-2f2d74388fff"),
				Direction: cmcall.DirectionOutgoing,
				Destination: cmaddress.Address{
					Type:   cmaddress.TypeSIP,
					Target: "test@voipbin.net",
				},
			},
			cmaddress.Address{
				Type:       defaultSourceType,
				Target:     defaultSourceTarget,
				TargetName: "",
				Name:       "",
				Detail:     "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			res := h.getSource(tt.call)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}

}
