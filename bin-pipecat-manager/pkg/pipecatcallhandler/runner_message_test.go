package pipecatcallhandler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"monorepo/bin-pipecat-manager/internal/logtest"
	"monorepo/bin-pipecat-manager/models/message"
	"monorepo/bin-pipecat-manager/models/pipecatcall"
)

func Test_receiveMessageFrameTypeMessage_rejectedInput(t *testing.T) {
	tests := []struct {
		name string

		m []byte

		expectErrContains string
		expectErrIsSyntax bool
	}{
		{
			name:              "not json",
			m:                 []byte(`{"label": `),
			expectErrIsSyntax: true,
		},
		{
			name:              "bot-transcription with a malformed data field",
			m:                 []byte(`{"label": "rtvi-ai", "type": "bot-transcription", "data": "text"}`),
			expectErrContains: "could not unmarshal bot-transcription message",
		},
		{
			name:              "user-transcription with a malformed data field",
			m:                 []byte(`{"label": "rtvi-ai", "type": "user-transcription", "data": "text"}`),
			expectErrContains: "could not unmarshal user-transcription message",
		},
		{
			name:              "user-llm-text with a malformed data field",
			m:                 []byte(`{"label": "rtvi-ai", "type": "user-llm-text", "data": "text"}`),
			expectErrContains: "could not unmarshal user-llm-text message",
		},
		{
			name:              "bot-llm-text with a malformed data field",
			m:                 []byte(`{"label": "rtvi-ai", "type": "bot-llm-text", "data": "text"}`),
			expectErrContains: "could not unmarshal bot-llm-text message",
		},
		{
			name:              "error-response with a malformed data field",
			m:                 []byte(`{"label": "rtvi-ai", "type": "error-response", "data": "text"}`),
			expectErrContains: "could not unmarshal error-response message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no expectation on any collaborator: a rejected message must not publish anything.
			h := &pipecatcallHandler{
				notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
				utilHandler:   utilhandler.NewMockUtilHandler(mc),
			}
			se := &pipecatcall.Session{
				Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a1b2c3d4-3003-4000-8000-000000000001")},
				Ctx:      context.Background(),
			}

			err := h.receiveMessageFrameTypeMessage(se, tt.m)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
			if tt.expectErrIsSyntax {
				var syntaxErr *json.SyntaxError
				if !errors.As(err, &syntaxErr) {
					t.Errorf("Wrong error. expect a json syntax error, got: %v", err)
				}
				return
			}
			if !strings.Contains(err.Error(), tt.expectErrContains) {
				t.Errorf("Wrong error. expect contains: %q, got: %v", tt.expectErrContains, err)
			}
		})
	}
}

func Test_receiveMessageFrameTypeMessage_ignoredInput(t *testing.T) {
	tests := []struct {
		name string

		m []byte
	}{
		{
			name: "another label",
			m:    []byte(`{"label": "other-ai", "type": "bot-transcription", "data": {"text": "hello"}}`),
		},
		{
			name: "user transcription that is not final",
			m:    []byte(`{"label": "rtvi-ai", "type": "user-transcription", "data": {"text": "hel", "final": false}}`),
		},
		{
			name: "unrecognized rtvi type",
			m:    []byte(`{"label": "rtvi-ai", "type": "metrics", "data": {}}`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// no expectation on any collaborator: nothing may be published or generated.
			h := &pipecatcallHandler{
				notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
				utilHandler:   utilhandler.NewMockUtilHandler(mc),
			}
			se := &pipecatcall.Session{
				Identity: commonidentity.Identity{ID: uuid.FromStringOrNil("a1b2c3d4-3003-4000-8000-000000000002")},
				Ctx:      context.Background(),
			}

			if err := h.receiveMessageFrameTypeMessage(se, tt.m); err != nil {
				t.Errorf("expected nil, got: %v", err)
			}
		})
	}
}

func Test_receiveMessageFrameTypeMessage_userLLMText(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockUtil := utilhandler.NewMockUtilHandler(mc)
	h := &pipecatcallHandler{notifyHandler: mockNotify, utilHandler: mockUtil}

	se := &pipecatcall.Session{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a1b2c3d4-3003-4000-8000-000000000003"),
			CustomerID: uuid.FromStringOrNil("b2c3d4e5-3003-4000-8000-000000000003"),
		},
		Ctx: context.Background(),
	}
	messageID := uuid.FromStringOrNil("c3d4e5f6-3003-4000-8000-000000000003")
	mockUtil.EXPECT().UUIDCreate().Return(messageID)

	// the event is published from a goroutine: wait for it before the controller finishes.
	var wg sync.WaitGroup
	wg.Add(1)
	mockNotify.EXPECT().PublishEvent(se.Ctx, message.EventTypeUserLLM, &message.Message{
		Identity: commonidentity.Identity{
			ID:         messageID,
			CustomerID: se.CustomerID,
		},
		PipecatcallID: se.ID,
		Text:          "what is the weather",
	}).Do(func(any, any, any) { wg.Done() })

	m := []byte(`{"label": "rtvi-ai", "type": "user-llm-text", "data": {"text": "what is the weather"}}`)
	if err := h.receiveMessageFrameTypeMessage(se, m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wg.Wait()
}

func Test_receiveMessageFrameTypeMessage_tokenChannelFull(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	logs := logtest.New(t)

	// no collaborator is expected: the generation is already armed, so nothing starts.
	h := &pipecatcallHandler{
		notifyHandler: notifyhandler.NewMockNotifyHandler(mc),
		utilHandler:   utilhandler.NewMockUtilHandler(mc),
	}
	se := &pipecatcall.Session{
		Identity:     commonidentity.Identity{ID: uuid.FromStringOrNil("a1b2c3d4-3003-4000-8000-000000000004")},
		Ctx:          context.Background(),
		LLMTokenChan: make(chan string, 1),
	}
	se.LLMFlushing.Store(true)
	se.LLMTokenChan <- "queued"

	m := []byte(`{"label": "rtvi-ai", "type": "bot-llm-text", "data": {"text": "dropped"}}`)
	if err := h.receiveMessageFrameTypeMessage(se, m); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(logs.Dump(), "LLM token channel full, dropping intermediate token.") {
		t.Errorf("the drop warning is missing")
	}
	if len(se.LLMTokenChan) != 1 || <-se.LLMTokenChan != "queued" {
		t.Errorf("the queued token must be left untouched")
	}
}
