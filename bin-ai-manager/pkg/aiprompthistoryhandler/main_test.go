package aiprompthistoryhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/assert"

	"monorepo/bin-ai-manager/models/aiprompthistory"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/utilhandler"
)

func TestList(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	h := New(mockDB, utilhandler.NewUtilHandler())

	aiID := uuid.Must(uuid.NewV4())
	now := time.Now()

	mockDB.EXPECT().
		AIPromptHistoryGetsByAIID(gomock.Any(), aiID, uint64(10), "").
		Return([]*aiprompthistory.AIPromptHistory{
			{Identity: identity.Identity{ID: uuid.Must(uuid.NewV4())}, AIID: aiID, Prompt: "v1", TMCreate: &now},
		}, nil)

	res, err := h.List(context.Background(), aiID, 10, "")
	assert.NoError(t, err)
	assert.Len(t, res, 1)
}

func TestList_Empty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	h := New(mockDB, utilhandler.NewUtilHandler())

	aiID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().
		AIPromptHistoryGetsByAIID(gomock.Any(), aiID, uint64(10), "").
		Return([]*aiprompthistory.AIPromptHistory{}, nil)

	res, err := h.List(context.Background(), aiID, 10, "")
	assert.NoError(t, err)
	assert.Empty(t, res)
}

func TestGet_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	h := New(mockDB, utilhandler.NewUtilHandler())

	aiID := uuid.Must(uuid.NewV4())
	historyID := uuid.Must(uuid.NewV4())
	now := time.Now()

	mockDB.EXPECT().
		AIPromptHistoryGet(gomock.Any(), historyID).
		Return(&aiprompthistory.AIPromptHistory{
			Identity: identity.Identity{ID: historyID},
			AIID:     aiID,
			Prompt:   "v1",
			TMCreate: &now,
		}, nil)

	res, err := h.Get(context.Background(), aiID, historyID)
	assert.NoError(t, err)
	assert.Equal(t, historyID, res.ID)
}

func TestGet_AIID_Mismatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	h := New(mockDB, utilhandler.NewUtilHandler())

	aiID := uuid.Must(uuid.NewV4())
	differentAIID := uuid.Must(uuid.NewV4())
	historyID := uuid.Must(uuid.NewV4())
	now := time.Now()

	mockDB.EXPECT().
		AIPromptHistoryGet(gomock.Any(), historyID).
		Return(&aiprompthistory.AIPromptHistory{
			Identity: identity.Identity{ID: historyID},
			AIID:     differentAIID, // belongs to a different AI
			Prompt:   "v1",
			TMCreate: &now,
		}, nil)

	_, err := h.Get(context.Background(), aiID, historyID)
	assert.ErrorIs(t, err, dbhandler.ErrNotFound)
}

func TestGet_NotFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := dbhandler.NewMockDBHandler(ctrl)
	h := New(mockDB, utilhandler.NewUtilHandler())

	aiID := uuid.Must(uuid.NewV4())
	historyID := uuid.Must(uuid.NewV4())

	mockDB.EXPECT().
		AIPromptHistoryGet(gomock.Any(), historyID).
		Return(nil, dbhandler.ErrNotFound)

	_, err := h.Get(context.Background(), aiID, historyID)
	assert.ErrorIs(t, err, dbhandler.ErrNotFound)
}
