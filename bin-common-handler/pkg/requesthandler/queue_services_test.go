package requesthandler

import (
	"context"
	"reflect"
	"testing"

	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"
	qmservice "monorepo/bin-queue-manager/models/service"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_QueueV1ServiceTypeQueuecallStart(t *testing.T) {

	tests := []struct {
		name string

		queueID       uuid.UUID
		activeflowID  uuid.UUID
		referenceType qmqueuecall.ReferenceType
		referenceID   uuid.UUID
		exitActionID  uuid.UUID

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *qmservice.Service
	}{
		{
			"normal",

			uuid.FromStringOrNil("c1f0da68-acff-11ed-8d64-a311f65c0693"),
			uuid.FromStringOrNil("c216db46-acff-11ed-b9df-5b096e3173b4"),
			qmqueuecall.ReferenceTypeCall,
			uuid.FromStringOrNil("c246902a-acff-11ed-bf99-3f8e11dad28d"),
			uuid.FromStringOrNil("c273699c-acff-11ed-9a0a-e7ea36c32ff0"),

			"bin-manager.queue-manager.request",
			&sock.Request{
				URI:      "/v1/services/type/queuecall",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"queue_id":"c1f0da68-acff-11ed-8d64-a311f65c0693","activeflow_id":"c216db46-acff-11ed-b9df-5b096e3173b4","reference_type":"call","reference_id":"c246902a-acff-11ed-bf99-3f8e11dad28d","exit_action_id":"c273699c-acff-11ed-9a0a-e7ea36c32ff0"}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c2a26184-acff-11ed-a0ae-236e54ac3818"}`),
			},
			&qmservice.Service{
				ID: uuid.FromStringOrNil("c2a26184-acff-11ed-a0ae-236e54ac3818"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QueueV1ServiceTypeQueuecallStart(ctx, tt.queueID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.exitActionID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
