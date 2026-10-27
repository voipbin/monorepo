package listenhandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/internal/logtest"
	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/conversationhandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
)

var errBrokerCoverage = errors.New("broker failure")

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockAccount := accounthandler.NewMockAccountHandler(mc)
	mockConversation := conversationhandler.NewMockConversationHandler(mc)
	mockMessage := messagehandler.NewMockMessageHandler(mc)

	res, ok := NewListenHandler(mockSock, mockAccount, mockConversation, mockMessage).(*listenHandler)
	if !ok {
		t.Fatalf("NewListenHandler() did not return a *listenHandler")
	}

	if res.sockHandler != mockSock || res.accountHandler != mockAccount || res.conversationHandler != mockConversation || res.messageHandler != mockMessage {
		t.Errorf("NewListenHandler() did not keep the given handlers")
	}
	if res.utilHandler == nil {
		t.Errorf("NewListenHandler() did not create the util handler")
	}
}

// consumeSignal is a logrus hook that signals when an entry with the given text is logged.
type consumeSignal struct {
	text string
	seen chan struct{}
}

func (s *consumeSignal) Levels() []logrus.Level { return logrus.AllLevels }

func (s *consumeSignal) Fire(e *logrus.Entry) error {
	if strings.Contains(e.Message, s.text) {
		select {
		case s.seen <- struct{}{}:
		default:
		}
	}

	return nil
}

func Test_Run(t *testing.T) {
	const queue = "bin-manager.conversation-manager.request.test"

	tests := []struct {
		name string

		queueErr   error
		consumeErr error

		expectErr bool
	}{
		{name: "the queue is created and the consumer is started"},
		{name: "a consumer failure is only logged", consumeErr: errBrokerCoverage},
		{name: "a queue failure is returned and no consumer is started", queueErr: errBrokerCoverage, expectErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the consumer failure is logged by a goroutine, so the log is waited on instead of a fixed delay
			logtest.Capture(t)
			logged := &consumeSignal{text: "Could not consume the request message correctly", seen: make(chan struct{}, 1)}
			logrus.AddHook(logged)

			mockSock := sockhandler.NewMockSockHandler(mc)
			consuming := make(chan struct{})
			mockSock.EXPECT().QueueCreate(queue, "normal").Return(tt.queueErr).Times(1)
			if tt.queueErr == nil {
				mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, constCosumerName, false, false, false, 10, gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
						close(consuming)
						return tt.consumeErr
					}).Times(1)
			}

			h := NewListenHandler(mockSock, accounthandler.NewMockAccountHandler(mc), conversationhandler.NewMockConversationHandler(mc), messagehandler.NewMockMessageHandler(mc))

			err := h.Run(queue, "unused-delay-exchange")
			if tt.expectErr {
				if !errors.Is(err, errBrokerCoverage) || !strings.Contains(err.Error(), "could not declare the queue for listenHandler") {
					t.Errorf("Run() error = %v, want the queue failure wrapped", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run() error = %v, want nil", err)
			}

			select {
			case <-consuming:
			case <-time.After(5 * time.Second):
				t.Fatalf("the rpc consumer was not started")
			}

			if tt.consumeErr != nil {
				select {
				case <-logged.seen:
				case <-time.After(5 * time.Second):
					t.Fatalf("the consumer failure was not logged")
				}
			}
		})
	}
}

// A failure of a route handler is converted by processRequest: a typed error and a not found error keep their
// status, any other error becomes 400.
func Test_processRequest_errorConversion(t *testing.T) {
	const body = `{"customer_id":"3f7a1b9c-0d52-4c1e-8a0b-6e2d4f8a9b10","type":"line","secret":"s","token":"t"}`

	tests := []struct {
		name string
		err  error

		expectStatus   int
		expectDataType string
	}{
		{
			name:           "typed error keeps its status and envelope",
			err:            cerrors.InvalidArgument(commonoutline.ServiceNameConversationManager, "TEST_REASON", "test message"),
			expectStatus:   400,
			expectDataType: string(cerrors.DataTypeVoipbinError),
		},
		{
			name:         "not found becomes 404",
			err:          fmt.Errorf("lookup: %w", dbhandler.ErrNotFound),
			expectStatus: 404,
		},
		{
			name:         "any other error becomes 400",
			err:          errors.New("connection refused"),
			expectStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newErrorPathHandler(mc)
			m.acc.EXPECT().Create(gomock.Any(), gomock.Any(), account.TypeLine, "", "", "s", "t", gomock.Any(), gomock.Any()).Return(nil, tt.err)

			res, err := h.processRequest(newReq("/v1/accounts", sock.RequestMethodPost, body))
			if err != nil {
				t.Fatalf("processRequest() error = %v, want nil", err)
			}
			if res.StatusCode != tt.expectStatus {
				t.Errorf("status = %d, want %d", res.StatusCode, tt.expectStatus)
			}
			if res.DataType != tt.expectDataType {
				t.Errorf("data type = %q, want %q", res.DataType, tt.expectDataType)
			}
		})
	}
}

func Test_uriPath(t *testing.T) {
	tests := []struct {
		name string
		uri  string

		expectRes string
	}{
		{name: "path without query", uri: "https://hook.example.com/v1.0/hooks/abc", expectRes: "/v1.0/hooks/abc"},
		{name: "the query is dropped", uri: "https://hook.example.com/v1.0/hooks/abc?hub.verify_token=x", expectRes: "/v1.0/hooks/abc"},
		{name: "an unparsable uri gives an empty path", uri: "://not-a-uri", expectRes: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uriPath(tt.uri); got != tt.expectRes {
				t.Errorf("uriPath() = %q, want %q", got, tt.expectRes)
			}
		})
	}
}
