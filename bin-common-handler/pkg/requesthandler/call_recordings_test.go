package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cmrecording "monorepo/bin-call-manager/models/recording"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"

	"monorepo/bin-common-handler/pkg/rabbitmqhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func Test_CallV1RecordingGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		expectURL     string
		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     []cmrecording.Recording
	}{
		{
			"normal",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			"/v1/recordings?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c9c63840-8ebf-11ed-8f4c-534a60a32848"}]`),
			},
			[]cmrecording.Recording{
				{
					ID: uuid.FromStringOrNil("c9c63840-8ebf-11ed-8f4c-534a60a32848"),
				},
			},
		},
		{
			"2 items",

			"2020-09-20T03:23:20.995000",
			10,
			map[string]string{
				"deleted": "false",
			},

			"/v1/recordings?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10&filter_deleted=false",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ca1558b2-8ebf-11ed-9014-33c1de740f04"},{"id":"e445e45e-8ebf-11ed-89f3-8b24e2aee52e"}]`),
			},
			[]cmrecording.Recording{
				{
					ID: uuid.FromStringOrNil("ca1558b2-8ebf-11ed-9014-33c1de740f04"),
				},
				{
					ID: uuid.FromStringOrNil("e445e45e-8ebf-11ed-89f3-8b24e2aee52e"),
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
			mockSock.EXPECT().PublishRPC(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := h.CallV1RecordingGets(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1RecordingGet(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		response      *rabbitmqhandler.Response
		expectRes     *cmrecording.Recording
	}{
		{
			"normal",

			uuid.FromStringOrNil("32154990-8ec0-11ed-98c2-7f6a7e0cc03e"),

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings/32154990-8ec0-11ed-98c2-7f6a7e0cc03e",
				Method:   rabbitmqhandler.RequestMethodGet,
				DataType: "application/json",
			},
			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"32154990-8ec0-11ed-98c2-7f6a7e0cc03e"}`),
			},
			&cmrecording.Recording{
				ID: uuid.FromStringOrNil("32154990-8ec0-11ed-98c2-7f6a7e0cc03e"),
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

			res, err := reqHandler.CallV1RecordingGet(ctx, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_CallV1RecordingDelete(t *testing.T) {

	tests := []struct {
		name string

		callID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cmrecording.Recording
	}{
		{
			"normal",

			uuid.FromStringOrNil("570ddfbe-8ec0-11ed-9dd8-1f8e11bf6de2"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"570ddfbe-8ec0-11ed-9dd8-1f8e11bf6de2"}`),
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings/570ddfbe-8ec0-11ed-9dd8-1f8e11bf6de2",
				Method:   rabbitmqhandler.RequestMethodDelete,
				DataType: "application/json",
			},
			&cmrecording.Recording{
				ID: uuid.FromStringOrNil("570ddfbe-8ec0-11ed-9dd8-1f8e11bf6de2"),
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

			res, err := reqHandler.CallV1RecordingDelete(ctx, tt.callID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CallV1RecordingStart(t *testing.T) {

	tests := []struct {
		name string

		referenceType cmrecording.ReferenceType
		referenceID   uuid.UUID
		format        cmrecording.Format
		endOfSilence  int
		endOfKey      string
		duration      int

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cmrecording.Recording
	}{
		{
			"normal",

			cmrecording.ReferenceTypeCall,
			uuid.FromStringOrNil("a49bea54-90ce-11ed-9bfb-67a5f5309240"),
			cmrecording.FormatWAV,
			10000,
			"#",
			100000,

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a4d5b57c-90ce-11ed-a125-b38f2f6766f4"}`),
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"owner_id":"00000000-0000-0000-0000-000000000000","reference_type":"call","reference_id":"a49bea54-90ce-11ed-9bfb-67a5f5309240","format":"wav","end_of_silence":10000,"end_of_key":"#","duration":100000}`),
			},
			&cmrecording.Recording{
				ID: uuid.FromStringOrNil("a4d5b57c-90ce-11ed-a125-b38f2f6766f4"),
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

			res, err := reqHandler.CallV1RecordingStart(ctx, tt.referenceType, tt.referenceID, tt.format, tt.endOfSilence, tt.endOfKey, tt.duration)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}

func Test_CallV1RecordingStop(t *testing.T) {

	tests := []struct {
		name string

		recordingID uuid.UUID

		response *rabbitmqhandler.Response

		expectTarget  string
		expectRequest *rabbitmqhandler.Request
		expectResult  *cmrecording.Recording
	}{
		{
			"normal",

			uuid.FromStringOrNil("b843ba34-90d6-11ed-872b-9fc8addbbe5e"),

			&rabbitmqhandler.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a4d5b57c-90ce-11ed-a125-b38f2f6766f4"}`),
			},

			"bin-manager.call-manager.request",
			&rabbitmqhandler.Request{
				URI:      "/v1/recordings/b843ba34-90d6-11ed-872b-9fc8addbbe5e/stop",
				Method:   rabbitmqhandler.RequestMethodPost,
				DataType: "",
			},
			&cmrecording.Recording{
				ID: uuid.FromStringOrNil("a4d5b57c-90ce-11ed-a125-b38f2f6766f4"),
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

			res, err := reqHandler.CallV1RecordingStop(ctx, tt.recordingID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectResult, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectResult, *res)
			}
		})
	}
}
