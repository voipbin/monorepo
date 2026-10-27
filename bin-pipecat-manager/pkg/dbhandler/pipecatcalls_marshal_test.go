package dbhandler

import (
	"context"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-pipecat-manager/models/pipecatcall"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// unserializableLLMMessages returns llm messages that json can not encode, so the json columns of
// a pipecatcall can not be prepared.
func unserializableLLMMessages() []map[string]any {
	return []map[string]any{{"content": make(chan int)}}
}

func Test_PipecatcallCreate_prepareFieldsFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mock, _ := newMockedHandler(t, mc)

	pc := &pipecatcall.Pipecatcall{
		Identity:    commonidentity.Identity{ID: uuid.FromStringOrNil("5a1e0c1e-6a52-4b6e-9d0a-2f7a1c000011")},
		LLMMessages: unserializableLLMMessages(),
	}

	// no statement is expected: the failure happens before the database is reached.
	err := h.PipecatcallCreate(context.Background(), pc)
	if err == nil || !strings.Contains(err.Error(), "could not prepare fields") {
		t.Errorf("Wrong match. expect: a prepare fields error, got: %v", err)
	}
	if errMet := mock.ExpectationsWereMet(); errMet != nil {
		t.Errorf("unmet expectations: %v", errMet)
	}
}

func Test_PipecatcallUpdate_prepareFieldsFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, mock, mockCache := newMockedHandler(t, mc)
	_ = mockCache // no cache call is expected

	id := uuid.FromStringOrNil("5a1e0c1e-6a52-4b6e-9d0a-2f7a1c000012")
	fields := map[pipecatcall.Field]any{
		pipecatcall.FieldLLMMessages: unserializableLLMMessages(),
	}

	err := h.PipecatcallUpdate(context.Background(), id, fields)
	if err == nil || !strings.Contains(err.Error(), "prepare fields failed") {
		t.Errorf("Wrong match. expect: a prepare fields error, got: %v", err)
	}
	if errMet := mock.ExpectationsWereMet(); errMet != nil {
		t.Errorf("unmet expectations: %v", errMet)
	}
}
