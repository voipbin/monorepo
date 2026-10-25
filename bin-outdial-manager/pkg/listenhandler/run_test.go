package listenhandler

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"monorepo/bin-outdial-manager/pkg/outdialhandler"
	"monorepo/bin-outdial-manager/pkg/outdialtargethandler"
)

const runWaitTimeout = 5 * time.Second

var errRunInjected = errors.New("injected failure")

// consumeArgs holds the arguments the mock received for ConsumeRPC.
type consumeArgs struct {
	queue      string
	consumer   string
	exclusive  bool
	noLocal    bool
	noWait     bool
	workerNum  int
	hasHandler bool
}

// logHook sends the message of every error entry to a buffered channel.
type logHook struct {
	ch chan string
}

func (h *logHook) Levels() []logrus.Level { return []logrus.Level{logrus.ErrorLevel} }

func (h *logHook) Fire(e *logrus.Entry) error {
	select {
	case h.ch <- e.Message:
	default:
	}
	return nil
}

func Test_NewListenHandler(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockOutdial := outdialhandler.NewMockOutdialHandler(mc)
	mockTarget := outdialtargethandler.NewMockOutdialTargetHandler(mc)

	h, ok := NewListenHandler(mockSock, mockOutdial, mockTarget).(*listenHandler)
	if !ok {
		t.Fatalf("expected a *listenHandler")
	}

	if h.sockHandler != sockhandler.SockHandler(mockSock) {
		t.Errorf("the sock handler was not stored")
	}
	if h.outdialHandler != outdialhandler.OutdialHandler(mockOutdial) {
		t.Errorf("the outdial handler was not stored")
	}
	if h.outdialTargetHandler != outdialtargethandler.OutdialTargetHandler(mockTarget) {
		t.Errorf("the outdial target handler was not stored")
	}
}

func Test_Run_queueCreateFails(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockSock := sockhandler.NewMockSockHandler(mc)
	h := &listenHandler{sockHandler: mockSock}

	// ConsumeRPC has no expectation: a call fails the test
	mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(errRunInjected)

	err := h.Run("test-queue", "test-delay")
	if !errors.Is(err, errRunInjected) {
		t.Errorf("expected the queue failure in the error chain, got: %v", err)
	}
	if err != nil && !strings.Contains(err.Error(), "could not declare the queue") {
		t.Errorf("expected the declare queue message, got: %v", err)
	}
}

func Test_Run_consume(t *testing.T) {
	tests := []struct {
		name       string
		consumeErr error
	}{
		{"consume starts", nil},
		{"consume fails and is logged", errRunInjected},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			// the hook sees the error log of the consume goroutine; the original hooks come back at the end
			hook := &logHook{ch: make(chan string, 16)}
			origHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
			logrus.AddHook(hook)
			t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(origHooks) })

			mockSock := sockhandler.NewMockSockHandler(mc)
			h := &listenHandler{sockHandler: mockSock}

			called := make(chan consumeArgs, 1)
			mockSock.EXPECT().QueueCreate("test-queue", "normal").Return(nil)
			mockSock.EXPECT().ConsumeRPC(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				DoAndReturn(func(_ any, queue, consumer string, exclusive, noLocal, noWait bool, workerNum int, cb sock.CbMsgRPC) error {
					called <- consumeArgs{queue, consumer, exclusive, noLocal, noWait, workerNum, cb != nil}
					return tt.consumeErr
				})

			if err := h.Run("test-queue", "test-delay"); err != nil {
				t.Fatalf("expected no error, got: %v", err)
			}

			// the consume runs in a goroutine; the arguments are asserted here, not inside the mock
			select {
			case got := <-called:
				expect := consumeArgs{"test-queue", "outdial-manager", false, false, false, 10, true}
				if got != expect {
					t.Errorf("wrong ConsumeRPC arguments. expect: %+v, got: %+v", expect, got)
				}
			case <-time.After(runWaitTimeout):
				t.Fatalf("ConsumeRPC was not called")
			}

			if tt.consumeErr == nil {
				return
			}

			deadline := time.After(runWaitTimeout)
			for {
				select {
				case msg := <-hook.ch:
					if strings.Contains(msg, "Could not consume the request message") {
						return
					}
				case <-deadline:
					t.Fatalf("the consume failure was not logged")
				}
			}
		})
	}
}
