package listenhandler

import (
	"errors"
	"strings"
	"testing"
	"time"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-call-manager/models/call"
	"monorepo/bin-call-manager/models/channel"
	"monorepo/bin-call-manager/pkg/callhandler"
	"monorepo/bin-call-manager/pkg/channelhandler"
)

// signalHook notifies via channel when a log entry containing the keyword is fired.
type signalHook struct {
	keyword string
	ch      chan string
}

func (s *signalHook) Levels() []logrus.Level { return logrus.AllLevels }

func (s *signalHook) Fire(e *logrus.Entry) error {
	if strings.Contains(e.Message, s.keyword) {
		select {
		case s.ch <- e.Message:
		default:
		}
	}
	return nil
}

// installSignalHook replaces the standard logger hooks and returns the restore function.
func installSignalHook(keyword string) (chan string, func()) {
	hook := &signalHook{keyword: keyword, ch: make(chan string, 1)}
	hooks := make(logrus.LevelHooks)
	hooks.Add(hook)
	old := logrus.StandardLogger().ReplaceHooks(hooks)
	return hook.ch, func() { logrus.StandardLogger().ReplaceHooks(old) }
}

func Test_processV1CallsIDActionNextPost_cov2_actionError(t *testing.T) {
	callID := uuid.FromStringOrNil("37b3a214-0afd-11eb-88ea-7bdd69288e90")

	tests := []struct {
		name string

		data string

		expectKeyword string
	}{
		{name: "action next error is logged", data: `{"force":false}`, expectKeyword: "Could not execute the action next. err: next failed"},
		{name: "action next force error is logged", data: `{"force":true}`, expectKeyword: "Could not execute the action next force. err: next failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockCall := callhandler.NewMockCallHandler(mc)
			h := &listenHandler{callHandler: mockCall}

			ch, restore := installSignalHook("Could not execute the action next")
			defer restore()

			c := &call.Call{}
			c.ID = callID
			mockCall.EXPECT().Get(gomock.Any(), callID).Return(c, nil)
			if strings.Contains(tt.data, "true") {
				mockCall.EXPECT().ActionNextForce(gomock.Any(), c).Return(errors.New("next failed"))
			} else {
				mockCall.EXPECT().ActionNext(gomock.Any(), c).Return(errors.New("next failed"))
			}

			res, err := h.processRequest(&sock.Request{
				URI:      "/v1/calls/" + callID.String() + "/action-next",
				Method:   sock.RequestMethodPost,
				DataType: "application/json",
				Data:     []byte(tt.data),
			})
			if err != nil || res == nil || res.StatusCode != 200 {
				t.Errorf("Wrong match. expect: 200, got: %v, err: %v", res, err)
			}

			msg := waitSignalMsg(t, ch)
			if !strings.Contains(msg, tt.expectKeyword) {
				t.Errorf("Wrong match. expect: %s, got: %s", tt.expectKeyword, msg)
			}
		})
	}
}

func Test_processRequest_cov2(t *testing.T) {
	tests := []struct {
		name string

		request *sock.Request

		expectChannelGet bool
		expectStatus     int
	}{
		{
			name:             "channel get route",
			request:          &sock.Request{URI: "/v1/channels/ch-1", Method: sock.RequestMethodGet},
			expectChannelGet: true,
			expectStatus:     200,
		},
		{
			name: "handler error is turned into 400",
			request: &sock.Request{
				URI:    "/v1/calls/37b3a214-0afd-11eb-88ea-7bdd69288e90/action-next",
				Method: sock.RequestMethodPost,
				Data:   []byte(`{invalid`),
			},
			expectStatus: 400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockChannel := channelhandler.NewMockChannelHandler(mc)
			h := &listenHandler{channelHandler: mockChannel}

			if tt.expectChannelGet {
				mockChannel.EXPECT().Get(gomock.Any(), "ch-1").Return(&channel.Channel{ID: "ch-1"}, nil)
			}

			res, err := h.processRequest(tt.request)
			if err != nil {
				t.Errorf("Wrong match. expect: nil, got: %v", err)
			}
			if res == nil || res.StatusCode != tt.expectStatus {
				t.Errorf("Wrong match. expect status: %d, got: %v", tt.expectStatus, res)
			}
		})
	}
}

func Test_Run_cov2(t *testing.T) {
	t.Run("queue create error", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &listenHandler{sockHandler: mockSock}

		qErr := errors.New("q")
		mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(qErr)

		err := h.Run("test-queue", "")
		if err == nil || !strings.Contains(err.Error(), "could not declare the queue for listenHandler") || !errors.Is(err, qErr) {
			t.Errorf("Wrong match. got: %v", err)
		}
	})

	t.Run("consume error is logged", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockSock := sockhandler.NewMockSockHandler(mc)
		h := &listenHandler{sockHandler: mockSock}

		ch, restore := installSignalHook("Could not consume the request message correctly")
		defer restore()

		mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
		mockSock.EXPECT().ConsumeRPC(gomock.Any(), "test-queue", gomock.Any(), false, false, false, 10, gomock.Any()).Return(errors.New("consume failed"))

		if err := h.Run("test-queue", ""); err != nil {
			t.Fatalf("Wrong match. expect: nil, got: %v", err)
		}

		msg := waitSignalMsg(t, ch)
		if !strings.Contains(msg, "consume failed") {
			t.Errorf("Wrong match. got: %s", msg)
		}
	})
}

// waitSignalMsg waits for the hooked log message and fails the test instead of blocking forever.
func waitSignalMsg(t *testing.T, ch <-chan string) string {
	t.Helper()

	select {
	case msg := <-ch:
		return msg
	case <-time.After(5 * time.Second):
		t.Fatalf("Timed out waiting for the expected log message")
		return ""
	}
}
