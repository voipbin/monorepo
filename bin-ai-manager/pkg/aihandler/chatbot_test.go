package aihandler

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func TestCreate(t *testing.T) {
	tests := []struct {
		name        string
		customerID  uuid.UUID
		aiName      string
		engineModel ai.EngineModel
		setupMock   func(*dbhandler.MockDBHandler)
		wantError   bool
		errorMsg    string
	}{
		{
			name:        "creates_ai_with_valid_model",
			customerID:  uuid.Must(uuid.NewV4()),
			aiName:      "Test AI",
			engineModel: ai.EngineModelOpenaiGPT4O,
			setupMock: func(m *dbhandler.MockDBHandler) {
				testAI := &ai.AI{Name: "Test AI"}
				testAI.ID = uuid.Must(uuid.NewV4())
				m.EXPECT().AICreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(testAI, nil).Times(1)
			},
			wantError: false,
		},
		{
			name:        "fails_with_invalid_model",
			customerID:  uuid.Must(uuid.NewV4()),
			aiName:      "Test AI",
			engineModel: ai.EngineModel("invalid.model"),
			setupMock: func(m *dbhandler.MockDBHandler) {
				// Should not call database
			},
			wantError: true,
			errorMsg:  "invalid engine model",
		},
		{
			name:        "handles_database_error",
			customerID:  uuid.Must(uuid.NewV4()),
			aiName:      "Test AI",
			engineModel: ai.EngineModelDialogflowCX,
			setupMock: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().AICreate(gomock.Any(), gomock.Any()).Return(errors.New("database error")).Times(1)
			},
			wantError: true,
			errorMsg:  "could not create ai",
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

			result, err := h.Create(
				context.Background(),
				tt.customerID,
				tt.aiName,
				"Test detail",
				ai.EngineTypeNone,
				tt.engineModel,
				nil,
				"test-key",
				"Test prompt",
				ai.TTSTypeNone,
				"",
				ai.STTTypeNone,
				[]tool.ToolName{},
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
		name        string
		aiID        uuid.UUID
		aiName      string
		engineModel ai.EngineModel
		setupMock   func(*dbhandler.MockDBHandler)
		wantError   bool
		errorMsg    string
	}{
		{
			name:        "updates_ai_with_valid_model",
			aiID:        uuid.Must(uuid.NewV4()),
			aiName:      "Updated AI",
			engineModel: ai.EngineModelOpenaiGPT4OMini,
			setupMock: func(m *dbhandler.MockDBHandler) {
				updatedAI := &ai.AI{Name: "Updated AI"}
				updatedAI.ID = uuid.Must(uuid.NewV4())
				m.EXPECT().AIUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				m.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(updatedAI, nil).Times(1)
			},
			wantError: false,
		},
		{
			name:        "fails_with_invalid_model",
			aiID:        uuid.Must(uuid.NewV4()),
			aiName:      "Updated AI",
			engineModel: ai.EngineModel("unknown.invalid"),
			setupMock: func(m *dbhandler.MockDBHandler) {
				// Should not call database
			},
			wantError: true,
			errorMsg:  "invalid engine model",
		},
		{
			name:        "handles_database_error",
			aiID:        uuid.Must(uuid.NewV4()),
			aiName:      "Updated AI",
			engineModel: ai.EngineModelDialogflowES,
			setupMock: func(m *dbhandler.MockDBHandler) {
				m.EXPECT().AIUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("update failed")).Times(1)
			},
			wantError: true,
			errorMsg:  "could not update ai",
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
			}

			result, err := h.Update(
				context.Background(),
				tt.aiID,
				tt.aiName,
				"Updated detail",
				ai.EngineTypeNone,
				tt.engineModel,
				nil,
				"updated-key",
				"Updated prompt",
				ai.TTSTypeOpenAI,
				"voice-id",
				ai.STTTypeDeepgram,
				[]tool.ToolName{},
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
