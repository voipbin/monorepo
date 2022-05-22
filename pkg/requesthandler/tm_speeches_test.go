package requesthandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_TMV1SpeecheCreate(t *testing.T) {

	tests := []struct {
		name string

		callID   uuid.UUID
		text     string
		gender   string
		language string
		timeout  int

		response *rabbitmqhandler.Response

		expectRequest *rabbitmqhandler.Request
		expectURL     string
	}{
		{
			"normal",

			uuid.FromStringOrNil("cf0413d8-921a-11ec-96ed-7f0948b70d4e"),
			"hello world",
			"male",
			"en-US",
			3000,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"filename": "tts/tmp_filename.wav"}`),
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/speeches",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"call_id":"cf0413d8-921a-11ec-96ed-7f0948b70d4e","text":"hello world","gender":"male","language":"en-US"}`),
			},
			"tts/tmp_filename.wav",
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

			mockSock.EXPECT().PublishRPC(gomock.Any(), "bin-manager.tts-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.TMV1SpeecheCreate(context.Background(), tt.callID, tt.text, tt.gender, tt.language, tt.timeout)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectURL {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectURL, res)
			}
		})
	}
}
