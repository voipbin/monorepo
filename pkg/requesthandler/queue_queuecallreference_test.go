package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	qmqueuecallreference "gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecallreference"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_AMV1QueuecallReferenceGet(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *qmqueuecallreference.QueuecallReference
	}{
		{
			"normal",

			uuid.FromStringOrNil("a2764422-6159-11ec-8d87-975236f7d7b7"),

			"bin-manager.queue-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/queuecallreferences/a2764422-6159-11ec-8d87-975236f7d7b7",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a2764422-6159-11ec-8d87-975236f7d7b7"}`),
			},
			&qmqueuecallreference.QueuecallReference{
				ID: uuid.FromStringOrNil("a2764422-6159-11ec-8d87-975236f7d7b7"),
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.QMV1QueuecallReferenceGet(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
