package listenhandler

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"monorepo/bin-common-handler/models/sock"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/speakinghandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_v1SpeakingsPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseSpeaking *speaking.Speaking

		expectCustomerID    uuid.UUID
		expectReferenceType streaming.ReferenceType
		expectReferenceID   uuid.UUID
		expectLanguage      string
		expectProvider      string
		expectVoiceID       string
		expectDirection     streaming.Direction
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:    "/v1/speakings",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"customer_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", "reference_type": "call", "reference_id": "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12", "language": "en-US", "direction": "in"}`),
			},

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				ReferenceType: streaming.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
				Language:      "en-US",
				Direction:     streaming.DirectionIncoming,
				Status:        speaking.StatusActive,
			},

			expectCustomerID:    uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			expectReferenceType: streaming.ReferenceTypeCall,
			expectReferenceID:   uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
			expectLanguage:      "en-US",
			expectProvider:      "",
			expectVoiceID:       "",
			expectDirection:     streaming.DirectionIncoming,
		},
		{
			name: "with all optional fields",

			request: &sock.Request{
				URI:    "/v1/speakings",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"customer_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11", "reference_type": "call", "reference_id": "b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12", "language": "en-US", "provider": "elevenlabs", "voice_id": "voice123", "direction": "out"}`),
			},

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					CustomerID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				ReferenceType: streaming.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
				Language:      "en-US",
				Provider:      "elevenlabs",
				VoiceID:       "voice123",
				Direction:     streaming.DirectionOutgoing,
				Status:        speaking.StatusActive,
			},

			expectCustomerID:    uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			expectReferenceType: streaming.ReferenceTypeCall,
			expectReferenceID:   uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
			expectLanguage:      "en-US",
			expectProvider:      "elevenlabs",
			expectVoiceID:       "voice123",
			expectDirection:     streaming.DirectionOutgoing,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSpeaking := speakinghandler.NewMockSpeakingHandler(mc)

			h := &listenHandler{
				speakingHandler: mockSpeaking,
			}

			mockSpeaking.EXPECT().Create(
				gomock.Any(),
				tt.expectCustomerID,
				tt.expectReferenceType,
				tt.expectReferenceID,
				tt.expectLanguage,
				tt.expectProvider,
				tt.expectVoiceID,
				tt.expectDirection,
			).Return(tt.responseSpeaking, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Wrong status code. expect: 200, got: %d", res.StatusCode)
			}

			var got speaking.Speaking
			if errJSON := json.Unmarshal(res.Data, &got); errJSON != nil {
				t.Errorf("Could not unmarshal response. err: %v", errJSON)
			}
			if got.ID != tt.responseSpeaking.ID {
				t.Errorf("Wrong ID. expect: %s, got: %s", tt.responseSpeaking.ID, got.ID)
			}
		})
	}
}

func Test_v1SpeakingsGet(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseSpeakings []*speaking.Speaking

		expectToken   string
		expectSize    uint64
		expectFilters map[speaking.Field]any
	}{
		{
			name: "normal with default filters",

			request: &sock.Request{
				URI:    "/v1/speakings",
				Method: sock.RequestMethodGet,
			},

			responseSpeakings: []*speaking.Speaking{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					},
					Status: speaking.StatusActive,
				},
			},

			expectToken: "",
			expectSize:  uint64(100),
			expectFilters: map[speaking.Field]any{
				speaking.FieldDeleted: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSpeaking := speakinghandler.NewMockSpeakingHandler(mc)

			h := &listenHandler{
				speakingHandler: mockSpeaking,
			}

			mockSpeaking.EXPECT().Gets(gomock.Any(), tt.expectToken, tt.expectSize, tt.expectFilters).Return(tt.responseSpeakings, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Wrong status code. expect: 200, got: %d", res.StatusCode)
			}

			var got []*speaking.Speaking
			if errJSON := json.Unmarshal(res.Data, &got); errJSON != nil {
				t.Errorf("Could not unmarshal response. err: %v", errJSON)
			}
			if len(got) != len(tt.responseSpeakings) {
				t.Errorf("Wrong count. expect: %d, got: %d", len(tt.responseSpeakings), len(got))
			}
		})
	}
}

