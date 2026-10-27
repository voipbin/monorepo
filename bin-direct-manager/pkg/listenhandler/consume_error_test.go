package listenhandler

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	cerrors "monorepo/bin-common-handler/models/errors"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-common-handler/pkg/sockhandler"

	"github.com/sirupsen/logrus"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-direct-manager/pkg/directhandler"
)

// entryHook forwards the log entries to a channel without ever blocking the logger.
type entryHook struct {
	entries chan *logrus.Entry
}

func (h *entryHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *entryHook) Fire(e *logrus.Entry) error {
	select {
	case h.entries <- e:
	default:
	}
	return nil
}

func Test_Run_consumeError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	hook := &entryHook{entries: make(chan *logrus.Entry, 256)}
	logrus.AddHook(hook)
	t.Cleanup(func() {
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	})

	mockSock := sockhandler.NewMockSockHandler(mc)
	mockSock.EXPECT().QueueCreate(testListenQueue, "normal").Return(nil)
	mockSock.EXPECT().ConsumeRPC(gomock.Any(), testListenQueue, "direct-manager", false, false, false, 10, gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string, _ string, _ bool, _ bool, _ bool, _ int, _ sock.CbMsgRPC) error {
			return errHandlerTest
		}).Times(1)

	h := NewListenHandler(mockSock, directhandler.NewMockDirectHandler(mc))

	// a consumer failure happens in the background goroutine and is only logged
	if err := h.Run(testListenQueue, "delay"); err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	timeout := time.After(5 * time.Second)
	for {
		select {
		case e := <-hook.entries:
			if e.Level != logrus.ErrorLevel || !strings.Contains(e.Message, "Could not consume the request message correctly") {
				continue
			}
			if !strings.Contains(e.Message, errHandlerTest.Error()) {
				t.Errorf("log message = %q, want it to carry the consumer error %q", e.Message, errHandlerTest.Error())
			}
			return
		case <-timeout:
			t.Fatalf("the consumer failure was not logged within the time limit")
		}
	}
}

func Test_errorResponse_unmarshalableDetails(t *testing.T) {
	typed := cerrors.NotFound(commonoutline.ServiceNameDirectManager, "DIRECT_NOT_FOUND", "x")
	typed.Details = []map[string]any{{"value": make(chan int)}}

	resp := errorResponse(typed)
	if resp == nil {
		t.Fatalf("errorResponse() = nil, want a response")
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("errorResponse() status = %d, want %d", resp.StatusCode, http.StatusInternalServerError)
	}
	if len(resp.Data) != 0 {
		t.Errorf("errorResponse() body = %q, want empty", resp.Data)
	}
}
