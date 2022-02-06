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

func TestProcessV1CallRecordingsPost(t *testing.T) {
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

	type test struct {
		name string

		customerID    uuid.UUID
		referenceID   uuid.UUID
		language      string
		webhookURI    string
		webhookMethod string
		request       *rabbitmqhandler.Request

		responseRecording []*transcribe.Transcribe
		expectRes         *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("2f5a3352-85c8-11ec-a223-a394e8ca795b"),
			uuid.FromStringOrNil("e3677524-9b38-11eb-8acb-c3e7da47b5e6"),
			"en-US",
			"http://test.com/webhook",
			"POST",
			&rabbitmqhandler.Request{
				URI:      "/v1/call_recordings",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"2f5a3352-85c8-11ec-a223-a394e8ca795b","reference_id":"e3677524-9b38-11eb-8acb-c3e7da47b5e6","language":"en-US","webhook_uri":"http://test.com/webhook","webhook_method":"POST"}`),
			},

			[]*transcribe.Transcribe{
				{
					ID: uuid.FromStringOrNil("ef7a7baa-873b-11ec-be2f-5b793e69214c"),
				},
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ef7a7baa-873b-11ec-be2f-5b793e69214c","customer_id":"00000000-0000-0000-0000-000000000000","type":"","reference_id":"00000000-0000-0000-0000-000000000000","host_id":"00000000-0000-0000-0000-000000000000","language":"","direction":"","transcripts":null,"tm_create":"","tm_update":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockTranscribe.EXPECT().CallRecording(gomock.Any(), tt.customerID, tt.referenceID, tt.language).Return(tt.responseRecording, nil)
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
