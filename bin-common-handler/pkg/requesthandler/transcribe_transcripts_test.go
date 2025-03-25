package requesthandler

import (
	"context"
	"reflect"
	"testing"

	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_TranscribeV1TranscriptGets(t *testing.T) {

	type test struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		response *sock.Response

		expectedURL     string
		expectedTarget  string
		expectedRequest *sock.Request
		expectedRes     []tmtranscript.Transcript
	}

	tests := []test{
		{
			name: "normal",

			pageToken: "2020-09-20T03:23:20.995000",
			pageSize:  10,
			filters: map[string]string{
				"transcribe_id": "8fe05f90-8229-11ed-a215-a78ed418d1c0",
			},

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"9021680a-8229-11ed-a360-0792bc711080"}]`),
			},

			expectedURL:    "/v1/transcripts?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			expectedTarget: "bin-manager.transcribe-manager.request",
			expectedRequest: &sock.Request{
				URI:      "/v1/transcripts?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_transcribe_id=8fe05f90-8229-11ed-a215-a78ed418d1c0",
				Method:   sock.RequestMethodGet,
				DataType: ContentTypeJSON,
			},
			expectedRes: []tmtranscript.Transcript{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("9021680a-8229-11ed-a360-0792bc711080"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			h := requestHandler{
				sock:        mockSock,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockUtil.EXPECT().URLMergeFilters(tt.expectedURL, tt.filters).Return(utilhandler.URLMergeFilters(tt.expectedURL, tt.filters))
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectedTarget, tt.expectedRequest).Return(tt.response, nil)

			res, err := h.TranscribeV1TranscriptGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectedRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectedRes, res)
			}
		})
	}
}
