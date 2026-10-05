package aihandler

import (
	"context"
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

func Test_Create_EngineModelPolicy(t *testing.T) {
	tests := []struct {
		name        string
		engineModel ai.EngineModel
		wantError   bool
	}{
		{"rejects_anthropic_not_in_catalog", "anthropic.claude-opus-4", true},
		{"rejects_raw_openrouter", "openrouter.meta-llama/llama-3-70b", true},
		{"rejects_internal_type", "platform_openrouter.x", true},
		{"accepts_catalog_openrouter_model", "anthropic.claude-haiku-4.5", false},
		{"accepts_direct_passthrough", "openai.gpt-4o", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			if !tt.wantError {
				created := &ai.AI{Name: "n"}
				created.ID = uuid.Must(uuid.NewV4())
				mockReq.EXPECT().DirectV1DirectCreate(gomock.Any(), gomock.Any(), dmdirect.ResourceTypeAI, gomock.Any()).Return(&dmdirect.Direct{Hash: "a1b2c3d4e5f6"}, nil).Times(1)
				mockDB.EXPECT().AICreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
				mockDB.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(created, nil).Times(1)
				mockDB.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			}

			h := &aiHandler{db: mockDB, reqHandler: mockReq, notifyHandler: mockNotify, utilHandler: utilhandler.NewUtilHandler()}

			_, err := h.Create(context.Background(), uuid.Must(uuid.NewV4()), "n", "", ai.TypeNormal, tt.engineModel, nil, "key", uuid.Nil, "prompt",
				ai.TTSTypeNone, "", ai.STTTypeNone, "", []tool.ToolName{}, nil, false, false)
			if (err != nil) != tt.wantError {
				t.Errorf("Create() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func Test_Update_EngineModelValidatedOnlyOnChange(t *testing.T) {
	tests := []struct {
		name        string
		stored      ai.EngineModel
		requested   ai.EngineModel
		wantError   bool
		expectWrite bool
	}{
		{"unchanged_legacy_value_passes", "anthropic.claude-opus-4", "anthropic.claude-opus-4", false, true},
		{"unchanged_legacy_openrouter_value_passes", "openrouter.x/y", "openrouter.x/y", false, true},
		{"change_to_invalid_fails", "openai.gpt-5", "anthropic.claude-opus-4", true, false},
		{"change_from_legacy_to_invalid_fails", "anthropic.claude-opus-4", "unknown.model", true, false},
		{"change_to_catalog_passes", "anthropic.claude-opus-4", "anthropic.claude-haiku-4.5", false, true},
		{"change_to_direct_passthrough_passes", "openai.gpt-5", "openai.gpt-4o", false, true},
		{"stored_empty_requested_empty_passes", "", "", false, true},
		{"change_to_empty_fails", "openai.gpt-5", "", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			stored := &ai.AI{Name: "n", EngineModel: tt.stored}
			stored.ID = uuid.Must(uuid.NewV4())

			if tt.expectWrite {
				// pre-fetch + post-update fetch
				mockDB.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(stored, nil).Times(2)
				mockDB.EXPECT().AIUpdate(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1)
				mockDB.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
			} else {
				mockDB.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(stored, nil).Times(1)
			}

			h := &aiHandler{db: mockDB, notifyHandler: mockNotify, utilHandler: utilhandler.NewUtilHandler()}

			_, err := h.Update(context.Background(), stored.ID, "n", "", ai.TypeNormal, tt.requested, nil, "key", uuid.Nil, "prompt",
				ai.TTSTypeNone, "", ai.STTTypeNone, "", []tool.ToolName{}, nil, false, false)
			if (err != nil) != tt.wantError {
				t.Errorf("Update() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
