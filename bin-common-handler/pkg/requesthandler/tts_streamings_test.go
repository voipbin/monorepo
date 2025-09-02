package requesthandler

import (
	"context"
	"reflect"
	"testing"

	tmstreaming "monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"
)

func Test_TTSV1StreamingCreate(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		activeflowID  uuid.UUID
		referenceType tmstreaming.ReferenceType
		referenceID   uuid.UUID
		language      string
		gender        tmstreaming.Gender
		direction     tmstreaming.Direction

		response *sock.Response

		expectRequest *sock.Request
		expectRes     *tmstreaming.Streaming
	}{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("5d348388-5b01-11f0-bdba-43ddb95acbae"),
			activeflowID:  uuid.FromStringOrNil("7bd9de5e-87cb-11f0-bd50-633c3bd413e3"),
			referenceType: tmstreaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("6504fcc8-5b01-11f0-9107-2fa36ae9f966"),
			language:      "en-US",
			gender:        tmstreaming.GenderMale,
			direction:     tmstreaming.DirectionOutgoing,

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"65251224-5b01-11f0-8b4c-2759d8d6f460"}`),
			},

			expectRequest: &sock.Request{
				URI:      "/v1/streamings",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"customer_id":"5d348388-5b01-11f0-bdba-43ddb95acbae","activeflow_id":"7bd9de5e-87cb-11f0-bd50-633c3bd413e3","reference_type":"call","reference_id":"6504fcc8-5b01-11f0-9107-2fa36ae9f966","language":"en-US","gender":"male","direction":"out"}`),
			},
			expectRes: &tmstreaming.Streaming{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("65251224-5b01-11f0-8b4c-2759d8d6f460"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.tts-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.TTSV1StreamingCreate(context.Background(), tt.customerID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.language, tt.gender, tt.direction)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TTSV1StreamingDelete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		response *sock.Response

		expectRequest *sock.Request
		expectRes     *tmstreaming.Streaming
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("65450fca-5b01-11f0-a32f-9b5f26a6e601"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"65450fca-5b01-11f0-a32f-9b5f26a6e601"}`),
			},
			expectRequest: &sock.Request{
				URI:    "/v1/streamings/65450fca-5b01-11f0-a32f-9b5f26a6e601",
				Method: sock.RequestMethodDelete,
			},

			expectRes: &tmstreaming.Streaming{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("65450fca-5b01-11f0-a32f-9b5f26a6e601"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), "bin-manager.tts-manager.request", tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.TTSV1StreamingDelete(context.Background(), tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TTSV1StreamingSayInit(t *testing.T) {

	tests := []struct {
		name string

		podID     string
		id        uuid.UUID
		messageID uuid.UUID

		response *sock.Response

		expectQueue   string
		expectRequest *sock.Request
		expectRes     *tmstreaming.Streaming
	}{
		{
			name: "normal",

			podID:     "03ce0a14-87a5-11f0-883e-3f4085c4a940",
			id:        uuid.FromStringOrNil("040466ae-87a5-11f0-9b76-236f64436c4e"),
			messageID: uuid.FromStringOrNil("0428ed1c-87a5-11f0-a8e4-9719efbaab7c"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"040466ae-87a5-11f0-9b76-236f64436c4e"}`),
			},

			expectQueue: "bin-manager.tts-manager.request.03ce0a14-87a5-11f0-883e-3f4085c4a940",
			expectRequest: &sock.Request{
				URI:      "/v1/streamings/040466ae-87a5-11f0-9b76-236f64436c4e/say_init",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"message_id":"0428ed1c-87a5-11f0-a8e4-9719efbaab7c"}`),
			},
			expectRes: &tmstreaming.Streaming{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("040466ae-87a5-11f0-9b76-236f64436c4e"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			res, err := reqHandler.TTSV1StreamingSayInit(context.Background(), tt.podID, tt.id, tt.messageID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_TTSV1StreamingSayAdd(t *testing.T) {

	tests := []struct {
		name string

		podID     string
		id        uuid.UUID
		messageID uuid.UUID
		text      string

		response *sock.Response

		expectQueue   string
		expectRequest *sock.Request
		expectRes     *tmstreaming.Streaming
	}{
		{
			name: "normal",

			podID:     "f22f0194-83d7-11f0-9b8c-737f0a365b94",
			id:        uuid.FromStringOrNil("f25d4e1e-83d7-11f0-92a1-3f6fa96b2ecf"),
			messageID: uuid.FromStringOrNil("f286eefe-83d7-11f0-971b-d7b8ef521291"),
			text:      "hello world",

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"f25d4e1e-83d7-11f0-92a1-3f6fa96b2ecf"}`),
			},

			expectQueue: "bin-manager.tts-manager.request.f22f0194-83d7-11f0-9b8c-737f0a365b94",
			expectRequest: &sock.Request{
				URI:      "/v1/streamings/f25d4e1e-83d7-11f0-92a1-3f6fa96b2ecf/say_add",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"message_id":"f286eefe-83d7-11f0-971b-d7b8ef521291","text":"hello world"}`),
			},
			expectRes: &tmstreaming.Streaming{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("f25d4e1e-83d7-11f0-92a1-3f6fa96b2ecf"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.TTSV1StreamingSayAdd(context.Background(), tt.podID, tt.id, tt.messageID, tt.text); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_TTSV1StreamingSayStop(t *testing.T) {

	tests := []struct {
		name string

		podID string
		id    uuid.UUID

		response *sock.Response

		expectQueue   string
		expectRequest *sock.Request
		expectRes     *tmstreaming.Streaming
	}{
		{
			name: "normal",

			podID: "184c7e7c-8286-11f0-b4c5-77e7c515c4f2",
			id:    uuid.FromStringOrNil("187a294e-8286-11f0-a61c-ef87402cb1a0"),

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"187a294e-8286-11f0-a61c-ef87402cb1a0"}`),
			},

			expectQueue: "bin-manager.tts-manager.request.184c7e7c-8286-11f0-b4c5-77e7c515c4f2",
			expectRequest: &sock.Request{
				URI:      "/v1/streamings/187a294e-8286-11f0-a61c-ef87402cb1a0/say_stop",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeNone,
			},
			expectRes: &tmstreaming.Streaming{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("187a294e-8286-11f0-a61c-ef87402cb1a0"),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.TTSV1StreamingSayStop(context.Background(), tt.podID, tt.id); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
