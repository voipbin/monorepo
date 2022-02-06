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

func TestProcessV1StreamingsPost(t *testing.T) {
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

		customerID     uuid.UUID
		referenceID    uuid.UUID
		transcribeType transcribe.Type
		language       string

		request       *rabbitmqhandler.Request
		transcribeRes *transcribe.Transcribe

		expectRes *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("45afd578-7ffe-11ec-9430-3bdf65368563"),
			uuid.FromStringOrNil("02c7a132-0be1-11ec-ba15-ebb66c983fba"),
			"call",
			"en-US",

			&rabbitmqhandler.Request{
				URI:      "/v1/streamings",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"reference_id":"02c7a132-0be1-11ec-ba15-ebb66c983fba","customer_id":"45afd578-7ffe-11ec-9430-3bdf65368563","type":"call","language":"en-US"}`),
			},
			&transcribe.Transcribe{
				ID:          uuid.FromStringOrNil("5f8cbc4e-0be2-11ec-8cf2-7f5531d8f428"),
				CustomerID:  uuid.FromStringOrNil("45afd578-7ffe-11ec-9430-3bdf65368563"),
				Type:        transcribe.TypeCall,
				ReferenceID: uuid.FromStringOrNil("02c7a132-0be1-11ec-ba15-ebb66c983fba"),
				HostID:      uuid.FromStringOrNil("3a02f50c-0be6-11ec-9fa7-8792d3dfbd60"),
				Language:    "en-US",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"5f8cbc4e-0be2-11ec-8cf2-7f5531d8f428","customer_id":"45afd578-7ffe-11ec-9430-3bdf65368563","type":"call","reference_id":"02c7a132-0be1-11ec-ba15-ebb66c983fba","host_id":"3a02f50c-0be6-11ec-9fa7-8792d3dfbd60","language":"en-US","direction":"","transcripts":null,"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockTranscribe.EXPECT().StreamingTranscribeStart(gomock.Any(), tt.customerID, tt.referenceID, tt.transcribeType, tt.language).Return(tt.transcribeRes, nil)

			// mockTranscribe.EXPECT().Recording(tt.referenceID, tt.language).Return(tt.transcribe, nil)
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
