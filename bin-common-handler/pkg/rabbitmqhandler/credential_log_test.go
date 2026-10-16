package rabbitmqhandler

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/sirupsen/logrus"
	logrustest "github.com/sirupsen/logrus/hooks/test"
)

const (
	testSecretUser = "secretuser"
	testSecretPass = "secretpass"
)

// testCapturedLogs returns everything the hook captured (message and fields) as one string.
func testCapturedLogs(hook *logrustest.Hook) string {
	var sb strings.Builder
	for _, e := range hook.AllEntries() {
		sb.WriteString(e.Message)
		sb.WriteString(" ")
		for k, v := range e.Data {
			fmt.Fprintf(&sb, "%s=%v ", k, v)
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// testHookLogger captures the standard logger for the test and restores it afterwards.
func testHookLogger(t *testing.T) *logrustest.Hook {
	oldLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	hook := logrustest.NewGlobal()
	t.Cleanup(func() {
		logrus.SetLevel(oldLevel)
		logrus.StandardLogger().ReplaceHooks(make(logrus.LevelHooks))
	})
	return hook
}

func Test_Close_doesNotLogCredentials(t *testing.T) {
	hook := testHookLogger(t)

	uri := fmt.Sprintf("amqp://%s:%s@broker.internal:5672/", testSecretUser, testSecretPass)
	r := &rabbit{
		uri:        uri,
		connection: newMockConnection(),
		queues:     make(map[string]*queue),
		exchanges:  make(map[string]*exchange),
		queueBinds: make(map[string][]*queueBind),
	}

	r.Close()

	logs := testCapturedLogs(hook)
	if logs == "" {
		t.Fatal("Expected Close() to log something")
	}
	for _, secret := range []string{testSecretUser, testSecretPass, "broker.internal"} {
		if strings.Contains(logs, secret) {
			t.Errorf("Close() logged %q. logs: %s", secret, logs)
		}
	}
}

func Test_connect_doesNotLogCredentials(t *testing.T) {
	hook := testHookLogger(t)

	uri := fmt.Sprintf("amqp://%s:%s@broker.internal:badport/", testSecretUser, testSecretPass)

	// The first dial fails with an error that echoes the URI, like a url.Parse error does.
	// The second dial blocks forever so the retry loop stays parked after the first failure.
	oldDial := amqpDial
	retried := make(chan struct{})
	calls := 0
	amqpDial = func(url string) (*amqp.Connection, error) {
		calls++
		if calls == 1 {
			return nil, fmt.Errorf("parse %q: invalid port \":badport\" after host", url)
		}
		close(retried)
		select {}
	}
	t.Cleanup(func() { amqpDial = oldDial })

	r := &rabbit{uri: uri}
	go r.connect()

	// wait for the retry (second dial) so the first failure was logged
	select {
	case <-retried:
	case <-time.After(5 * time.Second):
		t.Fatal("Expected connect() to retry after the first failure")
	}

	logs := testCapturedLogs(hook)
	if !strings.Contains(logs, "Could not connect to rabbitmq") {
		t.Fatalf("Expected the connect failure log. logs: %s", logs)
	}
	// the dial error text may carry the address; the credentials and a url field must not appear
	if strings.Contains(logs, "url=") {
		t.Errorf("connect() logs a url field. logs: %s", logs)
	}
	for _, secret := range []string{testSecretUser, testSecretPass} {
		if strings.Contains(logs, secret) {
			t.Errorf("connect() logged %q. logs: %s", secret, logs)
		}
	}
}

func Test_redactUserinfo(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect string
	}{
		{
			name:   "url parse error with userinfo",
			err:    fmt.Errorf(`parse "amqp://u:***@h:badport/": invalid port ":badport" after host`),
			expect: `parse "amqp://***@h:badport/": invalid port ":badport" after host`,
		},
		{
			name:   "userinfo with the raw password",
			err:    fmt.Errorf("bad uri amqps://user:pa%%40ss@host:5671/vhost"),
			expect: "bad uri amqps://***@host:5671/vhost",
		},
		{
			name:   "userinfo rejected by the parser keeps a space",
			err:    fmt.Errorf(`parse "amqp://u:pa ss@h/db": net/url: invalid userinfo`),
			expect: `parse "amqp://***@h/db": net/url: invalid userinfo`,
		},
		{
			name:   "password only userinfo",
			err:    fmt.Errorf(`parse "amqp://:pw@h:badport/": invalid port`),
			expect: `parse "amqp://***@h:badport/": invalid port`,
		},
		{
			name:   "userinfo with an escaped quote from a real url.Parse error",
			err:    testURLParseError("amqp://u:pa\"ss@h/db"),
			expect: `parse "amqp://***@h/db": net/url: invalid userinfo`,
		},
		{
			name:   "userinfo with a backslash",
			err:    testURLParseError("amqp://u:pa\\ss\x7f@h/db"),
			expect: `parse "amqp://***@h/db": net/url: invalid control character in URL`,
		},
		{
			name:   "no userinfo",
			err:    errors.New("dial tcp 127.0.0.1:1: connect: connection refused"),
			expect: "dial tcp 127.0.0.1:1: connect: connection refused",
		},
		{
			name:   "uri without userinfo",
			err:    errors.New(`parse "amqp://host:badport/": invalid port`),
			expect: `parse "amqp://host:badport/": invalid port`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := redactUserinfo(tt.err); got != tt.expect {
				t.Errorf("Wrong match.\nexpect: %s\ngot: %s", tt.expect, got)
			}
		})
	}
}

// testURLParseError returns the error of url.Parse for a malformed URL. The URL goes through a
// parameter so that static analysis does not flag the deliberately invalid literal.
func testURLParseError(raw string) error {
	_, err := url.Parse(raw)
	return err
}
