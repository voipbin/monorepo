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
