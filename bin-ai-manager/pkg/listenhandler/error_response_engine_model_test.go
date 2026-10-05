package listenhandler

import (
	"context"
	"net/http"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/tool"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
)

// The real aihandler error must come out of errorResponse as 400 + INVALID_ENGINE_MODEL.
func assertEngineModel400(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("expected error")
	}
	resp := errorResponse(err)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
	if resp.DataType != cerrors.DataTypeVoipbinError {
		t.Errorf("data type = %q, want %q", resp.DataType, cerrors.DataTypeVoipbinError)
	}
	if !containsStr(string(resp.Data), "INVALID_ENGINE_MODEL") {
		t.Errorf("body missing reason: %s", resp.Data)
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func Test_errorResponse_invalidEngineModel(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	h := aihandler.NewAIHandler(requesthandler.NewMockRequestHandler(ctrl), notifyhandler.NewMockNotifyHandler(ctrl), mockDB)

	t.Run("create", func(t *testing.T) {
		_, err := h.Create(context.Background(), uuid.Must(uuid.NewV4()), "n", "d", ai.TypeNormal,
			ai.EngineModel("invalid.model"), nil, "", uuid.Nil, "", ai.TTSTypeNone, "", ai.STTTypeNone, "",
			[]tool.ToolName{}, nil, false, false)
		assertEngineModel400(t, err)
	})

	t.Run("update", func(t *testing.T) {
		mockDB.EXPECT().AIGet(gomock.Any(), gomock.Any()).Return(&ai.AI{EngineModel: ai.EngineModelOpenaiGPT5}, nil).Times(1)
		_, err := h.Update(context.Background(), uuid.Must(uuid.NewV4()), "n", "d", ai.TypeNormal,
			ai.EngineModel("unknown.invalid"), nil, "", uuid.Nil, "", ai.TTSTypeNone, "", ai.STTTypeNone, "",
			[]tool.ToolName{}, nil, false, false)
		assertEngineModel400(t, err)
	})
}
