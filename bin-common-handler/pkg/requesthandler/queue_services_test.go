package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/service"
	qmqueuecall "monorepo/bin-queue-manager/models/queuecall"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_QueueV1ServiceTypeQueuecallStart(t *testing.T) {

	tests := []struct {
		name string

		queueID       uuid.UUID
		activeflowID  uuid.UUID
		referenceType qmqueuecall.ReferenceType
		referenceID   uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRes     *service.Service
	}{
		{
			name: "normal",

			queueID:       uuid.FromStringOrNil("c1f0da68-acff-11ed-8d64-a311f65c0693"),
			activeflowID:  uuid.FromStringOrNil("c216db46-acff-11ed-b9df-5b096e3173b4"),
			referenceType: qmqueuecall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("c246902a-acff-11ed-bf99-3f8e11dad28d"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"c2a26184-acff-11ed-a0ae-236e54ac3818"}`),
			},

			expectTarget: "bin-manager.queue-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/services/type/queuecall",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"queue_id":"c1f0da68-acff-11ed-8d64-a311f65c0693","activeflow_id":"c216db46-acff-11ed-b9df-5b096e3173b4","reference_type":"call","reference_id":"c246902a-acff-11ed-bf99-3f8e11dad28d"}`),
			},
			expectRes: &service.Service{
				ID: uuid.FromStringOrNil("c2a26184-acff-11ed-a0ae-236e54ac3818"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QueueV1ServiceTypeQueuecallStart(ctx, tt.queueID, tt.activeflowID, tt.referenceType, tt.referenceID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
