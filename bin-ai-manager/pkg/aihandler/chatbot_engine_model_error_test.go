package aihandler

import (
	"context"
	stderrors "errors"
	"net/http"
	"strings"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func assertInvalidEngineModel(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("expected *cerrors.VoipbinError, got %T: %v", err, err)
	}
	if ve.Status != cerrors.StatusInvalidArgument {
		t.Errorf("status = %v, want InvalidArgument", ve.Status)
	}
	if ve.Reason != "INVALID_ENGINE_MODEL" {
		t.Errorf("reason = %q, want INVALID_ENGINE_MODEL", ve.Reason)
	}
	if got := cerrors.HTTPStatusFor(ve.Status); got != http.StatusBadRequest {
		t.Errorf("http status = %d, want 400", got)
	}
	resp, e := cerrors.ToResponse(ve)
	if e != nil || resp.StatusCode != http.StatusBadRequest {
		t.Errorf("ToResponse = %v, %v; want 400", resp, e)
	}

	lower := strings.ToLower(ve.Message)
	for _, banned := range []string{"openrouter", "platform_", "api key", "engine_key"} {
		if strings.Contains(lower, banned) {
			t.Errorf("message leaks internal term %q: %s", banned, ve.Message)
		}
	}
	if !strings.Contains(ve.Message, "/ai_models") {
		t.Errorf("message should point to GET /ai_models: %s", ve.Message)
	}
}

func Test_Create_invalidEngineModelIsTyped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h := &aiHandler{
		db:            dbhandler.NewMockDBHandler(mc),
		reqHandler:    requesthandler.NewMockRequestHandler(mc),
		notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	_, err := h.Create(context.Background(), uuid.Must(uuid.NewV4()), "n", "d", ai.TypeNormal,
		ai.EngineModel("invalid.model"), nil, "", uuid.Nil, "", ai.TTSTypeNone, "", ai.STTTypeNone, "",
		[]tool.ToolName{}, nil, false, false)
	assertInvalidEngineModel(t, err)
}

func Test_Update_invalidEngineModelIsTyped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockDB.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(&ai.AI{EngineModel: ai.EngineModelOpenaiGPT5}, nil).Times(1)

	h := &aiHandler{
		db:            mockDB,
		reqHandler:    requesthandler.NewMockRequestHandler(mc),
		notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
		utilHandler:   utilhandler.NewUtilHandler(),
	}

	_, err := h.Update(context.Background(), uuid.Must(uuid.NewV4()), "n", "d", ai.TypeNormal,
		ai.EngineModel("unknown.invalid"), nil, "", uuid.Nil, "", ai.TTSTypeNone, "", ai.STTTypeNone, "",
		[]tool.ToolName{}, nil, false, false)
	assertInvalidEngineModel(t, err)
}

func assertInvalidCustomModelID(t *testing.T, err error, leaked string) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("expected *cerrors.VoipbinError, got %T: %v", err, err)
	}
	if ve.Status != cerrors.StatusInvalidArgument || ve.Reason != "INVALID_ENGINE_MODEL" {
		t.Errorf("status = %v, reason = %q, want InvalidArgument INVALID_ENGINE_MODEL", ve.Status, ve.Reason)
	}
	if got := cerrors.HTTPStatusFor(ve.Status); got != http.StatusBadRequest {
		t.Errorf("http status = %d, want 400", got)
	}

	if leaked != "" && strings.Contains(ve.Message, leaked) {
		t.Errorf("message echoes the input %q: %s", leaked, ve.Message)
	}
	lower := strings.ToLower(ve.Message)
	for _, banned := range []string{"platform_", "engine_key"} {
		if strings.Contains(lower, banned) {
			t.Errorf("message leaks internal term %q: %s", banned, ve.Message)
		}
	}
}

func assertEngineKeyRequired(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("expected *cerrors.VoipbinError, got %T: %v", err, err)
	}
	if ve.Status != cerrors.StatusInvalidArgument || ve.Reason != "ENGINE_KEY_REQUIRED" {
		t.Errorf("status = %v, reason = %q, want InvalidArgument ENGINE_KEY_REQUIRED", ve.Status, ve.Reason)
	}
	if got := cerrors.HTTPStatusFor(ve.Status); got != http.StatusBadRequest {
		t.Errorf("http status = %d, want 400", got)
	}
	resp, e := cerrors.ToResponse(ve)
	if e != nil || resp.StatusCode != http.StatusBadRequest {
		t.Errorf("ToResponse = %v, %v; want 400", resp, e)
	}
}

