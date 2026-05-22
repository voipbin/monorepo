package aihandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	dmdirect "monorepo/bin-direct-manager/models/direct"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func float64Ptr(v float64) *float64 {
	return &v
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name             string
		customerID       uuid.UUID
		aiName           string
		engineModel      ai.EngineModel
		ttsType          ai.TTSType
		sttType          ai.STTType
		vadConfig        *ai.VADConfig
		smartTurnEnabled bool
		setupMock        func(*dbhandler.MockDBHandler, *requesthandler.MockRequestHandler)
		wantError        bool
		errorMsg         string
	}{
		{
			name:        "creates_ai_with_valid_model",
			customerID:  uuid.Must(uuid.NewV4()),
			aiName:      "Test AI",
			engineModel: ai.EngineModelOpenaiGPT5,
			ttsType:     ai.TTSTypeNone,
			sttType:     ai.STTTypeNone,
			setupMock: func(m *dbhandler.MockDBHandler, r *requesthandler.MockRequestHandler) {
				testAI := &ai.AI{Name: "Test AI"}
				testAI.ID = uuid.Must(uuid.NewV4())
				r.EXPECT().DirectV1DirectCreate(gomock.Any(), gomock.Any(), dmdirect.ResourceTypeAI, gomock.Any()).Return(&dmdirect.Direct{Hash: "a1b2c3d4e5f6"}, nil).Times(1)
				m.EXPECT().AICreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(testAI, nil).Times(1)
				// initPrompt is "Test prompt" (non-empty) → history recorded
				m.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			wantError: false,
		},
		{
			name:        "fails_with_invalid_model",
			customerID:  uuid.Must(uuid.NewV4()),
			aiName:      "Test AI",
			engineModel: ai.EngineModel("invalid.model"),
			ttsType:     ai.TTSTypeNone,
			sttType:     ai.STTTypeNone,
			setupMock: func(m *dbhandler.MockDBHandler, r *requesthandler.MockRequestHandler) {
				// Should not call database
			},
			wantError: true,
			errorMsg:  "invalid engine model",
		},
		{
			name:        "fails_with_invalid_tts_type",
			customerID:  uuid.Must(uuid.NewV4()),
			aiName:      "Test AI",
			engineModel: ai.EngineModelOpenaiGPT5,
			ttsType:     ai.TTSType("gcp"),
			sttType:     ai.STTTypeNone,
			setupMock: func(m *dbhandler.MockDBHandler, r *requesthandler.MockRequestHandler) {
				// Should not call database
			},
			wantError: true,
			errorMsg:  "invalid tts_type",
		},
		{
			name:        "fails_with_invalid_stt_type",
			customerID:  uuid.Must(uuid.NewV4()),
			aiName:      "Test AI",
			engineModel: ai.EngineModelOpenaiGPT5,
			ttsType:     ai.TTSTypeNone,
			sttType:     ai.STTType("gcp"),
			setupMock: func(m *dbhandler.MockDBHandler, r *requesthandler.MockRequestHandler) {
				// Should not call database
			},
			wantError: true,
			errorMsg:  "invalid stt_type",
		},
		{
			name:        "handles_database_error",
			customerID:  uuid.Must(uuid.NewV4()),
			aiName:      "Test AI",
			engineModel: ai.EngineModelGeminiGemini2Dot5Flash,
			ttsType:     ai.TTSTypeNone,
			sttType:     ai.STTTypeNone,
			setupMock: func(m *dbhandler.MockDBHandler, r *requesthandler.MockRequestHandler) {
				r.EXPECT().DirectV1DirectCreate(gomock.Any(), gomock.Any(), dmdirect.ResourceTypeAI, gomock.Any()).Return(&dmdirect.Direct{Hash: "a1b2c3d4e5f6"}, nil).Times(1)
				m.EXPECT().AICreate(gomock.Any(), gomock.Any()).Return(errors.New("database error")).Times(1)
				r.EXPECT().DirectV1DirectDelete(gomock.Any(), gomock.Any()).Return(nil, nil).Times(1)
			},
			wantError: true,
			errorMsg:  "could not create ai",
		},
		{
			name:        "fails_with_invalid_vad_config",
			customerID:  uuid.Must(uuid.NewV4()),
			aiName:      "Test AI",
			engineModel: ai.EngineModelOpenaiGPT5,
			ttsType:     ai.TTSTypeNone,
			sttType:     ai.STTTypeNone,
			vadConfig:   &ai.VADConfig{Confidence: float64Ptr(1.5)},
			setupMock: func(m *dbhandler.MockDBHandler, r *requesthandler.MockRequestHandler) {
				// Should not call database
			},
			wantError: true,
			errorMsg:  "invalid vad_config",
		},
		{
			name:        "creates_ai_with_valid_vad_config",
			customerID:  uuid.Must(uuid.NewV4()),
			aiName:      "Test AI with VAD",
			engineModel: ai.EngineModelOpenaiGPT5,
			ttsType:     ai.TTSTypeNone,
			sttType:     ai.STTTypeNone,
			vadConfig:   &ai.VADConfig{Confidence: float64Ptr(0.8), StopSecs: float64Ptr(0.5)},
			setupMock: func(m *dbhandler.MockDBHandler, r *requesthandler.MockRequestHandler) {
				testAI := &ai.AI{Name: "Test AI with VAD"}
				testAI.ID = uuid.Must(uuid.NewV4())
				r.EXPECT().DirectV1DirectCreate(gomock.Any(), gomock.Any(), dmdirect.ResourceTypeAI, gomock.Any()).Return(&dmdirect.Direct{Hash: "a1b2c3d4e5f6"}, nil).Times(1)
				m.EXPECT().AICreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(testAI, nil).Times(1)
				// initPrompt is "Test prompt" (non-empty) → history recorded
				m.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			wantError: false,
		},
		{
			name:             "creates_ai_with_smart_turn_enabled",
			customerID:       uuid.Must(uuid.NewV4()),
			aiName:           "Test AI with Smart Turn",
			engineModel:      ai.EngineModelOpenaiGPT5,
			ttsType:          ai.TTSTypeNone,
			sttType:          ai.STTTypeNone,
			smartTurnEnabled: true,
			setupMock: func(m *dbhandler.MockDBHandler, r *requesthandler.MockRequestHandler) {
				testAI := &ai.AI{Name: "Test AI with Smart Turn", SmartTurnEnabled: true}
				testAI.ID = uuid.Must(uuid.NewV4())
				r.EXPECT().DirectV1DirectCreate(gomock.Any(), gomock.Any(), dmdirect.ResourceTypeAI, gomock.Any()).Return(&dmdirect.Direct{Hash: "a1b2c3d4e5f6"}, nil).Times(1)
				m.EXPECT().AICreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(testAI, nil).Times(1)
				// initPrompt is "Test prompt" (non-empty) → history recorded
				m.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := dbhandler.NewMockDBHandler(ctrl)
			mockReq := requesthandler.NewMockRequestHandler(ctrl)
			mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			tt.setupMock(mockDB, mockReq)

			h := &aiHandler{
				db:            mockDB,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				utilHandler:   utilhandler.NewUtilHandler(),
			}

			result, err := h.Create(
				context.Background(),
				tt.customerID,
				tt.aiName,
				"Test detail",
				tt.engineModel,
				nil,
				"test-key",
				uuid.Nil,
				"Test prompt",
				tt.ttsType,
				"",
				tt.sttType,
				"",
				[]tool.ToolName{},
				tt.vadConfig,
				tt.smartTurnEnabled,
			)

			if (err != nil) != tt.wantError {
				t.Errorf("Create() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError && tt.errorMsg != "" && err != nil {
				if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Create() error message = %v, want to contain %v", err.Error(), tt.errorMsg)
				}
			}

			if !tt.wantError && result == nil {
				t.Error("Create() returned nil result when expecting success")
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	tests := []struct {
		name             string
		aiID             uuid.UUID
		aiName           string
		engineModel      ai.EngineModel
		ttsType          ai.TTSType
		sttType          ai.STTType
		vadConfig        *ai.VADConfig
		smartTurnEnabled bool
		setupMock        func(*dbhandler.MockDBHandler)
		wantError        bool
		errorMsg         string
	}{
		{
			name:        "updates_ai_with_valid_model",
			aiID:        uuid.Must(uuid.NewV4()),
			aiName:      "Updated AI",
			engineModel: ai.EngineModelOpenaiGPT5Mini,
			ttsType:     ai.TTSTypeOpenAI,
			sttType:     ai.STTTypeDeepgram,
			setupMock: func(m *dbhandler.MockDBHandler) {
				updatedAI := &ai.AI{Name: "Updated AI"}
				updatedAI.ID = uuid.Must(uuid.NewV4())
				// pre-fetch (initPrompt is non-empty "Updated prompt") + post-update in dbUpdate
				m.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(updatedAI, nil).Times(2)
				m.EXPECT().AIUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			wantError: false,
		},
		{
			name:        "fails_with_invalid_model",
			aiID:        uuid.Must(uuid.NewV4()),
			aiName:      "Updated AI",
			engineModel: ai.EngineModel("unknown.invalid"),
			ttsType:     ai.TTSTypeOpenAI,
			sttType:     ai.STTTypeDeepgram,
			setupMock: func(m *dbhandler.MockDBHandler) {
				// Should not call database
			},
			wantError: true,
			errorMsg:  "invalid engine model",
		},
		{
			name:        "fails_with_invalid_tts_type",
			aiID:        uuid.Must(uuid.NewV4()),
			aiName:      "Updated AI",
			engineModel: ai.EngineModelOpenaiGPT5,
			ttsType:     ai.TTSType("gcp"),
			sttType:     ai.STTTypeDeepgram,
			setupMock: func(m *dbhandler.MockDBHandler) {
				// Should not call database
			},
			wantError: true,
			errorMsg:  "invalid tts_type",
		},
		{
			name:        "fails_with_invalid_stt_type",
			aiID:        uuid.Must(uuid.NewV4()),
			aiName:      "Updated AI",
			engineModel: ai.EngineModelOpenaiGPT5,
			ttsType:     ai.TTSTypeOpenAI,
			sttType:     ai.STTType("gcp"),
			setupMock: func(m *dbhandler.MockDBHandler) {
				// Should not call database
			},
			wantError: true,
			errorMsg:  "invalid stt_type",
		},
		{
			name:        "handles_database_error",
			aiID:        uuid.Must(uuid.NewV4()),
			aiName:      "Updated AI",
			engineModel: ai.EngineModelGeminiGeminiProLatest,
			ttsType:     ai.TTSTypeOpenAI,
			sttType:     ai.STTTypeDeepgram,
			setupMock: func(m *dbhandler.MockDBHandler) {
				// pre-fetch succeeds (initPrompt is non-empty "Updated prompt"), then AIUpdate fails
				currentAI := &ai.AI{}
				currentAI.ID = uuid.Must(uuid.NewV4())
				m.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(currentAI, nil).Times(1)
				m.EXPECT().AIUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("update failed")).Times(1)
			},
			wantError: true,
			errorMsg:  "could not update ai",
		},
		{
			name:        "fails_with_invalid_vad_config",
			aiID:        uuid.Must(uuid.NewV4()),
			aiName:      "Updated AI",
			engineModel: ai.EngineModelOpenaiGPT5,
			ttsType:     ai.TTSTypeOpenAI,
			sttType:     ai.STTTypeDeepgram,
			vadConfig:   &ai.VADConfig{MinVolume: float64Ptr(-0.1)},
			setupMock: func(m *dbhandler.MockDBHandler) {
				// Should not call database
			},
			wantError: true,
			errorMsg:  "invalid vad_config",
		},
		{
			name:        "updates_ai_with_valid_vad_config",
			aiID:        uuid.Must(uuid.NewV4()),
			aiName:      "Updated AI with VAD",
			engineModel: ai.EngineModelOpenaiGPT5,
			ttsType:     ai.TTSTypeOpenAI,
			sttType:     ai.STTTypeDeepgram,
			vadConfig:   &ai.VADConfig{StopSecs: float64Ptr(0.3), MinVolume: float64Ptr(0.5)},
			setupMock: func(m *dbhandler.MockDBHandler) {
				updatedAI := &ai.AI{Name: "Updated AI with VAD"}
				updatedAI.ID = uuid.Must(uuid.NewV4())
				// pre-fetch (initPrompt non-empty) + post-update in dbUpdate
				m.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(updatedAI, nil).Times(2)
				m.EXPECT().AIUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			wantError: false,
		},
		{
			name:             "updates_ai_with_smart_turn_enabled",
			aiID:             uuid.Must(uuid.NewV4()),
			aiName:           "Updated AI with Smart Turn",
			engineModel:      ai.EngineModelOpenaiGPT5,
			ttsType:          ai.TTSTypeOpenAI,
			sttType:          ai.STTTypeDeepgram,
			smartTurnEnabled: true,
			setupMock: func(m *dbhandler.MockDBHandler) {
				updatedAI := &ai.AI{Name: "Updated AI with Smart Turn", SmartTurnEnabled: true}
				updatedAI.ID = uuid.Must(uuid.NewV4())
				// pre-fetch (initPrompt non-empty) + post-update in dbUpdate
				m.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(updatedAI, nil).Times(2)
				m.EXPECT().AIUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockDB := dbhandler.NewMockDBHandler(ctrl)
			mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			tt.setupMock(mockDB)

			h := &aiHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				utilHandler:   utilhandler.NewUtilHandler(),
			}

			result, err := h.Update(
				context.Background(),
				tt.aiID,
				tt.aiName,
				"Updated detail",
				tt.engineModel,
				nil,
				"updated-key",
				uuid.Nil,
				"Updated prompt",
				tt.ttsType,
				"voice-id",
				tt.sttType,
				"",
				[]tool.ToolName{},
				tt.vadConfig,
				tt.smartTurnEnabled,
			)

			if (err != nil) != tt.wantError {
				t.Errorf("Update() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if tt.wantError && tt.errorMsg != "" && err != nil {
				if !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Update() error message = %v, want to contain %v", err.Error(), tt.errorMsg)
				}
			}

			if !tt.wantError && result == nil {
				t.Error("Update() returned nil result when expecting success")
			}
		})
	}
}

func TestCreate_RecordsPromptHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	createdAI := &ai.AI{Name: "Helpful AI"}
	createdAI.ID = uuid.Must(uuid.NewV4())

	mockReq.EXPECT().DirectV1DirectCreate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&dmdirect.Direct{Hash: "abc123"}, nil).Times(1)
	mockDB.EXPECT().AICreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockDB.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(createdAI, nil).Times(1)
	mockDB.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	h := &aiHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	_, err := h.Create(
		context.Background(),
		uuid.Must(uuid.NewV4()),
		"Helpful AI",
		"",
		ai.EngineModelOpenaiGPT5,
		nil,
		"key",
		uuid.Nil,
		"You are helpful.",
		ai.TTSTypeNone,
		"",
		ai.STTTypeNone,
		"",
		nil,
		nil,
		false,
	)
	if err != nil {
		t.Errorf("Create() unexpected error: %v", err)
	}
}

func TestCreate_EmptyPrompt_NoHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	createdAI := &ai.AI{Name: "AI no prompt"}
	createdAI.ID = uuid.Must(uuid.NewV4())

	mockReq.EXPECT().DirectV1DirectCreate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&dmdirect.Direct{Hash: "abc123"}, nil).Times(1)
	mockDB.EXPECT().AICreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockDB.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(createdAI, nil).Times(1)
	// AIPromptHistoryCreate must NOT be called

	h := &aiHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	_, err := h.Create(
		context.Background(),
		uuid.Must(uuid.NewV4()),
		"AI no prompt",
		"",
		ai.EngineModelOpenaiGPT5,
		nil,
		"key",
		uuid.Nil,
		"",
		ai.TTSTypeNone,
		"",
		ai.STTTypeNone,
		"",
		nil,
		nil,
		false,
	)
	if err != nil {
		t.Errorf("Create() unexpected error: %v", err)
	}
}

func TestUpdate_NewPrompt_RecordsHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	aiID := uuid.Must(uuid.NewV4())
	currentAI := &ai.AI{Name: "My AI"}
	currentAI.ID = aiID
	currentAI.InitPrompt = "old prompt"

	updatedAI := &ai.AI{Name: "My AI"}
	updatedAI.ID = aiID
	updatedAI.InitPrompt = "new prompt"

	// First AIGet is pre-fetch (before dbUpdate), second is post-update (inside dbUpdate)
	gomock.InOrder(
		mockDB.EXPECT().AIGet(gomock.Any(), aiID).Return(currentAI, nil).Times(1),
		mockDB.EXPECT().AIUpdate(gomock.Any(), aiID, gomock.Any()).Return(nil).Times(1),
		mockDB.EXPECT().AIGet(gomock.Any(), aiID).Return(updatedAI, nil).Times(1),
		mockDB.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1),
	)

	h := &aiHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	_, err := h.Update(
		context.Background(),
		aiID,
		"My AI",
		"",
		ai.EngineModelOpenaiGPT5,
		nil,
		"key",
		uuid.Nil,
		"new prompt",
		ai.TTSTypeNone,
		"",
		ai.STTTypeNone,
		"",
		nil,
		nil,
		false,
	)
	if err != nil {
		t.Errorf("Update() unexpected error: %v", err)
	}
}

