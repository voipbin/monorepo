package variablehandler

import (
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

func Test_NewVariableHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	res := NewVariableHandler(mockDB, mockReq)
	h, ok := res.(*variableHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}
	if h.db != mockDB || h.requestHandler != mockReq {
		t.Errorf("The given handlers are not wired to the variable handler")
	}
}
