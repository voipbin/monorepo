package main

import (
	"encoding/json"
	"flag"
	"os"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

// These tests share process-wide state (logrus hooks, os.Args, flag.CommandLine)
// and must not call t.Parallel().

const (
	testQueue     = "test-queue"
	testURI       = "/v1/numbers/renew"
	testDataType  = "text/plain"
	testData      = `{"days":28}`
	wantFlagsMask = propContentType | propCorrelationID | propReplyTo
)

func Test_run_success_and_status_boundaries(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		wantErr string
	}{
		{"200 ok", 200, ""},
		{"299 ok upper bound", 299, ""},
		{"199 below range", 199, "non-2xx response: 199"},
		{"300 redirect", 300, "non-2xx response: 300"},
		{"404 client error", 404, "non-2xx response: 404"},
		{"500 server error", 500, "non-2xx response: 500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := startFakeBroker(t, brokerConfig{status: tt.status})

			err := run(b.addr(), testQueue, testURI, "POST", testDataType, testData, 5000)
			switch {
			case tt.wantErr == "" && err != nil:
				t.Fatalf("Wrong match. expect: nil, got: %v", err)
			case tt.wantErr != "" && (err == nil || err.Error() != tt.wantErr):
				t.Fatalf("Wrong match. expect: %q, got: %v", tt.wantErr, err)
			}

			assertWireRequest(t, b)
		})
	}
}

// assertWireRequest checks what the broker actually received from the client.
func assertWireRequest(t *testing.T, b *fakeBroker) {
	t.Helper()

	b.mu.Lock()
	declares := append([]declareRecord(nil), b.declares...)
	consumes := append([]consumeRecord(nil), b.consumes...)
	b.mu.Unlock()

	if len(declares) != 1 {
		t.Fatalf("Wrong match. expect: 1 declare, got: %d", len(declares))
	}
	// bits: passive=0x01 durable=0x02 exclusive=0x04 auto-delete=0x08 no-wait=0x10
	if declares[0].queue != "" || declares[0].bits != 0x04|0x08 {
		t.Errorf("Wrong match. expect: anonymous exclusive auto-delete queue, got: %+v", declares[0])
	}

	if len(consumes) != 1 {
		t.Fatalf("Wrong match. expect: 1 consume, got: %d", len(consumes))
	}
	// bits: no-local=0x01 no-ack=0x02 exclusive=0x04 no-wait=0x08
	if consumes[0].queue != fakeGeneratedQueue || consumes[0].bits != 0x02 || consumes[0].tag == "" {
		t.Errorf("Wrong match. expect: auto-ack consume of %q, got: %+v", fakeGeneratedQueue, consumes[0])
	}

	pubs := b.waitPublished(1)
	if len(pubs) != 1 {
		t.Fatalf("Wrong match. expect: 1 publish, got: %d", len(pubs))
	}
	p := pubs[0]
	if p.exchange != "" {
		t.Errorf("Wrong match. expect: default exchange, got: %q", p.exchange)
	}
	if p.routingKey != testQueue {
		t.Errorf("Wrong match. expect: %q, got: %q", testQueue, p.routingKey)
	}
	if p.flags != wantFlagsMask {
		t.Errorf("Wrong match. expect: flags 0x%x, got: 0x%x", wantFlagsMask, p.flags)
	}
	if p.contentType != "application/json" {
		t.Errorf("Wrong match. expect: application/json, got: %q", p.contentType)
	}
	if p.corrID != "bin-trigger-sender-cronjob" {
		t.Errorf("Wrong match. expect: bin-trigger-sender-cronjob, got: %q", p.corrID)
	}
	if p.replyTo != fakeGeneratedQueue {
		t.Errorf("Wrong match. expect: %q, got: %q", fakeGeneratedQueue, p.replyTo)
	}

	var got map[string]json.RawMessage
	if err := json.Unmarshal(p.body, &got); err != nil {
		t.Fatalf("published body is not JSON: %v", err)
	}
	want := map[string]string{
		"uri":        `"/v1/numbers/renew"`,
		"method":     `"POST"`,
		"publisher":  `"bin-trigger-sender"`,
		"data_type":  `"text/plain"`,
		"request_id": `"bin-trigger-sender-cronjob"`,
		"data":       testData,
	}
	for k, v := range want {
		if string(got[k]) != v {
			t.Errorf("Wrong match. field %s expect: %s, got: %s", k, v, got[k])
		}
	}
	if len(got) != len(want) {
		t.Errorf("Wrong match. expect: %d fields, got: %d", len(want), len(got))
	}
}

