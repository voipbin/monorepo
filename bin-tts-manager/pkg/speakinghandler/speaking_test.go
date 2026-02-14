package speakinghandler

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-tts-manager/models/speaking"
	"monorepo/bin-tts-manager/models/streaming"
	"monorepo/bin-tts-manager/pkg/dbhandler"
	"monorepo/bin-tts-manager/pkg/streaminghandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		referenceType streaming.ReferenceType
		referenceID   uuid.UUID
		language      string
		provider      string
		voiceID       string
		direction     streaming.Direction

		responseExisting []*speaking.Speaking
		responseExistErr error
		responseCreateErr error
		responseStreaming *streaming.Streaming
		responseStreamErr error
		responseUpdateErr error
		responseGet      *speaking.Speaking
		responseGetErr   error

		expectErr bool
	}{
		{
			name: "normal",

			customerID:    uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
			language:      "en-US",
			provider:      "elevenlabs",
			voiceID:       "voice123",
			direction:     streaming.DirectionIncoming,

			responseExisting: []*speaking.Speaking{},
			responseStreaming: &streaming.Streaming{},
			responseGet: &speaking.Speaking{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				ReferenceType: streaming.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
				Status:        speaking.StatusActive,
			},

			expectErr: false,
		},
		{
			name: "existing active session blocks",

			customerID:    uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
			language:      "en-US",
			provider:      "elevenlabs",
			voiceID:       "",
			direction:     streaming.DirectionIncoming,

			responseExisting: []*speaking.Speaking{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					},
					Status: speaking.StatusActive,
				},
			},

			expectErr: true,
		},
		{
			name: "existing initiating session blocks",

			customerID:    uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
			language:      "en-US",
			provider:      "elevenlabs",
			voiceID:       "",
			direction:     streaming.DirectionIncoming,

			responseExisting: []*speaking.Speaking{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					},
					Status: speaking.StatusInitiating,
				},
			},

			expectErr: true,
		},
		{
			name: "existing stopped session allows",

			customerID:    uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
			language:      "en-US",
			provider:      "elevenlabs",
			voiceID:       "",
			direction:     streaming.DirectionIncoming,

			responseExisting: []*speaking.Speaking{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					},
					Status: speaking.StatusStopped,
				},
			},
			responseStreaming: &streaming.Streaming{},
			responseGet: &speaking.Speaking{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusActive,
			},

			expectErr: false,
		},
		{
			name: "DB create error",

			customerID:    uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
			language:      "en-US",
			provider:      "elevenlabs",
			voiceID:       "",
			direction:     streaming.DirectionIncoming,

			responseExisting:  []*speaking.Speaking{},
			responseCreateErr: fmt.Errorf("db error"),

			expectErr: true,
		},
		{
			name: "streaming start error rolls back status",

			customerID:    uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
			language:      "en-US",
			provider:      "elevenlabs",
			voiceID:       "",
			direction:     streaming.DirectionIncoming,

			responseExisting:  []*speaking.Speaking{},
			responseStreamErr: fmt.Errorf("streaming error"),

			expectErr: true,
		},
		{
			name: "default provider to elevenlabs",

			customerID:    uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			referenceType: streaming.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
			language:      "en-US",
			provider:      "",
			voiceID:       "",
			direction:     streaming.DirectionIncoming,

			responseExisting: []*speaking.Speaking{},
			responseStreaming: &streaming.Streaming{},
			responseGet: &speaking.Speaking{
				Identity: commonidentity.Identity{
					CustomerID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Provider: "elevenlabs",
				Status:   speaking.StatusActive,
			},

			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

			h := &speakingHandler{
				db:               mockDB,
				streamingHandler: mockStreaming,
				podID:            "test-pod",
			}
			ctx := context.Background()

			expectFilters := map[speaking.Field]any{
				speaking.FieldCustomerID:    tt.customerID,
				speaking.FieldReferenceType: string(tt.referenceType),
				speaking.FieldReferenceID:   tt.referenceID,
				speaking.FieldDeleted:       false,
			}
			preCheck := mockDB.EXPECT().SpeakingGets(ctx, "", uint64(100), expectFilters).Return(tt.responseExisting, tt.responseExistErr)

			// Check for existing active session
			hasActive := false
			for _, s := range tt.responseExisting {
				if s.Status == speaking.StatusActive || s.Status == speaking.StatusInitiating {
					hasActive = true
					break
				}
			}

			_ = preCheck
			if tt.responseExistErr == nil && !hasActive {
				expectProvider := tt.provider
				if expectProvider == "" {
					expectProvider = string(streaming.VendorNameElevenlabs)
				}
				mockDB.EXPECT().SpeakingCreate(ctx, gomock.Any()).DoAndReturn(func(_ context.Context, s *speaking.Speaking) error {
					if s.Provider != expectProvider {
						t.Errorf("Wrong provider. expect: %s, got: %s", expectProvider, s.Provider)
					}
					if s.PodID != "test-pod" {
						t.Errorf("Wrong pod_id. expect: test-pod, got: %s", s.PodID)
					}
					return tt.responseCreateErr
				})

				if tt.responseCreateErr == nil {
					// Post-create recheck for race condition (returns no competing sessions)
					mockDB.EXPECT().SpeakingGets(ctx, "", uint64(100), expectFilters).Return([]*speaking.Speaking{}, nil).After(preCheck)

					mockStreaming.EXPECT().StartWithID(
						ctx,
						gomock.Any(),
						tt.customerID,
						tt.referenceType,
						tt.referenceID,
						tt.language,
						expectProvider,
						tt.voiceID,
						tt.direction,
					).Return(tt.responseStreaming, tt.responseStreamErr)

					if tt.responseStreamErr != nil {
						mockDB.EXPECT().SpeakingUpdate(ctx, gomock.Any(), map[speaking.Field]any{
							speaking.FieldStatus: speaking.StatusStopped,
						}).Return(nil)
					} else {
						mockDB.EXPECT().SpeakingUpdate(ctx, gomock.Any(), map[speaking.Field]any{
							speaking.FieldStatus: speaking.StatusActive,
						}).Return(tt.responseUpdateErr)

						mockDB.EXPECT().SpeakingGet(ctx, gomock.Any()).Return(tt.responseGet, tt.responseGetErr)
					}
				}
			}

			_, err := h.Create(ctx, tt.customerID, tt.referenceType, tt.referenceID, tt.language, tt.provider, tt.voiceID, tt.direction)
			if tt.expectErr && err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseSpeaking *speaking.Speaking
		responseErr      error

		expectRes *speaking.Speaking
		expectErr bool
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusActive,
			},

			expectRes: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusActive,
			},
			expectErr: false,
		},
		{
			name: "not found",

			id: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),

			responseErr: fmt.Errorf("not found"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &speakingHandler{
				db: mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().SpeakingGet(ctx, tt.id).Return(tt.responseSpeaking, tt.responseErr)

			res, err := h.Get(ctx, tt.id)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		token   string
		size    uint64
		filters map[speaking.Field]any

		responseSpeakings []*speaking.Speaking
		responseErr       error

		expectRes []*speaking.Speaking
		expectErr bool
	}{
		{
			name: "normal",

			token: "2024-01-01T00:00:00.000000Z",
			size:  10,
			filters: map[speaking.Field]any{
				speaking.FieldCustomerID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				speaking.FieldDeleted:    false,
			},

			responseSpeakings: []*speaking.Speaking{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					},
				},
			},

			expectRes: []*speaking.Speaking{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("b0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("c0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13"),
					},
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &speakingHandler{
				db: mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().SpeakingGets(ctx, tt.token, tt.size, tt.filters).Return(tt.responseSpeakings, tt.responseErr)

			res, err := h.Gets(ctx, tt.token, tt.size, tt.filters)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_Say(t *testing.T) {

	tests := []struct {
		name string

		id   uuid.UUID
		text string

		responseSpeaking *speaking.Speaking
		responseGetErr   error
		responseInitStr  *streaming.Streaming
		responseInitErr  error
		responseSayErr   error

		expectErr bool
	}{
		{
			name: "normal",

			id:   uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			text: "hello world",

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusActive,
			},
			responseInitStr: &streaming.Streaming{},

			expectErr: false,
		},
		{
			name: "not found",

			id:   uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			text: "hello world",

			responseGetErr: fmt.Errorf("not found"),

			expectErr: true,
		},
		{
			name: "session stopped rejects",

			id:   uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			text: "hello world",

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusStopped,
			},

			expectErr: true,
		},
		{
			name: "SayInit error",

			id:   uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			text: "hello world",

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusActive,
			},
			responseInitErr: fmt.Errorf("vendor not ready"),

			expectErr: true,
		},
		{
			name: "streaming SayAdd error",

			id:   uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			text: "hello world",

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusActive,
			},
			responseInitStr: &streaming.Streaming{},
			responseSayErr:  fmt.Errorf("say error"),

			expectErr: true,
		},
		{
			name: "text too long",

			id:   uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
			text: string(make([]byte, maxTextLength+1)),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

			h := &speakingHandler{
				db:               mockDB,
				streamingHandler: mockStreaming,
			}
			ctx := context.Background()

			// Text length check happens before DB call
			if len(tt.text) > maxTextLength {
				_, err := h.Say(ctx, tt.id, tt.text)
				if err == nil {
					t.Errorf("Wrong match. expect: error, got: ok")
				}
				return
			}

			mockDB.EXPECT().SpeakingGet(ctx, tt.id).Return(tt.responseSpeaking, tt.responseGetErr)

			if tt.responseGetErr == nil && tt.responseSpeaking.Status == speaking.StatusActive {
				mockStreaming.EXPECT().SayInit(ctx, tt.id, uuid.Nil).Return(tt.responseInitStr, tt.responseInitErr)

				if tt.responseInitErr == nil {
					mockStreaming.EXPECT().SayAdd(ctx, tt.id, uuid.Nil, tt.text).Return(tt.responseSayErr)
				}
			}

			_, err := h.Say(ctx, tt.id, tt.text)
			if tt.expectErr && err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Flush(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseSpeaking *speaking.Speaking
		responseGetErr   error
		responseFlushErr error

		expectErr bool
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusActive,
			},

			expectErr: false,
		},
		{
			name: "not found",

			id: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),

			responseGetErr: fmt.Errorf("not found"),

			expectErr: true,
		},
		{
			name: "session stopped rejects",

			id: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusStopped,
			},

			expectErr: true,
		},
		{
			name: "streaming SayFlush error",

			id: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusActive,
			},
			responseFlushErr: fmt.Errorf("flush error"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

			h := &speakingHandler{
				db:               mockDB,
				streamingHandler: mockStreaming,
			}
			ctx := context.Background()

			mockDB.EXPECT().SpeakingGet(ctx, tt.id).Return(tt.responseSpeaking, tt.responseGetErr)

			if tt.responseGetErr == nil && tt.responseSpeaking.Status == speaking.StatusActive {
				mockStreaming.EXPECT().SayFlush(ctx, tt.id).Return(tt.responseFlushErr)
			}

			_, err := h.Flush(ctx, tt.id)
			if tt.expectErr && err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Stop(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseSpeaking    *speaking.Speaking
		responseGetErr      error
		responseStopStreaming *streaming.Streaming
		responseStopErr     error
		responseUpdateErr   error
		responseGetAfter    *speaking.Speaking
		responseGetAfterErr error

		expectErr bool
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusActive,
			},
			responseStopStreaming: &streaming.Streaming{},
			responseGetAfter: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusStopped,
			},

			expectErr: false,
		},
		{
			name: "already stopped idempotent",

			id: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusStopped,
			},

			expectErr: false,
		},
		{
			name: "streaming stop error non-fatal",

			id: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusActive,
			},
			responseStopErr: fmt.Errorf("streaming stop error"),
			responseGetAfter: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusStopped,
			},

			expectErr: false,
		},
		{
			name: "not found",

			id: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),

			responseGetErr: fmt.Errorf("not found"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockStreaming := streaminghandler.NewMockStreamingHandler(mc)

			h := &speakingHandler{
				db:               mockDB,
				streamingHandler: mockStreaming,
			}
			ctx := context.Background()

			mockDB.EXPECT().SpeakingGet(ctx, tt.id).Return(tt.responseSpeaking, tt.responseGetErr)

			if tt.responseGetErr == nil && tt.responseSpeaking.Status != speaking.StatusStopped {
				mockStreaming.EXPECT().Stop(ctx, tt.id).Return(tt.responseStopStreaming, tt.responseStopErr)

				mockDB.EXPECT().SpeakingUpdate(ctx, tt.id, map[speaking.Field]any{
					speaking.FieldStatus: speaking.StatusStopped,
				}).Return(tt.responseUpdateErr)

				if tt.responseUpdateErr == nil {
					mockDB.EXPECT().SpeakingGet(ctx, tt.id).Return(tt.responseGetAfter, tt.responseGetAfterErr)
				}
			}

			_, err := h.Stop(ctx, tt.id)
			if tt.expectErr && err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseSpeaking  *speaking.Speaking
		responseGetErr    error
		responseDeleteErr error
		responseGetAfter  *speaking.Speaking
		responseGetAfterErr error

		expectErr bool
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),

			responseSpeaking: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status: speaking.StatusActive,
			},
			responseGetAfter: &speaking.Speaking{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),
				},
				Status:   speaking.StatusActive,
				TMDelete: "2024-01-01T00:00:00.000000Z",
			},

			expectErr: false,
		},
		{
			name: "not found",

			id: uuid.FromStringOrNil("a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"),

			responseGetErr: fmt.Errorf("not found"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &speakingHandler{
				db: mockDB,
			}
			ctx := context.Background()

			mockDB.EXPECT().SpeakingGet(ctx, tt.id).Return(tt.responseSpeaking, tt.responseGetErr)

			if tt.responseGetErr == nil {
				mockDB.EXPECT().SpeakingDelete(ctx, tt.id).Return(tt.responseDeleteErr)

				if tt.responseDeleteErr == nil {
					mockDB.EXPECT().SpeakingGet(ctx, tt.id).Return(tt.responseGetAfter, tt.responseGetAfterErr)
				}
			}

			_, err := h.Delete(ctx, tt.id)
			if tt.expectErr && err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}
