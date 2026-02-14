package pipecatcallhandler

import (
	"context"
	"errors"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
	"monorepo/bin-pipecat-manager/pkg/dbhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func TestCreate(t *testing.T) {
	tests := []struct {
		name string

		id            uuid.UUID
		customerID    uuid.UUID
		activeflowID  uuid.UUID
		referenceType pipecatcall.ReferenceType
		referenceID   uuid.UUID
		llmType       pipecatcall.LLMType
		llmMessages   []map[string]any
		sttType       pipecatcall.STTType
		sttLanguage   string
		ttsType       pipecatcall.TTSType
		ttsLanguage   string
		ttsVoiceID    string

		dbCreateErr error
		dbGetErr    error

		expectErr bool
	}{
		{
			name: "success",

			id:            uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
			customerID:    uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
			activeflowID:  uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),
			referenceType: pipecatcall.ReferenceTypeAICall,
			referenceID:   uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c"),
			llmType:       pipecatcall.LLMType("openai.gpt-4"),
			llmMessages: []map[string]any{
				{"role": "system", "content": "test"},
			},
			sttType:     pipecatcall.STTTypeDeepgram,
			sttLanguage: "en-US",
			ttsType:     pipecatcall.TTSTypeElevenLabs,
			ttsLanguage: "en-US",
			ttsVoiceID:  "test-voice-id",

			dbCreateErr: nil,
			dbGetErr:    nil,

			expectErr: false,
		},
		{
			name: "db create error",

			id:         uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
			customerID: uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),

			dbCreateErr: errors.New("db error"),
			dbGetErr:    nil,

			expectErr: true,
		},
		{
			name: "db get error",

			id:         uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
			customerID: uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),

			dbCreateErr: nil,
			dbGetErr:    errors.New("db error"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &pipecatcallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				hostID:        "test-host",
			}

			ctx := context.Background()

			if tt.dbCreateErr == nil {
				mockDB.EXPECT().PipecatcallCreate(ctx, gomock.Any()).Return(tt.dbCreateErr)
			} else {
				mockDB.EXPECT().PipecatcallCreate(ctx, gomock.Any()).Return(tt.dbCreateErr)
			}

			if tt.dbCreateErr == nil {
				if tt.dbGetErr == nil {
					expectedPipecatcall := &pipecatcall.Pipecatcall{
						Identity: commonidentity.Identity{
							ID:         tt.id,
							CustomerID: tt.customerID,
						},
					}
					mockDB.EXPECT().PipecatcallGet(ctx, tt.id).Return(expectedPipecatcall, tt.dbGetErr)
					mockNotify.EXPECT().PublishEvent(ctx, pipecatcall.EventTypeCreated, expectedPipecatcall)
				} else {
					mockDB.EXPECT().PipecatcallGet(ctx, tt.id).Return(nil, tt.dbGetErr)
				}
			}

			result, err := h.Create(
				ctx,
				tt.id,
				tt.customerID,
				tt.activeflowID,
				tt.referenceType,
				tt.referenceID,
				tt.llmType,
				tt.llmMessages,
				tt.sttType,
				tt.sttLanguage,
				tt.ttsType,
				tt.ttsLanguage,
				tt.ttsVoiceID,
			)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Create() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Create() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Create() returned nil result")
				return
			}

			if result.ID != tt.id {
				t.Errorf("Create() ID = %v, want %v", result.ID, tt.id)
			}
		})
	}
}

func TestGet(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		dbGetResult *pipecatcall.Pipecatcall
		dbGetErr    error

		expectErr bool
	}{
		{
			name: "success",

			id: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),

			dbGetResult: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				},
			},
			dbGetErr: nil,

			expectErr: false,
		},
		{
			name: "db error",

			id: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),

			dbGetResult: nil,
			dbGetErr:    errors.New("db error"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &pipecatcallHandler{
				db: mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().PipecatcallGet(ctx, tt.id).Return(tt.dbGetResult, tt.dbGetErr)

			result, err := h.Get(ctx, tt.id)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Get() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Get() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Get() returned nil result")
				return
			}

			if result.ID != tt.id {
				t.Errorf("Get() ID = %v, want %v", result.ID, tt.id)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name string

		id uuid.UUID

		dbDeleteErr error
		dbGetResult *pipecatcall.Pipecatcall
		dbGetErr    error

		expectErr bool
	}{
		{
			name: "success",

			id: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),

			dbDeleteErr: nil,
			dbGetResult: &pipecatcall.Pipecatcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				},
			},
			dbGetErr: nil,

			expectErr: false,
		},
		{
			name: "db delete error",

			id: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),

			dbDeleteErr: errors.New("db error"),
			dbGetResult: nil,
			dbGetErr:    nil,

			expectErr: true,
		},
		{
			name: "db get error",

			id: uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),

			dbDeleteErr: nil,
			dbGetResult: nil,
			dbGetErr:    errors.New("db error"),

			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := &pipecatcallHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockDB.EXPECT().PipecatcallDelete(ctx, tt.id).Return(tt.dbDeleteErr)

			if tt.dbDeleteErr == nil {
				mockDB.EXPECT().PipecatcallGet(ctx, tt.id).Return(tt.dbGetResult, tt.dbGetErr)
				if tt.dbGetErr == nil {
					mockNotify.EXPECT().PublishEvent(ctx, pipecatcall.EventTypeDeleted, tt.dbGetResult)
				}
			}

			result, err := h.Delete(ctx, tt.id)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Delete() expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Delete() unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Delete() returned nil result")
				return
			}

			if result.ID != tt.id {
				t.Errorf("Delete() ID = %v, want %v", result.ID, tt.id)
			}
		})
	}
}
