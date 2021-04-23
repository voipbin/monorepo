package requesthandler

import (
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

func TestTMRecordingPost(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := rabbitmqhandler.NewMockRabbit(mc)
	reqHandler := requestHandler{
		sock:                   mockSock,
		exchangeDelay:          "bin-manager.delay",
		queueRequestTranscribe: "bin-manager.transcribe-manager.request",
	}

	type test struct {
		name string

		id       uuid.UUID
		language string

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectResult *tmtranscribe.Transcribe
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("138cbdc2-a3ea-11eb-9a91-3b876395af6e"),
			"en-US",

			"bin-manager.transcribe-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"reference_id":"138cbdc2-a3ea-11eb-9a91-3b876395af6e","language":"en-US"}`),
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"10e438e2-a3eb-11eb-889c-975ac37d96fe","type":"recording","reference_id":"138cbdc2-a3ea-11eb-9a91-3b876395af6e","language":"en-US","webhook_uri":"","webhook_method":"","transcription":"Hello, this is voipbin. Thank you."}`),
			},
			&tmtranscribe.Transcribe{
				ID:            uuid.FromStringOrNil("10e438e2-a3eb-11eb-889c-975ac37d96fe"),
				Type:          tmtranscribe.TypeRecording,
				ReferenceID:   uuid.FromStringOrNil("138cbdc2-a3ea-11eb-9a91-3b876395af6e"),
				Language:      "en-US",
				WebhookURI:    "",
				WebhookMethod: "",
				Transcription: "Hello, this is voipbin. Thank you.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.TMRecordingPost(tt.id, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(tt.expectResult, res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectResult, res)
			}
		})
	}
}
