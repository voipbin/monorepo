package listenhandler

import (
	reflect "reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-queue-manager/models/queuecall"
	"monorepo/bin-queue-manager/models/service"
	"monorepo/bin-queue-manager/pkg/queuecallhandler"
)

func Test_processV1ServicesTypeQueuecallPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		queueID       uuid.UUID
		activeflowID  uuid.UUID
		referenceType queuecall.ReferenceType
		referenceID   uuid.UUID
		exitActionID  uuid.UUID

		responseService *service.Service

		expectRes *sock.Response
	}{
		{
			"normal",
			&sock.Request{
				URI:      "/v1/services/type/queuecall",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"queue_id":"61ff907c-acfa-11ed-978c-f76de62bf9f4","activeflow_id":"622931d4-acfa-11ed-9689-7b028764e072","reference_type":"call","reference_id":"624fe626-acfa-11ed-a358-0b881bcb40b0","exit_action_id":"62739c88-acfa-11ed-b338-67d80143d77e"}`),
			},

			uuid.FromStringOrNil("61ff907c-acfa-11ed-978c-f76de62bf9f4"),
			uuid.FromStringOrNil("622931d4-acfa-11ed-9689-7b028764e072"),
			queuecall.ReferenceTypeCall,
			uuid.FromStringOrNil("624fe626-acfa-11ed-a358-0b881bcb40b0"),
			uuid.FromStringOrNil("62739c88-acfa-11ed-b338-67d80143d77e"),

			&service.Service{
				ID: uuid.FromStringOrNil("6299086a-acfa-11ed-a8ff-4f23e0ae71fd"),
			},

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"6299086a-acfa-11ed-a8ff-4f23e0ae71fd","type":"","push_actions":null}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockQueuecall := queuecallhandler.NewMockQueuecallHandler(mc)

			h := &listenHandler{
				rabbitSock:       mockSock,
				queuecallHandler: mockQueuecall,
			}

			mockQueuecall.EXPECT().ServiceStart(
				gomock.Any(),
				tt.queueID,
				tt.activeflowID,
				tt.referenceType,
				tt.referenceID,
				tt.exitActionID,
			).Return(tt.responseService, nil)

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
