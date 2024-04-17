package requesthandler

import (
	"context"
	"reflect"
	"testing"

	tmtts "monorepo/bin-tts-manager/models/tts"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
)

func Test_TTSV1SpeecheCreate(t *testing.T) {

	tests := []struct {
		name string

		callID   uuid.UUID
		text     string
		gender   tmtts.Gender
		language string
		timeout  int

		response *rabbitmqhandler.Response

		expectRequest *rabbitmqhandler.Request
		expectURL     string
		expectRes     *tmtts.TTS
	}{
		{
			"normal",

			uuid.FromStringOrNil("cf0413d8-921a-11ec-96ed-7f0948b70d4e"),
			"hello world",
			tmtts.GenderMale,
			"en-US",
			3000,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"gender":"male","text":"hello world","language":"en-US","media_filepath":"tts/tmp_filename.wav"}`),
			},
			&rabbitmqhandler.Request{
				URI:      "/v1/speeches",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"call_id":"cf0413d8-921a-11ec-96ed-7f0948b70d4e","text":"hello world","gender":"male","language":"en-US"}`),
			},
			"tts/tmp_filename.wav",

			&tmtts.TTS{
				Gender:        tmtts.GenderMale,
				Text:          "hello world",
				Language:      "en-US",
				MediaFilepath: "tts/tmp_filename.wav",
			},
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

			res, err := reqHandler.TTSV1SpeecheCreate(context.Background(), tt.callID, tt.text, tt.gender, tt.language, tt.timeout)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}
