package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/transcribehandler"
)

func TestProcessV1TranscribesIDDelete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)

	h := &listenHandler{
		rabbitSock:        mockSock,
		reqHandler:        mockReq,
		transcribeHandler: mockTranscribe,
	}

	tests := []struct {
		name string

		id         uuid.UUID
		customerID uuid.UUID

		request            *rabbitmqhandler.Request
		responseTranscribe *transcribe.Transcribe

		expectRes *rabbitmqhandler.Response
	}{
		{
			"normal",

			uuid.FromStringOrNil("a4f388dc-86ab-11ec-8d14-9bd962288757"),
			uuid.FromStringOrNil("45afd578-7ffe-11ec-9430-3bdf65368563"),

			&rabbitmqhandler.Request{
				URI:    "/v1/transcribes/a4f388dc-86ab-11ec-8d14-9bd962288757",
				Method: rabbitmqhandler.RequestMethodDelete,
			},
			&transcribe.Transcribe{
				ID: uuid.FromStringOrNil("a4f388dc-86ab-11ec-8d14-9bd962288757"),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a4f388dc-86ab-11ec-8d14-9bd962288757","customer_id":"00000000-0000-0000-0000-000000000000","type":"","reference_id":"00000000-0000-0000-0000-000000000000","host_id":"00000000-0000-0000-0000-000000000000","language":"","direction":"","transcripts":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockTranscribe.EXPECT().Delete(gomock.Any(), tt.id).Return(tt.responseTranscribe, nil)

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