func TestUpdate_SamePrompt_NoHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	aiID := uuid.Must(uuid.NewV4())
	currentAI := &ai.AI{Name: "My AI"}
	currentAI.ID = aiID
	currentAI.InitPrompt = "same prompt"

	updatedAI := &ai.AI{Name: "My AI"}
	updatedAI.ID = aiID
	updatedAI.InitPrompt = "same prompt"

	// First AIGet is pre-fetch, second is post-update (inside dbUpdate); no AIPromptHistoryCreate
	gomock.InOrder(
		mockDB.EXPECT().AIGet(gomock.Any(), aiID).Return(currentAI, nil).Times(1),
		mockDB.EXPECT().AIUpdate(gomock.Any(), aiID, gomock.Any()).Return(nil).Times(1),
		mockDB.EXPECT().AIGet(gomock.Any(), aiID).Return(updatedAI, nil).Times(1),
	)

	h := &aiHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	_, err := h.Update(
		context.Background(),
		aiID,
		"My AI",
		"",
		ai.EngineModelOpenaiGPT5,
		nil,
		"key",
		uuid.Nil,
		"same prompt",
		ai.TTSTypeNone,
		"",
		ai.STTTypeNone,
		"",
		nil,
		nil,
		false,
	)
	if err != nil {
		t.Errorf("Update() unexpected error: %v", err)
	}
}

