package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/stt-manager.git/pkg/stthandler"
)

func TestProcessV1CallRecordingsPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockSTT := stthandler.NewMockSTTHandler(mc)

	h := &listenHandler{
		rabbitSock: mockSock,
		reqHandler: mockReq,
		sttHandler: mockSTT,
	}

	type test struct {
		name          string
		sttType       string
		referenceID   uuid.UUID
		language      string
		webhookURI    string
		webhookMethod string
		request       *rabbitmqhandler.Request
		expectRes     *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",
			"recording",
			uuid.FromStringOrNil("e3677524-9b38-11eb-8acb-c3e7da47b5e6"),
			"en-US",
			"http://test.com/webhook",
			"POST",
			&rabbitmqhandler.Request{
				URI:      "/v1/call_recordings",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"reference_id":"e3677524-9b38-11eb-8acb-c3e7da47b5e6","language":"en-US","webhook_uri":"http://test.com/webhook","webhook_method":"POST"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSTT.EXPECT().CallRecording(tt.referenceID, tt.language, tt.webhookURI, tt.webhookMethod).Return(nil)
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
