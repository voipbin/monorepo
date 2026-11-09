package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"monorepo/bin-api-manager/models/common"
	"monorepo/bin-api-manager/pkg/servicehandler"

	"github.com/gin-gonic/gin"
	joonix "github.com/joonix/log"
	"github.com/sirupsen/logrus"
	"go.uber.org/mock/gomock"
)

// logSentinelA stands in for the value AuthLogin returns. It must never reach
// a log entry. The name and value are deliberately neutral.
const logSentinelA = "SENTINEL-X-MUST-NOT-LEAK-7c1e9a40"

// logCaptureHook records every entry of the global logrus logger together with
// the text the production formatter renders for it.
type logCaptureHook struct {
	mu        sync.Mutex
	formatter logrus.Formatter
	entries   []capturedLogEntry
}

type capturedLogEntry struct {
	message   string
	data      string
	dataJSON  string
	formatted string
	formatErr error
}

func (h *logCaptureHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *logCaptureHook) Fire(e *logrus.Entry) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	out, ferr := h.formatter.Format(e)
	c := capturedLogEntry{
		message:   e.Message,
		data:      fmt.Sprintf("%v", e.Data),
		formatted: string(out),
		formatErr: ferr,
	}
	// json.Marshal can fail on func-valued fields; the %v form is the primary check.
	if b, err := json.Marshal(e.Data); err == nil {
		c.dataJSON = string(b)
	}
	h.entries = append(h.entries, c)
	return nil
}

// captureGlobalLog switches the global logger to the production formatter at
// Debug level, restoring level, hooks, formatter and output on cleanup.
func captureGlobalLog(t *testing.T) *logCaptureHook {
	t.Helper()

	std := logrus.StandardLogger()
	prevLevel := logrus.GetLevel()
	prevFormatter := std.Formatter
	prevOut := std.Out
	prevHooks := std.ReplaceHooks(make(logrus.LevelHooks))

	h := &logCaptureHook{formatter: joonix.NewFormatter()}
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(h.formatter)
	logrus.SetOutput(io.Discard)
	std.AddHook(h)

	t.Cleanup(func() {
		std.ReplaceHooks(prevHooks)
		logrus.SetLevel(prevLevel)
		logrus.SetFormatter(prevFormatter)
		logrus.SetOutput(prevOut)
	})

	return h
}

func assertSentinelAbsent(t *testing.T, entries []capturedLogEntry, sentinel string) {
	t.Helper()

	byteForm := fmt.Sprintf("%v", []byte(sentinel))
	for i, e := range entries {
		if e.formatErr != nil {
			t.Fatalf("entry %d: the formatter returned an error: %v", i, e.formatErr)
		}
		if e.formatted == "" {
			t.Fatalf("entry %d: the formatter returned empty output, the check would be vacuous", i)
		}
		if !strings.Contains(e.formatted, `"username"`) {
			t.Fatalf("entry %d: the formatted output lacks the username field: %s", i, e.formatted)
		}

		for name, got := range map[string]string{
			"message":   e.message,
			"data":      e.data,
			"data_json": e.dataJSON,
			"formatted": e.formatted,
		} {
			if strings.Contains(got, sentinel) {
				t.Errorf("entry %d: %s contains the sentinel in plaintext", i, name)
			}
			if strings.Contains(got, byteForm) {
				t.Errorf("entry %d: %s contains the sentinel as a byte array", i, name)
			}
		}
	}
}

func countMessage(entries []capturedLogEntry, msg string) int {
	n := 0
	for _, e := range entries {
		if e.message == msg {
			n++
		}
	}
	return n
}

func Test_PostLogin_LogDoesNotContainToken(t *testing.T) {
	tests := []struct {
		name string

		authErr error

		expectCode       int
		expectLogged     string
		expectNotLogged  string
		expectTokenInRes bool
	}{
		{
			name: "success",

			expectCode:       http.StatusOK,
			expectLogged:     "User successfully logged in.",
			expectTokenInRes: true,
		},
		{
			name: "auth failure",

			authErr:         errors.New("auth failed"),
			expectCode:      http.StatusBadRequest,
			expectLogged:    "Login failed. err: auth failed",
			expectNotLogged: "User successfully logged in.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			hook := captureGlobalLog(t)

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockSvc := servicehandler.NewMockServiceHandler(mc)

			w := httptest.NewRecorder()
			_, r := gin.CreateTestContext(w)
			r.Use(func(c *gin.Context) {
				c.Set(common.OBJServiceHandler, mockSvc)
			})
			setupServer(r)

			reqBody := RequestBodyLoginPOST{Username: "test@test.com", Password: "pw"}
			body, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest("POST", "/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// the mock returns the sentinel in both cases, so a leak on the
			// error path would be caught as well
			mockSvc.EXPECT().AuthLogin(req.Context(), reqBody.Username, reqBody.Password).Return(logSentinelA, tt.authErr)

			r.ServeHTTP(w, req)
			if w.Code != tt.expectCode {
				t.Fatalf("Wrong match. expect: %d, got: %d", tt.expectCode, w.Code)
			}

			// the token is still returned to the caller; only the log lost it
			if tt.expectTokenInRes {
				if !strings.Contains(w.Body.String(), logSentinelA) {
					t.Errorf("Wrong match. the response body must carry the token")
				}
				if !strings.Contains(w.Header().Get("Set-Cookie"), logSentinelA) {
					t.Errorf("Wrong match. the cookie must carry the token")
				}
			}

			hook.mu.Lock()
			entries := append([]capturedLogEntry(nil), hook.entries...)
			hook.mu.Unlock()

			// positive control: the logger really captured this request
			if got := countMessage(entries, tt.expectLogged); got != 1 {
				t.Fatalf("Wrong match. expect exactly 1 entry %q, got: %d (entries: %d)", tt.expectLogged, got, len(entries))
			}
			if tt.expectNotLogged != "" && countMessage(entries, tt.expectNotLogged) != 0 {
				t.Errorf("Wrong match. unexpected entry: %q", tt.expectNotLogged)
			}

			assertSentinelAbsent(t, entries, logSentinelA)
		})
	}
}
