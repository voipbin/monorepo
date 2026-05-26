package message_test

import (
	"testing"

	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func TestFieldStructContainsActiveAIID(t *testing.T) {
	filters := map[string]any{
		"active_ai_id": "550e8400-e29b-41d4-a716-446655440000",
	}
	_, err := utilhandler.ConvertFilters[message.FieldStruct, message.Field](message.FieldStruct{}, filters)
	if err != nil {
		t.Errorf("expected active_ai_id to be a valid filter, got err: %v", err)
	}
}
