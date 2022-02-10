package listenhandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"

	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecall"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/models/queuecallreference"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallhandler"
	"gitlab.com/voipbin/bin-manager/queue-manager.git/pkg/queuecallreferencehandler"
)

func TestProcessV1QueuescallreferencesIDGet(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)

	mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)
	mockQueuecallReference := queuecallreferencehandler.NewMockQueuecallReferenceHandler(mc)

	h := &listenHandler{
		rabbitSock:                mockSock,
		queuecallHandler:          mockQueuecall,
		queuecallReferenceHandler: mockQueuecallReference,
	}

	tests := []struct {
		name string

		request *rabbitmqhandler.Request

		queuecallReferenceID uuid.UUID
		queuecallReference   *queuecallreference.QueuecallReference

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:      "/v1/queuecallreferences/74a40c58-60ac-11ec-96c8-03b200835240",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},

			uuid.FromStringOrNil("74a40c58-60ac-11ec-96c8-03b200835240"),
			&queuecallreference.QueuecallReference{
				ID:         uuid.FromStringOrNil("74a40c58-60ac-11ec-96c8-03b200835240"),
				CustomerID: uuid.FromStringOrNil("53f3c696-7ffc-11ec-9184-6b98b2fa94fb"),
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"74a40c58-60ac-11ec-96c8-03b200835240","customer_id":"53f3c696-7ffc-11ec-9184-6b98b2fa94fb","type":"","current_queuecall_id":"00000000-0000-0000-0000-000000000000","queuecall_ids":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockQueuecallReference.EXPECT().Get(gomock.Any(), tt.queuecallReferenceID).Return(tt.queuecallReference, nil)

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

func TestProcessV1QueuescallreferencesIDDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)

	mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

	h := &listenHandler{
		rabbitSock:       mockSock,
		queuecallHandler: mockQueuecall,
	}

	tests := []struct {
		name string

		request *rabbitmqhandler.Request

		queuecallID uuid.UUID

		responseQueuecall *queuecall.Queuecall

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

			&queuecall.Queuecall{
				ID: uuid.FromStringOrNil("74a40c58-60ac-11ec-96c8-03b200835240"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"74a40c58-60ac-11ec-96c8-03b200835240","customer_id":"00000000-0000-0000-0000-000000000000","queue_id":"00000000-0000-0000-0000-000000000000","reference_type":"","reference_id":"00000000-0000-0000-0000-000000000000","flow_id":"00000000-0000-0000-0000-000000000000","forward_action_id":"00000000-0000-0000-0000-000000000000","exit_action_id":"00000000-0000-0000-0000-000000000000","confbridge_id":"00000000-0000-0000-0000-000000000000","source":{"type":"","target":"","target_name":"","name":"","detail":""},"routing_method":"","tag_ids":null,"status":"","service_agent_id":"00000000-0000-0000-0000-000000000000","timeout_wait":0,"timeout_service":0,"tm_create":"","tm_service":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockQueuecall.EXPECT().KickByReferenceID(gomock.Any(), tt.queuecallID).Return(tt.responseQueuecall, nil)

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
