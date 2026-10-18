package aicallhandler

import (
	"context"
	"testing"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// ctxAliveMatcher matches a context that has not been cancelled or expired.
type ctxAliveMatcher struct{}

func (ctxAliveMatcher) Matches(x any) bool {
	c, ok := x.(context.Context)
	return ok && c.Err() == nil
}

func (ctxAliveMatcher) String() string { return "is a live (not cancelled) context" }

// Test_rollbackListenState_ExpiredContext verifies the DB read and write of the
// rollback run on a context detached from the caller's cancellation, so a
// caller whose own deadline already expired still undoes the speculative write.
func Test_rollbackListenState_ExpiredContext(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	config.SetListenDefaultsForTest()
	m := newStartListenHarness(mc)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	transcribeID := uuid.FromStringOrNil("22220000-0000-4000-8000-000000000002")
	cur := &aicall.AIcall{Metadata: map[string]any{"keep": "x", aicall.MetaKeyListenTranscribeID: transcribeID.String()}}

	m.cache.EXPECT().ListenAIcallIDRemove(gomock.Any(), transcribeID, ltAIcallID).Return(nil)
	m.db.EXPECT().AIcallGet(ctxAliveMatcher{}, ltAIcallID).Return(cur, nil)
	m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctxAliveMatcher{}, ltAIcallID, gomock.Any()).Return(nil)

	if err := m.h.rollbackListenState(ctx, ltAIcallID, transcribeID); err != nil {
		t.Errorf("expected rollback to succeed with an expired caller context, got: %v", err)
	}
}
