package listenhandler

import (
	"errors"
	"testing"

	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aicallhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_processV1MessagesDraftDecisionPost(t *testing.T) {
	aicallID := uuid.FromStringOrNil("d8000000-0000-11f0-0000-000000000001")
	draftID := uuid.FromStringOrNil("d8000000-0000-11f0-0000-000000000002")
	body := []byte(`{"aicall_id":"d8000000-0000-11f0-0000-000000000001","draft_id":"d8000000-0000-11f0-0000-000000000002","decision":"approve"}`)

	tests := []struct {
		name string
		data []byte

		callExpected bool
		response     *message.Message
		responseErr  error

		expectStatus int
	}{
		{name: "approved", data: body, callExpected: true, response: &message.Message{Identity: identity.Identity{ID: uuid.FromStringOrNil("d8000000-0000-11f0-0000-000000000003")}}, expectStatus: 200},
		{name: "invalid body", data: []byte(`{`), expectStatus: 400},
		{
			name: "typed error is passed through", data: body, callExpected: true,
			responseErr:  cerrors.AlreadyExists(commonoutline.ServiceNameAIManager, "DRAFT_ALREADY_DECIDED", "The draft was already decided."),
			expectStatus: 409,
		},
		{name: "untyped error is a 500", data: body, callExpected: true, responseErr: errors.New("boom"), expectStatus: 500},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockAIcall := aicallhandler.NewMockAIcallHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			h := &listenHandler{aicallHandler: mockAIcall, messageHandler: mockMessage}

			if tt.callExpected {
				mockAIcall.EXPECT().DraftDecision(gomock.Any(), aicallID, draftID, message.DraftDecisionApprove).Return(tt.response, tt.responseErr)
			}

			res, err := h.processRequest(&sock.Request{URI: "/v1/messages/draft-decision", Method: sock.RequestMethodPost, DataType: "application/json", Data: tt.data})
			if err != nil {
				t.Fatalf("processRequest() err = %v", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("status = %d, want %d", res.StatusCode, tt.expectStatus)
			}
		})
	}
}

// the fixed path must not be captured by the message id routes, and the id
// route must not capture the fixed path.
func Test_regV1MessagesDraftDecision_routing(t *testing.T) {
	if regV1MessagesID.MatchString("/v1/messages/draft-decision") {
		t.Errorf("the message id route must not match the draft-decision path")
	}
	if regV1Messages.MatchString("/v1/messages/draft-decision") {
		t.Errorf("the messages route must not match the draft-decision path")
	}
	if regV1MessagesDraftDecision.MatchString("/v1/messages/d8000000-0000-11f0-0000-000000000001") {
		t.Errorf("the draft-decision route must not match a message id path")
	}
	if !regV1MessagesDraftDecision.MatchString("/v1/messages/draft-decision") {
		t.Errorf("the draft-decision route must match its path")
	}
}
