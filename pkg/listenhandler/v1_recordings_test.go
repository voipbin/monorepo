package listenhandler

import (
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
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
		name string

		customerID  uuid.UUID
		referenceID uuid.UUID
		language    string
		transcribe  *transcribe.Transcribe
		request     *rabbitmqhandler.Request
		expectRes   *rabbitmqhandler.Response
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("3cecdf62-86a7-11ec-96b9-8f0113e14192"),
			uuid.FromStringOrNil("8c91343c-999f-11eb-a3f8-df8a947fe87e"),
			"en-US",
			&transcribe.Transcribe{
				ID:          uuid.FromStringOrNil("29254ec4-a32c-11eb-9123-eb204908f78c"),
				Type:        transcribe.TypeRecording,
				ReferenceID: uuid.FromStringOrNil("8c91343c-999f-11eb-a3f8-df8a947fe87e"),
				Language:    "en-US",
				Transcripts: []transcript.Transcript{
					{
						Message: "hello",
					},
				},
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id":"3cecdf62-86a7-11ec-96b9-8f0113e14192","reference_id":"8c91343c-999f-11eb-a3f8-df8a947fe87e","language":"en-US"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"29254ec4-a32c-11eb-9123-eb204908f78c","customer_id":"00000000-0000-0000-0000-000000000000","type":"recording","reference_id":"8c91343c-999f-11eb-a3f8-df8a947fe87e","host_id":"00000000-0000-0000-0000-000000000000","language":"en-US","direction":"","transcripts":[{"id":"00000000-0000-0000-0000-000000000000","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"hello","tm_create":""}],"tm_create":"","tm_update":"","tm_delete":""}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockTranscribe.EXPECT().Recording(gomock.Any(), tt.customerID, tt.referenceID, tt.language).Return(tt.transcribe, nil)
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