func Test_Create_customModelErrors(t *testing.T) {
	tests := []struct {
		name        string
		engineModel ai.EngineModel
		engineKey   string
		leaked      string // input that must not be echoed, empty to skip
		expectKey   bool   // expect ENGINE_KEY_REQUIRED, else a custom model ID error
	}{
		{"key_required", "openrouter.vendor/model-a", "", "", true},
		{"key_required_blank", "openrouter.vendor/model-a", "  ", "", true},
		{"pasted_key_without_slash", "openrouter.dummy-key-not-real", "dummy-new-key", "dummy-key-not-real", false},
		{"pasted_key_two_slashes", "openrouter.dummy-key-not-real/a/b", "dummy-new-key", "dummy-key-not-real", false},
		{"variant_suffix", "openrouter.zz-author/zz-model:free", "dummy-new-key", "zz-author", false},
		{"router_id", "openrouter.openrouter/auto", "dummy-new-key", "", false},
		{"upper_prefix_variant", "OpenRouter.dummy-key-not-real", "dummy-new-key", "dummy-key-not-real", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h := &aiHandler{
				db:            dbhandler.NewMockDBHandler(mc),
				reqHandler:    requesthandler.NewMockRequestHandler(mc),
				notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
				utilHandler:   utilhandler.NewUtilHandler(),
			}

			_, err := h.Create(context.Background(), uuid.Must(uuid.NewV4()), "n", "d", ai.TypeNormal,
				tt.engineModel, nil, tt.engineKey, uuid.Nil, "", ai.TTSTypeNone, "", ai.STTTypeNone, "",
				[]tool.ToolName{}, nil, false, false)
			if tt.expectKey {
				assertEngineKeyRequired(t, err)
				return
			}
			assertInvalidCustomModelID(t, err, tt.leaked)
		})
	}
}

func Test_Update_customModelErrors(t *testing.T) {
	tests := []struct {
		name        string
		stored      ai.EngineModel
		engineModel ai.EngineModel
		engineKey   string
		expectKey   bool
	}{
		{"changed_to_custom_empty_key", ai.EngineModelOpenaiGPT5, "openrouter.vendor/model-a", "", true},
		{"unchanged_custom_empty_key", "openrouter.vendor/model-a", "openrouter.vendor/model-a", "", true},
		{"unchanged_custom_blank_key", "openrouter.vendor/model-a", "openrouter.vendor/model-a", "   ", true},
		{"changed_to_bad_custom_id", ai.EngineModelOpenaiGPT5, "openrouter.dummy-key-not-real", "dummy-new-key", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockDB.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(&ai.AI{EngineModel: tt.stored}, nil).Times(1)

			h := &aiHandler{
				db:            mockDB,
				reqHandler:    requesthandler.NewMockRequestHandler(mc),
				notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
				utilHandler:   utilhandler.NewUtilHandler(),
			}

			_, err := h.Update(context.Background(), uuid.Must(uuid.NewV4()), "n", "d", ai.TypeNormal,
				tt.engineModel, nil, tt.engineKey, uuid.Nil, "", ai.TTSTypeNone, "", ai.STTTypeNone, "",
				[]tool.ToolName{}, nil, false, false)
			if tt.expectKey {
				assertEngineKeyRequired(t, err)
				return
			}
			assertInvalidCustomModelID(t, err, "dummy-key-not-real")
		})
	}
}

// Test_engineValidationError_customPrefixWithWhitespaceIsNotEchoed pins that an engine model with
// leading whitespace before the custom prefix is treated as a custom value and never echoed.
func Test_engineValidationError_customPrefixWithWhitespaceIsNotEchoed(t *testing.T) {
	tests := []struct {
		name  string
		model ai.EngineModel
	}{
		{"leading space", " openrouter.dummy-key-not-real"},
		{"leading tab", "\topenrouter.dummy-key-not-real"},
		{"leading space and upper case", "  OpenRouter.dummy-key-not-real"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engineValidationError(tt.model, ai.ErrInvalidEngineModel)
			if err == nil {
				t.Fatal("expected an error")
			}
			if strings.Contains(err.Error(), "dummy-key-not-real") {
				t.Errorf("error echoes the input: %v", err)
			}

			var ve *cerrors.VoipbinError
			if !stderrors.As(err, &ve) {
				t.Fatalf("expected *cerrors.VoipbinError, got %T", err)
			}
			if strings.Contains(ve.Message, "dummy-key-not-real") {
				t.Errorf("message echoes the input: %s", ve.Message)
			}
		})
	}
}
