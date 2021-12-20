package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/request-manager.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallhandler"
)

func TestProcessV1QueuescallreferencesIDDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)

	mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

	h := &listenHandler{
		rabbitSock:       mockSock,
		db:               mockDB,
		reqHandler:       mockReq,
		queuecallHandler: mockQueuecall,
	}

	tests := []struct {
		name string

		request *rabbitmqhandler.Request

		queuecallID uuid.UUID

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/queuecallreferences/74a40c58-60ac-11ec-96c8-03b200835240",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("74a40c58-60ac-11ec-96c8-03b200835240"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockQueuecall.EXPECT().KickByReferenceID(gomock.Any(), tt.queuecallID).Return(nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexepct: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
