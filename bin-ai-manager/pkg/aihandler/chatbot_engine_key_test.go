package aihandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

// Test_Update_EngineKey covers the omitted / empty / value contract of engine_key on all three
// write branches of Update (prompt unchanged, changed, cleared).
func Test_Update_EngineKey(t *testing.T) {
	branches := []struct {
		name         string
		storedPrompt string
		reqPrompt    string
		history      bool
	}{
		{"prompt unchanged", "same", "same", false},
		{"prompt changed", "old", "new", true},
		{"prompt cleared", "old", "", false},
	}

	tests := []struct {
		name        string
		stored      ai.EngineModel
		storedKey   string
		model       ai.EngineModel
		key         *string
		wantError   bool
		wantWritten bool   // FieldEngineKey is part of the UPDATE
		wantValue   string // value written when wantWritten
	}{
		{"omitted keeps the stored key", "openai.gpt-5", "dummy-stored", "openai.gpt-5", nil, false, false, ""},
		{"empty string deletes", "openai.gpt-5", "dummy-stored", "openai.gpt-5", strPtr(""), false, true, ""},
		{"value replaces", "openai.gpt-5", "dummy-stored", "openai.gpt-5", strPtr("dummy-new"), false, true, "dummy-new"},
		{"omitted with a prefix change to another keyed vendor deletes the key", "openai.gpt-5", "dummy-stored", "gemini.gemini-2.5-pro", nil, false, true, ""},
		{"omitted with a prefix change to a platform model keeps the key", "openai.gpt-5", "dummy-stored", "anthropic.claude-haiku-4.5", nil, false, false, ""},
		{"omitted custom openrouter model passes with a stored key", "openrouter.vendor/model-a", "dummy-stored", "openrouter.vendor/model-a", nil, false, false, ""},
		{"omitted custom openrouter model without a stored key is rejected", "openrouter.vendor/model-a", "", "openrouter.vendor/model-a", nil, true, false, ""},
		{"empty string for a custom openrouter model is rejected", "openrouter.vendor/model-a", "dummy-stored", "openrouter.vendor/model-a", strPtr(""), true, false, ""},
		{"omitted key does not move to a custom openrouter model", "openai.gpt-5", "dummy-stored", "openrouter.vendor/model-a", nil, true, false, ""},
	}

	for _, b := range branches {
		for _, tt := range tests {
			t.Run(b.name+"/"+tt.name, func(t *testing.T) {
				mc := gomock.NewController(t)
				defer mc.Finish()

				mockDB := dbhandler.NewMockDBHandler(mc)
				mockNotify := notifyhandler.NewMockNotifyHandler(mc)
				mockNotify.EXPECT().PublishWebhookEvent(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

				stored := &ai.AI{Name: "n", EngineModel: tt.stored, EngineKey: tt.storedKey, InitPrompt: b.storedPrompt}
				stored.ID = uuid.Must(uuid.NewV4())

				if tt.wantError {
					mockDB.EXPECT().AIGet(gomock.Any(), stored.ID).Return(stored, nil).Times(1)
				} else {
					mockDB.EXPECT().AIGet(gomock.Any(), stored.ID).Return(stored, nil).Times(2)
					mockDB.EXPECT().AIUpdate(gomock.Any(), stored.ID, gomock.Any()).DoAndReturn(
						func(_ context.Context, _ uuid.UUID, fields map[ai.Field]any) error {
							v, ok := fields[ai.FieldEngineKey]
							if ok != tt.wantWritten {
								t.Errorf("FieldEngineKey written = %v, want %v", ok, tt.wantWritten)
							}
							if ok && v != tt.wantValue {
								t.Errorf("FieldEngineKey value. expect: %q, got: %v", tt.wantValue, v)
							}
							return nil
						}).Times(1)
					if b.history {
						mockDB.EXPECT().AIPromptHistoryCreate(gomock.Any(), gomock.Any()).Return(nil).Times(1)
					}
				}

				h := &aiHandler{db: mockDB, notifyHandler: mockNotify, utilHandler: utilhandler.NewUtilHandler()}

				_, err := h.Update(context.Background(), stored.ID, "n", "", ai.TypeNormal, tt.model, nil, tt.key, "", nil, uuid.Nil, b.reqPrompt,
					ai.TTSTypeNone, "", ai.STTTypeNone, "", []tool.ToolName{}, nil, false, false)
				if (err != nil) != tt.wantError {
					t.Errorf("Update() error = %v, wantError %v", err, tt.wantError)
				}
			})
		}
	}
}

// Test_buildUpdateFields_engineKey: a nil key never reaches the UPDATE, so a stale pre-read value
// (for example from a cache entry that is not yet refreshed) cannot be written back over a key that
// a concurrent request just stored.
func Test_buildUpdateFields_engineKey(t *testing.T) {
	h := &aiHandler{}
	build := func(key *string) map[ai.Field]any {
		return h.buildUpdateFields("n", "d", ai.TypeNormal, ai.EngineModelOpenaiGPT5, nil, key, uuid.Nil, "",
			ai.TTSTypeNone, "", ai.STTTypeNone, "", nil, nil, false, false, endpointWrite{})
	}

	if _, ok := build(nil)[ai.FieldEngineKey]; ok {
		t.Errorf("a nil key must not be written")
	}
	if v, ok := build(strPtr(""))[ai.FieldEngineKey]; !ok || v != "" {
		t.Errorf("an empty key must be written as empty, got: %v %v", v, ok)
	}
	if v, ok := build(strPtr("dummy-new"))[ai.FieldEngineKey]; !ok || v != "dummy-new" {
		t.Errorf("a key must be written, got: %v %v", v, ok)
	}
}
