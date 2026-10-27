package reactionhandler

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	commonutil "monorepo/bin-common-handler/pkg/utilhandler"

	"monorepo/bin-talk-manager/models/message"
	"monorepo/bin-talk-manager/pkg/dbhandler"
)

// A creation time outside of the years 0 to 9999 cannot be encoded as JSON. The reaction must
// not be written, and the error must name the failed step.
func Test_ReactionAdd_marshalError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	messageID := uuid.FromStringOrNil("e8427fa8-17b2-4e9e-8855-90e516bcf1d3")
	ownerID := uuid.FromStringOrNil("91aed1d4-7fe2-11ec-848d-97c8e986acfc")

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockUtil := commonutil.NewMockUtilHandler(mc)
	h := &reactionHandler{dbHandler: mockDB, utilHandler: mockUtil}

	ctx := context.Background()
	outOfRange := timePtr(time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC))

	// no reaction yet, and no write expected: gomock fails on an unexpected MessageAddReactionAtomic
	mockDB.EXPECT().MessageGet(ctx, messageID).Return(&message.Message{Metadata: message.Metadata{Reactions: []message.Reaction{}}}, nil)
	mockUtil.EXPECT().TimeNow().Return(outOfRange)

	res, err := h.ReactionAdd(ctx, messageID, "👍", "agent", ownerID)
	if err == nil || !strings.Contains(err.Error(), "failed to marshal reaction") || !strings.Contains(err.Error(), "year outside of range") {
		t.Errorf("error = %v, want the marshal failure", err)
	}
	if res != nil {
		t.Errorf("result = %v, want nil", res)
	}
}
