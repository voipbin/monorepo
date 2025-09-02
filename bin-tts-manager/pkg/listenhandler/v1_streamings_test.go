package listenhandler

import (
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
	"monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/streaminghandler"
	"monorepo/bin-tts-manager/pkg/ttshandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_v1StreamingsPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseStreaming *streaming.Streaming

		expectCustomerID    uuid.UUID
		expectReferenceType streaming.ReferenceType
		expectReferenceID   uuid.UUID
		expectLanguage      string
		expectGender        streaming.Gender
		expectDirection     streaming.Direction
		expectRes           *sock.Response
	}{
		{
			name: "normal test",

			request: &sock.Request{
				URI:      "/v1/streamings",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"customer_id": "b685d36c-5af5-11f0-b36b-2be7103890ae", "reference_type": "call", "reference_id": "b6b3ba5c-5af5-11f0-bf7e-438fa6d3dba0", "language": "en-US", "gender": "female", "direction": "out"}`),
			},

			responseStreaming: &streaming.Streaming{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b6dcbc72-5af5-11f0-b0a7-fbe8aae41529"),
				},
			},

			expectCustomerID:    uuid.FromStringOrNil("b685d36c-5af5-11f0-b36b-2be7103890ae"),
			expectReferenceType: streaming.ReferenceTypeCall,
			expectReferenceID:   uuid.FromStringOrNil("b6b3ba5c-5af5-11f0-bf7e-438fa6d3dba0"),
			expectLanguage:      "en-US",
			expectGender:        streaming.GenderFemale,
			expectDirection:     streaming.DirectionOutgoing,
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b6dcbc72-5af5-11f0-b0a7-fbe8aae41529","customer_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","message_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTTS := ttshandler.NewMockTTSHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				ttsHandler:       mockTTS,
				streamingHandler: mockStreaming,
			}

			mockStreaming.EXPECT().Start(gomock.Any(), tt.expectCustomerID, tt.expectReferenceType, tt.expectReferenceID, tt.expectLanguage, tt.expectGender, tt.expectDirection).Return(tt.responseStreaming, nil)

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

func Test_v1StreamingsIDDelete(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseStreaming *streaming.Streaming

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			name: "normal test",

			request: &sock.Request{
				URI:    "/v1/streamings/08bd5438-5af7-11f0-bac3-77c3230b0c24",
				Method: sock.RequestMethodDelete,
			},

			responseStreaming: &streaming.Streaming{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("08bd5438-5af7-11f0-bac3-77c3230b0c24"),
				},
			},

			expectID: uuid.FromStringOrNil("08bd5438-5af7-11f0-bac3-77c3230b0c24"),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"08bd5438-5af7-11f0-bac3-77c3230b0c24","customer_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","message_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTTS := ttshandler.NewMockTTSHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				ttsHandler:       mockTTS,
				streamingHandler: mockStreaming,
			}

			mockStreaming.EXPECT().Stop(gomock.Any(), tt.expectID).Return(tt.responseStreaming, nil)

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

func Test_v1StreamingsIDSayAddPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseStreaming *streaming.Streaming

		expectID        uuid.UUID
		expectMessageID uuid.UUID
		expectText      string
		expectRes       *sock.Response
	}{
		{
			name: "normal test",

			request: &sock.Request{
				URI:      "/v1/streamings/4f5cf8c6-83d3-11f0-9853-bf795e7133de/say_add",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"message_id":"4f93bf3c-83d3-11f0-a55e-bbe3bb420e72","text": "Hello, this is a test message."}`),
			},

			responseStreaming: &streaming.Streaming{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4f5cf8c6-83d3-11f0-9853-bf795e7133de"),
				},
			},

			expectID:        uuid.FromStringOrNil("4f5cf8c6-83d3-11f0-9853-bf795e7133de"),
			expectMessageID: uuid.FromStringOrNil("4f93bf3c-83d3-11f0-a55e-bbe3bb420e72"),
			expectText:      "Hello, this is a test message.",
			expectRes: &sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTTS := ttshandler.NewMockTTSHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				ttsHandler:       mockTTS,
				streamingHandler: mockStreaming,
			}

			mockStreaming.EXPECT().SayAdd(gomock.Any(), tt.expectID, tt.expectMessageID, tt.expectText).Return(nil)

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

func Test_v1StreamingsIDSayStopPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseStreaming *streaming.Streaming

		expectID  uuid.UUID
		expectRes *sock.Response
	}{
		{
			name: "normal test",

			request: &sock.Request{
				URI:    "/v1/streamings/b349ca90-8283-11f0-bb51-9bcbfbadb7eb/say_stop",
				Method: sock.RequestMethodPost,
			},

			responseStreaming: &streaming.Streaming{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b349ca90-8283-11f0-bb51-9bcbfbadb7eb"),
				},
			},

			expectID: uuid.FromStringOrNil("b349ca90-8283-11f0-bb51-9bcbfbadb7eb"),
			expectRes: &sock.Response{
				StatusCode: 200,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTTS := ttshandler.NewMockTTSHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				ttsHandler:       mockTTS,
				streamingHandler: mockStreaming,
			}

			mockStreaming.EXPECT().SayStop(gomock.Any(), tt.expectID).Return(nil)

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

func Test_v1StreamingsIDSayInitPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseStreaming *streaming.Streaming

		expectID        uuid.UUID
		expectMessageID uuid.UUID
		expectRes       *sock.Response
	}{
		{
			name: "normal test",

			request: &sock.Request{
				URI:      "/v1/streamings/b7dcc5c6-87a1-11f0-857c-2797bdb40c77/say_init",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(`{"message_id":"b808a9d4-87a1-11f0-9b0c-e3dba6c685da"}`),
			},

			responseStreaming: &streaming.Streaming{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("b7dcc5c6-87a1-11f0-857c-2797bdb40c77"),
				},
			},

			expectID:        uuid.FromStringOrNil("b7dcc5c6-87a1-11f0-857c-2797bdb40c77"),
			expectMessageID: uuid.FromStringOrNil("b808a9d4-87a1-11f0-9b0c-e3dba6c685da"),
			expectRes: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"b7dcc5c6-87a1-11f0-857c-2797bdb40c77","customer_id":"00000000-0000-0000-0000-000000000000","reference_id":"00000000-0000-0000-0000-000000000000","message_id":"00000000-0000-0000-0000-000000000000"}`),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSock := sockhandler.NewMockSockHandler(mc)
			mockTTS := ttshandler.NewMockTTSHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

			h := &listenHandler{
				sockHandler:      mockSock,
				ttsHandler:       mockTTS,
				streamingHandler: mockStreaming,
			}

			mockStreaming.EXPECT().SayInit(gomock.Any(), tt.expectID, tt.expectMessageID).Return(tt.responseStreaming, nil)

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
