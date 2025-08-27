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
				Data:     []byte(`{"customer_id":"5d348388-5b01-11f0-bdba-43ddb95acbae","reference_type":"call","reference_id":"6504fcc8-5b01-11f0-9107-2fa36ae9f966","language":"en-US","gender":"male","direction":"out"}`),
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

			res, err := reqHandler.TTSV1StreamingCreate(context.Background(), tt.customerID, tt.referenceType, tt.referenceID, tt.language, tt.gender, tt.direction)
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

func Test_TTSV1StreamingSay(t *testing.T) {

	tests := []struct {
		name string

		podID string
		id    uuid.UUID
		text  string

		response *sock.Response

		expectQueue   string
		expectRequest *sock.Request
		expectRes     *tmstreaming.Streaming
	}{
		{
			name: "normal",

			podID: "7dbd5818-5b02-11f0-b594-c7683b9cdc6e",
			id:    uuid.FromStringOrNil("65450fca-5b01-11f0-a32f-9b5f26a6e601"),
			text:  "hello world",

			response: &sock.Response{
				StatusCode: 200,
				DataType:   "application/json",
				Data:       []byte(`{"id":"65450fca-5b01-11f0-a32f-9b5f26a6e601"}`),
			},

			expectQueue: "bin-manager.tts-manager.request.7dbd5818-5b02-11f0-b594-c7683b9cdc6e",
			expectRequest: &sock.Request{
				URI:      "/v1/streamings/65450fca-5b01-11f0-a32f-9b5f26a6e601/say",
				Method:   sock.RequestMethodPost,
				DataType: ContentTypeJSON,
				Data:     []byte(`{"text":"hello world"}`),
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

			mockSock.EXPECT().RequestPublish(gomock.Any(), tt.expectQueue, tt.expectRequest).Return(tt.response, nil)

			if err := reqHandler.TTSV1StreamingSay(context.Background(), tt.podID, tt.id, tt.text); err != nil {
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
