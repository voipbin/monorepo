package listenhandler

import (
	"reflect"
	"testing"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-transcribe-manager/models/transcript"
	"monorepo/bin-transcribe-manager/pkg/transcribehandler"
	"monorepo/bin-transcribe-manager/pkg/transcripthandler"
)

func Test_processV1TranscriptsGet(t *testing.T) {

	tests := []struct {
		name    string
		request *rabbitmqhandler.Request

		pageSize  uint64
		pageToken string

		responseFilters     map[string]string
		responseTranscripts []*transcript.Transcript
		expectRes           *rabbitmqhandler.Response
	}{
		{
			"normal",
			&rabbitmqhandler.Request{
				URI:    "/v1/transcripts?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_transcribe_id=4f08e520-821d-11ed-844e-67fdcb950f6f",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			10,
			"2020-05-03 21:35:02.809",

			map[string]string{
				"transcribe_id": "4f08e520-821d-11ed-844e-67fdcb950f6f",
			},
			[]*transcript.Transcript{
				{
					ID: uuid.FromStringOrNil("2afd749c-821e-11ed-9ba2-271e7b9600a1"),
				},
			},

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"2afd749c-821e-11ed-9ba2-271e7b9600a1","customer_id":"00000000-0000-0000-0000-000000000000","transcribe_id":"00000000-0000-0000-0000-000000000000","direction":"","message":"","tm_transcript":"","tm_create":"","tm_delete":""}]`),
			},
		},
		{
			"2 items",
			&rabbitmqhandler.Request{
				URI:    "/v1/transcripts?page_size=10&page_token=2020-05-03%2021:35:02.809&filter_transcribe_id=43b608e6-821e-11ed-9611-e329cba76cc9",
				Method: rabbitmqhandler.RequestMethodGet,
			},

			10,
			"2020-05-03 21:35:02.809",

			map[string]string{
				"transcribe_id": "43b608e6-821e-11ed-9611-e329cba76cc9",
			},
			[]*transcript.Transcript{
				{
					ID: uuid.FromStringOrNil("43e2dae2-821e-11ed-8cb9-ff5d144f9d22"),
				},
				{
					ID: uuid.FromStringOrNil("440fc214-821e-11ed-b83d-2f241266f784"),
				},
			},
			&rabbitmqhandler.Response{
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

			mockSock := rabbitmqhandler.NewMockRabbit(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockTranscribe := transcribehandler.NewMockTranscribeHandler(mc)
			mockTranscript := transcripthandler.NewMockTranscriptHandler(mc)

			h := &listenHandler{
				rabbitSock:        mockSock,
				utilHandler:       mockUtil,
				transcribeHandler: mockTranscribe,
				transcriptHandler: mockTranscript,
			}

			mockUtil.EXPECT().URLParseFilters(gomock.Any()).Return(tt.responseFilters)
			mockTranscript.EXPECT().Gets(gomock.Any(), tt.pageSize, tt.pageToken, tt.responseFilters).Return(tt.responseTranscripts, nil)
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
