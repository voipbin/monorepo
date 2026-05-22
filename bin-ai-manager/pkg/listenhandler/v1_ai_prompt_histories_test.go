package listenhandler

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aiprompthistory"
	aiprompthistoryhandler "monorepo/bin-ai-manager/pkg/aiprompthistoryhandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
)

func Test_processV1AIsIDPromptHistoriesGet_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockH := aiprompthistoryhandler.NewMockAIPromptHistoryHandler(ctrl)

	lh := &listenHandler{aiprompthistoryHandler: mockH}

	aiID := uuid.Must(uuid.NewV4())
	now := time.Now()

	mockH.EXPECT().
		List(gomock.Any(), aiID, uint64(10), "").
		Return([]*aiprompthistory.AIPromptHistory{
			{Identity: identity.Identity{ID: uuid.Must(uuid.NewV4())}, AIID: aiID, TMCreate: &now},
		}, nil)

	m := &sock.Request{
		Method: sock.RequestMethodGet,
		URI:    "/v1/ais/" + aiID.String() + "/prompt_histories?page_size=10&page_token=",
	}
	resp, err := lh.processV1AIsIDPromptHistoriesGet(context.Background(), m)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	var body []*aiprompthistory.AIPromptHistory
	assert.NoError(t, json.Unmarshal(resp.Data, &body))
	assert.Len(t, body, 1)
}

func Test_processV1AIsIDPromptHistoriesGet_InvalidAIID(t *testing.T) {
	lh := &listenHandler{}

	m := &sock.Request{
		Method: sock.RequestMethodGet,
		URI:    "/v1/ais/not-a-uuid/prompt_histories?page_size=10",
	}
	resp, err := lh.processV1AIsIDPromptHistoriesGet(context.Background(), m)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func Test_processV1AIsIDPromptHistoriesIDGet_OK(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockH := aiprompthistoryhandler.NewMockAIPromptHistoryHandler(ctrl)
	lh := &listenHandler{aiprompthistoryHandler: mockH}

	aiID := uuid.Must(uuid.NewV4())
	histID := uuid.Must(uuid.NewV4())
	now := time.Now()

	mockH.EXPECT().
		Get(gomock.Any(), aiID, histID).
		Return(&aiprompthistory.AIPromptHistory{
			Identity: identity.Identity{ID: histID},
			AIID:     aiID,
			Prompt:   "v1",
			TMCreate: &now,
		}, nil)

	m := &sock.Request{
		Method: sock.RequestMethodGet,
		URI:    "/v1/ais/" + aiID.String() + "/prompt_histories/" + histID.String(),
	}
	resp, err := lh.processV1AIsIDPromptHistoriesIDGet(context.Background(), m)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func Test_processV1AIsIDPromptHistoriesIDGet_InvalidHistoryID(t *testing.T) {
	lh := &listenHandler{}

	aiID := uuid.Must(uuid.NewV4())
	m := &sock.Request{
		Method: sock.RequestMethodGet,
		URI:    "/v1/ais/" + aiID.String() + "/prompt_histories/bad",
	}
	resp, err := lh.processV1AIsIDPromptHistoriesIDGet(context.Background(), m)
	assert.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}
