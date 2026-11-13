package logtest

import (
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
)

const sentence = "SENTINEL-SELF-TEST"

// failed runs the assertion on a recording T and reports whether it failed.
func failed(f func(t *testing.T)) (res bool) {
	rec := &testing.T{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = recover() }()
		f(rec)
	}()
	<-done
	return rec.Failed()
}

func Test_AssertAbsent_detects(t *testing.T) {
	tests := []struct {
		name string
		emit func()
	}{
		{"message", func() { logrus.Info("said " + sentence) }},
		{"field", func() { logrus.WithField("a", sentence).Info("x") }},
		{"bytes field", func() { logrus.WithField("a", []byte(sentence)).Info("x") }},
		{"formatted struct field", func() { logrus.WithField("a", fmt.Sprintf("%+v", struct{ S string }{sentence})).Info("x") }},
		{"nested bytes", func() { logrus.WithField("a", map[string][]byte{"k": []byte(sentence)}).Info("x") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(t)
			tt.emit()
			if !failed(func(rec *testing.T) { c.AssertAbsent(rec, sentence) }) {
				t.Errorf("the text written by %s must be detected", tt.name)
			}
		})
	}
}

func Test_AssertAbsent_passes(t *testing.T) {
	c := New(t)
	logrus.WithField("message_len", 10).Info("clean")
	if failed(func(rec *testing.T) { c.AssertAbsent(rec, sentence) }) {
		t.Errorf("a clean capture must pass")
	}
}

func Test_AssertAbsent_emptyCaptureFails(t *testing.T) {
	c := New(t)
	if !failed(func(rec *testing.T) { c.AssertAbsent(rec, sentence) }) {
		t.Errorf("an empty capture must fail")
	}
}

func Test_AssertField(t *testing.T) {
	c := New(t)
	logrus.WithField("message_len", 10).Info("clean")
	if failed(func(rec *testing.T) { c.AssertField(rec, "message_len") }) {
		t.Errorf("an existing field must pass")
	}
	if !failed(func(rec *testing.T) { c.AssertField(rec, "other") }) {
		t.Errorf("a missing field must fail")
	}
}
