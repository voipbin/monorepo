package requesthandler

import (
	"context"
	"reflect"
	"testing"

	tmtts "monorepo/bin-tts-manager/models/tts"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_TTSV1SpeecheCreate(t *testing.T) {

	tests := []struct {
		name string

		callID   uuid.UUID
		text     string
		language string
		provider tmtts.Provider
		voiceID  string
		timeout  int

		response *sock.Response

		expectRequest *sock.Request
		expectRes     *tmtts.TTS
	}{
		{
			name: "auto provider no voice_id",

			callID:   uuid.FromStringOrNil("cf0413d8-921a-11ec-96ed-7f0948b70d4e"),
			text:     "hello world",
			language: "en-US",
			provider: "",
			voiceID:  "",
			timeout:  3000,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"text":"hello world","language":"en-US","media_filepath":"tts/tmp_filename.wav"}`),
			},
			expectRequest: &sock.Request{
				URI:      "/v1/speeches",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"call_id":"cf0413d8-921a-11ec-96ed-7f0948b70d4e","text":"hello world","language":"en-US"}`),
			},

			expectRes: &tmtts.TTS{
				Text:          "hello world",
				Language:      "en-US",
				MediaFilepath: "tts/tmp_filename.wav",
			},
		},
		{
			name: "gcp provider with voice_id",

			callID:   uuid.FromStringOrNil("cf0413d8-921a-11ec-96ed-7f0948b70d4e"),
			text:     "hello world",
			language: "en-US",
			provider: tmtts.ProviderGCP,
			voiceID:  "en-US-Wavenet-A",
			timeout:  3000,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"provider":"gcp","voice_id":"en-US-Wavenet-A","text":"hello world","language":"en-US","media_filepath":"tts/gcp_filename.wav"}`),
			},
			expectRequest: &sock.Request{
				URI:      "/v1/speeches",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"call_id":"cf0413d8-921a-11ec-96ed-7f0948b70d4e","text":"hello world","language":"en-US","provider":"gcp","voice_id":"en-US-Wavenet-A"}`),
			},

			expectRes: &tmtts.TTS{
				Provider:      tmtts.ProviderGCP,
				VoiceID:       "en-US-Wavenet-A",
				Text:          "hello world",
				Language:      "en-US",
				MediaFilepath: "tts/gcp_filename.wav",
			},
		},
		{
			name: "aws provider no voice_id",

			callID:   uuid.FromStringOrNil("cf0413d8-921a-11ec-96ed-7f0948b70d4e"),
			text:     "hallo welt",
			language: "de-DE",
			provider: tmtts.ProviderAWS,
			voiceID:  "",
			timeout:  3000,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"provider":"aws","text":"hallo welt","language":"de-DE","media_filepath":"tts/aws_filename.wav"}`),
			},
			expectRequest: &sock.Request{
				URI:      "/v1/speeches",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"call_id":"cf0413d8-921a-11ec-96ed-7f0948b70d4e","text":"hallo welt","language":"de-DE","provider":"aws"}`),
			},

			expectRes: &tmtts.TTS{
				Provider:      tmtts.ProviderAWS,
				Text:          "hallo welt",
				Language:      "de-DE",
				MediaFilepath: "tts/aws_filename.wav",
			},
		},
		{
			name: "aws provider with voice_id",

			callID:   uuid.FromStringOrNil("cf0413d8-921a-11ec-96ed-7f0948b70d4e"),
			text:     "hello world",
			language: "en-US",
			provider: tmtts.ProviderAWS,
			voiceID:  "Joanna",
			timeout:  3000,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"provider":"aws","voice_id":"Joanna","text":"hello world","language":"en-US","media_filepath":"tts/aws_voice_filename.wav"}`),
			},
			expectRequest: &sock.Request{
				URI:      "/v1/speeches",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"call_id":"cf0413d8-921a-11ec-96ed-7f0948b70d4e","text":"hello world","language":"en-US","provider":"aws","voice_id":"Joanna"}`),
			},

			expectRes: &tmtts.TTS{
				Provider:      tmtts.ProviderAWS,
				VoiceID:       "Joanna",
				Text:          "hello world",
				Language:      "en-US",
				MediaFilepath: "tts/aws_voice_filename.wav",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			reqHandler := requestHandler{
				sock: mockSock,
			}

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.tts-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.TTSV1SpeecheCreate(context.Background(), tt.callID, tt.text, tt.language, tt.provider, tt.voiceID, tt.timeout)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expectRes, res)
			}
		})
	}
}
