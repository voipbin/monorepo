package requesthandler

import (
	"context"
	"reflect"
	"testing"

	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_TranscribeV1TranscriptGets(t *testing.T) {

	type test struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		expectURL     string
		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response

		expectRes []tmtranscript.Transcript
	}

	tests := []test{
		{
			"normal",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"transcribe_id": "8fe05f90-8229-11ed-a215-a78ed418d1c0",
			},

			"/v1/transcripts?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.transcribe-manager.request",
			&sock.Request{
				URI:      "/v1/transcripts?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_transcribe_id=8fe05f90-8229-11ed-a215-a78ed418d1c0",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			&sock.Response{
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectURL, tt.filters))
			mockSock.EXPECT().PublishRequest(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.TranscribeV1TranscriptGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}
