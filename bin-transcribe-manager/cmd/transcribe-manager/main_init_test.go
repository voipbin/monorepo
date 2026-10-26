package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gofrs/uuid"
	"github.com/sirupsen/logrus"

	"monorepo/bin-transcribe-manager/internal/config"
	"monorepo/bin-transcribe-manager/models/transcribe"
)

// promEndpointSeq makes every initProm call register a unique path, because http.Handle on the
// process-wide default mux panics on a duplicate pattern (for example under -count=N).
var promEndpointSeq atomic.Int64

// saveConfig restores the whole process-wide config once the test is over.
func saveConfig(t *testing.T) {
	t.Helper()

	saved := *config.Get()
	t.Cleanup(func() { *config.Get() = saved })
}

// closedAddr returns a loopback address that nothing listens on.
func closedAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	addr := l.Addr().String()
	_ = l.Close()

	return addr
}

// hookFunc is a logrus hook that forwards the message of every entry that matches.
type hookFunc struct {
	match string
	ch    chan struct{}
}

func (h *hookFunc) Levels() []logrus.Level { return logrus.AllLevels }

func (h *hookFunc) Fire(e *logrus.Entry) error {
	if strings.Contains(e.Message, h.match) {
		select {
		case h.ch <- struct{}{}:
		default:
		}
	}
	return nil
}

func Test_initCache(t *testing.T) {
	saveConfig(t)

	t.Run("connects to a reachable server", func(t *testing.T) {
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("could not start miniredis. err: %v", err)
		}
		t.Cleanup(mr.Close)

		config.Get().RedisAddress = mr.Addr()
		config.Get().RedisDatabase = 2

		res, err := initCache()
		if err != nil {
			t.Fatalf("initCache() returned an error. err: %v", err)
		}
		if res == nil {
			t.Fatalf("initCache() returned a nil handler")
		}

		// a write through the handler must land in the configured logical database.
		tr := &transcribe.Transcribe{}
		tr.ID = uuid.FromStringOrNil("8c0e5a40-0000-11ee-8c10-000000000001")
		if errSet := res.TranscribeSet(context.Background(), tr); errSet != nil {
			t.Fatalf("TranscribeSet() returned an error. err: %v", errSet)
		}
		if got := len(mr.DB(2).Keys()); got != 1 {
			t.Errorf("expected one key in redis database 2. got: %d", got)
		}
		if got := len(mr.DB(0).Keys()); got != 0 {
			t.Errorf("expected no key in redis database 0. got: %d", got)
		}
	})

	t.Run("wraps the connect error", func(t *testing.T) {
		config.Get().RedisAddress = closedAddr(t)

		res, err := initCache()
		if err == nil {
			t.Fatalf("initCache() expected an error")
		}
		if res != nil {
			t.Errorf("initCache() expected a nil handler on error. got: %v", res)
		}
		if !strings.Contains(err.Error(), "cache connect error") {
			t.Errorf("initCache() error was not wrapped. err: %v", err)
		}
	})
}

func Test_initSignal(t *testing.T) {
	t.Cleanup(func() {
		signal.Stop(chSigs)
		select {
		case <-chDone:
		default:
		}
	})

	initSignal()

	// no real OS signal is sent; the value goes straight to the channel the goroutine reads.
	chSigs <- os.Interrupt

	select {
	case <-chDone:
	case <-time.After(5 * time.Second):
		t.Fatalf("initSignal() did not report done after a signal")
	}
}

func Test_initProm_disabled(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		listen   string
	}{
		{"empty endpoint", "", "127.0.0.1:0"},
		{"empty listen address", "/disabled", ""},
		{"both empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// a disabled configuration must not register a handler: registering the same path
			// twice would panic, so calling it twice proves nothing was registered.
			initProm(tt.endpoint, tt.listen)
			initProm(tt.endpoint, tt.listen)
		})
	}
}

func Test_initProm_listenError(t *testing.T) {
	busy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("could not listen. err: %v", err)
	}
	t.Cleanup(func() { _ = busy.Close() })

	h := &hookFunc{match: "Prometheus server error", ch: make(chan struct{}, 1)}
	orig := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	logrus.AddHook(h)
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(orig) })

	endpoint := fmt.Sprintf("/metrics-test-%d", promEndpointSeq.Add(1))
	initProm(endpoint, busy.Addr().String())

	select {
	case <-h.ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("initProm() did not report the listen error for an address in use")
	}
}
