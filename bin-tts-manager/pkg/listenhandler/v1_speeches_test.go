package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-tts-manager/models/tts"
	"monorepo/bin-tts-manager/pkg/ttshandler"
)

func Test_v1SpeechesPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseTTS *tts.TTS

		expectCallID   uuid.UUID
		expectText     string
		expectLanguage string
		expectProvider tts.Provider
		expectVoiceID  string
		expectRes      *sock.Response
	}{
		{
			name: "normal test",

			request: &sock.Request{
				URI:    "/v1/speeches",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"call_id": "107d1f0e-9665-11ed-b3f3-039937430300", "text": "hello world", "language": "en-US"}`),
			},

			responseTTS: &tts.TTS{
				Text:            "hello world",
				Language:        "en-US",
				MediaBucketName: "voipbin-tmp-bucket-europe-west4",
				MediaFilepath:   "temp/tts/11271770-9665-11ed-ba40-bf3763460bd6.wav",
			},

			expectCallID:   uuid.FromStringOrNil("107d1f0e-9665-11ed-b3f3-039937430300"),
			expectText:     "hello world",
			expectLanguage: "en-US",
			expectProvider: "",
			expectVoiceID:  "",
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"text":"hello world","language":"en-US","media_bucket_name":"voipbin-tmp-bucket-europe-west4","media_filepath":"temp/tts/11271770-9665-11ed-ba40-bf3763460bd6.wav"}`),
			},
		},
		{
			name: "with provider and voice_id",

			request: &sock.Request{
				URI:    "/v1/speeches",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"call_id": "107d1f0e-9665-11ed-b3f3-039937430300", "text": "hello world", "language": "en-US", "provider": "gcp", "voice_id": "en-US-Wavenet-D"}`),
			},

			responseTTS: &tts.TTS{
				Provider:        tts.ProviderGCP,
				VoiceID:         "en-US-Wavenet-D",
				Text:            "hello world",
				Language:        "en-US",
				MediaBucketName: "voipbin-tmp-bucket-europe-west4",
				MediaFilepath:   "temp/tts/11271770-9665-11ed-ba40-bf3763460bd6.wav",
			},

			expectCallID:   uuid.FromStringOrNil("107d1f0e-9665-11ed-b3f3-039937430300"),
			expectText:     "hello world",
			expectLanguage: "en-US",
			expectProvider: tts.ProviderGCP,
			expectVoiceID:  "en-US-Wavenet-D",
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"provider":"gcp","voice_id":"en-US-Wavenet-D","text":"hello world","language":"en-US","media_bucket_name":"voipbin-tmp-bucket-europe-west4","media_filepath":"temp/tts/11271770-9665-11ed-ba40-bf3763460bd6.wav"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTTS := ttshandler.NewMockTTSHandler(mc)

			h := &listenHandler{
				sockHandler: mockSock,
				ttsHandler:  mockTTS,
			}

			mockTTS.EXPECT().Create(gomock.Any(), tt.expectCallID, tt.expectText, tt.expectLanguage, tt.expectProvider, tt.expectVoiceID).Return(tt.responseTTS, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}
