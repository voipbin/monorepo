package main

import (
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// Test_main_unknownInterface verifies that main returns without connecting to
// the message broker when the instance identity cannot be derived from the
// configured interface.
func Test_main_unknownInterface(t *testing.T) {
	hook := isolateLogger(t)

	savedInterface := interfaceName
	savedAddress := rabbitMQAddress
	interfaceName = "nonexistent999"
	// Empty so a regression cannot reach out to a default broker address.
	rabbitMQAddress = ""

	done := make(chan struct{})
	go func() {
		defer close(done)
		main()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		// main is still running, so leave the globals alone to avoid racing it.
		t.Fatalf("main did not return for an unknown interface")
	}

	interfaceName = savedInterface
	rabbitMQAddress = savedAddress

	var found bool
	for _, e := range hook.AllEntries() {
		if e.Level == logrus.ErrorLevel && e.Data["func"] == "main" &&
			strings.Contains(e.Message, "Could not get kamailio ID") &&
			strings.Contains(e.Message, "nonexistent999") {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected an error log naming the unknown interface, got %d entries", len(hook.AllEntries()))
	}
}
