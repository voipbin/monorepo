package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// initPromEndpointSeq gives every initProm call its own path, because initProm registers on
// http.DefaultServeMux, which panics on a duplicate pattern.
var initPromEndpointSeq atomic.Int64

// errorHook sends the message of every error log entry to a buffered channel without blocking,
// because the listener retry goroutine keeps logging until the process ends.
type errorHook struct {
	messages chan string
}

func (h *errorHook) Levels() []logrus.Level {
	return []logrus.Level{logrus.ErrorLevel}
}

func (h *errorHook) Fire(e *logrus.Entry) error {
	select {
	case h.messages <- e.Message:
	default:
	}
	return nil
}

func Test_main_help(t *testing.T) {
	savedArgs := os.Args
	savedStdout := os.Stdout
	t.Cleanup(func() {
		os.Args = savedArgs
		os.Stdout = savedStdout
	})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create the pipe: %v", err)
	}
	outCh := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outCh <- buf.String()
	}()

	os.Args = []string{"tag-manager", "--help"}
	os.Stdout = w
	main()
	os.Stdout = savedStdout
	_ = w.Close()
	out := <-outCh

	for _, expect := range []string{"Usage:", "tag-manager", "Tag Manager is a microservice"} {
		if !strings.Contains(out, expect) {
			t.Errorf("The help output does not contain %q. got: %s", expect, out)
		}
	}
}

func Test_initProm_listenFails(t *testing.T) {
	hook := &errorHook{messages: make(chan string, 8)}
	savedHooks := logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	logrus.AddHook(hook)
	t.Cleanup(func() { logrus.StandardLogger().ReplaceHooks(savedHooks) })

	endpoint := fmt.Sprintf("/m-listen-fails-%d", initPromEndpointSeq.Add(1))

	// a listen address with an invalid port never binds, so the goroutine logs and retries
	initProm(endpoint, "127.0.0.1:99999")

	// two entries prove the retry: without the continue the loop ends after the first failure
	deadline := time.After(10 * time.Second)
	for i := 0; i < 2; i++ {
		select {
		case msg := <-hook.messages:
			if !strings.Contains(msg, "Could not start prometheus listener") {
				t.Errorf("Wrong log message. got: %s", msg)
			}
		case <-deadline:
			t.Fatalf("Expected 2 listener failure logs, got %d", i)
		}
	}
}
