package actionhandler

import (
	"context"
	"strings"
	"testing"

	"monorepo/bin-flow-manager/models/action"
)

func Test_GenerateFlowActions_invalidType_coverage90(t *testing.T) {
	h := NewActionHandler()

	res, err := h.GenerateFlowActions(context.Background(), []action.Action{
		{Type: action.TypeAnswer},
		{Type: action.Type("synthetic_unknown_type")},
	})
	if err == nil || !strings.Contains(err.Error(), "synthetic_unknown_type") {
		t.Fatalf("expected the unsupported type failure, got: %v", err)
	}
	if res != nil {
		t.Errorf("no action is expected with the error. got: %v", res)
	}
}
