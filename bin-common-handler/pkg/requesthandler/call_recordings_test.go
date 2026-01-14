package requesthandler

import (
	"context"
	"reflect"
	"testing"

	cmrecording "monorepo/bin-call-manager/models/recording"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_CallV1RecordingGets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[cmrecording.Field]any

		expectTarget  string
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     []cmrecording.Recording
	}{
		{
			"normal",

			"2020-09-20T03:23:20.995000",
			10,
			map[cmrecording.Field]any{
				cmrecording.FieldDeleted: false,
			},

			"bin-manager.call-manager.request",
			&sock.Request{
				URI:      "/v1/recordings?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"deleted":false}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"c9c63840-8ebf-11ed-8f4c-534a60a32848"}]`),
			},
			[]cmrecording.Recording{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c9c63840-8ebf-11ed-8f4c-534a60a32848"),
					},
				},
			},
		},
		{
			"2 items",

			"2020-09-20T03:23:20.995000",
			10,
			map[cmrecording.Field]any{
				cmrecording.FieldDeleted: false,
			},

			"bin-manager.call-manager.request",
			&sock.Request{
				URI:      "/v1/recordings?page_token=2020-09-20T03%3A23%3A20.995000&page_size=10",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
				Data:     []byte(`{"deleted":false}`),
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`[{"id":"ca1558b2-8ebf-11ed-9014-33c1de740f04"},{"id":"e445e45e-8ebf-11ed-89f3-8b24e2aee52e"}]`),
			},
			[]cmrecording.Recording{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("ca1558b2-8ebf-11ed-9014-33c1de740f04"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("e445e45e-8ebf-11ed-89f3-8b24e2aee52e"),
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
			h := requestHandler{
				sock: mockSock,
			}
			ctx := context.Background()

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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
		expectRequest *sock.Request
		response      *sock.Response
		expectRes     *cmrecording.Recording
	}{
		{
			"normal",

			uuid.FromStringOrNil("32154990-8ec0-11ed-98c2-7f6a7e0cc03e"),

			"bin-manager.call-manager.request",
			&sock.Request{
				URI:      "/v1/recordings/32154990-8ec0-11ed-98c2-7f6a7e0cc03e",
				Method:   sock.RequestMethodGet,
				DataType: "application/json",
			},
			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"32154990-8ec0-11ed-98c2-7f6a7e0cc03e"}`),
			},
			&cmrecording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("32154990-8ec0-11ed-98c2-7f6a7e0cc03e"),
				},
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

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *cmrecording.Recording
	}{
		{
			"normal",

			uuid.FromStringOrNil("570ddfbe-8ec0-11ed-9dd8-1f8e11bf6de2"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"570ddfbe-8ec0-11ed-9dd8-1f8e11bf6de2"}`),
			},

			"bin-manager.call-manager.request",
			&sock.Request{
				URI:      "/v1/recordings/570ddfbe-8ec0-11ed-9dd8-1f8e11bf6de2",
				Method:   sock.RequestMethodDelete,
				DataType: "application/json",
			},
			&cmrecording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("570ddfbe-8ec0-11ed-9dd8-1f8e11bf6de2"),
				},
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

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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

		activeflowID  uuid.UUID
		referenceType cmrecording.ReferenceType
		referenceID   uuid.UUID
		format        cmrecording.Format
		endOfSilence  int
		endOfKey      string
		duration      int
		onEndFlowID   uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectRe      *cmrecording.Recording
	}{
		{
			name: "normal",

			activeflowID:  uuid.FromStringOrNil("2789cd4c-075a-11f0-83eb-ff3c943e1393"),
			referenceType: cmrecording.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("a49bea54-90ce-11ed-9bfb-67a5f5309240"),
			format:        cmrecording.FormatWAV,
			endOfSilence:  10000,
			endOfKey:      "#",
			duration:      100000,
			onEndFlowID:   uuid.FromStringOrNil("0198bf9c-055e-11f0-a389-d7d10200060e"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a4d5b57c-90ce-11ed-a125-b38f2f6766f4"}`),
			},

			expectTarget: "bin-manager.call-manager.request",
			expectRequest: &sock.Request{
				URI:      "/v1/recordings",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"owner_id":"00000000-0000-0000-0000-000000000000","activeflow_id":"2789cd4c-075a-11f0-83eb-ff3c943e1393","reference_type":"call","reference_id":"a49bea54-90ce-11ed-9bfb-67a5f5309240","format":"wav","end_of_silence":10000,"end_of_key":"#","duration":100000,"on_end_flow_id":"0198bf9c-055e-11f0-a389-d7d10200060e"}`),
			},
			expectRe: &cmrecording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4d5b57c-90ce-11ed-a125-b38f2f6766f4"),
				},
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

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.CallV1RecordingStart(ctx, tt.activeflowID, tt.referenceType, tt.referenceID, tt.format, tt.endOfSilence, tt.endOfKey, tt.duration, tt.onEndFlowID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*tt.expectRe, *res) == false {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", *tt.expectRe, *res)
			}
		})
	}
}

func Test_CallV1RecordingStop(t *testing.T) {

	tests := []struct {
		name string

		recordingID uuid.UUID

		response *sock.Response

		expectTarget  string
		expectRequest *sock.Request
		expectResult  *cmrecording.Recording
	}{
		{
			"normal",

			uuid.FromStringOrNil("b843ba34-90d6-11ed-872b-9fc8addbbe5e"),

			&sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"a4d5b57c-90ce-11ed-a125-b38f2f6766f4"}`),
			},

			"bin-manager.call-manager.request",
			&sock.Request{
				URI:      "/v1/recordings/b843ba34-90d6-11ed-872b-9fc8addbbe5e/stop",
				Method:   sock.RequestMethodPost,
				DataType: "",
			},
			&cmrecording.Recording{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a4d5b57c-90ce-11ed-a125-b38f2f6766f4"),
				},
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

			ctx := context.Background()
			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectTarget, tt.expectRequest).Return(tt.response, nil)

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
