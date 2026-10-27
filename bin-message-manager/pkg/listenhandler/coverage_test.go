package listenhandler

import (
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/pkg/messagehandler"
)

// outOfRangeTime is a time that json cannot encode (the year must be in [0,9999]).
func outOfRangeTime() *time.Time {
	tm := time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)
	return &tm
}

// Test_processRequest_responseMarshalFailure covers the business handler results that cannot be serialized.
func Test_processRequest_responseMarshalFailure(t *testing.T) {
	const messageURI = "/v1/messages/1b0c3d7e-8f21-4a56-9c34-5d6e7f8091a2"

	tests := []struct {
		name string

		method sock.RequestMethod
		uri    string
		data   []byte

		setup func(mockMessage *messagehandler.MockMessageHandler, bad *message.Message)
	}{
		{
			name:   "list",
			method: sock.RequestMethodGet,
			uri:    "/v1/messages?page_size=10",
			data:   []byte(`{}`),
			setup: func(mockMessage *messagehandler.MockMessageHandler, bad *message.Message) {
				mockMessage.EXPECT().List(gomock.Any(), "", uint64(10), gomock.Any()).Return([]*message.Message{bad}, nil)
			},
		},
		{
			name:   "send",
			method: sock.RequestMethodPost,
			uri:    "/v1/messages",
			data:   []byte(`{}`),
			setup: func(mockMessage *messagehandler.MockMessageHandler, bad *message.Message) {
				mockMessage.EXPECT().Send(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(bad, nil)
			},
		},
		{
			name:   "get",
			method: sock.RequestMethodGet,
			uri:    messageURI,
			setup: func(mockMessage *messagehandler.MockMessageHandler, bad *message.Message) {
				mockMessage.EXPECT().Get(gomock.Any(), gomock.Any()).Return(bad, nil)
			},
		},
		{
			name:   "delete",
			method: sock.RequestMethodDelete,
			uri:    messageURI,
			setup: func(mockMessage *messagehandler.MockMessageHandler, bad *message.Message) {
				mockMessage.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(bad, nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()
			h, mockMessage := newErrorPathListenHandler(mc)

			tt.setup(mockMessage, &message.Message{TMCreate: outOfRangeTime()})

			res, err := h.processRequest(&sock.Request{URI: tt.uri, Method: tt.method, Data: tt.data})
			if err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}
			if res.StatusCode != http.StatusInternalServerError {
				t.Errorf("Wrong status code. expect: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
			}
			if len(res.Data) != 0 {
				t.Errorf("Wrong data. expect: empty, got: %s", res.Data)
			}
		})
	}
}

// Test_errorResponse_typedNil covers a typed error that holds a nil pointer: the typed branch is taken
// but the error response cannot be built, so the result falls back to the internal server error.
func Test_errorResponse_typedNil(t *testing.T) {
	var ve *cerrors.VoipbinError

	res := errorResponse(ve)
	if res.StatusCode != http.StatusInternalServerError {
		t.Errorf("Wrong status code. expect: %d, got: %d", http.StatusInternalServerError, res.StatusCode)
	}
	if res.DataType == cerrors.DataTypeVoipbinError {
		t.Errorf("Wrong data type. expect: not %s, got: %s", cerrors.DataTypeVoipbinError, res.DataType)
	}
}

// logSignalHook forwards the messages accepted by the filter to a channel, without blocking the logger.
type logSignalHook struct {
	filter func(msg string) bool
	ch     chan string
}

func (h *logSignalHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *logSignalHook) Fire(e *logrus.Entry) error {
	if !h.filter(e.Message) {
		return nil
	}

	select {
	case h.ch <- e.Message:
	default:
	}
	return nil
}

// Test_Run_consumeError covers the consumer goroutine that reports a failed ConsumeRPC.
func Test_Run_consumeError(t *testing.T) {
	const queue = "bin-manager.message-manager.request.test"
	const waitLimit = 5 * time.Second

	sentinel := errors.New("sentinel consume failure")

	hook := &logSignalHook{
		filter: func(msg string) bool {
			return strings.Contains(msg, "Could not consume the request message correctly") && strings.Contains(msg, sentinel.Error())
		},
		ch: make(chan string, 1),
	}
	origLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	origHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	logrus.AddHook(hook)
	t.Cleanup(func() {
		logrus.StandardLogger().ReplaceHooks(origHooks)
		logrus.SetLevel(origLevel)
	})

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &listenHandler{
		sockHandler:    mockSock,
		messageHandler: messagehandler.NewMockMessageHandler(mc),
	}

	mockSock.EXPECT().QueueCreate(queue, "normal").Return(nil)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), queue, constCosumerName, false, false, false, 10, gomock.Any()).Return(sentinel).Times(1)

	if err := h.Run(queue, ""); err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}

	select {
	case <-hook.ch:
	case <-time.After(waitLimit):
		t.Fatalf("Wrong match. expect: the consume failure log, got: timeout")
	}
}
