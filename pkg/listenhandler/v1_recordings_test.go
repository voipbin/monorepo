package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/pkg/transcribehandler"
)

func TestProcessV1TranscribesPost(t *testing.T) {
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
		name           string
		transcribeType string
		referenceID    uuid.UUID
		language       string
		webhookURI     string
		webhookMethod  string
		request        *rabbitmqhandler.Request
		expectRes      *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			"recording",
			uuid.FromStringOrNil("8c91343c-999f-11eb-a3f8-df8a947fe87e"),
			"en-US",
			"http://test.com/webhook",
			"POST",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"reference_id":"8c91343c-999f-11eb-a3f8-df8a947fe87e","language":"en-US","webhook_uri":"http://test.com/webhook","webhook_method":"POST"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockTranscribe.EXPECT().Recording(tt.referenceID, tt.language, tt.webhookURI, tt.webhookMethod).Return(nil)
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