func Test_run_reply_errors(t *testing.T) {
	tests := []struct {
		name    string
		cfg     brokerConfig
		wantErr string
	}{
		{"other correlation id", brokerConfig{corrID: "other"}, `unexpected CorrelationId on reply: "other"`},
		{"reply is not json", brokerConfig{rawBody: []byte("not json")}, "unmarshal response: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := startFakeBroker(t, tt.cfg)

			err := run(b.addr(), testQueue, testURI, "POST", "application/json", "", 5000)
			if err == nil || !strings.HasPrefix(err.Error(), tt.wantErr) {
				t.Fatalf("Wrong match. expect prefix: %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func Test_run_timeout_without_reply(t *testing.T) {
	b := startFakeBroker(t, brokerConfig{noReply: true})

	err := run(b.addr(), testQueue, testURI, "POST", "application/json", "", 200)
	if err == nil || err.Error() != "timeout waiting for response after 200 ms" {
		t.Fatalf("Wrong match. expect: timeout error, got: %v", err)
	}
	if got := len(b.waitPublished(1)); got != 1 {
		t.Errorf("Wrong match. expect: 1 publish before the timeout, got: %d", got)
	}
}

func Test_run_publish_error_when_context_already_done(t *testing.T) {
	b := startFakeBroker(t, brokerConfig{})

	err := run(b.addr(), testQueue, testURI, "POST", "application/json", "", 0)
	if err == nil || err.Error() != "publish: context deadline exceeded" {
		t.Fatalf("Wrong match. expect: publish deadline error, got: %v", err)
	}
	if got := len(b.published()); got != 0 {
		t.Errorf("Wrong match. expect: no publish reached the broker, got: %d", got)
	}
}

func Test_run_marshal_error_on_invalid_data(t *testing.T) {
	b := startFakeBroker(t, brokerConfig{})

	err := run(b.addr(), testQueue, testURI, "POST", "application/json", "{bad", 5000)
	if err == nil || !strings.HasPrefix(err.Error(), "marshal request: ") {
		t.Fatalf("Wrong match. expect prefix: marshal request, got: %v", err)
	}
	if got := len(b.published()); got != 0 {
		t.Errorf("Wrong match. expect: nothing published, got: %d", got)
	}
}

func Test_run_broker_rejections(t *testing.T) {
	tests := []struct {
		name    string
		cfg     brokerConfig
		wantErr string
	}{
		{"open channel rejected", brokerConfig{rejectOpen: true}, "open channel: "},
		{"declare rejected", brokerConfig{rejectDeclare: true}, "declare reply queue: "},
		{"consume rejected", brokerConfig{rejectConsume: true}, "consume reply queue: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := startFakeBroker(t, tt.cfg)

			err := run(b.addr(), testQueue, testURI, "POST", "application/json", "", 5000)
			if err == nil || !strings.HasPrefix(err.Error(), tt.wantErr) {
				t.Fatalf("Wrong match. expect prefix: %q, got: %v", tt.wantErr, err)
			}
			if got := len(b.published()); got != 0 {
				t.Errorf("Wrong match. expect: nothing published, got: %d", got)
			}
		})
	}
}

func Test_run_close_failures_are_logged_not_returned(t *testing.T) {
	hook := logtest.NewGlobal()
	t.Cleanup(func() { log.StandardLogger().ReplaceHooks(make(log.LevelHooks)) })

	b := startFakeBroker(t, brokerConfig{dropOnClose: true})

	if err := run(b.addr(), testQueue, testURI, "POST", "application/json", "", 5000); err != nil {
		t.Fatalf("Wrong match. expect: nil, got: %v", err)
	}

	var warns []string
	for _, e := range hook.AllEntries() {
		if e.Level == log.WarnLevel {
			warns = append(warns, e.Message)
		}
	}
	for _, prefix := range []string{"close channel", "close connection"} {
		found := false
		for _, m := range warns {
			found = found || strings.HasPrefix(m, prefix)
		}
		if !found {
			t.Errorf("Wrong match. expect: warning with prefix %q, got: %v", prefix, warns)
		}
	}
}

// runMain executes main() with the given arguments, restoring the globals afterwards.
func runMain(t *testing.T, args ...string) {
	t.Helper()
	oldArgs, oldFlags := os.Args, flag.CommandLine
	t.Cleanup(func() {
		os.Args, flag.CommandLine = oldArgs, oldFlags
	})
	os.Args = append([]string{"bin-trigger-sender"}, args...)
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.PanicOnError)

	main()
}

func Test_main_success_forwards_flags(t *testing.T) {
	tests := []struct {
		name     string
		delayMs  string
		minDelay time.Duration
	}{
		{"no delay", "0", 0},
		{"with delay", "50", 50 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := startFakeBroker(t, brokerConfig{})
			addr := b.addr()

			start := time.Now()
			runMain(t,
				"-rabbit_addr", addr,
				"-queue", testQueue,
				"-uri", testURI,
				"-method", "PUT",
				"-data_type", testDataType,
				"-data", testData,
				"-timeout", "5000",
				"-delay", tt.delayMs,
			)
			if elapsed := time.Since(start); elapsed < tt.minDelay {
				t.Errorf("Wrong match. expect: at least %v, got: %v", tt.minDelay, elapsed)
			}

			pubs := b.waitPublished(1)
			var req request
			if err := json.Unmarshal(pubs[0].body, &req); err != nil {
				t.Fatalf("published body is not JSON: %v", err)
			}
			if pubs[0].routingKey != testQueue || req.URI != testURI || req.Method != "PUT" ||
				req.DataType != testDataType || string(req.Data) != testData {
				t.Errorf("Wrong match. flags were not forwarded, routing key %q, request %+v", pubs[0].routingKey, req)
			}
		})
	}
}

func Test_main_flag_defaults(t *testing.T) {
	b := startFakeBroker(t, brokerConfig{})

	runMain(t, "-rabbit_addr", b.addr(), "-queue", testQueue, "-uri", testURI)

	pubs := b.waitPublished(1)
	var req request
	if err := json.Unmarshal(pubs[0].body, &req); err != nil {
		t.Fatalf("published body is not JSON: %v", err)
	}
	if req.Method != "POST" || req.DataType != "application/json" || req.Data != nil {
		t.Errorf("Wrong match. expect: POST, application/json and no data, got: %+v", req)
	}
}