func Test_v1SpeakingsGetWithFilters(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseSpeakings []*speaking.Speaking

		expectToken   string
		expectSize    uint64
		expectFilters map[speaking.Field]any
	}{
		{
			name: "with customer_id and status filters",

			request: &sock.Request{
				URI:    "/v1/speakings?customer_id=a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11&status=active&page_size=5&page_token=2024-01-01",
				Method: sock.RequestMethodGet,
			},

			responseSpeakings: []*speaking.Speaking{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					},
					Status: speaking.StatusActive,
				},
			},

			expectToken: "2024-01-01",
			expectSize:  uint64(5),
			expectFilters: map[speaking.Field]any{
				speaking.FieldDeleted:    false,
				speaking.FieldCustomerID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				speaking.FieldStatus:     "active",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSpeaking := speakinghandler.NewMockSpeakingHandler(mc)

			h := &listenHandler{
				speakingHandler: mockSpeaking,
			}
			ctx := context.Background()

			mockSpeaking.EXPECT().Gets(gomock.Any(), tt.expectToken, tt.expectSize, tt.expectFilters).Return(tt.responseSpeakings, nil)

			res, err := h.v1SpeakingsGet(ctx, tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Wrong status code. expect: 200, got: %d", res.StatusCode)
			}

			var got []*speaking.Speaking
			if errJSON := json.Unmarshal(res.Data, &got); errJSON != nil {
				t.Errorf("Could not unmarshal response. err: %v", errJSON)
			}
			if len(got) != len(tt.responseSpeakings) {
				t.Errorf("Wrong count. expect: %d, got: %d", len(tt.responseSpeakings), len(got))
			}
		})
	}
}

func Test_v1SpeakingsIDGet(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseSpeaking *speaking.Speaking

		expectID uuid.UUID
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:    "/v1/speakings/c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13",
				Method: sock.RequestMethodGet,
			},

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
				},
				Status: speaking.StatusActive,
			},

			expectID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSpeaking := speakinghandler.NewMockSpeakingHandler(mc)

			h := &listenHandler{
				speakingHandler: mockSpeaking,
			}

			mockSpeaking.EXPECT().Get(gomock.Any(), tt.expectID).Return(tt.responseSpeaking, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Wrong status code. expect: 200, got: %d", res.StatusCode)
			}

			var got speaking.Speaking
			if errJSON := json.Unmarshal(res.Data, &got); errJSON != nil {
				t.Errorf("Could not unmarshal response. err: %v", errJSON)
			}
			if !reflect.DeepEqual(got, *tt.responseSpeaking) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", *tt.responseSpeaking, got)
			}
		})
	}
}

func Test_v1SpeakingsIDDelete(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseSpeaking *speaking.Speaking

		expectID uuid.UUID
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:    "/v1/speakings/c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13",
				Method: sock.RequestMethodDelete,
			},

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
				},
			},

			expectID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSpeaking := speakinghandler.NewMockSpeakingHandler(mc)

			h := &listenHandler{
				speakingHandler: mockSpeaking,
			}

			mockSpeaking.EXPECT().Delete(gomock.Any(), tt.expectID).Return(tt.responseSpeaking, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Wrong status code. expect: 200, got: %d", res.StatusCode)
			}

			var got speaking.Speaking
			if errJSON := json.Unmarshal(res.Data, &got); errJSON != nil {
				t.Errorf("Could not unmarshal response. err: %v", errJSON)
			}
			if got.ID != tt.expectID {
				t.Errorf("Wrong ID. expect: %s, got: %s", tt.expectID, got.ID)
			}
		})
	}
}

func Test_v1SpeakingsIDSayPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseSpeaking *speaking.Speaking

		expectID   uuid.UUID
		expectText string
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:    "/v1/speakings/c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13/say",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{"text": "hello world"}`),
			},

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
				},
				Status: speaking.StatusActive,
			},

			expectID:   uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
			expectText: "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSpeaking := speakinghandler.NewMockSpeakingHandler(mc)

			h := &listenHandler{
				speakingHandler: mockSpeaking,
			}

			mockSpeaking.EXPECT().Say(gomock.Any(), tt.expectID, tt.expectText).Return(tt.responseSpeaking, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Wrong status code. expect: 200, got: %d", res.StatusCode)
			}
		})
	}
}

func Test_v1SpeakingsIDFlushPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseSpeaking *speaking.Speaking

		expectID uuid.UUID
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:    "/v1/speakings/c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13/flush",
				Method: sock.RequestMethodPost,
			},

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
				},
				Status: speaking.StatusActive,
			},

			expectID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSpeaking := speakinghandler.NewMockSpeakingHandler(mc)

			h := &listenHandler{
				speakingHandler: mockSpeaking,
			}

			mockSpeaking.EXPECT().Flush(gomock.Any(), tt.expectID).Return(tt.responseSpeaking, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Wrong status code. expect: 200, got: %d", res.StatusCode)
			}
		})
	}
}

func Test_v1SpeakingsIDStopPost(t *testing.T) {

	tests := []struct {
		name string

		request *sock.Request

		responseSpeaking *speaking.Speaking

		expectID uuid.UUID
	}{
		{
			name: "normal",

			request: &sock.Request{
				URI:    "/v1/speakings/c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13/stop",
				Method: sock.RequestMethodPost,
			},

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
				},
				Status: speaking.StatusStopped,
			},

			expectID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSpeaking := speakinghandler.NewMockSpeakingHandler(mc)

			h := &listenHandler{
				speakingHandler: mockSpeaking,
			}

			mockSpeaking.EXPECT().Stop(gomock.Any(), tt.expectID).Return(tt.responseSpeaking, nil)

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if res.StatusCode != 200 {
				t.Errorf("Wrong status code. expect: 200, got: %d", res.StatusCode)
			}
		})
	}
}
