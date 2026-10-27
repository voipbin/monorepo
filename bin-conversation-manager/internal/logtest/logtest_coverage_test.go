package logtest

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/sirupsen/logrus"

	"monorepo/bin-common-handler/models/sock"
	"monorepo/bin-conversation-manager/models/account"
)

// The tests of this package use the global logger and must not run in parallel.

func Test_Capture(t *testing.T) {
	std := logrus.StandardLogger()

	prevLevel := std.GetLevel()
	prevOut := std.Out
	prevHooks := std.ReplaceHooks(make(logrus.LevelHooks))
	std.SetLevel(logrus.WarnLevel)
	var sink bytes.Buffer
	std.SetOutput(&sink)
	t.Cleanup(func() {
		std.ReplaceHooks(prevHooks)
		std.SetLevel(prevLevel)
		std.SetOutput(prevOut)
	})

	t.Run("records debug entries and discards the output", func(t *testing.T) {
		hook := Capture(t)

		if std.GetLevel() != logrus.DebugLevel {
			t.Errorf("level = %v, want %v", std.GetLevel(), logrus.DebugLevel)
		}

		logrus.Debug("a debug entry")
		logrus.WithField("key", "value").Info("an info entry")

		entries := hook.AllEntries()
		if len(entries) != 2 {
			t.Fatalf("recorded entries = %d, want 2", len(entries))
		}
		if entries[0].Message != "a debug entry" || entries[1].Data["key"] != "value" {
			t.Errorf("unexpected entries: %q, %v", entries[0].Message, entries[1].Data)
		}
		if sink.Len() != 0 {
			t.Errorf("the output was not discarded: %q", sink.String())
		}
	})

	// the subtest ended, so its cleanup has restored the logger
	if std.GetLevel() != logrus.WarnLevel {
		t.Errorf("level after the cleanup = %v, want %v", std.GetLevel(), logrus.WarnLevel)
	}
	if std.Out != &sink {
		t.Errorf("the output was not restored")
	}
	logrus.Warn("after the cleanup")
	if sink.Len() == 0 {
		t.Errorf("the restored output did not receive the entry")
	}
	if len(std.Hooks) != 0 {
		t.Errorf("the hooks of the capture remained after the cleanup: %v", std.Hooks)
	}
}

func Test_variants(t *testing.T) {
	tests := []struct {
		name     string
		sentinel string
		expect   []string
	}{
		{
			name:     "plain text",
			sentinel: "abc",
			expect:   []string{"abc", "YWJj", "[97 98 99]", "abc"},
		},
		{
			name:     "characters that change in the url form",
			sentinel: "a b&c",
			expect:   []string{"a b&c", "YSBiJmM=", "[97 32 98 38 99]", "a+b%26c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := variants(tt.sentinel)
			if !reflect.DeepEqual(got, tt.expect) {
				t.Errorf("variants() = %q, want %q", got, tt.expect)
			}
		})
	}
}

func Test_containsSentinel(t *testing.T) {
	tests := []struct {
		name       string
		renderings []string
		sentinels  []string
		expect     bool
	}{
		{name: "plain match", renderings: []string{"x", "has secret-1 inside"}, sentinels: []string{"secret-1"}, expect: true},
		{name: "base64 match", renderings: []string{"c2VjcmV0LTE="}, sentinels: []string{"secret-1"}, expect: true},
		{name: "decimal list match", renderings: []string{"[115 101 99 114 101 116 45 49]"}, sentinels: []string{"secret-1"}, expect: true},
		{name: "second sentinel matches", renderings: []string{"only other-2 here"}, sentinels: []string{"secret-1", "other-2"}, expect: true},
		{name: "no match", renderings: []string{"nothing to see"}, sentinels: []string{"secret-1"}, expect: false},
		{name: "no renderings", renderings: nil, sentinels: []string{"secret-1"}, expect: false},
		{name: "no sentinels", renderings: []string{"secret-1"}, sentinels: nil, expect: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsSentinel(tt.renderings, tt.sentinels); got != tt.expect {
				t.Errorf("containsSentinel() = %v, want %v", got, tt.expect)
			}
		})
	}
}

func Test_AssertNoLeak(t *testing.T) {
	const sentinel = "sentinel-value-1"

	tests := []struct {
		name      string
		emit      func()
		expectBad bool
	}{
		{
			name:      "no entry is rejected as vacuous",
			emit:      func() {},
			expectBad: true,
		},
		{
			name:      "clean entries pass",
			emit:      func() { logrus.WithField("count", 3).Info("nothing secret here") },
			expectBad: false,
		},
		{
			name:      "the sentinel in the message is detected",
			emit:      func() { logrus.Infof("token is %s", sentinel) },
			expectBad: true,
		},
		{
			name:      "the sentinel in a string field is detected",
			emit:      func() { logrus.WithField("token", sentinel).Info("a message") },
			expectBad: true,
		},
		{
			name:      "the sentinel in a byte field is detected in its encoded forms",
			emit:      func() { logrus.WithField("raw", []byte(sentinel)).Info("a message") },
			expectBad: true,
		},
		{
			name:      "the url escaped sentinel is detected",
			emit:      func() { logrus.Info("redirect?token=sentinel-value-1&x=1") },
			expectBad: true,
		},
		{
			name:      "an account value is rejected whatever its content",
			emit:      func() { logrus.WithField("account", &account.Account{}).Info("a message") },
			expectBad: true,
		},
		{
			name:      "a request value is rejected whatever its content",
			emit:      func() { logrus.WithField("request", &sock.Request{}).Info("a message") },
			expectBad: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hook := Capture(t)
			tt.emit()

			// a zero value T records the failure without failing the real test
			fake := &testing.T{}
			AssertNoLeak(fake, hook, sentinel)

			if fake.Failed() != tt.expectBad {
				t.Errorf("AssertNoLeak() failed = %v, want %v", fake.Failed(), tt.expectBad)
			}
		})
	}
}
