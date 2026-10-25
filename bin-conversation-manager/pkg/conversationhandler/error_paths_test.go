package conversationhandler

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
)

// Test_Event_errors covers the dispatch rejections that happen before any
// downstream handler is used, so the handler is built without dependencies.
// A payload with a numeric id is rejected by the decoder of the routed event type,
// and the decoder error names the target struct, which shows where Event routed it.
func Test_Event_errors(t *testing.T) {
	tests := []struct {
		name string

		referenceType conversation.Type
		data          []byte

		expectContains string
	}{
		{"unknown reference type", conversation.Type("unknown"), []byte(`{}`), "reference type handler not found"},
		{"sms event with malformed payload", conversation.TypeMessage, []byte(`{"id":5}`), "Go struct field Message.id"},
		{"sms event with an unknown direction", conversation.TypeMessage, []byte(`{"direction":"sideways"}`), "could not find the direction. direction: sideways"},
		{"webchat event with malformed payload", conversation.TypeWebchat, []byte(`{"id":5}`), "Go struct field webchatMessage.id"},
		{"webchat event with an unknown direction", conversation.TypeWebchat, []byte(`{"direction":"sideways"}`), "could not find the direction. direction: sideways"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &conversationHandler{}

			err := h.Event(context.Background(), tt.referenceType, tt.data)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tt.expectContains) {
				t.Errorf("Wrong error. expect to contain: %q, got: %v", tt.expectContains, err)
			}
		})
	}
}

// Test_Event_createFailure verifies that a failed message insert is reported to the caller
// for both event types. The payloads are valid only for their own type: if Event routed an
// event to the other decoder, the downstream mocks would not be called and the test fails.
func Test_Event_createFailure(t *testing.T) {
	customerID := uuid.FromStringOrNil("c1b2c3d4-0000-0000-0000-000000000001")
	msgID := uuid.FromStringOrNil("db596422-07f5-11f0-9afe-e7cd6b75aeac")
	convID := uuid.FromStringOrNil("44ebbd2e-82d8-11eb-8a4e-f7957fea9f50")

	tests := []struct {
		name string

		referenceType conversation.Type
		data          string
	}{
		{
			"sms outbound",
			conversation.TypeMessage,
			`{"id":"db596422-07f5-11f0-9afe-e7cd6b75aeac","customer_id":"c1b2c3d4-0000-0000-0000-000000000001","source":{"type":"tel","target":"+15550100001"},"targets":[{"destination":{"type":"tel","target":"+15550100002"}}],"text":"hello","direction":"outbound"}`,
		},
		{
			"webchat outbound",
			conversation.TypeWebchat,
			`{"id":"db596422-07f5-11f0-9afe-e7cd6b75aeac","customer_id":"c1b2c3d4-0000-0000-0000-000000000001","widget_id":"aa847807-6cc4-4713-9dec-53a42840e74c","session_id":"876defde-ad5e-11ed-a8c3-7bc19647b03f","text":"hello","direction":"outbound"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			h := &conversationHandler{db: mockDB, messageHandler: mockMessage}

			createErr := fmt.Errorf("insert failed")
			cv := &conversation.Conversation{
				Identity: commonidentity.Identity{ID: convID, CustomerID: customerID},
				Type:     tt.referenceType,
			}

			mockDB.EXPECT().ConversationGetBySelfAndPeer(gomock.Any(), gomock.Any(), gomock.Any()).Return(cv, nil)
			mockMessage.EXPECT().Get(gomock.Any(), msgID).Return(nil, dbhandler.ErrNotFound)
			mockMessage.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil, createErr)

			err := h.Event(context.Background(), tt.referenceType, []byte(tt.data))
			if !stderrors.Is(err, createErr) {
				t.Errorf("expected the insert error to be reported, got: %v", err)
			}
		})
	}
}
