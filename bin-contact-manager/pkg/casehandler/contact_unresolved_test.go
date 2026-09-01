package casehandler

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-contact-manager/models/kase"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// Test_CaseListAll_DelegatesToDB verifies CaseListAll is a thin,
// unfiltered delegation to dbhandler.CaseListAll (case-control's --all
// reconcile-contact sweep, CLI-only).
func Test_CaseListAll_DelegatesToDB(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &caseHandler{utilHandler: mockUtil, reqHandler: mockReq, db: mockDB, notifyHandler: mockNotify}
	ctx := context.Background()

	want := []*kase.Case{{}}
	mockDB.EXPECT().CaseListAll(ctx).Return(want, nil)

	res, err := h.CaseListAll(ctx)
	if err != nil {
		t.Fatalf("CaseListAll() error = %v", err)
	}
	if len(res) != len(want) {
		t.Errorf("expected %d cases, got %d", len(want), len(res))
	}
}