func TestUpdate_EmptyPrompt_NoPreFetch_NoHistory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	aiID := uuid.Must(uuid.NewV4())
	updatedAI := &ai.AI{Name: "My AI"}
	updatedAI.ID = aiID

	// No pre-fetch AIGet (empty prompt skips it); only post-update AIGet inside dbUpdate
	// No AIPromptHistoryCreate
	mockDB.EXPECT().AIUpdate(gomock.Any(), aiID, gomock.Any()).Return(nil).Times(1)
	mockDB.EXPECT().AIGet(gomock.Any(), aiID).Return(updatedAI, nil).Times(1)

	h := &aiHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	_, err := h.Update(
		context.Background(),
		aiID,
		"My AI",
		"",
		ai.EngineModelOpenaiGPT5,
		nil,
		"key",
		uuid.Nil,
		"",
		ai.TTSTypeNone,
		"",
		ai.STTTypeNone,
		"",
		nil,
		nil,
		false,
	)
	if err != nil {
		t.Errorf("Update() unexpected error: %v", err)
	}
}

func TestCreate_HistoryFails_CreateSucceeds(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	mockReq := requesthandler.NewMockRequestHandler(ctrl)
	mockNotify := notifyhandler.NewMockNotifyHandler(ctrl)
	mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	createdAI := &ai.AI{Name: "AI with history failure"}
	createdAI.ID = uuid.Must(uuid.NewV4())

	mockReq.EXPECT().DirectV1DirectCreate(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(&dmdirect.Direct{Hash: "abc123"}, nil).Times(1)
	mockDB.EXPECT().AICreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockDB.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(createdAI, nil).Times(1)
	// History create fails, but Create should still succeed (best-effort)
	mockDB.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).Return(errors.New("history db error")).Times(1)

	h := &aiHandler{
		db:            mockDB,
		reqHandler:    mockReq,
		notifyHandler: mockNotify,
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	result, err := h.Create(
		context.Background(),
		uuid.Must(uuid.NewV4()),
		"AI with history failure",
		"",
		ai.EngineModelOpenaiGPT5,
		nil,
		"key",
		uuid.Nil,
		"Some prompt",
		ai.TTSTypeNone,
		"",
		ai.STTTypeNone,
		"",
		nil,
		nil,
		false,
	)
	if err != nil {
		t.Errorf("Create() should succeed even when history fails, got error: %v", err)
	}
	if result == nil {
		t.Error("Create() returned nil result when expecting success")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
