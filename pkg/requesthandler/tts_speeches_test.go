package requesthandler

import (
	"testing"

	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func TestTTSSpeechesPOST(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:           mockSock,
		exchangeDelay:  "bin-manager.delay",
		queueCall:      "bin-manager.call-manager.request",
		queueFlow:      "bin-manager.flow-manager.request",
		queueTTS:       "bin-manager.tts-manager.request",
		queueRegistrar: "bin-manager.registrar-manager.request",
	}

	type test struct {
		name     string
		text     string
		gender   string
		language string

		response *rabbitmqhandler.Response

		expectRequest *rabbitmqhandler.Request
		expectURL     string
	}

	tests := []test{
		{
			"normal",
			"hello world",
			"male",
			"en-US",
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"url": "https://test.wav"}`),
			},

			&rabbitmqhandler.Request{
				URI:      "/v1/speeches",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"text":"hello world","gender":"male","language":"en-US"}`),
			},
			"https://test.wav",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mockSock.EXPECT().PublishRPC(gomock.Any(), "bin-manager.tts-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.TTSSpeechesPOST(tt.text, tt.gender, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res != tt.expectURL {
				t.Errorf("Wrong match. expect: ok, got: %v", res)
			}
		})
	}
}
