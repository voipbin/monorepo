package listenhandler

import (
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/internal/logtest"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/pkg/aicallhandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/sock"
)

const logPolicySentinel = "SENTINEL-CONVERSATION-TEXT-9f3c"

// A line written by a handler must not carry the request data. A message POST carries the user's
// sentence. The error path is used because the request is a field of the handler's logger.
func Test_logPolicy_messagesPost_requestData(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	logs := logtest.New(t)

	mockAIcall := aicallhandler.NewMockAIcallHandler(mc)
	h := &listenHandler{aicallHandler: mockAIcall}

	mockAIcall.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), logPolicySentinel, gomock.Any(), gomock.Any()).
		Return(nil, errors.New("send failed"))

	req := &sock.Request{
		URI:      "/v1/messages",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"aicall_id": "0615dec8-f25e-11ef-b878-0fb5e7ac2aee", "role": "user", "content": "` + logPolicySentinel + `"}`),
	}
	if _, err := h.processRequest(req); err != nil {
		t.Fatalf("processRequest: %v", err)
	}

	logs.AssertAbsent(t, logPolicySentinel)
	logs.AssertField(t, "data_len")
}

// A response that cannot be marshaled is logged at Error level. It must not print the response model.
func Test_logPolicy_aicallsPost_marshalFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	logs := logtest.New(t)

	mockAIcall := aicallhandler.NewMockAIcallHandler(mc)
	h := &listenHandler{aicallHandler: mockAIcall}

	// a channel value cannot be marshaled to JSON
	res := &aicall.AIcall{
		Identity: identity.Identity{ID: uuid.Must(uuid.NewV4())},
		Metadata: map[string]any{"prompt": logPolicySentinel, "bad": make(chan int)},
	}
	mockAIcall.EXPECT().Start(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(res, nil)

	req := &sock.Request{
		URI:      "/v1/aicalls",
		Method:   sock.RequestMethodPost,
		DataType: "application/json",
		Data:     []byte(`{"assistance_type":"ai","assistance_id":"0615dec8-f25e-11ef-b878-0fb5e7ac2aee","reference_type":"call","reference_id":"1615dec8-f25e-11ef-b878-0fb5e7ac2aee"}`),
	}
	got, err := h.processRequest(req)
	if err != nil {
		t.Fatalf("processRequest: %v", err)
	}
	if got.StatusCode != 500 {
		t.Fatalf("expected the marshal failure to answer 500, got %d", got.StatusCode)
	}

	logs.AssertAbsent(t, logPolicySentinel)
}
