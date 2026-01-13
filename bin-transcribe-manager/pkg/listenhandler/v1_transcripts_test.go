package listenhandler

import (
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/transcribehandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

func Test_processV1TranscriptsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *sock.Request

		pageSize  uint64
		pageToken string

		responseFilters map[string]string
		expectFilters   map[transcript.Field]any

		responseTranscripts []*transcript.Transcript
		expectRes           *sock.Response
	}{
		{
			name: "normal",
			request: &sock.Request{
				URI:    "/v1/transcripts?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_transcribe_id=4f08e520-821d-11ed-844e-67fdcb950f6f",
				Method: sock.RequestMethodGet,
			},

			pageSize:  10,
			pageToken: "2020-05-03 21:35:02.809",

			responseFilters: map[string]string{
				"transcribe_id": "4f08e520-821d-11ed-844e-67fdcb950f6f",
			},
			expectFilters: map[transcript.Field]any{
				transcript.FieldTranscribeID: uuid.FromStringOrNil("4f08e520-821d-11ed-844e-67fdcb950f6f"),
			},
			responseTranscripts: []*transcript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("2afd749c-821e-11ed-9ba2-271e7b9600a1"),
					},
				},
			},

			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"2afd749c-821e-11ed-9ba2-271e7b9600a1","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":"","tm_create":"","tm_delete":""}]`),
			},
		},
		{
			name: "2 items",
			request: &sock.Request{
				URI:    "/v1/transcripts?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_transcribe_id=43b608e6-821e-11ed-9611-e329cba76cc9",
				Method: sock.RequestMethodGet,
			},

			pageSize:  10,
			pageToken: "2020-05-03 21:35:02.809",

			responseFilters: map[string]string{
				"transcribe_id": "43b608e6-821e-11ed-9611-e329cba76cc9",
			},
			expectFilters: map[transcript.Field]any{
				transcript.FieldTranscribeID: uuid.FromStringOrNil("43b608e6-821e-11ed-9611-e329cba76cc9"),
			},
			responseTranscripts: []*transcript.Transcript{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("43e2dae2-821e-11ed-8cb9-ff5d144f9d22"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("440fc214-821e-11ed-b83d-2f241266f784"),
					},
				},
			},
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"43e2dae2-821e-11ed-8cb9-ff5d144f9d22","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":"","tm_create":"","tm_delete":""},{"id":"440fc214-821e-11ed-b83d-2f241266f784","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":"","tm_create":"","tm_delete":""}]`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

			h := &listenHandler{
				sockHandler:       mockSock,
				utilHandler:       mockUtil,
				transcribeHandler: mockTranscribe,
				transcriptHandler: mockTranscript,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockTranscript.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, tt.expectFilters).Return(tt.responseTranscripts, nil)
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
