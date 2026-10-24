package subscribehandler

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	commonsock "monorepo/bin-common-handler/models/sock"
	emmemail "monorepo/bin-email-manager/models/email"
	wmmessage "monorepo/bin-webchat-manager/models/message"

	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
)

func Test_processEventEmailEmailUpdated(t *testing.T) {
	tests := []struct {
		name string
		data string

		expectCall bool
		handlerErr error
		expectErr  bool
	}{
		{
			name:       "handler is called with the decoded email",
			data:       `{"id":"e1000000-0000-0000-0000-000000000001","subject":"test subject"}`,
			expectCall: true,
		},
		{
			name:      "malformed data returns error without calling the handler",
			data:      `{"id":`,
			expectErr: true,
		},
		{
			name:       "handler error is returned",
			data:       `{"id":"e1000000-0000-0000-0000-000000000002"}`,
			expectCall: true,
			handlerErr: fmt.Errorf("update failed"),
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockConversation := conversationhandler.NewMockConversationHandler(mc)
			h := &subscribeHandler{conversationHandler: mockConversation}

			if tt.expectCall {
				mockConversation.EXPECT().EmailEventUpdated(gomock.Any(), gomock.Any()).DoAndReturn(
					func(_ context.Context, e *emmemail.Email) error {
						var want emmemail.Email
						if err := json.Unmarshal([]byte(tt.data), &want); err != nil {
							t.Fatalf("invalid test data: %v", err)
						}
						if e.ID != want.ID || e.Subject != want.Subject {
							t.Errorf("Wrong email passed. expect id: %s subject: %q, got id: %s subject: %q", want.ID, want.Subject, e.ID, e.Subject)
						}
						return tt.handlerErr
					})
			}

			err := h.processEventEmailEmailUpdated(context.Background(), &commonsock.Event{
				Publisher: publisherEmailManager,
				Type:      emmemail.EventTypeUpdated,
				Data:      json.RawMessage(tt.data),
			})
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_processEventEmailEmailCreated_errors(t *testing.T) {
	tests := []struct {
		name string
		data string

		expectCall bool
		handlerErr error
	}{
		{
			name: "malformed data returns error without calling the handler",
			data: `{"id":`,
		},
		{
			name:       "handler error is returned",
			data:       `{"id":"e1000000-0000-0000-0000-000000000003"}`,
			expectCall: true,
			handlerErr: fmt.Errorf("send failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockConversation := conversationhandler.NewMockConversationHandler(mc)
			h := &subscribeHandler{conversationHandler: mockConversation}

			if tt.expectCall {
				mockConversation.EXPECT().EmailEventSent(gomock.Any(), gomock.Any()).Return(tt.handlerErr)
			}

			err := h.processEventEmailEmailCreated(context.Background(), &commonsock.Event{
				Data: json.RawMessage(tt.data),
			})
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func Test_processEventMessageMessageCreated_errors(t *testing.T) {
	tests := []struct {
		name string
		data string

		expectCall bool
		handlerErr error
	}{
		{
			name: "malformed data returns error without calling the handler",
			data: `{"id":`,
		},
		{
			name:       "handler error is returned",
			data:       `{"id":"e1000000-0000-0000-0000-000000000004"}`,
			expectCall: true,
			handlerErr: fmt.Errorf("event failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockConversation := conversationhandler.NewMockConversationHandler(mc)
			h := &subscribeHandler{conversationHandler: mockConversation}

			if tt.expectCall {
				mockConversation.EXPECT().Event(gomock.Any(), conversation.TypeMessage, []byte(tt.data)).Return(tt.handlerErr)
			}

			err := h.processEventMessageMessageCreated(context.Background(), &commonsock.Event{
				Data: json.RawMessage(tt.data),
			})
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: ok")
			}
		})
	}
}

func Test_processEventWebchatMessageMessageCreated(t *testing.T) {
	tests := []struct {
		name string

		handlerErr error
		expectErr  bool
	}{
		{
			name: "payload is forwarded as a webchat event",
		},
		{
			name:       "handler error is returned",
			handlerErr: fmt.Errorf("event failed"),
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockConversation := conversationhandler.NewMockConversationHandler(mc)
			h := &subscribeHandler{conversationHandler: mockConversation}

			data := `{"id":"e1000000-0000-0000-0000-000000000005","text":"hello"}`
			mockConversation.EXPECT().Event(gomock.Any(), conversation.TypeWebchat, []byte(data)).Return(tt.handlerErr)

			err := h.processEventWebchatMessageMessageCreated(context.Background(), &commonsock.Event{
				Data: json.RawMessage(data),
			})
			if (err != nil) != tt.expectErr {
				t.Errorf("Wrong match. expect error: %v, got: %v", tt.expectErr, err)
			}
		})
	}
}

func Test_processEvent_dispatch(t *testing.T) {
	tests := []struct {
		name  string
		event *commonsock.Event

		expect func(mockConversation *conversationhandler.MockConversationHandler)
	}{
		{
			name: "email updated is routed to EmailEventUpdated",
			event: &commonsock.Event{
				Publisher: publisherEmailManager,
				Type:      emmemail.EventTypeUpdated,
				Data:      json.RawMessage(`{"id":"e1000000-0000-0000-0000-000000000006"}`),
			},
			expect: func(mockConversation *conversationhandler.MockConversationHandler) {
				mockConversation.EXPECT().EmailEventUpdated(gomock.Any(), gomock.Any()).Return(nil)
			},
		},
		{
			name: "webchat message created is routed to a webchat Event",
			event: &commonsock.Event{
				Publisher: publisherWebchatManager,
				Type:      wmmessage.EventTypeMessageCreated,
				Data:      json.RawMessage(`{"id":"e1000000-0000-0000-0000-000000000007"}`),
			},
			expect: func(mockConversation *conversationhandler.MockConversationHandler) {
				mockConversation.EXPECT().Event(gomock.Any(), conversation.TypeWebchat, gomock.Any()).Return(nil)
			},
		},
		{
			name: "handler error is only logged and does not panic",
			event: &commonsock.Event{
				Publisher: publisherEmailManager,
				Type:      emmemail.EventTypeUpdated,
				Data:      json.RawMessage(`{"id":"e1000000-0000-0000-0000-000000000008"}`),
			},
			expect: func(mockConversation *conversationhandler.MockConversationHandler) {
				mockConversation.EXPECT().EmailEventUpdated(gomock.Any(), gomock.Any()).Return(fmt.Errorf("update failed"))
			},
		},
		{
			name: "known publisher with an unrelated type is ignored",
			event: &commonsock.Event{
				Publisher: publisherEmailManager,
				Type:      "email_unrelated",
				Data:      json.RawMessage(`{}`),
			},
			expect: func(mockConversation *conversationhandler.MockConversationHandler) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockConversation := conversationhandler.NewMockConversationHandler(mc)
			h := &subscribeHandler{conversationHandler: mockConversation}

			tt.expect(mockConversation)

			h.processEvent(tt.event)
		})
	}
}
