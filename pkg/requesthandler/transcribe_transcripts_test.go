package requesthandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	tmtranscript "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"

	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/rabbitmqhandler"
)

func Test_TranscribeV1TranscriptGets(t *testing.T) {

	type test struct {
		name string

		transcribeID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response

		expectRes []tmtranscript.Transcript
	}

	tests := []test{
		{
			"normal",

			uuid.FromStringOrNil("8fe05f90-8229-11ed-a215-a78ed418d1c0"),

			"bin-manager.transcribe-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/transcripts?transcribe_id=8fe05f90-8229-11ed-a215-a78ed418d1c0",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"9021680a-8229-11ed-a360-0792bc711080"}]`),
			},
			[]tmtranscript.Transcript{
				{
					ID: uuid.FromStringOrNil("9021680a-8229-11ed-a360-0792bc711080"),
				},
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

			ctx := context.Background()
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.TranscribeV1TranscriptGets(ctx, tt.transcribeID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
